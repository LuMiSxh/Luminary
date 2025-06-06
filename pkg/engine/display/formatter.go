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

package display

import (
	"Luminary/pkg/engine/core"
	"Luminary/pkg/provider"
	"Luminary/pkg/util"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Level defines the level of detail for displaying information
type Level int

const (
	// Minimal shows only the most basic information (e.g., Title, ID)
	Minimal Level = iota
	// Standard shows common information (e.g., Title, ID, Authors, Tags/AltTitles)
	Standard
	// Detailed shows all available information
	Detailed
)

// Options controls how information is displayed
type Options struct {
	Level            Level  // Detail level
	IncludeAltTitles bool   // Whether to include alternative titles (only if Level >= Standard)
	ShowTags         bool   // Whether to show tags (only if Level >= Standard)
	ItemLimit        int    // Maximum number of items (tags, alt titles) to display (0 = all)
	Indent           string // Base indentation string (e.g., "  ")
	Prefix           string // Prefix for the primary line (e.g., "- ")
	// Note: Detail lines will automatically get an additional indent level.
}

// --- Helper Functions ---

// appendLine adds a formatted line to the strings.Builder with appropriate prefix and indentation.
// indentLevel 0 = base indent, indentLevel 1 = one extra indent, etc.
func appendLine(sb *strings.Builder, options Options, indentLevel int, format string, args ...interface{}) {
	totalIndent := options.Indent + strings.Repeat(options.Indent, indentLevel)
	line := fmt.Sprintf(format, args...)
	sb.WriteString(fmt.Sprintf("%s%s%s\n", options.Prefix, totalIndent, line))
}

// formatLimitedList formats a slice of strings with a label and an optional limit.
// Returns an empty string if items is empty.
func formatLimitedList(label string, items []string, limit int) string {
	if len(items) == 0 {
		return ""
	}

	// Deduplicate for display (useful for AltTitles)
	uniqueMap := make(map[string]struct{})
	var displayItems []string
	for _, item := range items {
		if _, exists := uniqueMap[item]; !exists {
			uniqueMap[item] = struct{}{}
			displayItems = append(displayItems, item)
		}
	}

	if len(displayItems) == 0 { // Should not happen if items was not empty, but safety check
		return ""
	}

	limitedItems := displayItems
	suffix := ""
	if limit > 0 && len(displayItems) > limit {
		limitedItems = displayItems[:limit]
		suffix = fmt.Sprintf(" (and %d more)", len(displayItems)-limit)
	}

	return fmt.Sprintf("%s: %s%s", label, strings.Join(limitedItems, ", "), suffix)
}

// formatNullableDate formats a nullable date, returning a user-friendly string
func formatNullableDate(date *time.Time) string {
	if date == nil {
		return "Not specified"
	}
	return util.FormatDate(*date)
}

// formatNullableLanguage formats a nullable language, returning a user-friendly string
func formatNullableLanguage(language *string) string {
	if language == nil {
		return "Not specified"
	}
	// Use the language code directly
	return *language
}

// --- Primary Formatting Functions ---

// Manga formats and prints manga information according to the specified options
func Manga(manga core.Manga, provider provider.Provider, options Options) string {
	var output strings.Builder
	mangaID := core.FormatMangaID(provider.ID(), manga.ID)

	// Line 1: Title and ID (always shown)
	appendLine(&output, options, 0, "%s (ID: %s)", manga.Title, mangaID)

	// Detail lines (indented) - only if Level > Minimal
	if options.Level >= Standard {
		// Authors
		if len(manga.Authors) > 0 {
			appendLine(&output, options, 1, "Authors: %s", strings.Join(manga.Authors, ", "))
		}

		// Alternative titles (if enabled and available)
		if options.IncludeAltTitles {
			altTitleStr := formatLimitedList("Also known as", manga.AltTitles, options.ItemLimit)
			if altTitleStr != "" {
				appendLine(&output, options, 1, altTitleStr)
			}
		}

		// Tags (if enabled and available)
		if options.ShowTags {
			tagStr := formatLimitedList("Tags", manga.Tags, options.ItemLimit)
			if tagStr != "" {
				appendLine(&output, options, 1, tagStr)
			}
		}
	}

	// More detail lines (indented) - only if Level >= Detailed
	if options.Level >= Detailed {
		// Status
		if manga.Status != "" {
			appendLine(&output, options, 1, "Status: %s", manga.Status)
		}

		// Description
		if manga.Description != "" {
			desc := manga.Description
			// Basic truncation for very long descriptions in list views etc.
			if len(desc) > 300 {
				desc = desc[:300] + "..."
			}
			// Replace newlines in description for compact display
			desc = strings.ReplaceAll(desc, "\n", " ")
			appendLine(&output, options, 1, "Description: %s", desc)
		}
	}

	return output.String()
}

// Chapter formats and prints chapter information according to the specified options
func Chapter(chapter core.ChapterInfo, providerID string, options Options) string {
	var output strings.Builder
	chapterID := core.FormatMangaID(providerID, chapter.ID)

	// Line 1: Title and ID (always shown)
	title := chapter.Title
	if title == "" {
		title = fmt.Sprintf("Chapter %g", chapter.Number) // Fallback title
	}
	appendLine(&output, options, 0, "%s (ID: %s)", title, chapterID)

	// Detail lines (indented) - Chapter details are usually minimal or standard
	// We always show Chapter Number, Date, and Language if available, regardless of Level
	// as they are fundamental to a chapter.

	// Format date
	dateStr := formatNullableDate(chapter.Date)

	// Format language with more detail
	langStr := formatChapterLanguage(chapter.Language)

	// Format chapter number
	chapterNum := "?"
	if chapter.Number > 0 {
		chapterNum = fmt.Sprintf("%g", chapter.Number) // Use %g for clean number formatting
	}

	appendLine(&output, options, 1, "Chapter %s | Released: %s | Language: %s", chapterNum, dateStr, langStr)

	return output.String()
}

// formatChapterLanguage formats a chapter's language with more detail than the generic formatter
func formatChapterLanguage(language *string) string {
	if language == nil {
		return "Unknown"
	}

	langCode := *language

	// Map of common language codes to full names (can be extended)
	languageNames := map[string]string{
		"en":    "English",
		"ja":    "Japanese",
		"es":    "Spanish",
		"fr":    "French",
		"de":    "German",
		"pt":    "Portuguese",
		"ru":    "Russian",
		"ko":    "Korean",
		"zh":    "Chinese",
		"it":    "Italian",
		"ar":    "Arabic",
		"tr":    "Turkish",
		"th":    "Thai",
		"vi":    "Vietnamese",
		"id":    "Indonesian",
		"pl":    "Polish",
		"nl":    "Dutch",
		"sv":    "Swedish",
		"da":    "Danish",
		"no":    "Norwegian",
		"fi":    "Finnish",
		"hu":    "Hungarian",
		"cs":    "Czech",
		"sk":    "Slovak",
		"bg":    "Bulgarian",
		"hr":    "Croatian",
		"sr":    "Serbian",
		"sl":    "Slovenian",
		"et":    "Estonian",
		"lv":    "Latvian",
		"lt":    "Lithuanian",
		"ro":    "Romanian",
		"el":    "Greek",
		"he":    "Hebrew",
		"fa":    "Persian",
		"hi":    "Hindi",
		"bn":    "Bengali",
		"ta":    "Tamil",
		"te":    "Telugu",
		"ml":    "Malayalam",
		"kn":    "Kannada",
		"gu":    "Gujarati",
		"pa":    "Punjabi",
		"ur":    "Urdu",
		"uk":    "Ukrainian",
		"zh-cn": "Chinese (Simplified)",
		"zh-tw": "Chinese (Traditional)",
		"zh-hk": "Chinese (Hong Kong)",
		"pt-br": "Portuguese (Brazil)",
		"es-la": "Spanish (Latin America)",
	}

	if fullName, exists := languageNames[langCode]; exists {
		return fmt.Sprintf("%s (%s)", fullName, langCode)
	}

	// Return the code as-is if we don't have a mapping
	return langCode
}

// MangaInfo formats and prints detailed manga information including chapters.
// It uses Standard options for the main manga details by default, but can be customized.
func MangaInfo(manga *core.MangaInfo, provider provider.Provider, mainOptions Options) string {
	var output strings.Builder

	// --- Manga Details Section ---
	// Use the Manga formatter for the main details, ensuring Detailed level for full info.
	// Override prefix/indent for this specific section header if needed, or use defaults.
	mangaCore := core.Manga{ // Adapt MangaInfo to Manga struct for the formatter
		ID:          manga.ID,
		Title:       manga.Title,
		AltTitles:   manga.AltTitles,
		Description: manga.Description,
		Authors:     manga.Authors,
		Status:      manga.Status,
		Tags:        manga.Tags,
	}
	// Force detailed level for MangaInfo display, but respect other options like limits
	infoOptions := mainOptions
	infoOptions.Level = Detailed
	infoOptions.Prefix = ""   // No prefix for the main block
	infoOptions.Indent = ""   // No base indent for the main block
	infoOptions.ItemLimit = 0 // Show all tags/alt-titles in detailed view

	output.WriteString("--- Manga Details ---\n")
	mangaDetails := Manga(mangaCore, provider, infoOptions)
	// Manually indent the details block if desired (e.g., mainOptions.Indent)
	for _, line := range strings.Split(strings.TrimSuffix(mangaDetails, "\n"), "\n") {
		output.WriteString(mainOptions.Indent + line + "\n")
	}

	// Display last updated information if available
	if mainOptions.Level >= Detailed && manga.LastUpdated != nil {
		lastUpdatedStr := formatNullableDate(manga.LastUpdated)
		output.WriteString(fmt.Sprintf("%s%sLast Updated: %s\n", mainOptions.Prefix, mainOptions.Indent+mainOptions.Indent, lastUpdatedStr))
	}

	// Display full description separately if needed (Manga formatter truncates)
	if mainOptions.Level >= Detailed && manga.Description != "" {
		descHeader := fmt.Sprintf("%s%sDescription:", mainOptions.Prefix, mainOptions.Indent+mainOptions.Indent)
		output.WriteString(fmt.Sprintf("%s\n%s%s%s\n", descHeader, mainOptions.Prefix, mainOptions.Indent+mainOptions.Indent, strings.ReplaceAll(manga.Description, "\n", "\n"+mainOptions.Prefix+mainOptions.Indent+mainOptions.Indent)))
	}

	// --- Chapters Section ---
	if len(manga.Chapters) > 0 {
		output.WriteString(fmt.Sprintf("\n%s--- Chapters (%d) ---\n", mainOptions.Indent, len(manga.Chapters)))

		// Sort chapters (descending by number is common)
		sortedChapters := make([]core.ChapterInfo, len(manga.Chapters))
		copy(sortedChapters, manga.Chapters)
		sort.SliceStable(sortedChapters, func(i, j int) bool {
			// Sort primarily by number, descending
			if sortedChapters[i].Number != sortedChapters[j].Number {
				// Handle non-numeric chapters (e.g., number 0 or -1) by putting them last
				if sortedChapters[i].Number <= 0 {
					return false
				}
				if sortedChapters[j].Number <= 0 {
					return true
				}
				return sortedChapters[i].Number > sortedChapters[j].Number // Higher number first
			}
			// Secondary sort by title if numbers are equal
			return sortedChapters[i].Title < sortedChapters[j].Title
		})

		// Set options for displaying chapters within this list
		chapterOptions := Options{
			Level:  Minimal, // Keep the chapter list concise
			Indent: "  ",    // Indentation relative to the chapter list item
			Prefix: "- ",    // Use a list item prefix
			// ItemLimit, ShowTags, IncludeAltTitles not relevant for Chapter
		}

		for _, chapter := range sortedChapters {
			// Apply the overall indent from mainOptions to the prefix of the chapter line
			chapterOptions.Prefix = mainOptions.Indent + "- "
			chapterDisplay := Chapter(chapter, provider.ID(), chapterOptions)
			output.WriteString(chapterDisplay)
		}
	} else {
		output.WriteString(fmt.Sprintf("\n%sNo chapters available.\n", mainOptions.Indent))
	}

	return output.String()
}

// --- List Formatting Functions ---

// formatList formats a generic list of items using a provided formatter function.
// It handles numbering and avoids the TrimPrefix issue by controlling prefix/indent passed down.
func formatList[T any](
	items []T,
	providerID string, // Use provider.Provider interface if needed universally
	options Options, // Options for *each item* in the list
	formatter func(item T, providerID string, itemOptions Options) string,
	listName string, // e.g., "results", "manga titles", "chapters"
	emptyMsg string,
) string {
	var output strings.Builder

	if len(items) == 0 {
		appendLine(&output, options, 0, emptyMsg) // Use base indent/prefix for the empty message
		return output.String()
	}

	countPrefix := fmt.Sprintf("%s%s", options.Prefix, options.Indent) // Base prefix for count line
	output.WriteString(fmt.Sprintf("%sFound %d %s:\n", countPrefix, len(items), listName))

	// Options specifically for items within the list
	itemOptions := options
	itemOptions.Prefix = ""             // The number+dot will be the prefix, handled below
	itemOptions.Indent = options.Indent // Keep the item indentation relative

	basePrefix := fmt.Sprintf("%s%s", options.Prefix, options.Indent) // Base prefix for each numbered item line

	for i, item := range items {
		// Format the item itself using the provided formatter and itemOptions
		itemStr := formatter(item, providerID, itemOptions)

		// Add numbering and prefix, ensuring proper alignment
		lines := strings.Split(strings.TrimSuffix(itemStr, "\n"), "\n")
		if len(lines) > 0 {
			// Add number to the first line
			output.WriteString(fmt.Sprintf("%s%d. %s\n", basePrefix, i+1, lines[0]))
			// Indent subsequent lines of the same item further
			subsequentIndent := basePrefix + strings.Repeat(" ", len(fmt.Sprintf("%d. ", i+1)))
			for _, line := range lines[1:] {
				output.WriteString(fmt.Sprintf("%s%s\n", subsequentIndent, line))
			}
		}
	}
	return output.String()
}

// SearchResults formats and prints search results using the generic list formatter
func SearchResults(results []core.Manga, provider provider.Provider) string {
	listOptions := Options{ // Options for the overall list structure
		Level:            Standard, // Show standard details for search results
		IncludeAltTitles: true,
		ShowTags:         true,
		ItemLimit:        3, // Limit tags/alt-titles in lists
		Indent:           "  ",
		Prefix:           "  ",
	}

	// Adapt the Manga formatter slightly for the generic function signature
	mangaFormatter := func(item core.Manga, providerID string, itemOptions Options) string {
		// Need the provider object, not just ID, for Manga formatter
		return Manga(item, provider, itemOptions)
	}

	// Need to wrap the specific formatter to match the generic signature
	formatterAdapter := func(item core.Manga, pID string, itemOptions Options) string {
		// We ignore pID here because mangaFormatter captures the provider object
		return mangaFormatter(item, "", itemOptions)
	}

	return formatList(results, provider.ID(), listOptions, formatterAdapter, "results", "No results found")
}

// MangaList formats and prints a manga list using the generic list formatter
func MangaList(mangas []core.Manga, provider provider.Provider) string {
	listOptions := Options{ // Options for the overall list structure
		Level:  Minimal, // Minimal details for simple lists
		Indent: "  ",
		Prefix: "  ",
		// Other options like ItemLimit, ShowTags irrelevant for Minimal
	}

	// Adapt the Manga formatter slightly for the generic function signature
	mangaFormatter := func(item core.Manga, providerID string, itemOptions Options) string {
		// Need the provider object, not just ID, for Manga formatter
		return Manga(item, provider, itemOptions)
	}

	// Need to wrap the specific formatter to match the generic signature
	formatterAdapter := func(item core.Manga, pID string, itemOptions Options) string {
		// We ignore pID here because mangaFormatter captures the provider object
		return mangaFormatter(item, "", itemOptions)
	}

	return formatList(mangas, provider.ID(), listOptions, formatterAdapter, "manga titles", "No manga found")
}

// ChapterList formats and prints a list of chapters using the generic list formatter
func ChapterList(chapters []core.ChapterInfo, providerID string) string {
	listOptions := Options{ // Options for the overall list structure
		Level:  Minimal, // Minimal details for chapter lists
		Indent: "  ",
		Prefix: "  ",
	}

	// The Chapter formatter already matches the generic signature
	chapterFormatter := func(item core.ChapterInfo, pID string, itemOptions Options) string {
		return Chapter(item, pID, itemOptions)
	}

	return formatList(chapters, providerID, listOptions, chapterFormatter, "chapters", "No chapters found")
}
