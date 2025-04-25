package cmd

import (
	"Luminary/agents"
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
		allAgents := agents.All()

		// Sort agents alphabetically
		sort.Slice(allAgents, func(i, j int) bool {
			return allAgents[i].Name() < allAgents[j].Name()
		})

		if apiMode {
			// Output machine-readable JSON
			fmt.Print(`{"agents":[`)
			for i, agent := range allAgents {
				if i > 0 {
					fmt.Print(",")
				}
				// Get tags if available
				tags := "[]"
				if tagger, ok := agent.(interface{ Tags() []string }); ok {
					tagSlice := tagger.Tags()
					if len(tagSlice) > 0 {
						tags = fmt.Sprintf("%q", tagSlice)
					}
				}

				fmt.Printf(`{"id":"%s","name":"%s","description":"%s","status":"%s","tags":%s}`,
					agent.ID(), agent.Name(), agent.Description(), agent.Status(), tags)
			}
			fmt.Println(`]}`)
		} else {
			// User-friendly output
			fmt.Println("Available manga source agents:")
			fmt.Println("")

			format := "%-12s %-20s %-10s %s\n"
			fmt.Printf(format, "ID", "NAME", "STATUS", "DESCRIPTION")
			fmt.Println(strings.Repeat("-", 80))

			for _, agent := range allAgents {
				fmt.Printf(format,
					agent.ID(),
					agent.Name(),
					string(agent.Status()),
					agent.Description())
			}

			fmt.Println("")
			fmt.Println("Use --agent flag with search/read commands to specify a particular agent")
		}
	},
}

func init() {
	rootCmd.AddCommand(agentsCmd)
}
