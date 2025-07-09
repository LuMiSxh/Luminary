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

package html

import (
	"Luminary/pkg/errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
)

// Selector provides methods for querying HTML elements
type Selector struct {
	parser   *Parser
	selector string
}

// First returns the first element matching the selector
func (s *Selector) First() (*Element, error) {
	selection := s.parser.doc.Find(s.selector).First()
	if selection.Length() == 0 {
		return nil, errors.Track(fmt.Errorf("no elements found")).
			WithContext("selector", s.selector).
			Error()
	}
	return &Element{selection: selection}, nil
}

// FirstOrNil returns the first element or nil if not found
func (s *Selector) FirstOrNil() *Element {
	selection := s.parser.doc.Find(s.selector).First()
	if selection.Length() == 0 {
		return nil
	}
	return &Element{selection: selection}
}

// All returns all elements matching the selector
func (s *Selector) All() ([]*Element, error) {
	var elements []*Element

	s.parser.doc.Find(s.selector).Each(func(i int, sel *goquery.Selection) {
		elements = append(elements, &Element{selection: sel})
	})

	if len(elements) == 0 {
		return nil, errors.Track(fmt.Errorf("no elements found")).
			WithContext("selector", s.selector).
			Error()
	}

	return elements, nil
}

// AllOrEmpty returns all elements or empty slice if none found
func (s *Selector) AllOrEmpty() []*Element {
	var elements []*Element

	s.parser.doc.Find(s.selector).Each(func(i int, sel *goquery.Selection) {
		elements = append(elements, &Element{selection: sel})
	})

	return elements
}

// Count returns the number of elements matching the selector
func (s *Selector) Count() int {
	return s.parser.doc.Find(s.selector).Length()
}

// Exists checks if any elements match the selector
func (s *Selector) Exists() bool {
	return s.Count() > 0
}

// Each iterates over all matching elements
func (s *Selector) Each(fn func(int, *Element)) {
	s.parser.doc.Find(s.selector).Each(func(i int, sel *goquery.Selection) {
		fn(i, &Element{selection: sel})
	})
}

// Map transforms matching elements using the provided function
func (s *Selector) Map(fn func(*Element) interface{}) []interface{} {
	var results []interface{}

	s.Each(func(_ int, elem *Element) {
		if result := fn(elem); result != nil {
			results = append(results, result)
		}
	})

	return results
}

// MapString is a convenience method for mapping to strings
func (s *Selector) MapString(fn func(*Element) string) []string {
	var results []string

	s.Each(func(_ int, elem *Element) {
		if result := fn(elem); result != "" {
			results = append(results, result)
		}
	})

	return results
}

// Filter returns elements that match the predicate
func (s *Selector) Filter(predicate func(*Element) bool) []*Element {
	var filtered []*Element

	s.Each(func(_ int, elem *Element) {
		if predicate(elem) {
			filtered = append(filtered, elem)
		}
	})

	return filtered
}

// MultiSelector allows trying multiple selectors in order
type MultiSelector struct {
	parser    *Parser
	selectors []string
}

// MultiSelect creates a selector that tries multiple CSS selectors
func (p *Parser) MultiSelect(selectors ...string) *MultiSelector {
	return &MultiSelector{
		parser:    p,
		selectors: selectors,
	}
}

// First returns the first element found using any of the selectors
func (m *MultiSelector) First() (*Element, error) {
	for _, selector := range m.selectors {
		elem := m.parser.Select(selector).FirstOrNil()
		if elem != nil {
			return elem, nil
		}
	}

	return nil, errors.Track(fmt.Errorf("no elements found")).
		WithContext("selectors", m.selectors).
		Error()
}

// All returns all elements found using all selectors
func (m *MultiSelector) All() []*Element {
	seen := make(map[*goquery.Selection]bool)
	var elements []*Element

	for _, selector := range m.selectors {
		m.parser.Select(selector).Each(func(_ int, elem *Element) {
			if !seen[elem.selection] {
				seen[elem.selection] = true
				elements = append(elements, elem)
			}
		})
	}

	return elements
}
