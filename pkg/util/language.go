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
	"regexp"
	"strings"
)

// Language detection patterns for extracting language from titles and URLs
var languagePatterns = map[string]*regexp.Regexp{
	"en": regexp.MustCompile(`(?i)\b(english|eng|en)\b`),
	"es": regexp.MustCompile(`(?i)\b(spanish|español|esp|es)\b`),
	"fr": regexp.MustCompile(`(?i)\b(french|français|fra|fr)\b`),
	"de": regexp.MustCompile(`(?i)\b(german|deutsch|ger|de)\b`),
	"pt": regexp.MustCompile(`(?i)\b(portuguese|português|port|pt)\b`),
	"it": regexp.MustCompile(`(?i)\b(italian|italiano|ita|it)\b`),
	"ru": regexp.MustCompile(`(?i)\b(russian|русский|rus|ru)\b`),
	"ja": regexp.MustCompile(`(?i)\b(japanese|日本語|jpn|ja)\b`),
	"ko": regexp.MustCompile(`(?i)\b(korean|한국어|kor|ko)\b`),
	"zh": regexp.MustCompile(`(?i)\b(chinese|中文|chi|zh|mandarin|cantonese)\b`),
	"ar": regexp.MustCompile(`(?i)\b(arabic|العربية|ara|ar)\b`),
	"tr": regexp.MustCompile(`(?i)\b(turkish|türkçe|tur|tr)\b`),
	"nl": regexp.MustCompile(`(?i)\b(dutch|nederlands|nld|nl)\b`),
	"pl": regexp.MustCompile(`(?i)\b(polish|polski|pol|pl)\b`),
	"th": regexp.MustCompile(`(?i)\b(thai|ไทย|tha|th)\b`),
	"vi": regexp.MustCompile(`(?i)\b(vietnamese|tiếng việt|vie|vi)\b`),
	"id": regexp.MustCompile(`(?i)\b(indonesian|bahasa indonesia|ind|id)\b`),
}

// Additional patterns for scanlation groups that might indicate language
var scanlationPatterns = map[string]*regexp.Regexp{
	"en": regexp.MustCompile(`(?i)\b(scan|scans|scanlation|translation|trans|tl)\b`),
	"es": regexp.MustCompile(`(?i)\b(lectura|manga)\b`),
}

// Handle common non-standard language codes that might appear in URLs or data
var nonStandardCodeMap = map[string]string{
	"es-la": "es",    // Spanish (Latin America) -> Spanish
	"uk":    "uk",    // Ukrainian
	"cs":    "cs",    // Czech
	"zh-hk": "zh-hk", // Chinese (Hong Kong)
	"zh-tw": "zh-tw", // Chinese (Taiwan)
	"zh-cn": "zh-cn", // Chinese (Simplified)
	"pt-br": "pt",    // Portuguese (Brazil) -> Portuguese
}

// DetectLanguageFromText attempts to detect language from title or URL
// Returns a pointer to the detected language code using ISO 639-1 standard, or nil if no language is detected
func DetectLanguageFromText(text string) *string {
	if text == "" {
		return nil
	}

	// First check for direct language codes in the text (like "en", "es-la", etc.)
	for nonStandard, standard := range nonStandardCodeMap {
		if strings.Contains(strings.ToLower(text), nonStandard) {
			return &standard
		}
	}

	// Check against known language patterns
	for langCode, pattern := range languagePatterns {
		if pattern.MatchString(text) {
			return &langCode
		}
	}

	// Check for common scanlation group patterns that might indicate language
	for langCode, pattern := range scanlationPatterns {
		if pattern.MatchString(text) {
			return &langCode
		}
	}

	return nil
}
