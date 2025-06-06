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

package util

import (
	"Luminary/pkg/engine/core"
	"fmt"
	"sort"
	"strings"
)

// LanguageFilter holds configuration for filtering chapters by language
type LanguageFilter struct {
	Languages []string // List of language codes/names to include
}

// NewLanguageFilter creates a new language filter from a comma-separated string
func NewLanguageFilter(languageStr string) *LanguageFilter {
	if languageStr == "" {
		return nil
	}

	// Split by comma and clean up
	parts := strings.Split(languageStr, ",")
	var languages []string
	for _, part := range parts {
		lang := strings.TrimSpace(part)
		if lang != "" {
			languages = append(languages, strings.ToLower(lang))
		}
	}

	if len(languages) == 0 {
		return nil
	}

	return &LanguageFilter{
		Languages: languages,
	}
}

// MatchesLanguage checks if a chapter's language matches any of the filter languages
func (lf *LanguageFilter) MatchesLanguage(chapterLanguage *string) bool {
	if lf == nil || len(lf.Languages) == 0 {
		return true // No filter means all languages pass
	}

	if chapterLanguage == nil {
		// Chapter has no language specified
		// Check if filter includes "unknown", "none", or empty
		for _, filterLang := range lf.Languages {
			if filterLang == "unknown" || filterLang == "none" || filterLang == "" {
				return true
			}
		}
		return false
	}

	chapterLang := strings.ToLower(*chapterLanguage)

	// Check direct match with language codes
	for _, filterLang := range lf.Languages {
		if filterLang == chapterLang {
			return true
		}
	}

	// Check if any filter language matches the full language name
	// Convert chapter language code to full name and check
	if fullName, exists := languageCodeToFullName[chapterLang]; exists {
		fullNameLower := strings.ToLower(fullName)
		for _, filterLang := range lf.Languages {
			if filterLang == fullNameLower {
				return true
			}
		}
	}

	// Check if any filter language is a code that matches our chapter's full name
	for _, filterLang := range lf.Languages {
		if fullName, exists := languageCodeToFullName[filterLang]; exists {
			if strings.ToLower(fullName) == chapterLang {
				return true
			}
		}
	}

	return false
}

// FilterChapters filters a slice of chapters based on language
func (lf *LanguageFilter) FilterChapters(chapters []core.ChapterInfo) []core.ChapterInfo {
	if lf == nil {
		return chapters
	}

	var filtered []core.ChapterInfo
	for _, chapter := range chapters {
		if lf.MatchesLanguage(chapter.Language) {
			filtered = append(filtered, chapter)
		}
	}

	return filtered
}

// GetAvailableLanguages extracts all unique languages from a slice of chapters
func GetAvailableLanguages(chapters []core.ChapterInfo) []string {
	languageSet := make(map[string]struct{})

	for _, chapter := range chapters {
		if chapter.Language != nil {
			languageSet[*chapter.Language] = struct{}{}
		} else {
			languageSet["unknown"] = struct{}{}
		}
	}

	var languages []string
	for lang := range languageSet {
		languages = append(languages, lang)
	}

	sort.Strings(languages)
	return languages
}

// FormatAvailableLanguages formats available languages for display
func FormatAvailableLanguages(chapters []core.ChapterInfo) string {
	languages := GetAvailableLanguages(chapters)
	if len(languages) == 0 {
		return "None"
	}

	var formatted []string
	for _, lang := range languages {
		if lang == "unknown" {
			formatted = append(formatted, "unknown")
		} else if fullName, exists := languageCodeToFullName[lang]; exists {
			formatted = append(formatted, fmt.Sprintf("%s (%s)", fullName, lang))
		} else {
			formatted = append(formatted, lang)
		}
	}

	return strings.Join(formatted, ", ")
}

// languageCodeToFullName maps common language codes to full names
var languageCodeToFullName = map[string]string{
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
