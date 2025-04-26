package cmd

import (
	"Luminary/utils"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List all available manga source agents",
	Long:  `Display a list of all configured manga source agents that Luminary can use to search and read manga.`,
	Run: func(cmd *cobra.Command, args []string) {
		allAgents := appEngine.AllAgents()

		// Sort agents alphabetically
		sort.Slice(allAgents, func(i, j int) bool {
			return allAgents[i].Name() < allAgents[j].Name()
		})

		if apiMode {
			// Create a slice to hold agent data
			type AgentData struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			}

			agentList := make([]AgentData, 0, len(allAgents))

			for _, agent := range allAgents {
				agentData := AgentData{
					ID:          agent.ID(),
					Name:        agent.Name(),
					Description: agent.Description(),
				}

				agentList = append(agentList, agentData)
			}

			// Output machine-readable JSON using our utility
			utils.OutputJSON("success", map[string]interface{}{
				"agents": agentList,
			}, nil)
		} else {
			// User-friendly output
			fmt.Println("Available manga source agents:")
			fmt.Println("")

			format := "%-12s %-20s %s\n"
			fmt.Printf(format, "ID", "NAME", "DESCRIPTION")
			fmt.Println(strings.Repeat("-", 80))

			for _, agent := range allAgents {
				fmt.Printf(format,
					agent.ID(),
					agent.Name(),
					agent.Description())
			}

			fmt.Println("")
			fmt.Println("Use --agent flag with the search command to specify a particular agent")
		}
	},
}

func init() {
	rootCmd.AddCommand(agentsCmd)
}
