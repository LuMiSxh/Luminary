package cmd

import (
	"Luminary/engine"
	"Luminary/errors" // Import our custom errors package
	"Luminary/utils"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	downloadOutput    string
	downloadVolume    int  // Volume flag
	downloadHasVolume bool // Track if the volume flag was provided
	downloadDebugMode bool // Debug flag for detailed error information
)

// DownloadInfo represents download info for API responses
type DownloadInfo struct {
	ChapterID  string `json:"chapter_id"`
	Agent      string `json:"agent"`
	AgentName  string `json:"agent_name"`
	OutputDir  string `json:"output_dir"`
	Concurrent int    `json:"concurrent"`
	Volume     *int   `json:"volume,omitempty"` // Volume in API response
}

var downloadCmd = &cobra.Command{
	Use:   "download [chapter-ids...]",
	Short: "Download manga chapters",
	Long:  `Download one or more manga chapters by their IDs.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Create base context
		baseCtx := context.Background()

		// Create a context with concurrency settings
		ctx := engine.WithConcurrency(baseCtx, maxConcurrency)

		// Add volume override to context if provided
		if downloadHasVolume {
			ctx = engine.WithVolumeOverride(ctx, downloadVolume)
		}

		for _, combinedID := range args {
			// Parse the chapter ID format "agent:id"
			agentID, chapterID, err := utils.ParseMangaID(combinedID)
			if err != nil {
				if apiMode {
					utils.OutputJSON("error", nil, err)
					return
				}

				fmt.Println("Error: invalid chapter ID format, must be 'agent:id'")
				return
			}

			// Validate that the agent exists
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

			// Use the output directory directly - no intermediate folders
			outputDir := downloadOutput

			if apiMode {
				// Prepare download info for API response
				downloadInfo := DownloadInfo{
					ChapterID:  chapterID,
					Agent:      agent.ID(),
					AgentName:  agent.Name(),
					OutputDir:  outputDir,
					Concurrent: maxConcurrency,
				}

				// Add volume info if provided
				if downloadHasVolume {
					downloadInfo.Volume = &downloadVolume
				}

				utils.OutputJSON("downloading", downloadInfo, nil)
			} else {
				// Interactive mode for CLI users
				fmt.Printf("Downloading chapter %s from agent %s (%s)...\n",
					chapterID, agent.ID(), agent.Name())
				fmt.Printf("Output directory: %s\n", downloadOutput)
				fmt.Printf("Concurrent downloads: %d\n", maxConcurrency)

				// Print volume info if provided
				if downloadHasVolume {
					fmt.Printf("Volume override: %d\n", downloadVolume)
				}
			}

			// Create a context with timeout
			downloadCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)

			// Perform the download directly to the base output directory
			err = agent.DownloadChapter(downloadCtx, chapterID, outputDir)

			// Always cancel the context when done to release resources
			cancel()

			if err != nil {
				handleDownloadError(err, agentID, chapterID, agent.Name(), outputDir)
				return
			}

			// If we're in CLI mode, provide a success message
			if !apiMode {
				fmt.Printf("Successfully downloaded chapter to %s\n", outputDir)
			} else {
				// Report successful download in API mode
				utils.OutputJSON("success", map[string]interface{}{
					"message":    fmt.Sprintf("Successfully downloaded chapter %s", chapterID),
					"chapter_id": chapterID,
					"agent":      agent.ID(),
					"output_dir": outputDir,
				}, nil)
			}
		}
	},
}

// handleDownloadError provides user-friendly error messages based on error type
func handleDownloadError(err error, agentID, chapterID, agentName, outputDir string) {
	// Check for specific error types in order of specificity
	if errors.IsNotFound(err) {
		// Not found error
		if apiMode {
			utils.OutputJSON("error", nil, fmt.Errorf("chapter '%s' not found on %s", chapterID, agentName))
		} else {
			fmt.Printf("Error: Chapter '%s' not found on %s\n", chapterID, agentName)
		}
		return
	}

	// Check for server errors
	if errors.IsServerError(err) {
		if apiMode {
			utils.OutputJSON("error", nil, fmt.Errorf("server error from %s: %v", agentName, err))
		} else {
			fmt.Printf("Error: Server error from %s. Please try again later.\n", agentName)
			if downloadDebugMode {
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

	// File system errors
	var ioErr *os.PathError
	if errors.As(err, &ioErr) {
		if apiMode {
			utils.OutputJSON("error", nil, fmt.Errorf("file system error: %v", ioErr))
		} else {
			fmt.Printf("Error: Failed to access output directory '%s': %v\n", outputDir, ioErr)
			fmt.Println("Make sure the directory exists and you have write permissions.")
		}
		return
	}

	// Generic error handling
	if apiMode {
		utils.OutputJSON("error", nil, err)
	} else {
		fmt.Printf("Error downloading chapter: %v\n", err)
		if downloadDebugMode {
			// Print more detailed error info in debug mode
			fmt.Println("\nDebug error details:")
			fmt.Printf("  Agent: %s\n", agentID)
			fmt.Printf("  Chapter ID: %s\n", chapterID)
			fmt.Printf("  Error type: %T\n", err)
			fmt.Printf("  Full error: %+v\n", err)
		}
	}
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Flags
	downloadCmd.Flags().StringVar(&downloadOutput, "output", "./downloads", "Output directory")
	downloadCmd.Flags().BoolVar(&downloadDebugMode, "debug", false, "Show detailed error information")

	// Add volume flag
	downloadCmd.Flags().IntVar(&downloadVolume, "vol", 0, "Set or override the volume number")

	// Hook to track when the volume flag is explicitly set
	downloadCmd.PreRun = func(cmd *cobra.Command, args []string) {
		downloadHasVolume = cmd.Flags().Changed("vol")
	}
}
