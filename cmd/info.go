package cmd

import (
	"Luminary/agents"
	"Luminary/utils"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ChapterInfo represents chapter data for API responses
type ChapterInfo struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Number float64   `json:"number"`
	Date   time.Time `json:"date"`
}

// MangaInfo represents manga data for API responses
type MangaInfo struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Agent       string        `json:"agent"`
	AgentName   string        `json:"agent_name"`
	Description string        `json:"description"`
	Authors     []string      `json:"authors,omitempty"`
	Status      string        `json:"status,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Chapters    []ChapterInfo `json:"chapters,omitempty"`
}

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

		// Get manga info
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		manga, err := agent.GetManga(ctx, mangaID)
		if err != nil {
			if apiMode {
				utils.OutputJSON("error", nil, err)
				return
			}

			fmt.Printf("Error retrieving manga: %v\n", err)
			return
		}

		if apiMode {
			// Create structured manga info for API response
			mangaInfo := MangaInfo{
				ID:          utils.FormatMangaID(agentID, mangaID),
				Title:       manga.Title,
				Agent:       agentID,
				AgentName:   agent.Name(),
				Description: manga.Description,
				Authors:     manga.Authors,
				Status:      manga.Status,
				Tags:        manga.Tags,
			}

			// Add chapters info
			for _, chapter := range manga.Chapters {
				chapterInfo := ChapterInfo{
					ID:     utils.FormatMangaID(agentID, chapter.ID),
					Title:  chapter.Title,
					Number: chapter.Number,
					Date:   chapter.Date,
				}
				mangaInfo.Chapters = append(mangaInfo.Chapters, chapterInfo)
			}

			// Output the manga info
			utils.OutputJSON("success", map[string]interface{}{
				"manga": mangaInfo,
			}, nil)
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

func init() {
	rootCmd.AddCommand(infoCmd)
}
