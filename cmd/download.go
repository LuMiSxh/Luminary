package cmd

import (
	"Luminary/agents"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	downloadOutput     string
	downloadFormat     string
	downloadConcurrent int
)

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

		for _, combinedID := range args {
			// Parse the chapter ID format "agent:id"
			parts := strings.SplitN(combinedID, ":", 2)
			if len(parts) != 2 {
				fmt.Println("Error: invalid chapter ID format, must be 'agent:id'")
				return
			}

			agentID, chapterID := parts[0], parts[1]

			// Validate that the agent exists
			agent := agents.Get(agentID)
			if agent == nil {
				fmt.Printf("Error: Agent '%s' not found\n", agentID)
				fmt.Println("Available agents:")
				for _, a := range agents.All() {
					fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
				}
				return
			}

			if apiMode {
				// Output machine-readable JSON for Palaxy
				fmt.Printf(`{"status":"downloading","chapter_id":"%s","agent":"%s","agent_name":"%s","concurrent":%d}`,
					chapterID, agent.ID(), agent.Name(), downloadConcurrent)
			} else {
				// Interactive mode for CLI users
				fmt.Printf("Downloading chapter %s from agent %s (%s)...\n",
					chapterID, agent.ID(), agent.Name())
				fmt.Printf("Output directory: %s\n", downloadOutput)
				fmt.Printf("Image format: %s\n", downloadFormat)
				fmt.Printf("Concurrent downloads: %d\n", downloadConcurrent)
			}

			// Create output directory with agent and chapter ID to keep downloads organized
			outputDir := fmt.Sprintf("%s/%s/%s", downloadOutput, agent.ID(), chapterID)

			// Create a context with timeout
			downloadCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)

			// Perform the download
			err := agent.DownloadChapter(downloadCtx, "", chapterID, outputDir)

			// Always cancel the context when done to release resources
			cancel()

			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error downloading chapter: %v\n", err)
				return
			}

			// If we're in CLI mode, provide a success message
			if !apiMode {
				fmt.Printf("Successfully downloaded chapter to %s\n", outputDir)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Flags
	downloadCmd.Flags().StringVar(&downloadOutput, "output", "./downloads", "Output directory")
	downloadCmd.Flags().StringVar(&downloadFormat, "format", "jpeg", "Image format (jpeg, png, webp)")
	downloadCmd.Flags().IntVar(&downloadConcurrent, "concurrent", 5, "Number of concurrent downloads")
}
