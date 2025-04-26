package cmd

import (
	"Luminary/engine"
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

		// Create search options from flags
		options := engine.SearchOptions{
			Limit:   searchLimit,
			Fields:  searchFields,
			Filters: fieldFilters,
			Sort:    searchSort,
		}

		// Determine which agents to search
		var agentIDs []string
		if searchAgent != "" {
			// Validate the agent if specified
			if !appEngine.AgentExists(searchAgent) {
				if apiMode {
					utils.OutputJSON("error", nil, fmt.Errorf("agent '%s' not found", searchAgent))
					return
				}

				fmt.Printf("Error: Agent '%s' not found\n", searchAgent)
				fmt.Println("Available agents:")
				for _, a := range appEngine.AllAgents() {
					fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
				}
				return
			}

			agentIDs = []string{searchAgent}
		}

		// Execute the search using the search service
		results, err := appEngine.Search.SearchAcrossProviders(
			ctx,
			appEngine,
			query,
			options,
			agentIDs,
		)

		if err != nil {
			if apiMode {
				utils.OutputJSON("error", nil, err)
			} else {
				fmt.Printf("Error searching: %v\n", err)
			}
			return
		}

		if apiMode {
			outputAPIResults(results, query)
		} else {
			displayConsoleResults(results, query, options)
		}
	},
}

// outputAPIResults formats and outputs search results as JSON for API mode
func outputAPIResults(results map[string][]engine.Manga, query string) {
	var allResults []MangaSearchResult

	// Convert all results to our standardized format
	for agentID, mangaList := range results {
		agent, _ := appEngine.GetAgent(agentID)

		for _, manga := range mangaList {
			result := MangaSearchResult{
				ID:        utils.FormatMangaID(agentID, manga.ID),
				Title:     manga.Title,
				Agent:     agentID,
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

	// Output the search results
	utils.OutputJSON("success", map[string]interface{}{
		"query":   query,
		"results": allResults,
		"count":   len(allResults),
	}, nil)
}

// displayConsoleResults shows search results in an interactive, user-friendly format
func displayConsoleResults(results map[string][]engine.Manga, query string, options engine.SearchOptions) {
	// Calculate total result count
	totalCount := 0
	for _, mangaList := range results {
		totalCount += len(mangaList)
	}

	// Print search information
	fmt.Printf("Searching for: %s\n", query)

	if len(options.Fields) > 0 {
		fmt.Printf("Search fields: %s\n", strings.Join(options.Fields, ", "))
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
	fmt.Printf("Result limit: %d\n", options.Limit)

	// Display field-specific filters
	if len(options.Filters) > 0 {
		fmt.Println("Filters:")
		for field, value := range options.Filters {
			fmt.Printf("  %s: %s\n", field, value)
		}
	}

	fmt.Printf("\nTotal results found: %d\n\n", totalCount)

	// Display results for each agent
	if len(results) == 1 && searchAgent != "" {
		// Display results for a single agent
		for agentID, mangaList := range results {
			agent, _ := appEngine.GetAgent(agentID)
			fmt.Printf("Results from %s (%s):\n", agent.ID(), agent.Name())
			displaySearchResults(mangaList, agent)
		}
	} else {
		// Display results from all agents
		fmt.Println("Results across all agents:")
		for agentID, mangaList := range results {
			if len(mangaList) == 0 {
				continue
			}

			agent, _ := appEngine.GetAgent(agentID)
			fmt.Printf("\nFrom %s (%s):\n", agent.ID(), agent.Name())
			displaySearchResults(mangaList, agent)
		}
	}
}

// Helper function to display search results in a user-friendly format
func displaySearchResults(results []engine.Manga, agent engine.Agent) {
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
	searchCmd.Flags().StringVar(&searchSort, "sort", "relevance", "Sort by (relevance, name, newest, updated)")
	searchCmd.Flags().StringSliceVar(&searchFields, "fields", []string{}, "Fields to search in (title, author, genre), empty means all")
	searchCmd.Flags().BoolVar(&includeAltTitles, "alt-titles", true, "Include alternative titles in search")
	searchCmd.Flags().BoolVar(&includeAllLangs, "all-langs", true, "Search across all language versions of titles")

	// Field-specific filters
	fieldFilters = make(map[string]string)
	searchCmd.Flags().StringToStringVar(&fieldFilters, "filter", nil, "Field-specific filters (e.g., --filter genre=fantasy,author=oda)")
}
