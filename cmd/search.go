package cmd

import (
	"Luminary/pkg"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	searchAgent  string
	searchLimit  int
	searchSort   string
	searchFields []string
	fieldFilters map[string]string
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for manga",
	Long:  `Search for manga by title, genre, or author.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		// Validate the agent if specified
		if searchAgent != "" {
			agent := pkg.GetAgentByID(searchAgent)
			if agent == nil {
				fmt.Printf("Error: Agent '%s' not found\n", searchAgent)
				fmt.Println("Available agents:")
				for _, a := range pkg.GetAgents() {
					fmt.Printf("  - %s (%s)\n", a.ID, a.Name)
				}
				return
			}
		}

		// Determine which fields to search (defaults to all if none specified)
		fieldsToSearch := searchFields
		if len(fieldsToSearch) == 0 {
			fieldsToSearch = []string{"title", "genre", "author"}
		}

		if apiMode {
			// Output machine-readable JSON for Palaxy
			fmt.Printf(`{"results":[{"id":"manga-1","title":"Example Manga matching '%s'","agent":"mangadex","searchFields":%q,"filters":%q}]}`,
				query, strings.Join(fieldsToSearch, ","), fieldFilters)
		} else {
			// Interactive mode for CLI users
			fmt.Printf("Searching for: %s\n", query)
			fmt.Printf("Search fields: %s\n", strings.Join(fieldsToSearch, ", "))

			// Display field-specific filters
			if len(fieldFilters) > 0 {
				fmt.Println("Filters:")
				for field, value := range fieldFilters {
					fmt.Printf("  %s: %s\n", field, value)
				}
			}

			// Display additional settings and agent info
			if searchAgent != "" {
				agent := pkg.GetAgentByID(searchAgent)
				fmt.Printf("Agent: %s (%s)\n", agent.ID, agent.Name)
			} else {
				fmt.Println("Searching across all agents:")
				for _, a := range pkg.GetAgents() {
					fmt.Printf("  - %s (%s)\n", a.ID, a.Name)
				}
			}
			fmt.Printf("Limit: %d\n", searchLimit)
			fmt.Printf("Sort: %s\n", searchSort)
		}

		// Here you would implement the actual search logic to query
		// across the specified fields (title, author, genre) based on fieldsToSearch
		// and apply the fieldFilters as exact match constraints
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Flags
	searchCmd.Flags().StringVar(&searchAgent, "agent", "", "Search using specific agent")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Limit number of results")
	searchCmd.Flags().StringVar(&searchSort, "sort", "relevance", "Sort by (relevance, popularity, name)")
	searchCmd.Flags().StringSliceVar(&searchFields, "fields", []string{}, "Fields to search in (title, author, genre), empty means all")

	// Field-specific filters
	fieldFilters = make(map[string]string)
	searchCmd.Flags().StringToStringVar(&fieldFilters, "filter", nil, "Field-specific filters (e.g., --filter genre=fantasy,author=oda)")
}
