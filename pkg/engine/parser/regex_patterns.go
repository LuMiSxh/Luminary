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
