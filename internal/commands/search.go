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
	"Luminary/pkg/cli"
	"Luminary/pkg/engine/core"
	"Luminary/pkg/provider"
	"context"
	"fmt"
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
		if timeoutDuration > time.Minute {
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

		// Use the unified formatter
		formatter := cli.DefaultFormatter

		// Determine which providers to search
		var providers []provider.Provider
		if searchProvider != "" {
			// Validate the provider if specified
			if !appEngine.ProviderExists(searchProvider) {
				formatter.PrintError(fmt.Sprintf("Error: Provider '%s' not found", searchProvider))
				formatter.PrintInfo("Available providers:")
				for _, a := range appEngine.AllProvider() {
					formatter.PrintDetail("  "+a.ID(), a.Name())
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

		// Display search parameters
		formatter.PrintSearchInfo(
			query,
			options,
			includeAltTitles,
			includeAllLangs,
			maxConcurrency,
		)

		// Execute the search using the search service
		results, err := appEngine.Search.SearchAcrossProviders(
			ctx,
			providers,
			query,
			options,
		)

		if formatter.HandleError(err) {
			return
		}

		// Display search results
		formatter.PrintSearchResults(results, query, options)
	},
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
