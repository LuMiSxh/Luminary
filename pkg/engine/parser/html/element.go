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
	"github.com/PuerkitoBio/goquery"
	"strings"
)

// Element wraps a goquery selection for easier access
type Element struct {
	selection *goquery.Selection
}

// Text returns the text content of the element
func (e *Element) Text() string {
	return strings.TrimSpace(e.selection.Text())
}

// HTML returns the HTML content of the element
func (e *Element) HTML() (string, error) {
	return e.selection.Html()
}

// InnerHTML returns the inner HTML content
func (e *Element) InnerHTML() string {
	html, _ := e.HTML()
	return html
}

// Attr returns an attribute value
func (e *Element) Attr(name string) (string, bool) {
	return e.selection.Attr(name)
}

// AttrOr returns an attribute value or default if not found
func (e *Element) AttrOr(name string, defaultValue string) string {
	if val, exists := e.Attr(name); exists {
		return val
	}
	return defaultValue
}

// HasAttr checks if an attribute exists
func (e *Element) HasAttr(name string) bool {
	_, exists := e.Attr(name)
	return exists
}

// ID returns the element's ID attribute
func (e *Element) ID() string {
	return e.AttrOr("id", "")
}

// Classes returns all classes of the element
func (e *Element) Classes() []string {
	classStr := e.AttrOr("class", "")
	if classStr == "" {
		return nil
	}
	return strings.Fields(classStr)
}

// HasClass checks if the element has a specific class
func (e *Element) HasClass(class string) bool {
	return e.selection.HasClass(class)
}

// Extract returns an extractor for the element
func (e *Element) Extract() *Extractor {
	return &Extractor{element: e}
}

// Find searches within this element
func (e *Element) Find(selector string) *Selector {
	subDoc := &Parser{doc: &goquery.Document{Selection: e.selection}}
	return subDoc.Select(selector)
}

// Parent returns the parent element
func (e *Element) Parent() *Element {
	parent := e.selection.Parent()
	if parent.Length() == 0 {
		return nil
	}
	return &Element{selection: parent}
}

// Next returns the next sibling element
func (e *Element) Next() *Element {
	next := e.selection.Next()
	if next.Length() == 0 {
		return nil
	}
	return &Element{selection: next}
}

// Prev returns the previous sibling element
func (e *Element) Prev() *Element {
	prev := e.selection.Prev()
	if prev.Length() == 0 {
		return nil
	}
	return &Element{selection: prev}
}

// Children returns all child elements
func (e *Element) Children() []*Element {
	var children []*Element

	e.selection.Children().Each(func(i int, s *goquery.Selection) {
		children = append(children, &Element{selection: s})
	})

	return children
}

// Is checks if the element matches a selector
func (e *Element) Is(selector string) bool {
	return e.selection.Is(selector)
}
