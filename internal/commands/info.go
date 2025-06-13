// internal/commands/info.go
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
	"Luminary/pkg/util"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	infoLanguageFilter string
	infoShowLanguages  bool
)

var infoCmd = &cobra.Command{
	Use:   "info [provider:manga-id]",
	Short: "Get detailed information about a manga",
	Long:  `Get comprehensive information about a manga, including all chapters.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		providerID, mangaID, err := core.ParseMangaID(args[0])
		if err != nil {
			fmt.Println("Error: Invalid manga ID format, must be 'provider:id'")
			return
		}

		prov, exists := appEngine.GetProvider(providerID)
		if !exists {
			providerErr := fmt.Errorf("provider '%s' not found", providerID)
			if _, err := fmt.Fprintf(os.Stderr, "%s\n", errors.FormatCLI(errors.T(providerErr))); err != nil {
				return
			}
			return
		}

		timeoutDuration := 60 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		manga, err := prov.GetManga(ctx, mangaID)
		if err != nil {
			if _, err := fmt.Fprintln(os.Stderr, errors.FormatCLI(err)); err != nil {
				return
			}
			return
		}

		if manga == nil || manga.Title == "" {
			providerErr := fmt.Errorf("manga '%s' not found on provider '%s'", mangaID, prov.Name())
			if _, err := fmt.Fprintf(os.Stderr, "%s\n", errors.FormatCLI(errors.T(providerErr))); err != nil {
				return
			}
			return
		}
		if manga.Chapters == nil {
			manga.Chapters = []core.ChapterInfo{}
		}

		// Handle language filtering
		originalChapterCount := len(manga.Chapters)
		if infoLanguageFilter != "" {
			languageFilter := util.NewLanguageFilter(infoLanguageFilter)
			if languageFilter != nil {
				manga.Chapters = languageFilter.FilterChapters(manga.Chapters)

				// Show filtering results
				if len(manga.Chapters) == 0 {
					fmt.Printf("No chapters found matching language filter: %s\n", infoLanguageFilter)

					// Re-fetch to get all chapters for available languages display
					originalManga, refetchErr := prov.GetManga(ctx, mangaID)
					if refetchErr == nil && originalManga != nil {
						fmt.Printf("Available languages in this manga: %s\n", util.FormatAvailableLanguages(originalManga.Chapters))
					} else {
						fmt.Println("Available languages: Could not retrieve language information")
					}
					return
				} else if len(manga.Chapters) < originalChapterCount {
					fmt.Printf("Showing %d of %d chapters (filtered by language: %s)\n",
						len(manga.Chapters), originalChapterCount, infoLanguageFilter)
				}
			}
		}

		// Show available languages if requested
		if infoShowLanguages {
			// Need to get original chapters for a language list if filtering was applied
			if infoLanguageFilter != "" {
				// Re-fetch to get all chapters for language display
				originalManga, err := prov.GetManga(ctx, mangaID)
				if err == nil && originalManga != nil {
					fmt.Printf("Available languages: %s\n\n", util.FormatAvailableLanguages(originalManga.Chapters))
				}
			} else {
				fmt.Printf("Available languages: %s\n\n", util.FormatAvailableLanguages(manga.Chapters))
			}
		}

		displayOptions := display.Options{
			Level:            display.Detailed,
			IncludeAltTitles: true,
			ShowTags:         true,
			ItemLimit:        0,
			Indent:           "  ",
			Prefix:           "",
		}

		fmt.Print(display.MangaInfo(manga, prov, displayOptions))
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Add language filtering flags
	infoCmd.Flags().StringVar(&infoLanguageFilter, "language", "", "Filter chapters by language (comma-separated codes/names, e.g., 'en,ja' or 'english,japanese')")
	infoCmd.Flags().StringVar(&infoLanguageFilter, "lang", "", "Alias for --language")
	infoCmd.Flags().BoolVar(&infoShowLanguages, "show-languages", false, "Show available languages for this manga")

	// Mark the lang flag as an alias (hidden from help)
	err := infoCmd.Flags().MarkHidden("lang")
	if err != nil {
		return
	}
}
