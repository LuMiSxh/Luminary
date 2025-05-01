package cmd

import (
	"Luminary/engine"
	"Luminary/errors" // Import our custom errors package
	"Luminary/utils"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var (
	listAgent     string
	listLimit     int
	listPages     int
	listDebugMode bool // Debug flag for detailed error information
)

// MangaListItem represents a manga item for API responses
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
		// Determine if we're using multiple agents
		multipleAgents := listAgent == ""

		// Calculate appropriate timeout
		timeoutDuration := calculateListTimeout(listLimit, listPages, multipleAgents)

		// Create context with dynamic timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// Set the Concurrency limit for the engine
		ctx = engine.WithConcurrency(ctx, maxConcurrency)

		// Inform user about extended timeout if applicable
		if timeoutDuration > time.Minute && !apiMode {
			fmt.Printf("Note: Using extended timeout of %v for this request.\n",
				timeoutDuration.Round(time.Second))
		}

		// Validate the agent if specified
		var selectedAgent engine.Agent
		if listAgent != "" {
			var exists bool
			selectedAgent, exists = appEngine.GetAgent(listAgent)
			if !exists {
				if apiMode {
					utils.OutputJSON("error", nil, fmt.Errorf("agent '%s' not found", listAgent))
					return
				}

				fmt.Printf("Error: Agent '%s' not found\n", listAgent)
				fmt.Println("Available agents:")
				for _, a := range appEngine.AllAgents() {
					fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
				}
				return
			}
		}

		// Create search options using the engine type
		options := engine.SearchOptions{
			Limit: listLimit,
			Pages: listPages,
			// We use empty search to get a list of manga
		}

		if apiMode {
			var allMangas []MangaListItem
			var mu sync.Mutex
			var wg sync.WaitGroup
			errorChan := make(chan error, len(appEngine.AllAgents()))

			// Create semaphore for concurrency control
			concurrency := maxConcurrency
			if concurrency <= 0 {
				concurrency = 5 // Default fallback
			}
			semaphore := make(chan struct{}, concurrency)

			// Function to list manga from a single agent concurrently
			listAgentMangas := func(agent engine.Agent) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release semaphore

				// Use empty search to get list of manga
				mangas, err := agent.Search(ctx, "", options)
				if err != nil {
					errorChan <- fmt.Errorf("error from agent %s: %w", agent.ID(), err)
					return
				}

				// Safely add results to the shared slice
				mu.Lock()
				for _, manga := range mangas {
					mangaItem := MangaListItem{
						ID:        utils.FormatMangaID(agent.ID(), manga.ID),
						Title:     manga.Title,
						Agent:     agent.ID(),
						AgentName: agent.Name(),
					}
					allMangas = append(allMangas, mangaItem)
				}
				mu.Unlock()
			}

			if selectedAgent != nil {
				wg.Add(1)
				semaphore <- struct{}{} // Acquire semaphore
				go listAgentMangas(selectedAgent)
			} else {
				// List manga from all agents concurrently
				for _, agent := range appEngine.AllAgents() {
					wg.Add(1)
					semaphore <- struct{}{} // Acquire semaphore
					go listAgentMangas(agent)
				}
			}

			// Wait for all operations to complete
			wg.Wait()
			close(errorChan)

			// Process any errors
			var errors []error
			for err := range errorChan {
				errors = append(errors, err)
				// Handle errors but continue
				if listDebugMode {
					fmt.Printf("Debug: %v\n", err)
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
			if selectedAgent != nil {
				// Single agent case - no need for parallelism
				fmt.Printf("Listing manga from agent: %s (%s)\n", selectedAgent.ID(), selectedAgent.Name())
				fmt.Printf("Limit: %d manga per page\n", options.Limit)
				if options.Pages > 0 {
					fmt.Printf("Pages: %d\n", options.Pages)
				} else {
					fmt.Println("Pages: all available")
				}
				fmt.Println()

				// Use empty search to get the list of manga
				mangas, err := selectedAgent.Search(ctx, "", options)
				if err != nil {
					handleListError(err, selectedAgent.ID(), selectedAgent.Name(), false)
					return
				}

				displayMangaList(mangas, selectedAgent)
			} else {
				// Multiple agents - use parallel processing
				fmt.Println("Listing manga from all agents:")
				fmt.Printf("Limit: %d manga per page\n", options.Limit)
				if options.Pages > 0 {
					fmt.Printf("Pages: %d\n", options.Pages)
				} else {
					fmt.Println("Pages: all available")
				}
				fmt.Printf("Concurrency: %d\n", maxConcurrency)
				fmt.Println("\nFetching data from agents in parallel. Results will appear as they complete...")

				// Create synchronization primitives
				var wg sync.WaitGroup
				var mu sync.Mutex
				concurrency := maxConcurrency
				if concurrency <= 0 {
					concurrency = 3 // Default fallback
				}
				semaphore := make(chan struct{}, concurrency)

				// Process each agent concurrently
				for _, agent := range appEngine.AllAgents() {
					wg.Add(1)
					semaphore <- struct{}{} // Acquire semaphore

					go func(a engine.Agent) {
						defer wg.Done()
						defer func() { <-semaphore }() // Release semaphore

						// Use empty search to get list of manga
						mangas, err := a.Search(ctx, "", options)

						// Lock while we print to avoid interleaved output
						mu.Lock()
						defer mu.Unlock()

						fmt.Printf("\n--- From agent: %s (%s) ---\n", a.ID(), a.Name())

						if err != nil {
							// Show error but continue with other agents
							handleListError(err, a.ID(), a.Name(), true)
							return
						}

						if len(mangas) == 0 {
							fmt.Println("No manga found.")
							return
						}

						// Display results
						displayMangaList(mangas, a)
					}(agent)
				}

				// Wait for all agents to complete
				wg.Wait()
				fmt.Println("\nAll agents completed processing.")
			}
		}
	},
}

// handleListError provides user-friendly error messages based on error type
func handleListError(err error, agentID, agentName string, continueOnError bool) {
	// For API mode or when we should continue despite errors
	if apiMode || continueOnError {
		if apiMode {
			// In API mode, we generally don't report errors for individual agents
			// but we could add them to a "errors" section in the response if needed
			return
		}

		// For CLI with continue flag, show brief error but continue
		if errors.IsServerError(err) {
			fmt.Printf("  Error: Server error from %s. Skipping.\n", agentName)
		} else if errors.Is(err, errors.ErrRateLimit) {
			fmt.Printf("  Error: Rate limit exceeded for %s. Skipping.\n", agentName)
		} else if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("  Error: Timeout fetching from %s. Skipping.\n", agentName)
		} else {
			fmt.Printf("  Error: %v\n", err)
		}

		if listDebugMode {
			fmt.Printf("  Debug details: %v\n", err)
		}
		return
	}

	// For CLI mode with a single agent (no continuation)
	if errors.IsServerError(err) {
		fmt.Printf("Error: Server error from %s. Please try again later.\n", agentName)
	} else if errors.Is(err, errors.ErrRateLimit) {
		fmt.Printf("Error: Rate limit exceeded for %s. Please try again later.\n", agentName)
	} else if errors.Is(err, context.DeadlineExceeded) {
		fmt.Printf("Error: Timeout while fetching manga list. Try reducing the number of pages.\n")
	} else {
		fmt.Printf("Error: %v\n", err)
	}

	if listDebugMode {
		// Print more detailed error info in debug mode
		fmt.Println("\nDebug error details:")
		fmt.Printf("  Agent: %s\n", agentID)
		fmt.Printf("  Error type: %T\n", err)
		fmt.Printf("  Full error: %+v\n", err)
	}
}

// Calculate an appropriate timeout based on pagination parameters for list command
func calculateListTimeout(limit, pages int, multipleAgents bool) time.Duration {
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

	// Add extra time if querying multiple agents
	if multipleAgents {
		timeoutDuration += 2 * time.Minute
	}

	return timeoutDuration
}

// Helper function to display a manga list in a user-friendly format
func displayMangaList(mangas []engine.Manga, agent engine.Agent) {
	// Use the standardized engine display function
	output := engine.DisplayMangaList(mangas, agent)
	fmt.Print(output)
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Flags
	listCmd.Flags().StringVar(&listAgent, "agent", "", "Specific agent to list manga from (default: all)")
	listCmd.Flags().IntVar(&listLimit, "limit", 50, "Limit number of results per page (limit 0 for all)")
	listCmd.Flags().IntVar(&listPages, "pages", 1, "Number of pages to fetch (0 for all pages)")
	listCmd.Flags().BoolVar(&listDebugMode, "debug", false, "Show detailed error information")
}
