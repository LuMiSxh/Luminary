package commands

import (
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/display"
	"Luminary/pkg/errors"
	"Luminary/pkg/util"
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [provider:manga-id]",
	Short: "Get detailed information about a manga",
	Long:  `Get comprehensive information about a manga, including all chapters.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		providerID, mangaID, err := core.ParseMangaID(args[0])
		if err != nil {
			if apiMode {
				util.OutputJSON("error", nil, err)
				return
			}
			fmt.Println("Error: Invalid manga ID format, must be 'provider:id'")
			return
		}

		prov, exists := appEngine.GetProvider(providerID)
		if !exists {
			if apiMode {
				util.OutputJSON("error", nil, fmt.Errorf("provider '%s' not found", providerID))
				return
			}
			fmt.Printf("Error: Provider '%s' not found\n", providerID)
			fmt.Println("Available providers:")
			for _, p := range appEngine.AllProvider() {
				fmt.Printf("  - %s (%s)\n", p.ID(), p.Name())
			}
			return
		}

		timeoutDuration := 60 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		manga, err := prov.GetManga(ctx, mangaID)
		if err != nil {
			handleMangaError(err, providerID, mangaID, prov.Name())
			return
		}

		if manga == nil || manga.Title == "" {
			if apiMode {
				util.OutputJSON("error", nil, fmt.Errorf("retrieved empty or invalid manga data"))
			} else {
				fmt.Println("Error: Retrieved empty or invalid manga data")
			}
			return
		}
		if manga.Chapters == nil {
			manga.Chapters = []core.ChapterInfo{}
		}

		if apiMode {
			chapters := make([]map[string]interface{}, len(manga.Chapters))
			for i, ch := range manga.Chapters {
				chapters[i] = map[string]interface{}{
					"id":     core.FormatMangaID(providerID, ch.ID),
					"title":  ch.Title,
					"number": ch.Number,
					"date":   ch.Date,
				}
			}

			apiResponse := map[string]interface{}{
				"manga": map[string]interface{}{
					"id":            core.FormatMangaID(providerID, manga.ID),
					"title":         manga.Title,
					"provider":      providerID,
					"provider_name": prov.Name(),
					"description":   manga.Description,
					"authors":       manga.Authors,
					"status":        manga.Status,
					"tags":          manga.Tags,
					"chapters":      chapters,
					"chapter_count": len(manga.Chapters),
				},
			}
			util.OutputJSON("success", apiResponse, nil)

		} else {
			displayOptions := display.Options{
				Level:            display.Detailed,
				IncludeAltTitles: true,
				ShowTags:         true,
				ItemLimit:        0,
				Indent:           "  ",
				Prefix:           "",
			}

			// Call the refactored MangaInfo function with the options
			outputString := display.MangaInfo(manga, prov, displayOptions)
			fmt.Print(outputString) // Print the formatted string
		}
	},
}

// handleMangaError provides user-friendly error messages based on error type
func handleMangaError(err error, providerID, mangaID, providerName string) {
	// Check for specific error types in order of specificity
	if errors.IsNotFound(err) {
		// Not found error
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("manga '%s' not found on %s", mangaID, providerName))
		} else {
			fmt.Printf("Error: Manga '%s' not found on %s\n", mangaID, providerName)
		}
		return
	}

	// Check for server errors
	if errors.IsServerError(err) {
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("server error from %s: %v", providerName, err))
		} else {
			fmt.Printf("Error: Server error from %s. Please try again later.\n", providerName)
		}
		return
	}

	// Check for rate limiting
	if errors.Is(err, errors.ErrRateLimit) {
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("rate limit exceeded for %s", providerName))
		} else {
			fmt.Printf("Error: Rate limit exceeded for %s. Please try again later.\n", providerName)
		}
		return
	}

	// Generic error handling
	if apiMode {
		util.OutputJSON("error", nil, err)
	} else {
		fmt.Printf("Error retrieving manga: %v\n", err)
	}
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
