package cmd

import (
	"fmt"
	"liarslie/peer"
	"liarslie/reader"
	"os"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(standard)
	standard.AddCommand(start)
	standard.AddCommand(play)
	standard.AddCommand(stop)
	start.PersistentFlags().String("value", "", "True value of the network")
	start.PersistentFlags().String("max-value", "", "Max value that a liar can broadcast")
	start.PersistentFlags().String("num-agents", "", "Total number of agents in the network")
	start.PersistentFlags().String("liar-ratio", "", "Ratio between liars and truth-tellers in the network")
}

var standard = &cobra.Command{
	Use:     "standard",
	Aliases: []string{"std", "stan"},
	Short:   "Start liarslie in standard mode",
	Long:    `This command starts liarslie with a fixed set of bootstrapped agents defined in agents.config in standard mode.`,
}

var start = &cobra.Command{
	Use:   "start",
	Short: "Start the liarslie game and produce agents.config file.",
	Long:  `This command starts liarslie game and produces agents.config file with a set of bootstrapped agent addresses.`,
	Run: func(cmd *cobra.Command, args []string) {
		/// Steps 1. Generate config file
		///       2. For each player also randomly allocate true/false network value based on liar-ratio on storage

		// get number of agents
		num, _ := cmd.Flags().GetString("num-agents")
		value, _ := cmd.Flags().GetString("value")
		maxValue, _ := cmd.Flags().GetString("max-value")
		liarRatio, _ := cmd.Flags().GetString("liar-ratio")

		// preliminary setup
		val, valConversionError := strconv.Atoi(value)
		agents, agentConversionError := strconv.Atoi(num)
		max, maxConversionError := strconv.Atoi(maxValue)
		ratio, liarRatioConversionError := strconv.ParseFloat(liarRatio, 32)
		config := "agents.json"

		// remove config if exists
		os.Remove(config)
		// remove storage
		os.RemoveAll("storage/")

		if valConversionError != nil || agentConversionError != nil || maxConversionError != nil || liarRatioConversionError != nil {
			fmt.Println("Error in value conversion.")
			return
		}

		// call append file from reader.go to generate config
		appendError := reader.AddAgentsToConfig(agents, val, max, ratio, config)
		if appendError != nil {
			fmt.Println("Error in saving agents.json.")
			return
		}
		fmt.Println("Ready...")
	},
}

var play = &cobra.Command{
	Use:   "play",
	Short: "Start computing the true network value",
	Long:  `This command starts p2p networking among the agents and computes the true network value`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("********************************************************************************************")
		fmt.Println("Starting liarslie in standard mode... Attempting to compute network value for only one round")
		fmt.Println("********************************************************************************************")

		// get agents from config
		agents := reader.GetCurrentParticipants("agents.json")
		numAgents := len(agents)

		var wg sync.WaitGroup
		wg.Add(numAgents)
		// initialize network value as -1
		truthValue := -1
		// start standard mode network value computation
		for i := 0; i < numAgents; i++ {
			go func(i int, agents []reader.ParticipantSet) {
				defer wg.Done()
				value, _ := strconv.Atoi(peer.RunAsStandard(i, agents, numAgents))
				if truthValue < value {
					truthValue = value
				}
			}(i, agents)
		}

		wg.Wait()

		if truthValue < 0 {
			fmt.Println(" ")
			fmt.Println("*******************************************")
			fmt.Println("Please check your agents config. Its empty!")
			fmt.Println("*******************************************")
		} else {
			fmt.Println(" ")
			fmt.Println("*****************************************")
			fmt.Println("The computed network value is", truthValue)
			fmt.Println("*****************************************")
		}

		fmt.Println(" ")
		fmt.Println("One round of play in standard mode is complete.. Liarslie is shutting down..")

	},
}

var stop = &cobra.Command{
	Use:   "stop",
	Short: "stop liarslie",
	Long:  `This command can be used to stop the game`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(" ")
		fmt.Println("*********************************************************")
		fmt.Println("All artifacts from liarslie are being succesfully removed")
		fmt.Println("*********************************************************")

		err := os.Remove("agents.json")
		if err != nil {
			fmt.Println(err)
		}
		os.RemoveAll("storage/")
		e := KillProcess("liarslie")
		if e != nil {
			fmt.Println(e)
		}

		fmt.Println(" ")
		fmt.Println("*****************************")
		fmt.Println("Successfully removed liarslie")
		fmt.Println("*****************************")
	},
}
