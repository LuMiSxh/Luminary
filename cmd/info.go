package cmd

import (
	"Luminary/agents"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [agent:manga-id]",
	Short: "Get detailed information about a manga",
	Long:  `Get comprehensive information about a manga, including all chapters.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Parse the manga ID format "agent:id"
		parts := strings.SplitN(args[0], ":", 2)
		if len(parts) != 2 {
			fmt.Println("Error: Invalid manga ID format, must be 'agent:id'")
			return
		}

		agentID, mangaID := parts[0], parts[1]

		// Get the agent
		agent := agents.Get(agentID)
		if agent == nil {
			fmt.Printf("Error: Agent '%s' not found\n", agentID)
			fmt.Println("Available agents:")
			for _, a := range agents.All() {
				fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
			}
			return
		}

		// Get manga info
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		manga, err := agent.GetManga(ctx, mangaID)
		if err != nil {
			fmt.Printf("Error retrieving manga: %v\n", err)
			return
		}

		if apiMode {
			// Output machine-readable JSON
			fmt.Printf(`{"manga":{"id":"%s","title":"%s","agent":"%s","description":"%s"`,
				mangaID, manga.Title, agentID, manga.Description)

			// Output authors if available
			if len(manga.Authors) > 0 {
				fmt.Printf(`,"authors":["%s"]`, strings.Join(manga.Authors, `","`))
			}

			// Output chapters
			fmt.Print(`,"chapters":[`)
			for i, chapter := range manga.Chapters {
				if i > 0 {
					fmt.Print(",")
				}
				fmt.Printf(`{"id":"%s","title":"%s","number":%g,"date":"%s"}`,
					chapter.ID, chapter.Title, chapter.Number, chapter.Date.Format(time.RFC3339))
			}
			fmt.Println(`]}}`)
		} else {
			// Interactive mode for CLI users
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
				fmt.Printf("- %s (Chapter %g, %s)\n",
					chapter.Title,
					chapter.Number,
					chapter.Date.Format("2006-01-02"))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
