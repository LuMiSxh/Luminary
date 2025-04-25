package cmd

import (
	"Luminary/pkg"
	"fmt"
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
		for _, combinedID := range args {
			agentID, id := pkg.ParseAgentID(combinedID)
			if agentID == "" {
				fmt.Println("Error: invalid chapter ID format, must be 'agent:id'")
				return
			}

			// Validate that the agent exists
			agent := pkg.GetAgentByID(agentID)
			if agent == nil {
				fmt.Printf("Error: Agent '%s' not found\n", agentID)
				fmt.Println("Available agents:")
				for _, a := range pkg.GetAgents() {
					fmt.Printf("  - %s (%s)\n", a.ID, a.Name)
				}
				return
			}

			// Process download with extracted agent and ID
			if apiMode {
				// Output machine-readable JSON for Palaxy
				fmt.Printf(`{"status":"downloading","chapter_id":"%s","agent":"%s","agent_name":"%s"}`,
					id, agent.ID, agent.Name)
			} else {
				// Interactive mode for CLI users
				fmt.Printf("Downloading chapter %s from agent %s (%s)...\n", id, agent.ID, agent.Name)
				fmt.Printf("Output directory: %s\n", downloadOutput)
				fmt.Printf("Image format: %s\n", downloadFormat)
				fmt.Printf("Concurrent downloads: %d\n", downloadConcurrent)
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
