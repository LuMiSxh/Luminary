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
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Extractor provides data extraction utilities for elements
type Extractor struct {
	element *Element
}

// Href returns the href attribute, commonly used for links
func (e *Extractor) Href() string {
	return e.element.AttrOr("href", "")
}

// AbsHref returns an absolute URL from href attribute
func (e *Extractor) AbsHref(baseURL string) string {
	href := e.Href()
	if href == "" {
		return ""
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}

	// Parse href
	u, err := url.Parse(href)
	if err != nil {
		return href
	}

	// Resolve against base
	return base.ResolveReference(u).String()
}

// Src returns the src attribute, commonly used for images
func (e *Extractor) Src() string {
	return e.element.AttrOr("src", "")
}

// AbsSrc returns an absolute URL from src attribute
func (e *Extractor) AbsSrc(baseURL string) string {
	src := e.Src()
	if src == "" {
		return ""
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return src
	}

	u, err := url.Parse(src)
	if err != nil {
		return src
	}

	return base.ResolveReference(u).String()
}

// Alt returns the alt attribute
func (e *Extractor) Alt() string {
	return e.element.AttrOr("alt", "")
}

// Title returns the title attribute
func (e *Extractor) Title() string {
	return e.element.AttrOr("title", "")
}

// Value returns the value attribute
func (e *Extractor) Value() string {
	return e.element.AttrOr("value", "")
}

// Data returns a data attribute value
func (e *Extractor) Data(key string) string {
	return e.element.AttrOr("data-"+key, "")
}

// Number extracts a number from the element's text
func (e *Extractor) Number() (float64, error) {
	text := e.element.Text()

	// Try to find a number in the text
	re := regexp.MustCompile(`[\d,]+\.?\d*`)
	match := re.FindString(text)
	if match == "" {
		return 0, nil
	}

	// Remove commas
	match = strings.ReplaceAll(match, ",", "")

	return strconv.ParseFloat(match, 64)
}

// NumberOr returns a number or default value
func (e *Extractor) NumberOr(defaultValue float64) float64 {
	n, err := e.Number()
	if err != nil {
		return defaultValue
	}
	return n
}

// Int extracts an integer from the element's text
func (e *Extractor) Int() (int, error) {
	n, err := e.Number()
	return int(n), err
}

// IntOr returns an integer or default value
func (e *Extractor) IntOr(defaultValue int) int {
	n, _ := e.Int()
	if n == 0 && e.element.Text() != "0" {
		return defaultValue
	}
	return n
}

// TextNodes returns only direct text nodes, excluding child element text
func (e *Extractor) TextNodes() []string {
	var texts []string

	e.element.selection.Contents().Each(func(i int, s *goquery.Selection) {
		if s.Nodes != nil && len(s.Nodes) > 0 && s.Nodes[0].Type == 1 { // Text node
			if text := strings.TrimSpace(s.Text()); text != "" {
				texts = append(texts, text)
			}
		}
	})

	return texts
}

// Text returns the full text content of the element
func (e *Extractor) Text() string {
	return e.element.Text()
}

// CleanText returns text with normalized whitespace
func (e *Extractor) CleanText() string {
	text := e.element.Text()

	// Normalize whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// Links extracts all links from the element
func (e *Extractor) Links() []Link {
	var links []Link

	e.element.Find("a[href]").Each(func(_ int, elem *Element) {
		href := elem.Extract().Href()
		if href != "" {
			links = append(links, Link{
				URL:  href,
				Text: elem.Text(),
			})
		}
	})

	return links
}

// Images extracts all images from the element
func (e *Extractor) Images() []Image {
	var images []Image

	e.element.Find("img[src]").Each(func(_ int, elem *Element) {
		src := elem.Extract().Src()
		if src != "" {
			images = append(images, Image{
				URL: src,
				Alt: elem.Extract().Alt(),
			})
		}
	})

	return images
}

// Table extracts data from a table element
func (e *Extractor) Table() [][]string {
	var rows [][]string

	e.element.Find("tr").Each(func(_ int, row *Element) {
		var cells []string

		row.Find("td, th").Each(func(_ int, cell *Element) {
			cells = append(cells, cell.Text())
		})

		if len(cells) > 0 {
			rows = append(rows, cells)
		}
	})

	return rows
}

// Link represents an extracted link
type Link struct {
	URL  string
	Text string
}

// Image represents an extracted image
type Image struct {
	URL string
	Alt string
}
