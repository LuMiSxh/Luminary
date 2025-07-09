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

package cli

import (
	"Luminary/pkg/core"
	"Luminary/pkg/engine"
	"Luminary/pkg/errors"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

// Color styles for consistent UI
var (
	// Headers and section titles
	headerStyle   = color.New(color.Bold, color.FgCyan)
	titleStyle    = color.New(color.Bold, color.FgWhite)
	subtitleStyle = color.New(color.FgHiWhite)
	sectionStyle  = color.New(color.Underline, color.FgHiCyan)

	// Content elements
	highlightStyle = color.New(color.FgMagenta)
	infoStyle      = color.New(color.FgBlue)
	successStyle   = color.New(color.FgGreen)
	warningStyle   = color.New(color.FgYellow)
	errorStyle     = color.New(color.FgRed)

	// Details and supplementary information
	labelStyle     = color.New(color.FgHiBlue)
	valueStyle     = color.New(color.FgWhite)
	secondaryStyle = color.New(color.FgHiBlack)

	// Decorations
	bulletStyle  = color.New(color.FgHiCyan)
	dividerColor = color.New(color.FgHiBlack)
)

// NewSearchCommand creates the search command
func NewSearchCommand(eng *engine.Engine) cli.ActionFunc {
	return func(c *cli.Context) error {
		if c.NArg() == 0 {
			return errors.New("search query is required").Error()
		}

		query := c.Args().First()
		provider := c.String("provider")
		limit := c.Int("limit")

		// Parse additional options from README examples
		fields := c.String("fields")
		filter := c.String("filter")
		sort := c.String("sort")

		eng.Logger.Debug("Search parameters: query=%s, provider=%s, limit=%d, fields=%s, filter=%s, sort=%s",
			query, provider, limit, fields, filter, sort)

		ctx := context.Background()
		options := core.SearchOptions{
			Query: query,
			Limit: limit,
			Pages: 1,
		}

		// Add fields if specified
		if fields != "" {
			options.Fields = strings.Split(fields, ",")
			eng.Logger.Debug("Using fields: %v", options.Fields)
		}

		// Add filters if specified
		if filter != "" {
			// Simple parsing for filter string
			eng.Logger.Debug("Filter string: %s", filter)
			// Future: Implement filter parsing
		}

		// Add sort if specified
		if sort != "" {
			options.Sort = sort
			eng.Logger.Debug("Using sort: %s", sort)
		}

		_, _ = headerStyle.Printf("Searching for: ")
		_, _ = titleStyle.Printf("%s\n", query)
		_, _ = dividerColor.Println(strings.Repeat("─", 50))

		if provider != "" {
			// Search single provider
			p, err := eng.GetProvider(provider)
			if err != nil {
				return err // Let the ExitErrHandler format this
			}

			eng.Logger.Debug("Searching provider: %s", p.ID())
			results, err := p.Search(ctx, query, options)
			if err != nil {
				return err // Let the ExitErrHandler format this
			}

			printSearchResults(p.Name(), results)
		} else {
			// Search all providers
			eng.Logger.Debug("Searching all providers")
			for _, p := range eng.AllProviders() {
				results, err := p.Search(ctx, query, options)
				if err != nil {
					// Just log errors from individual providers rather than failing
					fmt.Println(eng.FormatError(err))
					continue
				}

				printSearchResults(p.Name(), results)
			}
		}

		return nil
	}
}

// NewInfoCommand creates the info command
func NewInfoCommand(eng *engine.Engine) cli.ActionFunc {
	return func(c *cli.Context) error {
		if c.NArg() == 0 {
			return errors.New("manga ID is required").Error()
		}

		mangaID := c.Args().First()
		langFilter := c.String("lang")

		eng.Logger.Debug("Info request: manga=%s, lang=%s", mangaID, langFilter)

		// Parse combined ID
		parts := strings.SplitN(mangaID, ":", 2)
		if len(parts) != 2 {
			return errors.Newf("invalid manga ID format: %s", mangaID).Error()
		}

		providerID, id := parts[0], parts[1]

		// Get provider
		provider, err := eng.GetProvider(providerID)
		if err != nil {
			return err // Let the ExitErrHandler format this
		}

		// Get manga info
		ctx := context.Background()
		eng.Logger.Debug("Fetching manga info from provider: %s, id: %s", providerID, id)
		info, err := provider.GetManga(ctx, id)
		if err != nil {
			return err // Let the ExitErrHandler format this
		}

		// Print manga info
		_, _ = titleStyle.Printf("%s\n", info.Title)

		_, _ = labelStyle.Printf("Provider: ")
		_, _ = valueStyle.Printf("%s\n", provider.Name())

		if info.Description != "" {
			_, _ = labelStyle.Printf("Description: ")
			_, _ = valueStyle.Printf("%s\n", info.Description)
		}

		if len(info.Authors) > 0 {
			_, _ = labelStyle.Printf("Authors: ")
			_, _ = valueStyle.Printf("%s\n", strings.Join(info.Authors, ", "))
		}

		if info.Status != "" {
			_, _ = labelStyle.Printf("Status: ")

			// Color status based on its value
			switch strings.ToLower(info.Status) {
			case "completed", "complete":
				_, _ = successStyle.Printf("%s\n", info.Status)
			case "ongoing":
				_, _ = infoStyle.Printf("%s\n", info.Status)
			case "hiatus":
				_, _ = warningStyle.Printf("%s\n", info.Status)
			case "cancelled", "dropped":
				_, _ = errorStyle.Printf("%s\n", info.Status)
			default:
				_, _ = valueStyle.Printf("%s\n", info.Status)
			}
		}

		if len(info.Tags) > 0 {
			_, _ = labelStyle.Printf("Tags: ")

			// Print tags with highlight color
			for i, tag := range info.Tags {
				if i > 0 {
					fmt.Print(", ")
				}
				_, _ = highlightStyle.Printf("%s", tag)
			}
			fmt.Println()
		}

		_, _ = dividerColor.Println(strings.Repeat("─", 50))

		// Filter chapters if requested
		chapters := info.Chapters
		if langFilter != "" {
			languages := strings.Split(langFilter, ",")
			eng.Logger.Debug("Filtering chapters by languages: %v", languages)
			chapters = filterChaptersByLanguage(chapters, languages)
		}

		// Print chapters
		_, _ = sectionStyle.Printf("Chapters (%d):\n", len(chapters))

		for i, ch := range chapters {
			if i >= 10 {
				_, _ = secondaryStyle.Printf("... and %d more chapters\n", len(chapters)-10)
				break
			}

			_, _ = bulletStyle.Printf("  • ")
			_, _ = infoStyle.Printf("[%s:%s]", providerID, ch.ID)
			_, _ = valueStyle.Printf(" Ch.%g", ch.Number)

			if ch.Title != "" {
				_, _ = titleStyle.Printf(" - %s", ch.Title)
			}

			if ch.Date != nil {
				_, _ = secondaryStyle.Printf(" (%s)", ch.Date.Format("2006-01-02"))
			}

			fmt.Println()
		}

		return nil
	}
}

// NewDownloadCommand creates the download command
func NewDownloadCommand(eng *engine.Engine) cli.ActionFunc {
	return func(c *cli.Context) error {
		if c.NArg() == 0 {
			return errors.New("chapter ID is required").Error()
		}

		// Support multiple chapters as shown in README
		chapterIDs := c.Args().Slice()
		outputDir := c.String("output")
		format := c.String("format")
		concurrent := c.Int("concurrent")

		eng.Logger.Debug("Download request: chapters=%v, output=%s, format=%s, concurrent=%d",
			chapterIDs, outputDir, format, concurrent)

		ctx := context.Background()
		start := time.Now()

		hasErrors := false
		successCount := 0

		_, _ = headerStyle.Printf("Download started to: ")
		_, _ = valueStyle.Printf("%s\n", outputDir)

		if len(chapterIDs) > 1 {
			_, _ = infoStyle.Printf("Processing %d chapters...\n", len(chapterIDs))
		}

		_, _ = dividerColor.Println(strings.Repeat("─", 50))

		for _, chapterID := range chapterIDs {
			// Parse combined ID
			parts := strings.SplitN(chapterID, ":", 2)
			if len(parts) != 2 {
				err := errors.Newf("invalid chapter ID format: %s", chapterID).Error()
				fmt.Println(eng.FormatError(err))
				hasErrors = true
				continue
			}

			providerID, id := parts[0], parts[1]

			// Get provider
			provider, err := eng.GetProvider(providerID)
			if err != nil {
				fmt.Println(eng.FormatError(err))
				hasErrors = true
				continue
			}

			// Download chapter
			_, _ = infoStyle.Printf("Downloading: ")
			_, _ = titleStyle.Printf("%s ", chapterID)
			_, _ = secondaryStyle.Printf("from %s\n", provider.Name())

			eng.Logger.Debug("Downloading chapter: provider=%s, id=%s, output=%s",
				providerID, id, outputDir)

			if err := provider.DownloadChapter(ctx, id, outputDir); err != nil {
				fmt.Println(eng.FormatError(err))
				hasErrors = true
				continue
			}

			_, _ = successStyle.Printf("✓ Chapter %s downloaded successfully\n", chapterID)
			successCount++
		}

		_, _ = dividerColor.Println(strings.Repeat("─", 50))

		elapsed := time.Since(start)

		if successCount > 0 {
			if successCount == len(chapterIDs) {
				_, _ = successStyle.Printf("All %d chapter(s) downloaded successfully ", successCount)
			} else {
				_, _ = warningStyle.Printf("%d of %d chapter(s) downloaded ", successCount, len(chapterIDs))
			}
			_, _ = secondaryStyle.Printf("in %s\n", formatDuration(elapsed))
		}

		if hasErrors {
			return errors.New("some downloads failed").
				WithMessage("Some chapters could not be downloaded. See above for details.").Error()
		}

		return nil
	}
}

// NewProvidersCommand creates the providers command
func NewProvidersCommand(eng *engine.Engine) cli.ActionFunc {
	return func(c *cli.Context) error {
		providers := eng.AllProviders()
		eng.Logger.Debug("Listing %d providers", len(providers))

		_, _ = headerStyle.Printf("Available providers ")
		_, _ = titleStyle.Printf("(%d)\n", len(providers))
		_, _ = dividerColor.Println(strings.Repeat("─", 50))

		for _, p := range providers {
			_, _ = highlightStyle.Printf("[%s] ", p.ID())
			_, _ = titleStyle.Printf("%s\n", p.Name())
			_, _ = secondaryStyle.Printf("    %s\n", p.Description())
			_, _ = infoStyle.Printf("    URL: ")
			_, _ = valueStyle.Printf("%s\n", p.SiteURL())
			fmt.Println()
		}

		return nil
	}
}

// Helper functions

func printSearchResults(providerName string, results []core.Manga) {
	if len(results) == 0 {
		_, _ = secondaryStyle.Printf("\n[%s] ", providerName)
		_, _ = warningStyle.Println("No results found\n")
		return
	}

	_, _ = sectionStyle.Printf("\n[%s] ", providerName)
	_, _ = titleStyle.Printf("Found %d results:\n", len(results))

	for _, manga := range results {
		_, _ = bulletStyle.Print("  • ")
		_, _ = titleStyle.Printf("%s ", manga.Title)
		_, _ = secondaryStyle.Printf("(ID: %s)\n", manga.ID)

		if manga.Description != "" {
			desc := manga.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			_, _ = valueStyle.Printf("    %s\n", desc)
		}
	}
	fmt.Println()
}

func filterChaptersByLanguage(chapters []core.ChapterInfo, languages []string) []core.ChapterInfo {
	var filtered []core.ChapterInfo

	langMap := make(map[string]bool)
	for _, lang := range languages {
		langMap[strings.ToLower(lang)] = true
	}

	for _, ch := range chapters {
		if ch.Language == "" || langMap[strings.ToLower(ch.Language)] {
			filtered = append(filtered, ch)
		}
	}

	return filtered
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d.Seconds() < 60.0 {
		return fmt.Sprintf("%.2f seconds", d.Seconds())
	} else if d.Minutes() < 60.0 {
		return fmt.Sprintf("%.1f minutes", d.Minutes())
	} else {
		return fmt.Sprintf("%.1f hours", d.Hours())
	}
}
