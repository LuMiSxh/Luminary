package cmd

import (
	"Luminary/pkg"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	listAgent string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available manga",
	Long:  `List all manga from all agents or a specific agent.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate the agent if specified
		if listAgent != "" {
			agent := pkg.GetAgentByID(listAgent)
			if agent == nil {
				fmt.Printf("Error: Agent '%s' not found\n", listAgent)
				fmt.Println("Available agents:")
				for _, a := range pkg.GetAgents() {
					fmt.Printf("  - %s (%s)\n", a.ID, a.Name)
				}
				return
			}
		}

		if apiMode {
			// Output machine-readable JSON for Palaxy
			fmt.Println(`{"mangas":[]}`) // Placeholder
		} else {
			// Interactive mode for CLI users
			fmt.Println("Listing manga...")
			if listAgent != "" {
				fmt.Printf("From agent: %s\n", listAgent)
			} else {
				fmt.Println("From all agents")
				fmt.Println("Available agents:")
				for _, a := range pkg.GetAgents() {
					fmt.Printf("  - %s (%s)\n", a.ID, a.Name)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Flags
	listCmd.Flags().StringVar(&listAgent, "agent", "", "Specific agent to list manga from (default: all)")
}
