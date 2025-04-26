package engine

import (
	"regexp"
)

type ParserService struct {
	RegexPatterns map[string]*regexp.Regexp
}
