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
	"bytes"
	"io"
	"strings"

	"Luminary/pkg/errors"
	"github.com/PuerkitoBio/goquery"
)

// Parser wraps goquery document for HTML parsing
type Parser struct {
	doc *goquery.Document
}

// Parse creates a new parser from HTML content
func Parse(content []byte) (*Parser, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		return nil, errors.Track(err).
			WithContext("operation", "html_parse").
			WithContext("content_size", len(content)).
			Error()
	}
	return &Parser{doc: doc}, nil
}

// ParseReader creates a new parser from an io.Reader
func ParseReader(r io.Reader) (*Parser, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, errors.Track(err).
			WithContext("operation", "html_parse_reader").
			Error()
	}
	return &Parser{doc: doc}, nil
}

// ParseString creates a new parser from a string
func ParseString(html string) (*Parser, error) {
	return Parse([]byte(html))
}

// Select returns a selector for querying elements
func (p *Parser) Select(selector string) *Selector {
	return &Selector{
		parser:   p,
		selector: selector,
	}
}

// Find is an alias for Select for familiarity
func (p *Parser) Find(selector string) *Selector {
	return p.Select(selector)
}

// Text returns all text content in the document
func (p *Parser) Text() string {
	return strings.TrimSpace(p.doc.Text())
}

// HTML returns the HTML content of the document
func (p *Parser) HTML() (string, error) {
	return p.doc.Html()
}

// Title returns the document title
func (p *Parser) Title() string {
	return p.doc.Find("title").Text()
}

// Meta returns a map of meta tags
func (p *Parser) Meta() map[string]string {
	meta := make(map[string]string)

	p.doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, exists := s.Attr("name"); exists {
			if content, exists := s.Attr("content"); exists {
				meta[name] = content
			}
		}

		// Also handle property attribute (for og: tags)
		if property, exists := s.Attr("property"); exists {
			if content, exists := s.Attr("content"); exists {
				meta[property] = content
			}
		}
	})

	return meta
}

// Links returns all links in the document
func (p *Parser) Links() []string {
	var links []string

	p.doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && href != "" {
			links = append(links, href)
		}
	})

	return links
}

// Images returns all image URLs in the document
func (p *Parser) Images() []string {
	var images []string

	p.doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			images = append(images, src)
		}
	})

	return images
}
