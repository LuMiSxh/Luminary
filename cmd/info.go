package cmd

import (
	"Luminary/errors" // Add our custom errors package
	"Luminary/utils"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Add --debug flag for detailed error information when needed
var infoDebugMode bool

var infoCmd = &cobra.Command{
	Use:   "info [agent:manga-id]",
	Short: "Get detailed information about a manga",
	Long:  `Get comprehensive information about a manga, including all chapters.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Parse the manga ID format "agent:id"
		agentID, mangaID, err := utils.ParseMangaID(args[0])
		if err != nil {
			if apiMode {
				utils.OutputJSON("error", nil, err)
				return
			}

			fmt.Println("Error: Invalid manga ID format, must be 'agent:id'")
			return
		}

		// Get the agent
		agent, exists := appEngine.GetAgent(agentID)
		if !exists {
			if apiMode {
				utils.OutputJSON("error", nil, fmt.Errorf("agent '%s' not found", agentID))
				return
			}

			fmt.Printf("Error: Agent '%s' not found\n", agentID)
			fmt.Println("Available agents:")
			for _, a := range appEngine.AllAgents() {
				fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
			}
			return
		}

		// Calculate an appropriate timeout based on the operation
		// Fetching manga info with all chapters can be time-consuming
		timeoutDuration := 60 * time.Second

		// Create context with the calculated timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// Get manga info
		manga, err := agent.GetManga(ctx, mangaID)
		if err != nil {
			// Use our handleMangaError helper which handles different error types
			handleMangaError(err, agentID, mangaID, agent.Name())
			return
		}

		// Verify we got valid manga data
		if manga == nil || manga.Title == "" {
			if apiMode {
				utils.OutputJSON("error", nil, fmt.Errorf("retrieved empty or invalid manga data"))
			} else {
				fmt.Println("Error: Retrieved empty or invalid manga data")
			}
			return
		}

		if apiMode {
			// Format chapters for API output
			chapters := make([]map[string]interface{}, len(manga.Chapters))
			for i, ch := range manga.Chapters {
				chapters[i] = map[string]interface{}{
					"id":     utils.FormatMangaID(agentID, ch.ID),
					"title":  ch.Title,
					"number": ch.Number,
					"date":   ch.Date,
				}
			}

			apiResponse := map[string]interface{}{
				"manga": map[string]interface{}{
					"id":          utils.FormatMangaID(agentID, manga.ID),
					"title":       manga.Title,
					"agent":       agentID,
					"agent_name":  agent.Name(),
					"description": manga.Description,
					"authors":     manga.Authors,
					"status":      manga.Status,
					"tags":        manga.Tags,
					"chapters":    chapters,
				},
			}

			// Output the manga info
			utils.OutputJSON("success", apiResponse, nil)
		} else {
			// Interactive CLI output
			fmt.Printf("Manga: %s\n", manga.Title)
			fmt.Printf("ID: %s:%s\n", agentID, mangaID)

			if len(manga.Authors) > 0 {
				fmt.Printf("Authors: %s\n", strings.Join(manga.Authors, ", "))
			}

			if manga.Status != "" {
				fmt.Printf("Status: %s\n", manga.Status)
			}

			if len(manga.Tags) > 0 {
				fmt.Printf("Tags: %s\n", strings.Join(manga.Tags, ", "))
			}

			fmt.Printf("\nDescription:\n%s\n\n", manga.Description)

			// Display chapters
			fmt.Printf("Chapters (%d):\n", len(manga.Chapters))
			for _, chapter := range manga.Chapters {
				// Include the chapter ID in the output with the format: agent:chapterID
				fmt.Printf("- %s:%s: %s (Chapter %g, %s)\n",
					agentID,
					chapter.ID,
					chapter.Title,
					chapter.Number,
					chapter.Date.Format("2006-01-02"))
			}
		}
	},
}

// handleMangaError provides user-friendly error messages based on error type
func handleMangaError(err error, agentID, mangaID, agentName string) {
	// Check for specific error types in order of specificity
	if errors.IsNotFound(err) {
		// Not found error
		if apiMode {
			utils.OutputJSON("error", nil, fmt.Errorf("manga '%s' not found on %s", mangaID, agentName))
		} else {
			fmt.Printf("Error: Manga '%s' not found on %s\n", mangaID, agentName)
		}
		return
	}

	// Check for server errors
	if errors.IsServerError(err) {
		if apiMode {
			utils.OutputJSON("error", nil, fmt.Errorf("server error from %s: %v", agentName, err))
		} else {
			fmt.Printf("Error: Server error from %s. Please try again later.\n", agentName)
			if infoDebugMode {
				fmt.Printf("Debug details: %v\n", err)
			}
		}
		return
	}

	// Check for rate limiting
	if errors.Is(err, errors.ErrRateLimit) {
		if apiMode {
			utils.OutputJSON("error", nil, fmt.Errorf("rate limit exceeded for %s", agentName))
		} else {
			fmt.Printf("Error: Rate limit exceeded for %s. Please try again later.\n", agentName)
		}
		return
	}

	// Generic error handling
	if apiMode {
		utils.OutputJSON("error", nil, err)
	} else {
		fmt.Printf("Error retrieving manga: %v\n", err)
		if infoDebugMode {
			// Print more detailed error info in debug mode
			fmt.Println("\nDebug error details:")
			fmt.Printf("  Agent: %s\n", agentID)
			fmt.Printf("  Manga ID: %s\n", mangaID)
			fmt.Printf("  Error type: %T\n", err)
			fmt.Printf("  Full error: %+v\n", err)
		}
	}
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Add debug flag for detailed error information
	infoCmd.Flags().BoolVar(&infoDebugMode, "debug", false, "Show detailed error information")
}
