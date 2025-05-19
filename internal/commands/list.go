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
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var (
	listProvider string
	listLimit    int
	listPages    int
)

// MangaListItem represents a manga item for API responses
type MangaListItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Provider     string `json:"provider"`
	ProviderName string `json:"provider_name,omitempty"`
}

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
		if timeoutDuration > time.Minute && !apiMode {
			fmt.Printf("Note: Using extended timeout of %v for this request.\n",
				timeoutDuration.Round(time.Second))
		}

		// Validate the provider if specified
		var selectedProvider provider.Provider
		if listProvider != "" {
			var exists bool
			selectedProvider, exists = appEngine.GetProvider(listProvider)
			if !exists {
				if apiMode {
					util.OutputJSON("error", nil, fmt.Errorf("provider '%s' not found", listProvider))
					return
				}

				fmt.Printf("Error: Provider '%s' not found\n", listProvider)
				fmt.Println("Available providers:")
				for _, a := range appEngine.AllProvider() {
					fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
				}
				return
			}
		}

		// Create search options using the engine type
		options := core.SearchOptions{
			Limit: listLimit,
			Pages: listPages,
			// We use empty search to get a list of manga
		}

		if apiMode {
			var allMangas []MangaListItem
			var mu sync.Mutex
			var wg sync.WaitGroup
			errorChan := make(chan error, len(appEngine.AllProvider()))

			// Create semaphore for concurrency control
			concurrency := maxConcurrency
			if concurrency <= 0 {
				concurrency = 5 // Default fallback
			}
			semaphore := make(chan struct{}, concurrency)

			// Function to list manga from a single provider concurrently
			listProviderMangas := func(provider provider.Provider) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release semaphore

				// Use empty search to get list of manga
				mangas, err := provider.Search(ctx, "", options)
				if err != nil {
					errorChan <- fmt.Errorf("error from provider %s: %w", provider.ID(), err)
					return
				}

				// Safely add results to the shared slice
				mu.Lock()
				for _, manga := range mangas {
					mangaItem := MangaListItem{
						ID:           core.FormatMangaID(provider.ID(), manga.ID),
						Title:        manga.Title,
						Provider:     provider.ID(),
						ProviderName: provider.Name(),
					}
					allMangas = append(allMangas, mangaItem)
				}
				mu.Unlock()
			}

			if selectedProvider != nil {
				wg.Add(1)
				semaphore <- struct{}{} // Acquire semaphore
				go listProviderMangas(selectedProvider)
			} else {
				// List manga from all providers concurrently
				for _, provider := range appEngine.AllProvider() {
					wg.Add(1)
					semaphore <- struct{}{} // Acquire semaphore
					go listProviderMangas(provider)
				}
			}

			// Wait for all operations to complete
			wg.Wait()
			close(errorChan)

			// Process any errors
			var errors []error
			for err := range errorChan {
				errors = append(errors, err)
			}

			// Create response data with provider filter info if applicable
			responseData := map[string]interface{}{
				"mangas": allMangas,
				"count":  len(allMangas),
			}

			if selectedProvider != nil {
				responseData["provider"] = selectedProvider.ID()
				responseData["provider_name"] = selectedProvider.Name()
			}

			util.OutputJSON("success", responseData, nil)
		} else {
			if selectedProvider != nil {
				// Single provider case - no need for parallelism
				fmt.Printf("Listing manga from provider: %s (%s)\n", selectedProvider.ID(), selectedProvider.Name())
				fmt.Printf("Limit: %d manga per page\n", options.Limit)
				if options.Pages > 0 {
					fmt.Printf("Pages: %d\n", options.Pages)
				} else {
					fmt.Println("Pages: all available")
				}
				fmt.Println()

				// Use empty search to get the list of manga
				mangas, err := selectedProvider.Search(ctx, "", options)
				if err != nil {
					handleListError(err, selectedProvider.ID(), selectedProvider.Name(), false)
					return
				}

				displayMangaList(mangas, selectedProvider)
			} else {
				// Multiple providers - use parallel processing
				fmt.Println("Listing manga from all providers:")
				fmt.Printf("Limit: %d manga per page\n", options.Limit)
				if options.Pages > 0 {
					fmt.Printf("Pages: %d\n", options.Pages)
				} else {
					fmt.Println("Pages: all available")
				}
				fmt.Printf("Concurrency: %d\n", maxConcurrency)
				fmt.Println("\nFetching data from providers in parallel. Results will appear as they complete...")

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

						fmt.Printf("\n--- From provider: %s (%s) ---\n", p.ID(), p.Name())

						if err != nil {
							// Show error but continue with other providers
							handleListError(err, p.ID(), p.Name(), true)
							return
						}

						// Display results
						displayMangaList(mangas, p)
					}(prov)
				}

				// Wait for all providers to complete
				wg.Wait()
				fmt.Println("\nAll providers completed processing.")
			}
		}
	},
}

// handleListError provides user-friendly error messages based on error type
func handleListError(err error, providerID, providerName string, continueOnError bool) {
	// For API mode or when we should continue despite errors
	if apiMode || continueOnError {
		if apiMode {
			// In API mode, we generally don't report errors for individual providers,
			// but we could add them to a "errors" section in the response if needed
			return
		}

		// For CLI with the continue flag, show brief error but continue
		if errors.IsServerError(err) {
			fmt.Printf("  Error: Server error from %s. Skipping.\n", providerName)
		} else if errors.Is(err, errors.ErrRateLimit) {
			fmt.Printf("  Error: Rate limit exceeded for %s. Skipping.\n", providerName)
		} else if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("  Error: Timeout fetching from %s. Skipping.\n", providerName)
		} else {
			fmt.Printf("  Error: %v\n", err)
		}
		return
	}

	// For CLI mode with a single provider (no continuation)
	if errors.IsServerError(err) {
		fmt.Printf("Error: Server error from %s. Please try again later.\n", providerName)
	} else if errors.Is(err, errors.ErrRateLimit) {
		fmt.Printf("Error: Rate limit exceeded for %s. Please try again later.\n", providerName)
	} else if errors.Is(err, context.DeadlineExceeded) {
		fmt.Printf("Error: Timeout while fetching manga list. Try reducing the number of pages.\n")
	} else {
		fmt.Printf("Error: %v\n", err)
	}
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

// Helper function to display a manga list in a user-friendly format
func displayMangaList(mangas []core.Manga, provider provider.Provider) {
	// Use the standardized engine display function
	output := display.MangaList(mangas, provider)
	fmt.Print(output)
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Flags
	listCmd.Flags().StringVar(&listProvider, "provider", "", "Specific provider to list manga from (default: all)")
	listCmd.Flags().IntVar(&listLimit, "limit", 50, "Limit number of results per page (limit 0 for all)")
	listCmd.Flags().IntVar(&listPages, "pages", 1, "Number of pages to fetch (0 for all pages)")
}
