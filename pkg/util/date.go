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
	"strings"
	"time"
)

// ParseNullableDate safely parses a date string, returning nil if empty or invalid
func ParseNullableDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// Try various date formats commonly used on manga sites
	formats := []string{
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"2006/01/02",
		"2 January 2006",
		"2 Jan 2006",
		"January 2006",
		"Jan 2006",
		"2006",
	}

	// Clean up the date string
	dateStr = strings.TrimSpace(dateStr)

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t
		}
	}

	return nil
}
