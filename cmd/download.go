package cmd

import (
	"Luminary/agents"
	"Luminary/utils"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	downloadOutput     string
	downloadConcurrent int
	downloadVolume     int  // Volume flag
	downloadHasVolume  bool // Track if the volume flag was provided
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
		ctx := agents.WithConcurrency(baseCtx, downloadConcurrent)

		// Add volume override to context if provided
		if downloadHasVolume {
			ctx = context.WithValue(ctx, "volume_override", downloadVolume)
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
			agent := agents.Get(agentID)
			if agent == nil {
				if apiMode {
					utils.OutputJSON("error", nil, fmt.Errorf("agent '%s' not found", agentID))
					return
				}

				fmt.Printf("Error: Agent '%s' not found\n", agentID)
				fmt.Println("Available agents:")
				for _, a := range agents.All() {
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
					Concurrent: downloadConcurrent,
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
				fmt.Printf("Concurrent downloads: %d\n", downloadConcurrent)

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
				if apiMode {
					utils.OutputJSON("error", nil, err)
					return
				}

				_, _ = fmt.Fprintf(os.Stderr, "Error downloading chapter: %v\n", err)
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

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Flags
	downloadCmd.Flags().StringVar(&downloadOutput, "output", "./downloads", "Output directory")
	downloadCmd.Flags().IntVar(&downloadConcurrent, "concurrent", 5, "Number of concurrent downloads")

	// Add volume flag
	downloadCmd.Flags().IntVar(&downloadVolume, "vol", 0, "Set or override the volume number")

	// Hook to track when the volume flag is explicitly set
	downloadCmd.PreRun = func(cmd *cobra.Command, args []string) {
		downloadHasVolume = cmd.Flags().Changed("vol")
	}
}
