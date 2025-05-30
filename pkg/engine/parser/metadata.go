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

package parser

import (
	"Luminary/pkg/util"
	"strings"
)

// MetadataService provides metadata extraction capabilities
type MetadataService struct {
	Parser *Service
}

// ExtractChapterInfo extracts chapter and volume numbers from a title
func (m *MetadataService) ExtractChapterInfo(title string) (chapterNum *float64, volumeNum *int) {
	// Extract chapter number
	if matches := m.Parser.RegexPatterns["chapterNumber"].FindStringSubmatch(title); len(matches) > 1 {
		if val, err := util.ParseFloat64(matches[1]); err == nil {
			chapterNum = &val
		}
	}

	// Extract volume number
	if matches := m.Parser.RegexPatterns["volumeNumber"].FindStringSubmatch(title); len(matches) > 1 {
		if val, err := util.ParseInt(matches[1]); err == nil {
			volumeNum = &val
		}
	}

	return chapterNum, volumeNum
}

// ExtractAuthors extracts author names from text
func (m *MetadataService) ExtractAuthors(text string) []string {
	// Could be improved with NLP or more sophisticated regex
	var authors []string

	// Look for common author patterns
	authorPatterns := []string{
		`(?i)(?:author|writer|creator|mangaka)[:]\s*([^,]+)`,
		`(?i)(?:by|written by)\s+([^,]+)`,
	}

	for _, pattern := range authorPatterns {
		re := m.Parser.CompilePattern(pattern)
		if matches := re.FindStringSubmatch(text); len(matches) > 1 {
			authors = append(authors, strings.TrimSpace(matches[1]))
		}
	}

	return authors
}
