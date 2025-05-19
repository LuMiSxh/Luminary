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

// FormatDate formats a time.Time value for display
func FormatDate(date time.Time) string {
	if date.IsZero() {
		return "Unknown"
	}
	return date.Format("2006-01-02")
}
