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
	"Luminary/pkg/errors"
	"Luminary/pkg/provider"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var (
	listProvider string
	listLimit    int
	listPages    int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available manga",
	Long:  `List all manga from all providers or a specific provider.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Determine if we're using multiple providers
		multipleProviders := listProvider == ""

		// Calculate appropriate timeout
		timeoutDuration := calculateListTimeout(listLimit, listPages, multipleProviders)

		// Create context with dynamic timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// Set the Concurrency limit for the engine
		ctx = core.WithConcurrency(ctx, maxConcurrency)

		// Inform user about extended timeout if applicable
		if timeoutDuration > time.Minute {
			fmt.Printf("Note: Using extended timeout of %v for this request.\n",
				timeoutDuration.Round(time.Second))
		}

		// Validate the provider if specified
		var selectedProvider provider.Provider
		if listProvider != "" {
			var exists bool
			selectedProvider, exists = appEngine.GetProvider(listProvider)
			if !exists {
				providerErr := fmt.Errorf("provider '%s' not found", listProvider)
				formatter.HandleError(providerErr)
				return
			}
		}

		// Create search options using the engine type
		options := core.SearchOptions{
			Limit: listLimit,
			Pages: listPages,
			// We use empty search to get a list of manga
		}

		// Use the unified formatter
		formatter := cli.DefaultFormatter

		if selectedProvider != nil {
			// Single provider case - no need for parallelism
			formatter.PrintHeader(fmt.Sprintf("Manga from %s (%s)", selectedProvider.ID(), selectedProvider.Name()))
			formatter.PrintDetail("Limit", fmt.Sprintf("%d manga per page", options.Limit))

			if options.Pages > 0 {
				formatter.PrintDetail("Pages", fmt.Sprintf("%d", options.Pages))
			} else {
				formatter.PrintDetail("Pages", "all available")
			}

			formatter.PrintNewLine()

			// Use empty search to get the list of manga
			mangas, err := selectedProvider.Search(ctx, "", options)
			if formatter.HandleError(err) {
				return
			}

			formatter.PrintMangaList(mangas, selectedProvider, "")
		} else {
			// Multiple providers - use parallel processing
			formatter.PrintHeader("Manga from All Providers")
			formatter.PrintDetail("Limit", fmt.Sprintf("%d manga per page", options.Limit))

			if options.Pages > 0 {
				formatter.PrintDetail("Pages", fmt.Sprintf("%d", options.Pages))
			} else {
				formatter.PrintDetail("Pages", "all available")
			}

			formatter.PrintDetail("Concurrency", fmt.Sprintf("%d", maxConcurrency))
			formatter.PrintInfo("Fetching data from providers in parallel. Results will appear as they complete...")

			// Create synchronization primitives
			var wg sync.WaitGroup
			var mu sync.Mutex
			concurrency := maxConcurrency
			if concurrency <= 0 {
				concurrency = 3 // Default fallback
			}
			semaphore := make(chan struct{}, concurrency)

			// Process each provider concurrently
			for _, prov := range appEngine.AllProvider() {
				wg.Add(1)
				semaphore <- struct{}{} // Acquire semaphore

				go func(p provider.Provider) {
					defer wg.Done()
					defer func() { <-semaphore }() // Release semaphore

					// Use empty search to get list of manga
					mangas, err := p.Search(ctx, "", options)

					// Lock while we print to avoid interleaved output
					mu.Lock()
					defer mu.Unlock()

					formatter.PrintSection(fmt.Sprintf("From provider: %s (%s)", p.ID(), p.Name()))

					if err != nil {
						formatter.PrintError(errors.FormatCLI(err))
						return
					}

					formatter.PrintMangaList(mangas, p, "")
				}(prov)
			}

			// Wait for all providers to complete
			wg.Wait()
			fmt.Println("\nAll providers completed processing.")
		}
	},
}

// Calculate an appropriate timeout based on pagination parameters for list command
func calculateListTimeout(limit, pages int, multipleProviders bool) time.Duration {
	// Base timeout
	timeoutDuration := 60 * time.Second

	// Adjust for pagination parameters
	if limit == 0 || pages == 0 {
		// For unlimited requests, start with a higher base timeout
		timeoutDuration = 5 * time.Minute

		// If both are unlimited, use a maximum timeout
		if limit == 0 && pages == 0 {
			timeoutDuration = 10 * time.Minute
		}
	} else if pages > 3 || limit > 50 {
		// For larger paginated requests
		timeoutDuration = 3 * time.Minute

		// Scale with number of pages
		if pages > 5 {
			pageTimeoutFactor := time.Duration(pages) / 5
			if pageTimeoutFactor > 1 {
				extraTimeout := pageTimeoutFactor * time.Minute
				if extraTimeout > 5*time.Minute {
					extraTimeout = 5 * time.Minute // Cap at 5 extra minutes
				}
				timeoutDuration += extraTimeout
			}
		}
	}

	// Add extra time if querying multiple providers
	if multipleProviders {
		timeoutDuration += 2 * time.Minute
	}

	return timeoutDuration
}
