package engine

import (
	"Luminary/utils"
	"fmt"
	"sort"
	"strings"
)

// DisplayLevel defines the level of detail for displaying information
type DisplayLevel int

const (
	// DisplayMinimal shows only the most basic information (title, ID)
	DisplayMinimal DisplayLevel = iota
	// DisplayStandard shows common information (title, ID, authors, tags)
	DisplayStandard
	// DisplayDetailed shows all available information
	DisplayDetailed
)

// DisplayOptions controls how information is displayed
type DisplayOptions struct {
	Level            DisplayLevel // Detail level
	IncludeAltTitles bool         // Whether to include alternative titles
	ShowTags         bool         // Whether to show tags
	TagLimit         int          // Maximum number of tags to display (0 = all)
	Indent           string       // Indentation string (e.g., "  " for two spaces)
	Prefix           string       // Prefix for each line (e.g., "- " for list items)
}

// DefaultDisplayOptions returns the default display options
func DefaultDisplayOptions() DisplayOptions {
	return DisplayOptions{
		Level:            DisplayStandard,
		IncludeAltTitles: true,
		ShowTags:         true,
		TagLimit:         5,
		Indent:           "  ",
		Prefix:           "",
	}
}

// DisplayManga formats and prints manga information according to the specified options
func DisplayManga(manga Manga, agent Agent, options DisplayOptions) string {
	var output strings.Builder

	// Format the manga ID
	mangaID := utils.FormatMangaID(agent.ID(), manga.ID)

	// Apply indentation and prefix to each line
	indent := options.Indent
	prefix := options.Prefix

	// Title and ID (always shown)
	output.WriteString(fmt.Sprintf("%s%s%s (ID: %s)\n", prefix, indent, manga.Title, mangaID))

	// Additional information based on display level
	if options.Level >= DisplayStandard {
		// Authors
		if len(manga.Authors) > 0 {
			output.WriteString(fmt.Sprintf("%s%s%sAuthors: %s\n", prefix, indent, indent, strings.Join(manga.Authors, ", ")))
		}

		// Alternative titles (if enabled)
		if options.IncludeAltTitles && len(manga.AltTitles) > 0 {
			// Deduplicate alternative titles
			uniqueTitles := make(map[string]bool)
			var displayTitles []string

			for _, title := range manga.AltTitles {
				if !uniqueTitles[title] {
					uniqueTitles[title] = true
					displayTitles = append(displayTitles, title)
				}
			}

			if len(displayTitles) > 0 {
				// Show up to a limited number of alternative titles
				limitedTitles := displayTitles
				if options.TagLimit > 0 && len(displayTitles) > options.TagLimit {
					limitedTitles = displayTitles[:options.TagLimit]
					output.WriteString(fmt.Sprintf("%s%s%sAlso known as: %s (and %d more)\n",
						prefix, indent, indent,
						strings.Join(limitedTitles, ", "),
						len(displayTitles)-options.TagLimit))
				} else {
					output.WriteString(fmt.Sprintf("%s%s%sAlso known as: %s\n",
						prefix, indent, indent,
						strings.Join(limitedTitles, ", ")))
				}
			}
		}

		// Tags (if enabled)
		if options.ShowTags && len(manga.Tags) > 0 {
			limitedTags := manga.Tags
			if options.TagLimit > 0 && len(manga.Tags) > options.TagLimit {
				limitedTags = manga.Tags[:options.TagLimit]
				output.WriteString(fmt.Sprintf("%s%s%sTags: %s (and %d more)\n",
					prefix, indent, indent,
					strings.Join(limitedTags, ", "),
					len(manga.Tags)-options.TagLimit))
			} else {
				output.WriteString(fmt.Sprintf("%s%s%sTags: %s\n",
					prefix, indent, indent,
					strings.Join(manga.Tags, ", ")))
			}
		}

		// Status (if available and detailed level)
		if options.Level >= DisplayDetailed && manga.Status != "" {
			output.WriteString(fmt.Sprintf("%s%s%sStatus: %s\n", prefix, indent, indent, manga.Status))
		}

		// Description (if available and detailed level)
		if options.Level >= DisplayDetailed && manga.Description != "" {
			// Format description with proper indentation
			desc := manga.Description
			// Truncate very long descriptions
			if len(desc) > 500 {
				desc = desc[:500] + "..."
			}
			output.WriteString(fmt.Sprintf("%s%s%sDescription: %s\n", prefix, indent, indent, desc))
		}
	}

	return output.String()
}

// DisplayChapter formats and prints chapter information according to the specified options
func DisplayChapter(chapter ChapterInfo, agentID string, options DisplayOptions) string {
	var output strings.Builder

	// Format the chapter ID
	chapterID := utils.FormatMangaID(agentID, chapter.ID)

	// Apply indentation and prefix to each line
	indent := options.Indent
	prefix := options.Prefix

	// Format date
	dateStr := "Unknown date"
	if !chapter.Date.IsZero() {
		dateStr = utils.FormatDate(chapter.Date)
	}

	// Format chapter number
	chapterNum := "?"
	if chapter.Number > 0 {
		chapterNum = fmt.Sprintf("%g", chapter.Number)
	}

	// Title and ID (always shown)
	output.WriteString(fmt.Sprintf("%s%s%s (ID: %s)\n", prefix, indent, chapter.Title, chapterID))

	// Chapter number and date (always shown for chapters)
	output.WriteString(fmt.Sprintf("%s%s%sChapter %s, Released: %s\n", prefix, indent, indent, chapterNum, dateStr))

	return output.String()
}

// DisplayMangaInfo formats and prints detailed manga information including chapters
func DisplayMangaInfo(manga *MangaInfo, agent Agent) string {
	var output strings.Builder

	// Format the manga ID
	mangaID := utils.FormatMangaID(agent.ID(), manga.ID)

	// Title and ID (always shown first for better readability)
	output.WriteString(fmt.Sprintf("Manga: %s\n", manga.Title))
	output.WriteString(fmt.Sprintf("ID: %s\n", mangaID))

	// Authors
	if len(manga.Authors) > 0 {
		output.WriteString(fmt.Sprintf("Authors: %s\n", strings.Join(manga.Authors, ", ")))
	}

	// Status
	if manga.Status != "" {
		output.WriteString(fmt.Sprintf("Status: %s\n", manga.Status))
	}

	// Alternative titles
	if len(manga.AltTitles) > 0 {
		output.WriteString(fmt.Sprintf("Also known as: %s\n", strings.Join(manga.AltTitles, ", ")))
	}

	// Tags
	if len(manga.Tags) > 0 {
		output.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(manga.Tags, ", ")))
	}

	// Description
	if manga.Description != "" {
		output.WriteString(fmt.Sprintf("\nDescription:\n%s\n", manga.Description))
	}

	// Chapters
	if len(manga.Chapters) > 0 {
		output.WriteString(fmt.Sprintf("\nChapters (%d):\n", len(manga.Chapters)))

		// Sort chapters by number if possible
		sortedChapters := make([]ChapterInfo, len(manga.Chapters))
		copy(sortedChapters, manga.Chapters)

		sort.SliceStable(sortedChapters, func(i, j int) bool {
			// If both have valid numbers, sort by number
			if sortedChapters[i].Number > 0 && sortedChapters[j].Number > 0 {
				return sortedChapters[i].Number < sortedChapters[j].Number
			}
			// If only one has a number, the one with number comes first
			if sortedChapters[i].Number > 0 {
				return true
			}
			if sortedChapters[j].Number > 0 {
				return false
			}
			// If neither has a number, sort by title
			return sortedChapters[i].Title < sortedChapters[j].Title
		})

		// Set options for displaying chapters
		chapterOptions := DisplayOptions{
			Level:  DisplayMinimal,
			Indent: "  ",
			Prefix: "- ",
		}

		for _, chapter := range sortedChapters {
			chapterDisplay := DisplayChapter(chapter, agent.ID(), chapterOptions)
			output.WriteString(chapterDisplay)
		}
	} else {
		output.WriteString("\nNo chapters available.\n")
	}

	return output.String()
}

// DisplaySearchResults formats and prints search results
func DisplaySearchResults(results []Manga, agent Agent) string {
	var output strings.Builder

	if len(results) == 0 {
		output.WriteString("  No results found\n")
		return output.String()
	}

	output.WriteString(fmt.Sprintf("  Found %d results:\n", len(results)))

	options := DefaultDisplayOptions()
	options.Indent = "  "
	options.Prefix = "  "

	for i, manga := range results {
		output.WriteString(fmt.Sprintf("  %d. ", i+1))
		// Remove the leading spaces that DisplayManga adds to compensate for our numbering
		mangaOutput := DisplayManga(manga, agent, options)
		output.WriteString(strings.TrimPrefix(mangaOutput, "    "))
	}

	return output.String()
}

// DisplayMangaList formats and prints a manga list
func DisplayMangaList(mangas []Manga, agent Agent) string {
	var output strings.Builder

	if len(mangas) == 0 {
		output.WriteString("  No manga found\n")
		return output.String()
	}

	// Use minimal display options for lists
	options := DisplayOptions{
		Level:  DisplayMinimal,
		Indent: "",
		Prefix: "  ",
	}

	for i, manga := range mangas {
		output.WriteString(fmt.Sprintf("  %d. ", i+1))
		// Remove the leading spaces that DisplayManga adds to compensate for our numbering
		mangaOutput := DisplayManga(manga, agent, options)
		output.WriteString(strings.TrimPrefix(mangaOutput, "    "))
	}

	output.WriteString(fmt.Sprintf("\n  Found %d manga titles\n", len(mangas)))
	return output.String()
}

// DisplayChapterList formats and prints a list of chapters
func DisplayChapterList(chapters []ChapterInfo, agentID string) string {
	var output strings.Builder

	if len(chapters) == 0 {
		output.WriteString("  No chapters found\n")
		return output.String()
	}

	output.WriteString(fmt.Sprintf("  Found %d chapters:\n", len(chapters)))

	// Use minimal display options for lists
	options := DisplayOptions{
		Level:  DisplayMinimal,
		Indent: "",
		Prefix: "  ",
	}

	for i, chapter := range chapters {
		output.WriteString(fmt.Sprintf("  %d. ", i+1))
		// Remove the leading spaces that DisplayChapter adds to compensate for our numbering
		chapterOutput := DisplayChapter(chapter, agentID, options)
		output.WriteString(strings.TrimPrefix(chapterOutput, "    "))
	}

	return output.String()
}

// Min returns the smaller of two integers (helper function)
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
