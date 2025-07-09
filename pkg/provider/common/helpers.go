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

package common

import (
	"strings"
	"time"
)

// ExtractBestTitle selects the most appropriate title from a map of localized strings.
// It prioritizes English ("en"), then the first available non-empty title as a fallback.
// This is useful for APIs that return multilingual data.
func ExtractBestTitle(titleMap map[string]string) string {
	if title, ok := titleMap["en"]; ok && strings.TrimSpace(title) != "" {
		return title
	}
	// Fallback to the first non-empty title available
	for _, title := range titleMap {
		if strings.TrimSpace(title) != "" {
			return title
		}
	}
	return "Untitled" // Default if no titles are found
}

// ParseDate attempts to parse a date string using a list of common layouts.
// It returns a pointer to a time.Time object on success, or nil if parsing fails.
// This is useful for handling nullable or inconsistently formatted date fields from APIs.
func ParseDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// A list of common date/time formats to try
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01-02-2006",
		"2 January 2006",
		"Jan 2, 2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return &t
		}
	}

	// Return nil if no format matches
	return nil
}
