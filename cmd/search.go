package cmd

import (
	"Luminary/agents"
	"Luminary/utils"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	searchAgent      string
	searchLimit      int
	searchSort       string
	searchFields     []string
	fieldFilters     map[string]string
	includeAltTitles bool
	includeAllLangs  bool
)

// MangaSearchResult represents a manga search result for API output
type MangaSearchResult struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Agent     string   `json:"agent"`
	AgentName string   `json:"agent_name,omitempty"`
	AltTitles []string `json:"alt_titles,omitempty"`
	Authors   []string `json:"authors,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for manga",
	Long:  `Search for manga by title, genre, or author.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Validate the agent if specified
		var selectedAgent agents.Agent
		if searchAgent != "" {
			selectedAgent = agents.Get(searchAgent)
			if selectedAgent == nil {
				if apiMode {
					utils.OutputJSON("error", nil, fmt.Errorf("agent '%s' not found", searchAgent))
					return
				}

				fmt.Printf("Error: Agent '%s' not found\n", searchAgent)
				fmt.Println("Available agents:")
				for _, a := range agents.All() {
					fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
				}
				return
			}
		}

		// Create search options from flags
		options := agents.SearchOptions{
			Limit:   searchLimit,
			Fields:  searchFields,
			Filters: fieldFilters,
			Sort:    searchSort,
		}

		if apiMode {
			var allResults []MangaSearchResult

			// When a specific agent is selected
			if selectedAgent != nil {
				results, err := selectedAgent.Search(ctx, query, options)
				if err != nil {
					utils.OutputJSON("error", nil, err)
					return
				}

				// Convert results to our standardized format
				for _, manga := range results {
					result := MangaSearchResult{
						ID:        utils.FormatMangaID(selectedAgent.ID(), manga.ID),
						Title:     manga.Title,
						Agent:     selectedAgent.ID(),
						AgentName: selectedAgent.Name(),
					}

					// Include additional fields if available
					if includeAltTitles && len(manga.AltTitles) > 0 {
						result.AltTitles = manga.AltTitles
					}
					if len(manga.Authors) > 0 {
						result.Authors = manga.Authors
					}
					if len(manga.Tags) > 0 {
						result.Tags = manga.Tags
					}

					allResults = append(allResults, result)
				}
			} else {
				// When searching across all agents, we combine results
				for _, agent := range agents.All() {
					results, err := agent.Search(ctx, query, options)
					if err != nil {
						continue // Skip agents with errors
					}

					for _, manga := range results {
						result := MangaSearchResult{
							ID:        utils.FormatMangaID(agent.ID(), manga.ID),
							Title:     manga.Title,
							Agent:     agent.ID(),
							AgentName: agent.Name(),
						}

						// Include additional fields if available
						if includeAltTitles && len(manga.AltTitles) > 0 {
							result.AltTitles = manga.AltTitles
						}
						if len(manga.Authors) > 0 {
							result.Authors = manga.Authors
						}
						if len(manga.Tags) > 0 {
							result.Tags = manga.Tags
						}

						allResults = append(allResults, result)
					}
				}
			}

			// Output the search results
			utils.OutputJSON("success", map[string]interface{}{
				"query":   query,
				"results": allResults,
				"count":   len(allResults),
			}, nil)
		} else {
			// Interactive mode for CLI users
			fmt.Printf("Searching for: %s\n", query)

			if len(searchFields) > 0 {
				fmt.Printf("Search fields: %s\n", strings.Join(searchFields, ", "))
			} else {
				fmt.Println("Search fields: all")
			}

			// Display search options
			if includeAltTitles {
				fmt.Println("Searching in alternative titles: enabled")
			}
			if includeAllLangs {
				fmt.Println("Searching across all languages: enabled")
			}
			fmt.Printf("Result limit: %d\n", searchLimit)

			// Display field-specific filters
			if len(fieldFilters) > 0 {
				fmt.Println("Filters:")
				for field, value := range fieldFilters {
					fmt.Printf("  %s: %s\n", field, value)
				}
			}

			// Execute the search
			if selectedAgent != nil {
				fmt.Printf("Searching using agent: %s (%s)\n", selectedAgent.ID(), selectedAgent.Name())
				results, err := selectedAgent.Search(ctx, query, options)
				if err != nil {
					fmt.Printf("Error searching: %v\n", err)
					return
				}

				// Display results
				displaySearchResults(results, selectedAgent)
			} else {
				fmt.Println("Searching across all agents...")

				// Search through all agents
				for _, agent := range agents.All() {
					fmt.Printf("Results from %s (%s):\n", agent.ID(), agent.Name())
					results, err := agent.Search(ctx, query, options)
					if err != nil {
						fmt.Printf("  Error: %v\n", err)
						continue
					}

					// Display results from this agent
					displaySearchResults(results, agent)
					fmt.Println()
				}
			}
		}
	},
}

// Helper function to display search results in a user-friendly format
func displaySearchResults(results []agents.Manga, agent agents.Agent) {
	if len(results) == 0 {
		fmt.Println("  No results found")
		return
	}

	fmt.Printf("  Found %d results:\n", len(results))
	for i, manga := range results {
		fmt.Printf("  %d. %s (ID: %s:%s)\n", i+1, manga.Title, agent.ID(), manga.ID)

		// Display alternative titles if available
		if len(manga.AltTitles) > 0 {
			// Deduplicate alternative titles (in case the same title appears in multiple languages)
			uniqueTitles := make(map[string]bool)
			var displayTitles []string

			for _, title := range manga.AltTitles {
				if !uniqueTitles[title] {
					uniqueTitles[title] = true
					displayTitles = append(displayTitles, title)
				}
			}

			if len(displayTitles) > 0 {
				// Show up to 3 alternative titles
				fmt.Printf("     Also known as: %s\n", strings.Join(displayTitles[:minInt(3, len(displayTitles))], ", "))
				if len(displayTitles) > 3 {
					fmt.Printf("     ...and %d more alternative titles\n", len(displayTitles)-3)
				}
			}
		}

		if len(manga.Authors) > 0 {
			fmt.Printf("     Authors: %s\n", strings.Join(manga.Authors, ", "))
		}
		if len(manga.Tags) > 0 {
			fmt.Printf("     Tags: %s\n", strings.Join(manga.Tags[:minInt(5, len(manga.Tags))], ", "))
			if len(manga.Tags) > 5 {
				fmt.Printf("     ...and %d more tags\n", len(manga.Tags)-5)
			}
		}
	}
}

// Helper function for min of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Flags
	searchCmd.Flags().StringVar(&searchAgent, "agent", "", "Search using specific agent")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Limit number of results")
	searchCmd.Flags().StringVar(&searchSort, "sort", "relevance", "Sort by (relevance, popularity, name)")
	searchCmd.Flags().StringSliceVar(&searchFields, "fields", []string{}, "Fields to search in (title, author, genre), empty means all")
	searchCmd.Flags().BoolVar(&includeAltTitles, "alt-titles", true, "Include alternative titles in search")
	searchCmd.Flags().BoolVar(&includeAllLangs, "all-langs", true, "Search across all language versions of titles")

	// Field-specific filters
	fieldFilters = make(map[string]string)
	searchCmd.Flags().StringToStringVar(&fieldFilters, "filter", nil, "Field-specific filters (e.g., --filter genre=fantasy,author=oda)")
}
