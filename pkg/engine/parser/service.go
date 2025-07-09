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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Luminary/pkg/engine/logger"
	"Luminary/pkg/engine/parser/html"
	"Luminary/pkg/errors"
)

// Service provides unified parsing capabilities
type Service struct {
	logger   logger.Logger
	patterns map[string]*regexp.Regexp
}

// NewService creates a new parser service
func NewService(logger logger.Logger) *Service {
	return &Service{
		logger: logger,
		patterns: map[string]*regexp.Regexp{
			"chapter_number": regexp.MustCompile(`(?i)(?:chapter|ch\.?|episode|ep\.?)[\s:]*(\d+(?:\.\d+)?)`),
			"volume_number":  regexp.MustCompile(`(?i)(?:volume|vol\.?)[\s:]*(\d+)`),
			"date":           regexp.MustCompile(`(\d{4}[-/]\d{2}[-/]\d{2})`),
			"year":           regexp.MustCompile(`\b(19|20)\d{2}\b`),
			"number":         regexp.MustCompile(`\d+(?:\.\d+)?`),
			"url":            regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`),
			"image_ext":      regexp.MustCompile(`\.(jpg|jpeg|png|gif|webp|bmp)$`),
		},
	}
}

// ParseHTML parses HTML content
func (s *Service) ParseHTML(content []byte) (*html.Parser, error) {
	return html.Parse(content)
}

// ParseHTMLString parses HTML from string
func (s *Service) ParseHTMLString(content string) (*html.Parser, error) {
	return html.ParseString(content)
}

// ParseJSON parses JSON content into the provided structure
func (s *Service) ParseJSON(content []byte, v interface{}) error {
	if err := json.Unmarshal(content, v); err != nil {
		return errors.Track(err).
			WithContext("content_preview", string(content[:min(len(content), 200)])).
			AsParser().
			Error()
	}
	return nil
}

// ExtractChapterNumber extracts chapter number from text
func (s *Service) ExtractChapterNumber(text string) (float64, error) {
	pattern := s.patterns["chapter_number"]
	matches := pattern.FindStringSubmatch(text)

	if len(matches) > 1 {
		num, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, errors.Track(err).
				WithContext("text", text).
				AsParser().
				Error()
		}
		return num, nil
	}

	// Fallback: try to find any number
	numPattern := s.patterns["number"]
	if match := numPattern.FindString(text); match != "" {
		num, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return 0, errors.Track(err).
				WithContext("text", text).
				AsParser().
				Error()
		}
		return num, nil
	}

	return 0, errors.Track(fmt.Errorf("no chapter number found")).
		WithContext("text", text).
		AsParser().
		Error()
}

// ExtractVolumeNumber extracts volume number from text
func (s *Service) ExtractVolumeNumber(text string) (int, error) {
	pattern := s.patterns["volume_number"]
	matches := pattern.FindStringSubmatch(text)

	if len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, errors.Track(err).
				WithContext("text", text).
				AsParser().
				Error()
		}
		return num, nil
	}

	return 0, errors.Track(fmt.Errorf("no volume number found")).
		WithContext("text", text).
		AsParser().
		Error()
}

// ExtractDate attempts to extract a date from text
func (s *Service) ExtractDate(text string) (*time.Time, error) {
	// Common date formats to try
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"02/01/2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"2 January 2006",
	}

	// Try each format
	for _, format := range formats {
		if t, err := time.Parse(format, text); err == nil {
			return &t, nil
		}
	}

	// Try regex pattern
	pattern := s.patterns["date"]
	if match := pattern.FindString(text); match != "" {
		// Normalize separators
		normalized := strings.ReplaceAll(match, "/", "-")
		if t, err := time.Parse("2006-01-02", normalized); err == nil {
			return &t, nil
		}
	}

	return nil, errors.Track(fmt.Errorf("no valid date found")).
		WithContext("text", text).
		AsParser().
		Error()
}

// ExtractURLs extracts all URLs from text
func (s *Service) ExtractURLs(text string) []string {
	pattern := s.patterns["url"]
	matches := pattern.FindAllString(text, -1)

	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0, len(matches))

	for _, url := range matches {
		if !seen[url] {
			seen[url] = true
			unique = append(unique, url)
		}
	}

	return unique
}

// ExtractImageURLs extracts URLs that appear to be images
func (s *Service) ExtractImageURLs(text string) []string {
	urls := s.ExtractURLs(text)
	pattern := s.patterns["image_ext"]

	var images []string
	for _, url := range urls {
		if pattern.MatchString(strings.ToLower(url)) {
			images = append(images, url)
		}
	}

	return images
}

// CleanText normalizes text by removing extra whitespace
func (s *Service) CleanText(text string) string {
	// Normalize whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Remove zero-width characters
	text = strings.ReplaceAll(text, "\u200b", "")
	text = strings.ReplaceAll(text, "\u200c", "")
	text = strings.ReplaceAll(text, "\u200d", "")
	text = strings.ReplaceAll(text, "\ufeff", "")

	return strings.TrimSpace(text)
}

// SanitizeFilename makes a string safe for use as a filename
func (s *Service) SanitizeFilename(name string) string {
	// Replace invalid characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)

	name = replacer.Replace(name)
	name = s.CleanText(name)

	// Limit length
	if len(name) > 200 {
		name = name[:200]
	}

	return name
}

// CompareStrings performs fuzzy string comparison
func (s *Service) CompareStrings(a, b string) float64 {
	a = strings.ToLower(s.CleanText(a))
	b = strings.ToLower(s.CleanText(b))

	if a == b {
		return 1.0
	}

	// Simple character-based similarity
	longer := a
	shorter := b
	if len(b) > len(a) {
		longer = b
		shorter = a
	}

	if len(longer) == 0 {
		return 0.0
	}

	matches := 0
	for _, ch := range shorter {
		if strings.ContainsRune(longer, ch) {
			matches++
		}
	}

	return float64(matches) / float64(len(longer))
}

// UrlJoin joins URL parts safely
func UrlJoin(base string, parts ...string) string {
	result := strings.TrimRight(base, "/")

	for _, part := range parts {
		part = strings.TrimLeft(part, "/")
		if part != "" {
			result += "/" + part
		}
	}

	return result
}
