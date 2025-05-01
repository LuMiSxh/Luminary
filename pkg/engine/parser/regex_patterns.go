package parser

import (
	"regexp"
)

type Service struct {
	RegexPatterns map[string]*regexp.Regexp
}

// CompilePattern compiles a regex pattern and caches it for future use
func (p *Service) CompilePattern(pattern string) *regexp.Regexp {
	re, found := p.RegexPatterns[pattern]
	if found {
		return re
	}

	// If not found, compile it
	re = regexp.MustCompile(pattern)
	p.RegexPatterns[pattern] = re
	return re
}
