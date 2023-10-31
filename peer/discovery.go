package peer

import (
	"context"
	"flag"
	"fmt"
	"liarslie/reader"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

var (
	topicNameFlag = flag.String("topicName", "liarslie", "name of topic to join")
)

// `RunAsExpert` runs the discovery process and updates network value for a host
func RunAsExpert(i int, agents []reader.ParticipantSet, numAgents int, computeValue bool) {
	ctx := context.Background()
	// create a new libp2p Host that listens on the allocated TCP port
	h, err := libp2p.New(libp2p.ListenAddrStrings(agents[i].IP))
	if err != nil {
		panic(err)
	}
	// discover peers in a separate thread
	go discoverPeers(ctx, h)

	// start gossipsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	// join the topic
	topic, err := ps.Join(*topicNameFlag)
	if err != nil {
		panic(err)
	}

	go publishTopic(ctx, topic, agents[i].IP)

	sub, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}

	if computeValue {
		computeNetworkValueExpert(h, ctx, sub, agents[i].IP, numAgents)
	}
}

// `RunAsStandard` runs updates network value for a host in Standard Mode
func RunAsStandard(i int, agents []reader.ParticipantSet, numAgents int) (value string) {
	value = computeNetworkValueStandard(i, agents, numAgents)
	return value
}

// `initDHT` starts a DHT, for use in peer discovery.
func initDHT(ctx context.Context, h host.Host) *dht.IpfsDHT {

	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		panic(err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerinfo); err != nil {
				fmt.Println("Bootstrap warning:", err)
			}
		}()
	}
	wg.Wait()

	return kademliaDHT
}

// `discoverPeers` initializes DHT and Look for others who have
// announced and attempt to connect to them
func discoverPeers(ctx context.Context, h host.Host) {
	kademliaDHT := initDHT(ctx, h)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, *topicNameFlag)
	anyConnected := false
	for !anyConnected {
		fmt.Println(h.ID().Pretty(), "is searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, *topicNameFlag)
		if err != nil {
			panic(err)
		}
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue
			}
			err := h.Connect(ctx, peer)
			if err != nil {
				continue
			} else {
				fmt.Println(h.ID().Pretty(), " is Connected to:", peer.ID.Pretty())
				anyConnected = true
			}
		}
		time.Sleep(10 * time.Second)
	}

	fmt.Println("Peer Discovery complete for host", h.ID().Pretty())
}

// `publishTopic` is used by the host to publish a message
// to the subscribed topic
func publishTopic(ctx context.Context, topic *pubsub.Topic, value string) {
	for {
		if err := topic.Publish(ctx, []byte(value)); err != nil {
			fmt.Println("### Publish error:", err)
		}
	}
}

// `computeNetworkValueExpert` computes network value for the agent
//  1. read current value from storage(there will always be some value stored at init)
//  2. if there is data and pub != sub then vote for the new message received
func computeNetworkValueExpert(h host.Host, ctx context.Context, sub *pubsub.Subscription, agent string, numAgents int) {
	voteCount := 0
	// a small in-memory map to keep count of votes from peers.
	truthMap := make(map[string]int)
	// a small in-memory agent map to remember all peers who have appeared earlier.
	peerMap := make(map[string]int)
	// get db instance
	db := reader.GetInstance()

	for {
		m, err := sub.Next(ctx)
		if err != nil {
			continue
		}
		currentValue, err := db.Get([]byte(agent))
		if err != nil {
			continue
		}
		_, ok := peerMap[m.ReceivedFrom.Pretty()]
		// if peer has already communicated earlier..continue
		if ok {
			continue
		}
		// host != peer and peer has not appeared before
		if h.ID().Pretty() != m.ReceivedFrom.Pretty() {
			// update local peerMap
			peerMap[m.ReceivedFrom.Pretty()] = 1
			// increase VoteCount
			voteCount = voteCount + 1
			// Get value of peer from vault
			recvdValue, err := db.Get([]byte(m.Message.Data))
			if err != nil {
				continue
			}
			// if received value != value in host vault
			// network needs to decide if the corresponding host
			// value needs to get updated or not.
			// For now, update in memory map frequency
			if string(currentValue) != string(recvdValue) && string(recvdValue) != "0" {
				_, ok := truthMap[string(recvdValue)]
				if ok {
					truthMap[string(recvdValue)] = truthMap[string(recvdValue)] + 1
				} else {
					truthMap[string(recvdValue)] = 0
				}
			}
		}

		// to decide if the corresponding host
		// received a value that is true or false, we wait till all votes are received and choose
		// the widely received value
		if voteCount == numAgents-1 {
			keys := make([]string, 0, len(truthMap))

			for key := range truthMap {
				keys = append(keys, key)
			}

			// sort the map to get the network value with highest frequency
			sort.SliceStable(keys, func(i, j int) bool {
				return truthMap[keys[i]] > truthMap[keys[j]]
			})

			for _, k := range keys {
				// update the agent(host) value with value which got the
				// highest frequency.
				// this way all rows in the vault will have the same value which
				// is the true value.
				db.Put([]byte(agent), []byte(k))
				break
			}
			break
		}
		time.Sleep(2 * time.Second)
	}

	fmt.Println(h.ID().Pretty(), "has received votes from all peers")
}

// `computeNetworkValueStandard` computes network value for the agent in standard mode
//  1. read current value from storage (there will always be some value stored at init)
//  2. compare with all other agents and decide
func computeNetworkValueStandard(id int, agents []reader.ParticipantSet, numAgents int) (k string) {
	truthMap := make(map[string]int)
	db := reader.GetInstance()
	numKeys := db.Len()

	for i := 0; i < numKeys; i++ {
		agentValue, _ := db.Get([]byte(agents[i].IP))
		if i == id {
			continue
		}
		_, ok := truthMap[string(agentValue)]
		if ok {
			truthMap[string(agentValue)] = truthMap[string(agentValue)] + 1
		} else {
			truthMap[string(agentValue)] = 0
		}
	}
	keys := make([]string, 0, len(truthMap))
	for key := range truthMap {
		keys = append(keys, key)
	}

	// sort the map to get the network value with highest frequency
	sort.SliceStable(keys, func(i, j int) bool {
		return truthMap[keys[i]] > truthMap[keys[j]]
	})
	for _, k := range keys {
		// return top value from cache.
		return k
	}

	return "-1"
}
