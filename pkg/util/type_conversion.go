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
	"strconv"
	"strings"
	"time"
)

func ParseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// CleanImageURL handles cleaning the image URLs by removing control characters
// and extracting URLs that might be embedded in the URL
func CleanImageURL(dirtyURL string) string {
	// First handle the case where the actual URL is embedded with control characters
	if strings.Contains(dirtyURL, "http") {
		// Find all URLs in the string
		re := regexp.MustCompile(`https?://[^\s\t\n\r]+`)
		matches := re.FindAllString(dirtyURL, -1)

		if len(matches) > 0 {
			// Take the last match, which is likely the actual image URL
			return matches[len(matches)-1]
		}
	}

	// Remove control characters
	return regexp.MustCompile(`[\t\n\r]+`).ReplaceAllString(dirtyURL, "")
}

// FormatNullableDate formats a nullable date pointer for display
func FormatNullableDate(date *time.Time) string {
	if date == nil {
		return "Not specified"
	}
	return FormatDate(*date)
}

// FormatNullableLanguage formats a nullable language pointer for display
func FormatNullableLanguage(language *string) string {
	if language == nil {
		return "Not specified"
	}
	return *language
}

// ParseNullableTime safely parses a time string, returning nil if empty or invalid
func ParseNullableTime(timeStr string) *time.Time {
	if timeStr == "" {
		return nil
	}

	// Try parsing RFC3339 format first (common for APIs)
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return &t
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
		"01/02/2006",
		"02/01/2006",
		"2006/01/02",
		"2 January 2006",
		"2 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return &t
		}
	}

	return nil
}

// TimeToNullableString converts a nullable time pointer to a nullable string pointer for JSON
func TimeToNullableString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	str := t.Format(time.RFC3339)
	return &str
}

// StringToNullableTime converts a nullable string pointer to a nullable time pointer
func StringToNullableTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	return ParseNullableTime(*s)
}
