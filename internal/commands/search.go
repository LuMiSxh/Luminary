// Luminary: A streamlined CLI tool for searching and downloading manga.
// Copyright (C) 2025 Luca M. Schmidt (LuMiSxh)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package commands

import (
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/display"
	"Luminary/pkg/errors"
	"Luminary/pkg/provider"
	"Luminary/pkg/util"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	searchProvider   string
	searchLimit      int
	searchPages      int
	searchSort       string
	searchFields     []string
	fieldFilters     map[string]string
	includeAltTitles bool
	includeAllLangs  bool
)

// MangaSearchResult represents a manga search result for API output
type MangaSearchResult struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Provider     string   `json:"provider"`
	ProviderName string   `json:"provider_name,omitempty"`
	AltTitles    []string `json:"alt_titles,omitempty"`
	Authors      []string `json:"authors,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for manga",
	Long:  `Search for manga by title, genre, or author.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		// Determine if we're using multiple providers
		multipleProviders := searchProvider == ""

		// Calculate appropriate timeout based on pagination parameters
		timeoutDuration := calculateTimeout(searchLimit, searchPages, multipleProviders)

		// Create context with dynamic timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// Set concurrency limit in the context
		ctx = core.WithConcurrency(ctx, maxConcurrency)

		// Inform user about extended timeout if applicable
		if timeoutDuration > time.Minute && !apiMode {
			fmt.Printf("Note: Using extended timeout of %v for this request.\n",
				timeoutDuration.Round(time.Second))
		}

		// Create search options from flags
		options := core.SearchOptions{
			Limit:   searchLimit,
			Pages:   searchPages,
			Fields:  searchFields,
			Filters: fieldFilters,
			Sort:    searchSort,
		}

		// Determine which providers to search
		var providers []provider.Provider
		if searchProvider != "" {
			// Validate the provider if specified
			if !appEngine.ProviderExists(searchProvider) {
				if apiMode {
					util.OutputJSON("error", nil, fmt.Errorf("provider '%s' not found", searchProvider))
					return
				}

				fmt.Printf("Error: Provider '%s' not found\n", searchProvider)
				fmt.Println("Available providers:")
				for _, a := range appEngine.AllProvider() {
					fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
				}
				return
			}

			// Get the provider instance instead of just the ID
			provider, _ := appEngine.GetProvider(searchProvider)
			providers = append(providers, provider)
		} else {
			// If no specific provider was requested, use all available providers
			providers = appEngine.AllProvider()
		}

		// Execute the search using the search service
		results, err := appEngine.Search.SearchAcrossProviders(
			ctx,
			providers,
			query,
			options,
		)

		if err != nil {
			handleSearchError(err, query, searchProvider)
			return
		}

		if apiMode {
			outputAPIResults(results, query)
		} else {
			displayConsoleResults(results, query, options)
		}
	},
}

// handleSearchError provides user-friendly error messages based on error type
func handleSearchError(err error, query, providerSpec string) {
	// Determine provider name for display
	providerName := "all providers"
	if providerSpec != "" {
		providerName = providerSpec
	}

	// Check for specific error types in order of specificity
	if errors.IsServerError(err) {
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("server error from %s: %v", providerName, err))
		} else {
			fmt.Printf("Error: Server error while searching with %s. Please try again later.\n", providerName)
		}
		return
	}

	// Check for rate limiting
	if errors.Is(err, errors.ErrRateLimit) {
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("rate limit exceeded for %s", providerName))
		} else {
			fmt.Printf("Error: Rate limit exceeded for %s. Please try again later.\n", providerName)
		}
		return
	}

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) {
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("search timed out for query '%s'", query))
		} else {
			fmt.Printf("Error: Search timed out. Try reducing the number of pages or increasing the time limit.\n")
		}
		return
	}

	// Generic error handling
	if apiMode {
		util.OutputJSON("error", nil, err)
	} else {
		fmt.Printf("Error searching: %v\n", err)
	}
}

// Calculate an appropriate timeout based on pagination parameters
func calculateTimeout(limit, pages int, multipleProviders bool) time.Duration {
	// Base timeout
	timeoutDuration := 30 * time.Second

	// Adjust for pagination parameters
	if limit == 0 || pages == 0 {
		// For unlimited requests, use a much higher base timeout
		timeoutDuration = 15 * time.Minute // Increased from 5 to 15 minutes

		// If both are unlimited, use a maximum timeout
		if limit == 0 && pages == 0 {
			timeoutDuration = 30 * time.Minute // Increased from 10 to 30 minutes
		}
	} else if pages > 3 || limit > 50 {
		// For larger paginated requests
		timeoutDuration = 5 * time.Minute // Increased from 3 to 5 minutes

		// Scale with number of pages
		if pages > 5 {
			pageTimeoutFactor := time.Duration(pages) / 5
			if pageTimeoutFactor > 1 {
				extraTimeout := pageTimeoutFactor * 2 * time.Minute // Doubled the extra time per page
				if extraTimeout > 10*time.Minute {
					extraTimeout = 10 * time.Minute // Cap at 10 extra minutes (increased from 5)
				}
				timeoutDuration += extraTimeout
			}
		}
	}

	// Add extra time if querying multiple providers
	if multipleProviders {
		timeoutDuration += 5 * time.Minute // Increased from 2 to 5 minutes
	}

	return timeoutDuration
}

// outputAPIResults formats and outputs search results as JSON for API mode
func outputAPIResults(results map[string][]core.Manga, query string) {
	var allResults []MangaSearchResult

	// Convert all results to our standardized format
	for providerID, mangaList := range results {
		prov, _ := appEngine.GetProvider(providerID)

		for _, manga := range mangaList {
			result := MangaSearchResult{
				ID:           core.FormatMangaID(providerID, manga.ID),
				Title:        manga.Title,
				Provider:     providerID,
				ProviderName: prov.Name(),
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
	util.OutputJSON("success", map[string]interface{}{
		"query":   query,
		"results": allResults,
		"count":   len(allResults),
	}, nil)
}

// displayConsoleResults shows search results in an interactive, user-friendly format
func displayConsoleResults(results map[string][]core.Manga, query string, options core.SearchOptions) {
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
	fmt.Printf("Result limit: %d per page\n", options.Limit)
	if options.Pages > 0 {
		fmt.Printf("Pages fetched: %d\n", options.Pages)
	} else {
		fmt.Println("Pages fetched: all available")
	}

	// Display field-specific filters
	if len(options.Filters) > 0 {
		fmt.Println("Filters:")
		for field, value := range options.Filters {
			fmt.Printf("  %s: %s\n", field, value)
		}
	}

	// Display concurrency information
	fmt.Printf("Concurrency: %d\n", maxConcurrency)

	fmt.Printf("\nTotal results found: %d\n\n", totalCount)

	// Display results for each provider using the standardized display functions
	if len(results) == 1 && searchProvider != "" {
		// Display results for a single provider
		for providerID, mangaList := range results {
			prov, _ := appEngine.GetProvider(providerID)
			fmt.Printf("Results from %s (%s):\n", prov.ID(), prov.Name())
			fmt.Print(display.SearchResults(mangaList, prov))
		}
	} else {
		// Display results from all providers
		fmt.Println("Results across all providers:")
		for providerID, mangaList := range results {
			if len(mangaList) == 0 {
				continue
			}

			prov, _ := appEngine.GetProvider(providerID)
			fmt.Printf("\nFrom %s (%s):\n", prov.ID(), prov.Name())
			fmt.Print(display.SearchResults(mangaList, prov))
		}
	}
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Flags
	searchCmd.Flags().StringVar(&searchProvider, "provider", "", "Search using specific provider")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Limit number of results per page")
	searchCmd.Flags().IntVar(&searchPages, "pages", 1, "Number of pages to fetch")
	searchCmd.Flags().StringVar(&searchSort, "sort", "relevance", "Sort by (relevance, name, newest, updated)")
	searchCmd.Flags().StringSliceVar(&searchFields, "fields", []string{}, "Fields to search in (title, author, genre), empty means all")
	searchCmd.Flags().BoolVar(&includeAltTitles, "alt-titles", true, "Include alternative titles in search")
	searchCmd.Flags().BoolVar(&includeAllLangs, "all-langs", true, "Search across all language versions of titles")

	// Field-specific filters
	fieldFilters = make(map[string]string)
	searchCmd.Flags().StringToStringVar(&fieldFilters, "filter", nil, "Field-specific filters (e.g., --filter genre=fantasy,author=oda)")
}
