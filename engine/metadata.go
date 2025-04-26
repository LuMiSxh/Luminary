package engine

import (
	"Luminary/utils"
	"strings"
)

// MetadataService provides metadata extraction capabilities
type MetadataService struct {
	Parser *ParserService
}

// ExtractChapterInfo extracts chapter and volume numbers from a title
func (m *MetadataService) ExtractChapterInfo(title string) (chapterNum *float64, volumeNum *int) {
	// Extract chapter number
	if matches := m.Parser.RegexPatterns["chapterNumber"].FindStringSubmatch(title); len(matches) > 1 {
		if val, err := utils.ParseFloat64(matches[1]); err == nil {
			chapterNum = &val
		}
	}

	// Extract volume number
	if matches := m.Parser.RegexPatterns["volumeNumber"].FindStringSubmatch(title); len(matches) > 1 {
		if val, err := utils.ParseInt(matches[1]); err == nil {
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
