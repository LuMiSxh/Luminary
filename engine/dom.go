package engine

import (
	"fmt"
	"golang.org/x/net/html"
	"strings"
)

// DOMService provides DOM parsing and querying capabilities
type DOMService struct {
	// Add configuration options here if needed
}

// Parse parses HTML content into a DOM
func (d *DOMService) Parse(content string) (*html.Node, error) {
	return html.Parse(strings.NewReader(content))
}

// QuerySelector finds the first element matching the selector
func (d *DOMService) QuerySelector(node *html.Node, selector string) (*html.Node, error) {
	// Implementation would depend on a CSS selector library
	// For now, a simplified version:
	var result *html.Node
	var f func(*html.Node) bool

	f = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == selector {
			result = n
			return true
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if f(c) {
				return true
			}
		}

		return false
	}

	f(node)

	if result == nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	return result, nil
}

// QuerySelectorAll finds all elements matching the selector
func (d *DOMService) QuerySelectorAll(node *html.Node, selector string) ([]*html.Node, error) {
	// Implementation would depend on a CSS selector library
	// For now, a simplified version:
	var results []*html.Node
	var f func(*html.Node)

	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == selector {
			results = append(results, n)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(node)

	return results, nil
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
