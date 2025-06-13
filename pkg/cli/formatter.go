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
	"Luminary/pkg/engine/core"
	pkgerrors "Luminary/pkg/errors" // Use alias for package errors
	"Luminary/pkg/provider"
	"Luminary/pkg/util"
	"errors"
	"fmt"
	"github.com/olekukonko/tablewriter/tw"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

const (
	OutputTypeText  = "text"
	OutputTypeTable = "table"
)

// Formatter handles all CLI output formatting
type Formatter struct {
	// Writer is where the formatted output will be written
	Writer io.Writer

	// DisableColor disables colorized output
	DisableColor bool

	// OutputType controls the type of output (text, table, simple)
	OutputType string

	// Styles for different elements
	HeaderStyle      *color.Color
	TitleStyle       *color.Color
	SubtitleStyle    *color.Color
	SuccessStyle     *color.Color
	ErrorStyle       *color.Color
	WarningStyle     *color.Color
	InfoStyle        *color.Color
	HighlightStyle   *color.Color
	SecondaryStyle   *color.Color
	ImportantStyle   *color.Color
	SectionStyle     *color.Color
	DetailLabelStyle *color.Color
	DetailValueStyle *color.Color
	CommandStyle     *color.Color
	IDStyle          *color.Color
	PathStyle        *color.Color
	ParameterStyle   *color.Color
	DateStyle        *color.Color
	NumberStyle      *color.Color
}

// NewFormatter creates a new CLI formatter with default settings
func NewFormatter() *Formatter {
	f := &Formatter{
		Writer:       os.Stdout,
		DisableColor: false,
		OutputType:   OutputTypeText,
	}

	// Initialize styles
	f.initStyles()

	return f
}

// initStyles sets up all the color styles
func (f *Formatter) initStyles() {
	// Don't use color if it's disabled
	if f.DisableColor {
		color.NoColor = true
	}

	// Configure styles
	f.HeaderStyle = color.New(color.Bold, color.FgCyan)
	f.TitleStyle = color.New(color.Bold, color.FgWhite)
	f.SubtitleStyle = color.New(color.FgHiWhite)
	f.SuccessStyle = color.New(color.FgGreen)
	f.ErrorStyle = color.New(color.FgRed)
	f.WarningStyle = color.New(color.FgYellow)
	f.InfoStyle = color.New(color.FgBlue)
	f.HighlightStyle = color.New(color.FgMagenta)
	f.SecondaryStyle = color.New(color.FgHiBlack)
	f.ImportantStyle = color.New(color.Bold, color.FgHiWhite)
	f.SectionStyle = color.New(color.Underline, color.FgHiCyan)
	f.DetailLabelStyle = color.New(color.FgHiBlue)
	f.DetailValueStyle = color.New(color.FgWhite)
	f.CommandStyle = color.New(color.FgHiYellow)
	f.IDStyle = color.New(color.FgHiMagenta)
	f.PathStyle = color.New(color.FgHiGreen)
	f.ParameterStyle = color.New(color.FgHiCyan)
	f.DateStyle = color.New(color.FgHiBlue)
	f.NumberStyle = color.New(color.FgHiYellow)
}

// PrintHeader prints a header section
func (f *Formatter) PrintHeader(text string) {
	_, err := f.HeaderStyle.Fprintln(f.Writer, text)
	if err != nil {
		return
	}
	f.PrintDivider()
}

// PrintTitle prints a title
func (f *Formatter) PrintTitle(text string) {
	_, err := f.TitleStyle.Fprintln(f.Writer, text)
	if err != nil {
		return
	}
}

// PrintSubtitle prints a subtitle
func (f *Formatter) PrintSubtitle(text string) {
	_, err := f.SubtitleStyle.Fprintln(f.Writer, text)
	if err != nil {
		return
	}
}

// PrintSuccess prints a success message
func (f *Formatter) PrintSuccess(text string) {
	_, err := f.SuccessStyle.Fprintln(f.Writer, text)
	if err != nil {
		return
	}
}

// PrintError prints an error message
func (f *Formatter) PrintError(text string) {
	_, err := f.ErrorStyle.Fprintln(f.Writer, text)
	if err != nil {
		return
	}
}

// PrintWarning prints a warning message
func (f *Formatter) PrintWarning(text string) {
	_, err := f.WarningStyle.Fprintln(f.Writer, text)
	if err != nil {
		return
	}
}

// PrintInfo prints an informational message
func (f *Formatter) PrintInfo(text string) {
	_, err := f.InfoStyle.Fprintln(f.Writer, text)
	if err != nil {
		return
	}
}

// PrintDetail prints a labeled detail
func (f *Formatter) PrintDetail(label, value string) {
	_, err := f.DetailLabelStyle.Fprintf(f.Writer, "%s: ", label)
	if err != nil {
		return
	}
	_, err = f.DetailValueStyle.Fprintln(f.Writer, value)
	if err != nil {
		return
	}
}

// PrintDivider prints a horizontal divider
func (f *Formatter) PrintDivider() {
	_, err := fmt.Fprintln(f.Writer, strings.Repeat("-", 80))
	if err != nil {
		return
	}
}

// PrintSection prints a section header
func (f *Formatter) PrintSection(text string) {
	_, err := fmt.Fprintln(f.Writer, "")
	if err != nil {
		return
	}
	_, err = f.SectionStyle.Fprintln(f.Writer, text)
	if err != nil {
		return
	}
	_, err = fmt.Fprintln(f.Writer, "")
	if err != nil {
		return
	}
}

// PrintNewLine prints a blank line
func (f *Formatter) PrintNewLine() {
	_, err := fmt.Fprintln(f.Writer, "")
	if err != nil {
		return
	}
}

// FormatID formats an ID string
func (f *Formatter) FormatID(id string) string {
	return f.IDStyle.Sprint(id)
}

// FormatPath formats a file path
func (f *Formatter) FormatPath(path string) string {
	return f.PathStyle.Sprint(path)
}

// FormatDate formats a date with styling
func (f *Formatter) FormatDate(date *time.Time) string {
	if date == nil {
		return f.SecondaryStyle.Sprint("Not specified")
	}
	return f.DateStyle.Sprint(util.FormatDate(*date))
}

// FormatNumber formats a number with styling
func (f *Formatter) FormatNumber(num interface{}) string {
	return f.NumberStyle.Sprintf("%v", num)
}

// FormatLanguage formats a language code with description
func (f *Formatter) FormatLanguage(language *string) string {
	if language == nil {
		return f.SecondaryStyle.Sprint("Unknown")
	}

	langCode := *language
	languageName := util.GetLanguageName(langCode)

	if languageName != "" {
		return fmt.Sprintf("%s (%s)", f.DetailValueStyle.Sprint(languageName), f.SecondaryStyle.Sprint(langCode))
	}

	return f.DetailValueStyle.Sprint(langCode)
}

// PrintTable prints data in a table format
func (f *Formatter) PrintTable(headers []string, data [][]string) {
	table := tablewriter.NewTable(f.Writer)
	table.Configure(func(tableConfig *tablewriter.Config) {
		tableConfig.Header.Alignment.Global = tw.AlignLeft
		tableConfig.Row.Alignment.Global = tw.AlignLeft
		tableConfig.Header.Padding.Global = tw.Padding{
			Left:  " ",
			Right: " ",
		}
		tableConfig.Row.Padding.Global = tw.Padding{
			Left:  " ",
			Right: " ",
		}
	})

	// Add headers and data
	table.Header(headers)
	err := table.Bulk(data)
	if err != nil {
		return
	}

	// Render the table
	err = table.Render()
	if err != nil {
		return
	}
}

// HandleError handles and formats any error, including regular Go errors
// It returns true if an error was handled, false otherwise
func (f *Formatter) HandleError(err error) bool {
	if err == nil {
		return false
	}

	var trackedError *pkgerrors.TrackedError
	if errors.As(err, &trackedError) {
		f.PrintError(pkgerrors.FormatCLI(err))
	} else {
		// For regular errors, format them in a consistent way
		f.PrintError(fmt.Sprintf("[ERROR] %s", err.Error()))
	}

	return true
}

// PrintProviderList formats and prints a list of providers
func (f *Formatter) PrintProviderList(provs []provider.Provider) {
	f.PrintHeader("Available Manga Source Providers")

	if len(provs) == 0 {
		f.PrintWarning("No providers available.")
		return
	}

	// Sort providers alphabetically
	sort.Slice(provs, func(i, j int) bool {
		return provs[i].Name() < provs[j].Name()
	})

	if f.OutputType == OutputTypeTable {
		headers := []string{"ID", "NAME", "DESCRIPTION"}
		data := make([][]string, len(provs))

		for i, prov := range provs {
			data[i] = []string{
				prov.ID(),
				prov.Name(),
				prov.Description(),
			}
		}

		f.PrintTable(headers, data)
	} else {
		// Text format
		for _, prov := range provs {
			_, err := f.TitleStyle.Fprintf(f.Writer, "%s ", prov.ID())
			if err != nil {
				return
			}
			_, err = f.SecondaryStyle.Fprintf(f.Writer, "(%s)\n", prov.Name())
			if err != nil {
				return
			}
			_, err = fmt.Fprintf(f.Writer, "  %s\n\n", prov.Description())
			if err != nil {
				return
			}
		}
	}

	_, err := f.InfoStyle.Fprintln(f.Writer, "Use --provider flag with the search command to specify a particular provider")
	if err != nil {
		return
	}
}

// PrintVersionInfo formats and prints version information
func (f *Formatter) PrintVersionInfo(version, goVersion, os, arch, logFile string) {
	f.PrintHeader("Luminary Version Information")

	f.PrintDetail("Version", version)
	f.PrintDetail("Go version", goVersion)
	f.PrintDetail("OS/Arch", fmt.Sprintf("%s/%s", os, arch))

	if logFile != "" {
		f.PrintDetail("Log file", f.FormatPath(logFile))
	} else {
		f.PrintDetail("Logging to file", "disabled")
	}
}

// PrintSearchInfo prints search parameters
func (f *Formatter) PrintSearchInfo(query string, options core.SearchOptions, includeAltTitles, includeAllLangs bool, maxConcurrency int) {
	f.PrintHeader("Search Parameters")

	f.PrintDetail("Query", f.HighlightStyle.Sprint(query))

	if len(options.Fields) > 0 {
		f.PrintDetail("Search fields", strings.Join(options.Fields, ", "))
	} else {
		f.PrintDetail("Search fields", "all")
	}

	if includeAltTitles {
		f.PrintDetail("Include alternative titles", "yes")
	}

	if includeAllLangs {
		f.PrintDetail("Search across all languages", "yes")
	}

	if options.Limit > 0 {
		f.PrintDetail("Result limit", f.FormatNumber(options.Limit)+" per page")
	} else {
		f.PrintDetail("Result limit", "unlimited")
	}

	if options.Pages > 0 {
		f.PrintDetail("Pages fetched", f.FormatNumber(options.Pages))
	} else {
		f.PrintDetail("Pages fetched", "all available")
	}

	// Display field-specific filters
	if len(options.Filters) > 0 {
		_, err := fmt.Fprintln(f.Writer, "")
		if err != nil {
			return
		}
		f.PrintSubtitle("Filters:")
		for field, value := range options.Filters {
			f.PrintDetail("  "+field, value)
		}
	}

	f.PrintDetail("Concurrency", f.FormatNumber(maxConcurrency))
}

// PrintMangaItem prints a single manga item in a formatted way
func (f *Formatter) PrintMangaItem(manga core.Manga, providerID string, number int) {
	// Format the item number and manga ID
	itemPrefix := ""
	if number > 0 {
		itemPrefix = fmt.Sprintf("%d. ", number)
	}

	// Generate manga ID using the provider ID
	mangaID := core.FormatMangaID(providerID, manga.ID)

	// Print the title line
	_, err := f.TitleStyle.Fprintf(f.Writer, "%s%s ", itemPrefix, manga.Title)
	if err != nil {
		return
	}
	_, err = f.IDStyle.Fprintf(f.Writer, "(ID: %s)\n", mangaID)
	if err != nil {
		return
	}

	// Prepare a compact detail line
	var details []string

	// Add authors if available
	if len(manga.Authors) > 0 {
		authorLimit := 2
		authorDisplay := manga.Authors
		if len(authorDisplay) > authorLimit {
			authorDisplay = authorDisplay[:authorLimit]
			details = append(details, fmt.Sprintf("Authors: %s +%d more",
				strings.Join(authorDisplay, ", "),
				len(manga.Authors)-authorLimit))
		} else {
			details = append(details, fmt.Sprintf("Authors: %s", strings.Join(manga.Authors, ", ")))
		}
	}

	// Add status if available
	if manga.Status != "" {
		details = append(details, fmt.Sprintf("Status: %s", manga.Status))
	}

	// Print the details line if we have any details
	if len(details) > 0 {
		_, err := f.DetailLabelStyle.Fprintf(f.Writer, "  %s\n", strings.Join(details, " | "))
		if err != nil {
			return
		}
	}

	// Print tags in a compact format
	if len(manga.Tags) > 0 {
		limit := 5 // Limit number of tags displayed
		tags := manga.Tags
		suffix := ""

		if len(tags) > limit {
			tags = tags[:limit]
			suffix = fmt.Sprintf(" +%d more", len(manga.Tags)-limit)
		}

		_, err := f.SecondaryStyle.Fprintf(f.Writer, "  Tags: %s%s\n",
			strings.Join(tags, ", "), suffix)
		if err != nil {
			return
		}
	}

	// Add a blank line after each manga item for readability
	_, err = fmt.Fprintln(f.Writer, "")
	if err != nil {
		return
	}
}

// PrintMangaList prints a list of manga items
func (f *Formatter) PrintMangaList(mangas []core.Manga, prov provider.Provider, title string) {
	if title != "" {
		f.PrintSection(title)
	}

	if len(mangas) == 0 {
		f.PrintWarning("No manga found.")
		return
	}

	f.PrintInfo(fmt.Sprintf("Found %d manga:", len(mangas)))
	f.PrintNewLine()

	for i, manga := range mangas {
		f.PrintMangaItem(manga, prov.ID(), i+1)
	}
}

// PrintSearchResults prints search results grouped by provider
func (f *Formatter) PrintSearchResults(results map[string][]core.Manga, query string, options core.SearchOptions) {
	// Calculate total result count
	totalCount := 0
	for _, mangaList := range results {
		totalCount += len(mangaList)
	}

	f.PrintHeader(fmt.Sprintf("Search Results for '%s'", query))
	f.PrintDetail("Total results", f.FormatNumber(totalCount))

	// No results case
	if totalCount == 0 {
		f.PrintWarning("No results found for your query.")
		return
	}

	// Display results for each provider
	for providerID, mangaList := range results {
		if len(mangaList) == 0 {
			continue
		}

		f.PrintSection(fmt.Sprintf("Results from %s (%d)", providerID, len(mangaList)))

		for i, manga := range mangaList {
			f.PrintMangaItem(manga, providerID, i+1)
		}
	}
}

// PrintChapterItem prints information about a single chapter
func (f *Formatter) PrintChapterItem(chapter core.ChapterInfo, providerID string, number int) {
	// Format the item number and chapter ID
	itemPrefix := ""
	if number > 0 {
		itemPrefix = fmt.Sprintf("%d. ", number)
	}

	chapterID := core.FormatMangaID(providerID, chapter.ID)

	// Format title (use chapter number if title is empty)
	title := chapter.Title
	if title == "" {
		title = fmt.Sprintf("Chapter %g", chapter.Number)
	}

	// Print title line
	_, err := f.TitleStyle.Fprintf(f.Writer, "%s%s ", itemPrefix, title)
	if err != nil {
		return
	}
	_, err = f.IDStyle.Fprintf(f.Writer, "(ID: %s)\n", chapterID)
	if err != nil {
		return
	}

	// Format details line
	chapterNum := "?"
	if chapter.Number > 0 {
		chapterNum = fmt.Sprintf("%g", chapter.Number)
	}

	// Print chapter details
	_, err = f.DetailLabelStyle.Fprintf(f.Writer, "  Chapter %s", chapterNum)
	if err != nil {
		return
	}
	_, err = f.SecondaryStyle.Fprintf(f.Writer, " | ")
	if err != nil {
		return
	}
	_, err = f.DetailLabelStyle.Fprintf(f.Writer, "Released: ")
	if err != nil {
		return
	}
	_, err = f.DetailValueStyle.Fprintf(f.Writer, "%s", f.FormatDate(chapter.Date))
	if err != nil {
		return
	}
	_, err = f.SecondaryStyle.Fprintf(f.Writer, " | ")
	if err != nil {
		return
	}
	_, err = f.DetailLabelStyle.Fprintf(f.Writer, "Language: ")
	if err != nil {
		return
	}
	_, err = f.DetailValueStyle.Fprintln(f.Writer, f.FormatLanguage(chapter.Language))
	if err != nil {
		return
	}

	// Add a blank line for readability
	_, err = fmt.Fprintln(f.Writer, "")
	if err != nil {
		return
	}
}

// PrintMangaInfo prints detailed manga information including chapters
func (f *Formatter) PrintMangaInfo(manga *core.MangaInfo, prov provider.Provider) {
	if manga == nil {
		f.PrintError("No manga information available.")
		return
	}

	// Print manga header
	f.PrintHeader(manga.Title)

	// Print manga ID
	mangaID := core.FormatMangaID(prov.ID(), manga.ID)
	f.PrintDetail("ID", mangaID)

	// Print provider info
	f.PrintDetail("Provider", fmt.Sprintf("%s (%s)", prov.ID(), prov.Name()))

	// Print alt titles if available
	if len(manga.AltTitles) > 0 {
		f.PrintDetail("Alternative Titles", strings.Join(manga.AltTitles, ", "))
	}

	// Print authors
	if len(manga.Authors) > 0 {
		f.PrintDetail("Authors", strings.Join(manga.Authors, ", "))
	}

	// Print status
	if manga.Status != "" {
		f.PrintDetail("Status", manga.Status)
	}

	// Print last updated
	if manga.LastUpdated != nil {
		f.PrintDetail("Last Updated", f.FormatDate(manga.LastUpdated))
	}

	// Print tags
	if len(manga.Tags) > 0 {
		f.PrintDetail("Tags", strings.Join(manga.Tags, ", "))
	}

	// Print description
	if manga.Description != "" {
		f.PrintNewLine()
		_, err := f.DetailLabelStyle.Fprintln(f.Writer, "Description:")
		if err != nil {
			return
		}
		_, err = fmt.Fprintln(f.Writer, manga.Description)
		if err != nil {
			return
		}
	}

	// Print chapters section
	if len(manga.Chapters) > 0 {
		f.PrintSection(fmt.Sprintf("Chapters (%d)", len(manga.Chapters)))

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

		for i, chapter := range sortedChapters {
			f.PrintChapterItem(chapter, prov.ID(), i+1)
		}
	} else {
		f.PrintWarning("No chapters available.")
	}
}

// PrintDownloadInfo prints information about a download
func (f *Formatter) PrintDownloadInfo(chapterID, providerID, providerName, outputDir string, maxConcurrency int, volumeOverride int, hasVolumeOverride bool) {
	f.PrintHeader("Download Information")

	f.PrintDetail("Chapter ID", f.FormatID(chapterID))
	f.PrintDetail("Provider", fmt.Sprintf("%s (%s)", providerID, providerName))
	f.PrintDetail("Output directory", f.FormatPath(outputDir))
	f.PrintDetail("Concurrent downloads", f.FormatNumber(maxConcurrency))

	if hasVolumeOverride {
		f.PrintDetail("Volume override", f.FormatNumber(volumeOverride))
	}
}

// DefaultFormatter Global instance for convenience
var DefaultFormatter = NewFormatter()
