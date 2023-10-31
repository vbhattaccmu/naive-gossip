package reader

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strconv"

	"github.com/goombaio/namegenerator"
)

// ParticipantSet defines the identity of
// an agent in a network
type ParticipantSet struct {
	USER string
	IP   string
}

// `AddAgentsToConfig` appends new agents to a
// config file. If the file does not exist, a blank
// file is created.
func AddAgentsToConfig(numAgents int, value int, max_value int, ratio float64, config string) error {
	err := checkFile(config)
	data := []ParticipantSet{}

	if err != nil {
		return err
	}
	db := GetInstance()

	file, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}

	json.Unmarshal(file, &data)
	numTruthSpeakers := int(float64(numAgents) * (1 - ratio))
	port, _ := pickRandomNumber(5)

	numKeys := db.Len()
	startIdx := 0
	endIdx := numAgents
	if numKeys > 0 {
		startIdx = numKeys + 1
		endIdx = numKeys + numAgents
	}

	// append data to struct and distribute true and false data to vault
	for i := startIdx; i < endIdx; i++ {
		nameGenerator := namegenerator.NewNameGenerator(int64(i))
		name := nameGenerator.Generate()
		newStruct := &ParticipantSet{
			USER: name,
			IP:   fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
		}

		if i <= numTruthSpeakers {
			// assign value v to truth speakers
			db.Put([]byte(newStruct.IP), []byte(strconv.Itoa(value)))
		} else {
			// assign false value to the rest of the agents
			db.Put([]byte(newStruct.IP), []byte(strconv.Itoa(int(float64(max_value)*ratio))))
		}

		data = append(data, *newStruct)
		port = port + 1
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(config, dataBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// `checkFile` checks and creates config if not
// present in path.
func checkFile(filename string) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		_, err := os.Create(filename)
		if err != nil {
			return err
		}
	}
	return nil
}

// `pickRandomNumber` generates random integer of n digits.
func pickRandomNumber(denomination int) (int, error) {
	maxLimit := int64(65536)
	lowLimit := int(math.Pow10(5 - 1))
	randomNumber, err := rand.Int(rand.Reader, big.NewInt(maxLimit))
	if err != nil {
		return 0, err
	}
	randomNumberInt := int(randomNumber.Int64())

	if randomNumberInt <= lowLimit {
		randomNumberInt += lowLimit
	}
	if randomNumberInt > int(maxLimit) {
		randomNumberInt = int(maxLimit)
	}

	return randomNumberInt, nil
}

// `GetCurrentParticipants` gets participants from config.
func GetCurrentParticipants(config string) []ParticipantSet {
	var agents []ParticipantSet
	file, err := ioutil.ReadFile(config)
	if err != nil {
		return agents
	}
	json.Unmarshal(file, &agents)

	return agents
}

// `GetParticpantIP` gets IP of a certain participant from config.
func GetParticpantIP(config string, id string) string {
	var agents []ParticipantSet
	file, _ := ioutil.ReadFile(config)
	json.Unmarshal(file, &agents)

	for i := 0; i < len(agents); i++ {
		if agents[i].USER == id {
			return agents[i].IP
		}
	}

	return ""
}
