package cmd

import (
	"Luminary/agents"
	"Luminary/engine"
	"Luminary/utils"
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	listAgent string
	listLimit int
)

// MangaListItem represents a manga item for API list responses
type MangaListItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Agent     string `json:"agent"`
	AgentName string `json:"agent_name,omitempty"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available manga",
	Long:  `List all manga from all agents or a specific agent.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Validate the agent if specified
		var selectedAgent agents.Agent
		if listAgent != "" {
			selectedAgent = agents.Get(listAgent)
			if selectedAgent == nil {
				if apiMode {
					utils.OutputJSON("error", nil, fmt.Errorf("agent '%s' not found", listAgent))
					return
				}

				fmt.Printf("Error: Agent '%s' not found\n", listAgent)
				fmt.Println("Available agents:")
				for _, a := range agents.All() {
					fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
				}
				return
			}
		}

		// Create search options using the engine type
		options := engine.SearchOptions{
			Limit: listLimit,
			// We use empty search to get list of manga
		}

		if apiMode {
			var allMangas []MangaListItem

			// Function to list manga from a single agent
			listAgentMangas := func(agent agents.Agent) {
				// Use empty search to get list of manga
				mangas, err := agent.Search(ctx, "", options)
				if err != nil {
					return
				}

				for _, manga := range mangas {
					mangaItem := MangaListItem{
						ID:        utils.FormatMangaID(agent.ID(), manga.ID),
						Title:     manga.Title,
						Agent:     agent.ID(),
						AgentName: agent.Name(),
					}
					allMangas = append(allMangas, mangaItem)
				}
			}

			if selectedAgent != nil {
				listAgentMangas(selectedAgent)
			} else {
				// List manga from all agents
				for _, agent := range agents.All() {
					listAgentMangas(agent)
				}
			}

			// Create response data with agent filter info if applicable
			responseData := map[string]interface{}{
				"mangas": allMangas,
				"count":  len(allMangas),
			}

			if selectedAgent != nil {
				responseData["agent"] = selectedAgent.ID()
				responseData["agent_name"] = selectedAgent.Name()
			}

			utils.OutputJSON("success", responseData, nil)
		} else {
			// Interactive mode for CLI users
			if selectedAgent != nil {
				fmt.Printf("Listing manga from agent: %s (%s)\n\n", selectedAgent.ID(), selectedAgent.Name())

				// Use empty search to get list of manga
				mangas, err := selectedAgent.Search(ctx, "", options)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					return
				}

				displayMangaList(mangas, selectedAgent)
			} else {
				fmt.Println("Listing manga from all agents:")

				for _, agent := range agents.All() {
					fmt.Printf("\nFrom agent: %s (%s)\n", agent.ID(), agent.Name())

					// Use empty search to get list of manga
					mangas, err := agent.Search(ctx, "", options)
					if err != nil {
						fmt.Printf("  Error: %v\n", err)
						continue
					}

					displayMangaList(mangas, agent)
				}
			}
		}
	},
}

// Helper function to display a manga list in a user-friendly format
func displayMangaList(mangas []agents.Manga, agent agents.Agent) {
	if len(mangas) == 0 {
		fmt.Println("  No manga found")
		return
	}

	for i, manga := range mangas {
		fmt.Printf("  %d. %s (ID: %s:%s)\n", i+1, manga.Title, agent.ID(), manga.ID)
	}

	fmt.Printf("\n  Found %d manga titles\n", len(mangas))
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Flags
	listCmd.Flags().StringVar(&listAgent, "agent", "", "Specific agent to list manga from (default: all)")
	listCmd.Flags().IntVar(&listLimit, "limit", 50, "Limit number of results per agent")
}
