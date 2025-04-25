package cmd

import (
	"Luminary/pkg"
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
		agents := pkg.GetAgents()

		// Sort agents alphabetically
		sort.Slice(agents, func(i, j int) bool {
			return agents[i].Name < agents[j].Name
		})

		if apiMode {
			// Output machine-readable JSON
			fmt.Print(`{"agents":[`)
			for i, agent := range agents {
				if i > 0 {
					fmt.Print(",")
				}
				fmt.Printf(`{"id":"%s","name":"%s","description":"%s","status":"%s"}`,
					agent.ID, agent.Name, agent.Description, agent.Status)
			}
			fmt.Println(`]}`)
		} else {
			// User-friendly output
			fmt.Println("Available manga source agents:")
			fmt.Println("")

			format := "%-12s %-20s %-10s %s\n"
			fmt.Printf(format, "ID", "NAME", "STATUS", "DESCRIPTION")
			fmt.Println(strings.Repeat("-", 80))

			for _, agent := range agents {
				fmt.Printf(format,
					agent.ID,
					agent.Name,
					agent.Description,
					agent.Status)
			}

			fmt.Println("")
			fmt.Println("Use --agent flag with search/read commands to specify a particular agent")
		}
	},
}

func init() {
	rootCmd.AddCommand(agentsCmd)
}
