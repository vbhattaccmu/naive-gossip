## LiarsLie

A programmatic solution to the Byzantine Generals' problem to determine a true value among a set of `n` agents where `m` (`m` <= `n`) agents always lie and the rest always speak the truth. The agents communicate over each other over a local network using go-libp2p. The algorithm details and CLI command structure is defined
[here](https://github.com/vbhattaccmu/liarslie/blob/main/algorithm.pdf).

## Modes

Standard - Agents do not gossip
Expert - Agents gossip their values to each other.

For more information, see `help` on CLI.

## To build

To build

```
go build
```

## Usage example in standard mode

```
 Case: liar ratio <= 0.5
 Start (config is cleaned)
 .\liarslie.exe standard start --value 5 --max-value 8 --num-agents 3 --liar-ratio 0.2
 Output :- Ready...

 Play
 .\liarslie.exe standard play
 Output :- The network value is is 5.
           One round of play in standard mode  is complete.. Liarslie is shutting down..

 Case: liar ratio > 0.5
 Start
 .\liarslie.exe standard start --value 5 --max-value 8 --num-agents 3 --liar-ratio 0.9
 Output :- Ready...

 Play
 .\liarslie.exe standard play
 Output :- The network value is is 7. (False value reported)
           One round of play in standard mode  is complete.. Liarslie is shutting down..

 Stop
 .\liarslie.exe standard stop
 Output :- All artifacts from liarslie are successfully removed.
```

## Usage example in expert mode

```
 Case liar-ratio <= 0.5
 Extend (Agents are appended)
 .\liarslie.exe expert extend --value 5 --max-value 8 --num-agents 3 --liar-ratio 0.2
 Output :- New agents added to the network...Vault has been updated.

 Playexpert
 .\liarslie.exe expert playexpert --num-agents 3 --liar-ratio 0.2
 Output :- The computed network value is is 5.
           One round of play in expert mode is complete.. Liarslie is shutting down..

 Case liar-ratio > 0.5
 Extend (Agents are appended)
 .\liarslie.exe expert extend --value 5 --max-value 8 --num-agents 3 --liar-ratio 0.8
 Output :- New agents added to the network...

 Playexpert
 .\liarslie.exe expert playexpert --num-agents 3 --liar-ratio 0.8
 Output :- The computed network value is is 7. (False value reported)
           One round of play in expert mode is complete.. Liarslie is shutting down..

 Kill
 .\liarslie.exe expert kill --id divine-cloud
 Output :- divine-cloud is removed from the network.

```

# Peer-to-peer (P2P) networking

The agents in liarslie maintain a peer-to-peer network (P2P). P2P implements _two_ high-level functionalities:

- **Messaging**: P2P enables replicas to send messages to each other to replicate the state machine.
- **Peer Discovery**: in order for messages to eventually be received by all of their intended recipients, the P2P network needs to be _connected_.

## libp2p

P2P is built using [libp2p](https://docs.libp2p.io/concepts/introduction/overview/). libp2p is a set of networking-related protocols initially developed for [IPFS](https://en.wikipedia.org/wiki/InterPlanetary_File_System) that is intended to help build peer-to-peer systems.

The libp2p protocols used by P2P fall into two categories:

1. **Substream protocols** define how to resolve domain names into IP addresses, which OSI level-4 transport to use, how to secure connections, and how to multiplex the use of a connection between multiple substreams.
2. **Application protocols** define when to create substreams and what to communicate over the substreams.

## Substreams

P2P substreams are built using four protocols, each implementing a specific functionality listed with before it in the table below:

| Functionality               | Protocol                                                                                                                                     | Details                                                                                                                                            |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Domain Name Resolution      | [`dns`](https://github.com/libp2p/specs/blob/master/addressing/README.md#ip-and-name-resolution)                                             |                                                                                                                                                    |
| OSI Level 4 Transport       | TCP                                                                                                                                          |                                                                                                                                                    |
| Authentication and Security | [Noise](https://github.com/libp2p/specs/blob/master/noise/README.md)                                                                         | Using the XX handshake pattern and the replica's keypair.                                                                                          |
| Connection Multiplexing     | [yamux](https://github.com/libp2p/specs/blob/master/yamux/README.md) or [mplex](https://github.com/libp2p/specs/blob/master/mplex/README.md) | Negotiated using [multistream-select](https://github.com/libp2p/specs/blob/master/connections/README.md#multistream-select), with yamux preferred. |

## Messaging

### GossipSub

P2P uses GossipSub for messaging. GossipSub is a topic-based publish-subscribe protocol. A topic is a UTF-8 string, and messages published to a topic is received by peers subscribed to that topic. GossipSub provides robust assurances that a peer subscribed to a topic eventually receives all messages published to that topic. In addition, GossipSub messages are digitally signed by default, and are therefore non-repudiable.

The following subsections describe how P2P configures GossipSub, and then lists the topics that P2P has available for different protocol functionalities, as well as the types in the corresponding message data.

#### Configuration

P2P retains most of the default GossipSub configuration as listed [here](https://docs.rs/libp2p-gossipsub/0.45.0/libp2p_gossipsub/struct.Config.html), but overrides three parameters:

| Overriden parameter | Value                                                  |
| ------------------- | ------------------------------------------------------ |
| Max transmit size   | 4 KB                                                   |
| Allow self origin   | true                                                   |
| Message ID[^1]      | Topic + Base64URL encoding of source + sequence number |

## Peer Discovery

### Kademlia

Kademlia is a Distributed Hash Table (DHT) with a network topology that has desirable properties for peer-to-peer networks with large numbers of participants. P2P _disables_ Kademlia's storage functionality and uses it solely for Peer Discovery. Kademlia maintains a table (a “routing table”) of peer address information in each peer.

Every time Kademlia opens a new connection to a peer, GossipSub is notified and considers opening a stream to that peer for itself, eventually creating a connected topology of GossipSub peers.
