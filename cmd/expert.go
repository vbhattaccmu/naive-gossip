package cmd

import (
	"fmt"
	"liarslie/peer"
	"liarslie/reader"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(expert)
	expert.AddCommand(extend)
	expert.AddCommand(playexpert)
	expert.AddCommand(kill)

	extend.PersistentFlags().String("value", "", "True value of the network")
	extend.PersistentFlags().String("max-value", "", "Max value that a liar can broadcast")
	extend.PersistentFlags().String("num-agents", "", "Total number of agents in the network")
	extend.PersistentFlags().String("liar-ratio", "", "Ratio between liars and truth-tellers in the network")

	playexpert.PersistentFlags().String("num-agents", "", "Total number of agents in the network")
	playexpert.PersistentFlags().String("liar-ratio", "", "Ratio between liars and truth-tellers in the network")

	kill.PersistentFlags().String("id", "", "Id of the agent")
}

var expert = &cobra.Command{
	Use:     "expert",
	Aliases: []string{"exp"},
	Short:   "Start liarslie in expert mode",
	Long:    `This command starts liarslie with a variable set of bootstrapped agents defined in agents.config in expert mode.`,
}

var extend = &cobra.Command{
	Use:   "extend",
	Short: "Extend number of agents in the network",
	Long: `Extend checks for the existence of agents.config, and, if present, extend the network by launching
	the specified agents and appending information about them into agents.config`,
	Run: func(cmd *cobra.Command, args []string) {

		/// Steps 1. Append new entries to config file
		///       2. For new players also randomly allocate true/false network value based on liar-ratio on storage

		// get number of agents
		num, _ := cmd.Flags().GetString("num-agents")
		value, _ := cmd.Flags().GetString("value")
		maxValue, _ := cmd.Flags().GetString("max-value")
		liarRatio, _ := cmd.Flags().GetString("liar-ratio")

		// convert string to integer
		val, valConversionError := strconv.Atoi(value)
		agents, agentConversionError := strconv.Atoi(num)
		max, maxConversionError := strconv.Atoi(maxValue)
		ratio, liarRatioConversionError := strconv.ParseFloat(liarRatio, 32)
		config := "agents.json"

		if valConversionError != nil || agentConversionError != nil || maxConversionError != nil || liarRatioConversionError != nil {
			fmt.Println("Error in value conversion.")
			return
		}

		// call reader append file to generate config.
		appendError := reader.AddAgentsToConfig(agents, val, max, ratio, config)
		if appendError != nil {
			fmt.Println("Error in saving agents.json.")
			return
		}

		fmt.Println("Updated agents.json with new agents.")

		fmt.Println("******************************************************************************************")
		fmt.Println("Starting liarslie in expert mode... Attempting to compute network value for only one round")
		fmt.Println("******************************************************************************************")

		// get agents from config
		latest_agents := reader.GetCurrentParticipants("agents.json")
		numAgents := len(latest_agents)

		var wg sync.WaitGroup
		wg.Add(numAgents)

		for i := 0; i < numAgents; i++ {
			go func(i int, latest_agents []reader.ParticipantSet) {
				defer wg.Done()
				peer.RunAsExpert(i, latest_agents, numAgents, true)
			}(i, latest_agents)
		}

		wg.Wait()

		fmt.Println(" ")
		fmt.Println("********************************************************")
		fmt.Println("New agents added to the network..Vault has been updated.")
		fmt.Println("********************************************************")
	},
}

var playexpert = &cobra.Command{
	Use:   "playexpert",
	Short: "Play liarslie as expert",
	Long:  `This command starts p2p networking among the agents and computes the true network value in expert mode`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("******************************************************************************************")
		fmt.Println("Starting liarslie in expert mode... Attempting to compute network value for only one round")
		fmt.Println("******************************************************************************************")

		// get data from arguments
		num, _ := cmd.Flags().GetString("num-agents")
		liarRatio, _ := cmd.Flags().GetString("liar-ratio")

		// get vault instance
		db := reader.GetInstance()

		// convert string to integer
		numAgents, agentConversionError := strconv.Atoi(num)
		_, liarRatioConversionError := strconv.ParseFloat(liarRatio, 32)

		if agentConversionError != nil || liarRatioConversionError != nil {
			fmt.Println("Error in value conversion.")
			return
		}
		agents := reader.GetCurrentParticipants("agents.json")

		truthValue := -1
		// go through the vault to compute the truth value of network
		// in expert mode, given a low liar-ratio, all the entries
		// of the vault is updated to the truest value if `extend` is called earlier.
		// In any other case `playexpert` may return a false value as well since numAgents
		// may or may not be equal to total number of keys in the vault and truth value is decided
		// by frequency.
		for i := 0; i < numAgents; i++ {
			networkValue, _ := db.Get([]byte(agents[i].IP))
			value, _ := strconv.Atoi(string(networkValue))
			if value > truthValue {
				truthValue = value
			}
		}
		db.Close()

		if truthValue < 0 {
			fmt.Println(" ")
			fmt.Println("*******************************************")
			fmt.Println("Please check your agents config. Its empty!")
			fmt.Println("*******************************************")
		} else {
			fmt.Println(" ")
			fmt.Println("****************************************")
			fmt.Println("The computed network value is", truthValue)
			fmt.Println("****************************************")
		}

		fmt.Println(" ")
		fmt.Println("One round of play in expert mode is complete.. Liarslie is shutting down..")
	},
}

var kill = &cobra.Command{
	Use:   "kill",
	Short: "kill the current agent",
	Long:  `This command can be used to remove a certain agent from a network.`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		IP := reader.GetParticpantIP("agents.json", id)
		if len(IP) == 0 {
			fmt.Println("No ID by name provided exists on the network")
		} else {
			// get vault instance
			db := reader.GetInstance()

			// remove data from vault
			// this triggers the discovery module to
			// remove the respective peerID from the network
			db.Delete([]byte(IP))

			fmt.Println(" ")
			fmt.Println(id, "is removed from the network")
		}
	},
}
