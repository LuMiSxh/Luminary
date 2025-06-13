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
	"Luminary/pkg/errors"
	"fmt"
	"golang.org/x/net/html"
	"regexp"
	"strconv"
	"strings"
)

// DOMService provides DOM parsing and querying capabilities
type DOMService struct {
	// Config options - Not used for this service
}

// Element represents an HTML element with helper methods
type Element struct {
	Node *html.Node
	DOM  *DOMService
}

// Parse parses HTML content into a DOM
func (d *DOMService) Parse(content string) (*html.Node, error) {
	return html.Parse(strings.NewReader(content))
}

// ParseHTML parses HTML content and returns the root element
func (d *DOMService) ParseHTML(content string) (*Element, error) {
	node, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, errors.T(err)
	}

	return &Element{Node: node, DOM: d}, nil
}

// QuerySelector finds the first element matching the selector
func (d *DOMService) QuerySelector(node *html.Node, selector string) (*html.Node, error) {
	elements, err := d.parseSelector(node, selector, true)
	if err != nil {
		return nil, errors.T(err)
	}

	if len(elements) == 0 {
		return nil, errors.T(fmt.Errorf("element not found: %s", selector))
	}

	return elements[0], nil
}

// QuerySelectorAll finds all elements matching the selector
func (d *DOMService) QuerySelectorAll(node *html.Node, selector string) ([]*html.Node, error) {
	return d.parseSelector(node, selector, false)
}

// QuerySelectorWithContext finds the first element matching the selector and returns an Element
func (d *DOMService) QuerySelectorWithContext(root *html.Node, selector string) (*Element, error) {
	elements, err := d.parseSelector(root, selector, true)
	if err != nil {
		return nil, errors.T(err)
	}

	if len(elements) == 0 {
		return nil, fmt.Errorf("no element found matching selector: %s", selector)
	}

	return &Element{Node: elements[0], DOM: d}, nil
}

// QuerySelectorAllWithContext finds all elements matching the selector and returns Elements
func (d *DOMService) QuerySelectorAllWithContext(root *html.Node, selector string) ([]*Element, error) {
	nodes, err := d.parseSelector(root, selector, false)
	if err != nil {
		return nil, errors.T(err)
	}

	elements := make([]*Element, len(nodes))
	for i, node := range nodes {
		elements[i] = &Element{Node: node, DOM: d}
	}

	return elements, nil
}

// GetText gets the text content of a node
func (d *DOMService) GetText(node *html.Node) string {
	var text string
	var f func(*html.Node)

	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			text += n.Data
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(node)

	return strings.TrimSpace(text)
}

// Attr gets an attribute value from an Element
func (e *Element) Attr(name string) string {
	if e.Node == nil || e.Node.Type != html.ElementNode {
		return ""
	}

	for _, attr := range e.Node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}

	return ""
}

// Text gets the text content of an Element
func (e *Element) Text() string {
	if e.Node == nil {
		return ""
	}

	return e.DOM.GetText(e.Node)
}

// Find finds elements matching a CSS selector within this Element
func (e *Element) Find(selector string) ([]*Element, error) {
	if e.Node == nil {
		return nil, errors.T(fmt.Errorf("element is nil"))
	}

	return e.DOM.QuerySelectorAllWithContext(e.Node, selector)
}

// FindOne finds the first element matching a CSS selector within this Element
func (e *Element) FindOne(selector string) (*Element, error) {
	if e.Node == nil {
		return nil, errors.T(fmt.Errorf("element is nil"))
	}

	return e.DOM.QuerySelectorWithContext(e.Node, selector)
}

// Parent returns the parent element of this element
func (e *Element) Parent() *Element {
	if e.Node == nil || e.Node.Parent == nil {
		return nil
	}

	return &Element{Node: e.Node.Parent, DOM: e.DOM}
}

// NextSibling returns the next sibling element
func (e *Element) NextSibling() *Element {
	if e.Node == nil || e.Node.NextSibling == nil {
		return nil
	}

	return &Element{Node: e.Node.NextSibling, DOM: e.DOM}
}

// PrevSibling returns the previous sibling element
func (e *Element) PrevSibling() *Element {
	if e.Node == nil || e.Node.PrevSibling == nil {
		return nil
	}

	return &Element{Node: e.Node.PrevSibling, DOM: e.DOM}
}

// FindSiblings finds elements that are siblings of this element and match the selector
func (e *Element) FindSiblings(selector string) ([]*Element, error) {
	if e.Node == nil || e.Node.Parent == nil {
		return nil, errors.T(fmt.Errorf("element has no parent"))
	}

	// Get all matching elements from parent
	parentElement := &Element{Node: e.Node.Parent, DOM: e.DOM}
	allElements, err := parentElement.Find(selector)
	if err != nil {
		return nil, errors.T(err)
	}

	// Filter out this element and keep only siblings
	var siblings []*Element
	for _, element := range allElements {
		if element.Node != e.Node {
			siblings = append(siblings, element)
		}
	}

	return siblings, nil
}

// parseSelector is a helper function that implements CSS selector parsing
func (d *DOMService) parseSelector(root *html.Node, selector string, firstOnly bool) ([]*html.Node, error) {
	// Handle complex selectors
	if strings.Contains(selector, " ") {
		return d.parseComplexSelector(root, selector, firstOnly)
	}

	// Handle simple selectors
	return d.parseSimpleSelector(root, selector, firstOnly)
}

// parseComplexSelector handles selectors with descendant combinators (space)
func (d *DOMService) parseComplexSelector(root *html.Node, selector string, firstOnly bool) ([]*html.Node, error) {
	parts := strings.Split(selector, " ")
	if len(parts) == 0 {
		return nil, errors.T(fmt.Errorf("empty selector"))
	}

	// Find all elements matching the first part
	currentMatches, err := d.parseSimpleSelector(root, parts[0], false)
	if err != nil {
		return nil, errors.T(err)
	}

	// For each following part, find matching descendants
	for i := 1; i < len(parts); i++ {
		if len(currentMatches) == 0 {
			return nil, nil // No matches, return empty
		}

		var nextMatches []*html.Node

		for _, match := range currentMatches {
			// Find descendants matching this part
			descendants, err := d.parseSimpleSelector(match, parts[i], false)
			if err != nil {
				continue
			}

			nextMatches = append(nextMatches, descendants...)

			// If we only need the first match and we found one, return it
			if firstOnly && len(nextMatches) > 0 {
				return nextMatches[:1], nil
			}
		}

		currentMatches = nextMatches
	}

	// If we only need the first match, return only the first one
	if firstOnly && len(currentMatches) > 0 {
		return currentMatches[:1], nil
	}

	return currentMatches, nil
}

// parseSimpleSelector handles simple selectors (tag, class, ID, attribute)
func (d *DOMService) parseSimpleSelector(root *html.Node, selector string, firstOnly bool) ([]*html.Node, error) {
	var matches []*html.Node

	// Handle ID selector (#id)
	if strings.Contains(selector, "#") {
		return d.parseIDSelector(root, selector, firstOnly)
	}

	// Handle class selector (.class)
	if strings.Contains(selector, ".") {
		return d.parseClassSelector(root, selector, firstOnly)
	}

	// Handle attribute selector ([attr=value])
	if strings.HasPrefix(selector, "[") && strings.HasSuffix(selector, "]") {
		return d.parseAttributeSelector(root, selector, firstOnly)
	}

	// Simple tag selector
	var matchFunc func(*html.Node) bool

	if selector == "*" {
		// Match any element
		matchFunc = func(n *html.Node) bool {
			return n.Type == html.ElementNode
		}
	} else {
		// Match specific tag
		matchFunc = func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == selector
		}
	}

	// Traverse the DOM
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if matchFunc(n) {
			matches = append(matches, n)
			if firstOnly {
				return
			}
		}

		// Don't traverse further if we only need the first match and found one
		if firstOnly && len(matches) > 0 {
			return
		}

		// Continue traversal
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(root)
	return matches, nil
}

// parseIDSelector handles ID selectors (#id)
func (d *DOMService) parseIDSelector(root *html.Node, selector string, firstOnly bool) ([]*html.Node, error) {
	parts := strings.SplitN(selector, "#", 2)
	tagName := parts[0]
	id := parts[1]

	var matches []*html.Node

	var matchFunc func(*html.Node) bool
	if tagName == "" {
		// Just match by ID
		matchFunc = func(n *html.Node) bool {
			if n.Type != html.ElementNode {
				return false
			}

			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == id {
					return true
				}
			}

			return false
		}
	} else {
		// Match by tag and ID
		matchFunc = func(n *html.Node) bool {
			if n.Type != html.ElementNode || n.Data != tagName {
				return false
			}

			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == id {
					return true
				}
			}

			return false
		}
	}

	// Traverse the DOM
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if matchFunc(n) {
			matches = append(matches, n)
			if firstOnly {
				return
			}
		}

		// Don't traverse further if we only need the first match and found one
		if firstOnly && len(matches) > 0 {
			return
		}

		// Continue traversal
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(root)
	return matches, nil
}

// parseClassSelector handles class selectors (.class)
func (d *DOMService) parseClassSelector(root *html.Node, selector string, firstOnly bool) ([]*html.Node, error) {
	parts := strings.SplitN(selector, ".", 2)
	tagName := parts[0]
	className := parts[1]

	var matches []*html.Node

	var matchFunc func(*html.Node) bool
	if tagName == "" {
		// Just match by class
		matchFunc = func(n *html.Node) bool {
			if n.Type != html.ElementNode {
				return false
			}

			for _, attr := range n.Attr {
				if attr.Key == "class" {
					classes := strings.Fields(attr.Val)
					for _, class := range classes {
						if class == className {
							return true
						}
					}
				}
			}

			return false
		}
	} else {
		// Match by tag and class
		matchFunc = func(n *html.Node) bool {
			if n.Type != html.ElementNode || n.Data != tagName {
				return false
			}

			for _, attr := range n.Attr {
				if attr.Key == "class" {
					classes := strings.Fields(attr.Val)
					for _, class := range classes {
						if class == className {
							return true
						}
					}
				}
			}

			return false
		}
	}

	// Traverse the DOM
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if matchFunc(n) {
			matches = append(matches, n)
			if firstOnly {
				return
			}
		}

		// Don't traverse further if we only need the first match and found one
		if firstOnly && len(matches) > 0 {
			return
		}

		// Continue traversal
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(root)
	return matches, nil
}

// parseAttributeSelector handles attribute selectors ([attr=value])
func (d *DOMService) parseAttributeSelector(root *html.Node, selector string, firstOnly bool) ([]*html.Node, error) {
	// Strip the brackets
	attrSelector := selector[1 : len(selector)-1]

	// Check for attribute existence
	if !strings.Contains(attrSelector, "=") {
		return d.parseAttributeExistsSelector(root, attrSelector, firstOnly)
	}

	// Parse operator and value
	var attrName, op, attrValue string

	if strings.Contains(attrSelector, "*=") {
		parts := strings.SplitN(attrSelector, "*=", 2)
		attrName = parts[0]
		op = "*="
		attrValue = strings.Trim(parts[1], "\"'")
	} else if strings.Contains(attrSelector, "^=") {
		parts := strings.SplitN(attrSelector, "^=", 2)
		attrName = parts[0]
		op = "^="
		attrValue = strings.Trim(parts[1], "\"'")
	} else if strings.Contains(attrSelector, "$=") {
		parts := strings.SplitN(attrSelector, "$=", 2)
		attrName = parts[0]
		op = "$="
		attrValue = strings.Trim(parts[1], "\"'")
	} else if strings.Contains(attrSelector, "~=") {
		parts := strings.SplitN(attrSelector, "~=", 2)
		attrName = parts[0]
		op = "~="
		attrValue = strings.Trim(parts[1], "\"'")
	} else if strings.Contains(attrSelector, "|=") {
		parts := strings.SplitN(attrSelector, "|=", 2)
		attrName = parts[0]
		op = "|="
		attrValue = strings.Trim(parts[1], "\"'")
	} else {
		parts := strings.SplitN(attrSelector, "=", 2)
		attrName = parts[0]
		op = "="
		attrValue = strings.Trim(parts[1], "\"'")
	}

	// Trim whitespace
	attrName = strings.TrimSpace(attrName)

	var matches []*html.Node

	// Create match function based on operator
	var matchFunc func(*html.Node) bool
	matchFunc = func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}

		for _, attr := range n.Attr {
			if attr.Key == attrName {
				switch op {
				case "=":
					return attr.Val == attrValue
				case "*=":
					return strings.Contains(attr.Val, attrValue)
				case "^=":
					return strings.HasPrefix(attr.Val, attrValue)
				case "$=":
					return strings.HasSuffix(attr.Val, attrValue)
				case "~=":
					words := strings.Fields(attr.Val)
					for _, word := range words {
						if word == attrValue {
							return true
						}
					}
					return false
				case "|=":
					return attr.Val == attrValue || strings.HasPrefix(attr.Val, attrValue+"-")
				}
			}
		}

		return false
	}

	// Traverse the DOM
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if matchFunc(n) {
			matches = append(matches, n)
			if firstOnly {
				return
			}
		}

		// Don't traverse further if we only need the first match and found one
		if firstOnly && len(matches) > 0 {
			return
		}

		// Continue traversal
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(root)
	return matches, nil
}

// parseAttributeExistsSelector handles attribute existence selectors ([attr])
func (d *DOMService) parseAttributeExistsSelector(root *html.Node, attrName string, firstOnly bool) ([]*html.Node, error) {
	var matches []*html.Node

	var matchFunc func(*html.Node) bool
	matchFunc = func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}

		for _, attr := range n.Attr {
			if attr.Key == attrName {
				return true
			}
		}

		return false
	}

	// Traverse the DOM
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if matchFunc(n) {
			matches = append(matches, n)
			if firstOnly {
				return
			}
		}

		// Don't traverse further if we only need the first match and found one
		if firstOnly && len(matches) > 0 {
			return
		}

		// Continue traversal
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(root)
	return matches, nil
}

// GetElementsByTagName gets all elements with the specified tag name
func (d *DOMService) GetElementsByTagName(root *html.Node, tagName string) ([]*Element, error) {
	nodes, err := d.parseSimpleSelector(root, tagName, false)
	if err != nil {
		return nil, errors.T(err)
	}

	elements := make([]*Element, len(nodes))
	for i, node := range nodes {
		elements[i] = &Element{Node: node, DOM: d}
	}

	return elements, nil
}

// GetElementById gets an element by its ID
func (d *DOMService) GetElementById(root *html.Node, id string) (*Element, error) {
	nodes, err := d.parseIDSelector(root, "#"+id, true)
	if err != nil {
		return nil, errors.T(err)
	}

	if len(nodes) == 0 {
		return nil, errors.T(fmt.Errorf("element with ID '%s' not found", id))
	}

	return &Element{Node: nodes[0], DOM: d}, nil
}

// GetElementsByClassName gets all elements with the specified class name
func (d *DOMService) GetElementsByClassName(root *html.Node, className string) ([]*Element, error) {
	nodes, err := d.parseClassSelector(root, "."+className, false)
	if err != nil {
		return nil, errors.T(err)
	}

	elements := make([]*Element, len(nodes))
	for i, node := range nodes {
		elements[i] = &Element{Node: node, DOM: d}
	}

	return elements, nil
}

// ExtractTextContent extracts clean text content from an HTML string
func (d *DOMService) ExtractTextContent(htmlContent string) (string, error) {
	doc, err := d.Parse(htmlContent)
	if err != nil {
		return "", errors.T(err)
	}

	return d.GetText(doc), nil
}

// ExtractMetaTags extracts all meta tags from an HTML document
func (d *DOMService) ExtractMetaTags(doc *html.Node) map[string]string {
	result := make(map[string]string)

	metaTags, err := d.parseSimpleSelector(doc, "meta", false)
	if err != nil {
		return result
	}

	for _, node := range metaTags {
		var name, content string

		for _, attr := range node.Attr {
			switch attr.Key {
			case "name", "property", "itemprop":
				name = attr.Val
			case "content":
				content = attr.Val
			}
		}

		if name != "" && content != "" {
			result[name] = content
		}
	}

	return result
}

// CascadeParseChapterNumber tries multiple methods to extract a chapter number
// from title and ID, returning the first successful result or 0 if all fail
func CascadeParseChapterNumber(title, chapterID string) float64 {
	// Try extracting from title using standard method first
	chapterNumber := ExtractChapterNumber(title)
	if chapterNumber > 0 {
		return chapterNumber
	}

	// If that fails, try specialized URL extraction
	chapterNumber = extractChapterNumberFromURL(chapterID)
	if chapterNumber > 0 {
		return chapterNumber
	}

	// If that fails too, try direct title pattern matching
	chapterNumber = extractChapterNumberFromTitle(title)
	if chapterNumber > 0 {
		return chapterNumber
	}

	// All methods failed, return 0
	return 0
}

// ExtractChapterNumber extracts the chapter number from a string
func ExtractChapterNumber(text string) float64 {
	// Special case for KissManga-style "chapter XX.Y" format
	kissMangaPattern := regexp.MustCompile(`(?i)chapter\s+(\d+)\.(\d+)`)
	matches := kissMangaPattern.FindStringSubmatch(text)
	if len(matches) > 2 {
		whole, err1 := strconv.ParseFloat(matches[1], 64)
		decimal, err2 := strconv.ParseFloat("."+matches[2], 64)

		if err1 == nil && err2 == nil {
			return whole + decimal
		}
	}

	// Common patterns for chapter numbers
	patterns := []string{
		`(?i)chapter\s*(\d+(\.\d+)?)`,
		`(?i)ch(\.|apter\s*)?(\d+(\.\d+)?)`,
		`(?i)episode\s*(\d+(\.\d+)?)`,
		`(?i)ep(\.|isode\s*)?(\d+(\.\d+)?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)

		if len(matches) > 0 {
			// Always prefer the first capture group, which should contain the full number
			if matches[1] != "" {
				number, err := strconv.ParseFloat(matches[1], 64)
				if err == nil {
					return number
				}
			}
		}
	}

	return 0
}

// extractChapterNumberFromURL extracts chapter numbers from URLs
// which commonly follow patterns like "ch-123" or "chapter-123" in the URL
func extractChapterNumberFromURL(chapterID string) float64 {
	// KissManga specific pattern: chapter-XX-Y for chapter XX.Y
	re := regexp.MustCompile(`chapter-(\d+)-(\d+)`)
	matches := re.FindStringSubmatch(chapterID)
	if len(matches) > 2 {
		whole, err1 := strconv.ParseFloat(matches[1], 64)
		decimal, err2 := strconv.ParseFloat("."+matches[2], 64)

		if err1 == nil && err2 == nil {
			return whole + decimal
		}
	}

	// ch-XX-Y pattern
	re = regexp.MustCompile(`ch-(\d+)-(\d+)`)
	matches = re.FindStringSubmatch(chapterID)
	if len(matches) > 2 {
		whole, err1 := strconv.ParseFloat(matches[1], 64)
		decimal, err2 := strconv.ParseFloat("."+matches[2], 64)

		if err1 == nil && err2 == nil {
			return whole + decimal
		}
	}

	// If the specific patterns didn't match, try other common patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`ch-(\d+(?:\.\d+)?)`),      // matches ch-123 or ch-123.5
		regexp.MustCompile(`chapter-(\d+(?:\.\d+)?)`), // matches chapter-123 or chapter-123.5
		regexp.MustCompile(`chap-(\d+(?:\.\d+)?)`),    // matches chap-123 or similar
		regexp.MustCompile(`c-(\d+(?:\.\d+)?)`),       // matches c-123 or similar
	}

	// Try each pattern
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(chapterID)
		if len(matches) > 1 {
			// Parse as a regular float
			if num, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return num
			}
		}
	}

	// No matches found with our special patterns, return 0
	return 0
}

// extractChapterNumberFromTitle attempts to extract chapter number from title patterns
// like "Ch. 3 - Title" or "Chapter 3.5: Title"
func extractChapterNumberFromTitle(title string) float64 {
	// First check specifically for "chapter X.Y" patterns (used by KissManga)
	decimalPattern := regexp.MustCompile(`(?i)chapter\s+(\d+)\.(\d+)`)
	matches := decimalPattern.FindStringSubmatch(title)
	if len(matches) > 2 {
		whole, err1 := strconv.ParseFloat(matches[1], 64)
		decimal, err2 := strconv.ParseFloat("."+matches[2], 64)

		if err1 == nil && err2 == nil {
			return whole + decimal
		}
	}

	// Also check for hyphenated decimal format (chapter 16-5)
	hyphenPattern := regexp.MustCompile(`(?i)chapter\s+(\d+)-(\d+)`)
	matches = hyphenPattern.FindStringSubmatch(title)
	if len(matches) > 2 {
		whole, err1 := strconv.ParseFloat(matches[1], 64)
		decimal, err2 := strconv.ParseFloat("."+matches[2], 64)

		if err1 == nil && err2 == nil {
			return whole + decimal
		}
	}

	// Common patterns for chapter numbers in titles
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^chapter\s+(\d+(?:\.\d+)?)`),            // "chapter 123" or "chapter 123.5"
		regexp.MustCompile(`(?i)^ch\.\s+(\d+(?:\.\d+)?)`),               // "Ch. 123" or "Ch. 123.5"
		regexp.MustCompile(`(?i)^vol\.\s+\d+\s+ch\.\s+(\d+(?:\.\d+)?)`), // "Vol. 1 Ch. 123"
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(title)
		if len(matches) > 1 && matches[1] != "" {
			if num, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return num
			}
		}
	}

	return 0
}

// UrlJoin joins URL parts
func UrlJoin(base string, parts ...string) string {
	if len(parts) == 0 {
		return base
	}

	// Ensure the base ends with a single slash if it doesn't have one
	if !strings.HasSuffix(base, "/") {
		base = base + "/"
	}

	var result strings.Builder
	result.WriteString(base)

	for i, part := range parts {
		// Trim slashes from the beginning and end of this part
		part = strings.Trim(part, "/")

		if part == "" {
			continue
		}

		result.WriteString(part)

		// Add a slash between parts except after the last part
		if i < len(parts)-1 {
			result.WriteString("/")
		}
	}

	return result.String()
}
