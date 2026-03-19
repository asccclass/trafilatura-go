// Package utils provides shared utility functions for trafilatura-go.
package utils

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// Noise elements that should be removed from the DOM before extraction.
var NoiseTagNames = map[string]bool{
	"aside": true, "embed": true, "footer": true,
	"head": true, "iframe": true, "menu": true, "nav": true,
	"script": true, "style": true, "noscript": true,
}

// Block-level tags that produce paragraph breaks.
var BlockTags = map[string]bool{
	"article": true, "blockquote": true, "dd": true, "div": true,
	"dl": true, "dt": true, "fieldset": true, "figcaption": true,
	"figure": true, "footer": true, "form": true, "h1": true,
	"h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"header": true, "li": true, "main": true, "nav": true,
	"ol": true, "p": true, "pre": true, "section": true,
	"summary": true, "table": true, "tr": true, "ul": true,
}

// HeadingTags are tags that represent titles/headings.
var HeadingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

var (
	reMultiSpace = regexp.MustCompile(`[ \t]+`)
	reMultiLines = regexp.MustCompile(`\n{3,}`)
	reLeadingWS  = regexp.MustCompile(`(?m)^[ \t]+`)
	reTrailingWS = regexp.MustCompile(`(?m)[ \t]+$`)
	reURLPattern = regexp.MustCompile(`https?://[^\s]+`)
)

// GetAttr returns the value of an attribute from an HTML node.
func GetAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// HasAttr checks whether an HTML node has a specific attribute.
func HasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

// GetAttrContains checks if an attribute's value contains the given string.
func GetAttrContains(n *html.Node, key, value string) bool {
	val := GetAttr(n, key)
	return strings.Contains(strings.ToLower(val), strings.ToLower(value))
}

// TextContent returns all text inside a node (recursively).
func TextContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		if cur.Type == html.TextNode {
			b.WriteString(cur.Data)
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// CleanText normalises whitespace in a text string.
func CleanText(s string) string {
	s = reMultiSpace.ReplaceAllString(s, " ")
	s = reLeadingWS.ReplaceAllString(s, "")
	s = reTrailingWS.ReplaceAllString(s, "")
	s = reMultiLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// approximateWordCount returns the approximate word count of a string,
// treating CJK characters as individual words.
func approximateWordCount(s string) int {
	words := strings.Fields(s)
	count := 0
	for _, w := range words {
		// A rough heuristic: count non-ASCII runes as words to handle CJK
		cjkRunes := 0
		for _, r := range w {
			if r > '\u007F' {
				cjkRunes++
			}
		}
		if cjkRunes > 0 {
			count += cjkRunes
		} else {
			count += 1
		}
	}
	return count
}

// WordCount returns the approximate word count of a string.
func WordCount(s string) int {
	return approximateWordCount(s)
}

// CharCount returns the rune count of a string.
func CharCount(s string) int {
	return len([]rune(s))
}

// IsNoiseElement returns true when an element is known boilerplate noise.
func IsNoiseElement(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	tag := strings.ToLower(n.Data)
	if NoiseTagNames[tag] {
		return true
	}

	// Specifically handle <form> tags: protect them if they are content-rich (e.g. wrapper for a generic table like SunnyBank do)
	if tag == "form" {
		text := TextContent(n)
		// If the form has a lot of text, or contains a table, preserve it.
		// Otherwise, it's likely just a search bar or login form.
		if len(text) > 200 {
			return false
		}
		if FindFirst(n, func(child *html.Node) bool { return IsElement(child, "table") }) != nil {
			return false
		}
		return true // Small forms without tables are noise
	}

	// class/id based heuristics
	for _, kw := range []string{"sidebar", "footer", "header", "nav", "menu",
		"advertisement", "banner", "cookie", "popup", "modal", "overlay",
		"social", "share", "comment-form", "related", "widget"} {
		if GetAttrContains(n, "class", kw) || GetAttrContains(n, "id", kw) {
			// Special case: PrimeFaces uses 'ui-widget' for all content containers, don't drop them
			if kw == "widget" {
				classVal := strings.ToLower(GetAttr(n, "class"))
				idVal := strings.ToLower(GetAttr(n, "id"))
				classNoUI := strings.ReplaceAll(classVal, "ui-widget", "")
				idNoUI := strings.ReplaceAll(idVal, "ui-widget", "")
				if !strings.Contains(classNoUI, "widget") && !strings.Contains(idNoUI, "widget") {
					continue
				}
			}
			return true
		}
	}
	return false
}

// ContentSignal returns a score for how content-like a text block is.
// Higher is better.
func ContentSignal(text string) float64 {
	if text == "" {
		return 0
	}
	wordCount := float64(approximateWordCount(text))
	if wordCount == 0 {
		return 0
	}

	// Link density heuristic: lower ratio of links = better
	linkLen := float64(len(reURLPattern.FindAllString(text, -1)))
	linkRatio := linkLen / wordCount

	// Sentence structure heuristic: presence of punctuation
	punctuation := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == '?' || r == ';' {
			punctuation++
		}
	}
	sentenceScore := float64(punctuation) / wordCount * 10

	// Uppercase ratio (low is better — not all-caps spam)
	upperCount := 0
	for _, r := range text {
		if unicode.IsUpper(r) {
			upperCount++
		}
	}
	upperRatio := float64(upperCount) / float64(len([]rune(text))+1)

	score := wordCount * (1 - linkRatio) * (1 + sentenceScore) * (1 - upperRatio*0.5)
	return score
}

// FindNodes walks the HTML tree and returns all nodes matching the predicate.
func FindNodes(root *html.Node, match func(*html.Node) bool) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if match(n) {
			results = append(results, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return results
}

// FindFirst finds the first node matching the predicate.
func FindFirst(root *html.Node, match func(*html.Node) bool) *html.Node {
	var result *html.Node
	var walk func(*html.Node) bool
	walk = func(n *html.Node) bool {
		if match(n) {
			result = n
			return true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(root)
	return result
}

// IsElement checks tag name (case-insensitive).
func IsElement(n *html.Node, tag string) bool {
	return n.Type == html.ElementNode && strings.EqualFold(n.Data, tag)
}

// RemoveNode detaches a node from its parent.
func RemoveNode(n *html.Node) {
	if n.Parent != nil {
		n.Parent.RemoveChild(n)
	}
}

// InnerHTML serialises a node's children to an HTML string.
func InnerHTML(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		html.Render(&b, c) //nolint:errcheck
	}
	return b.String()
}

// OuterHTML serialises a node including itself.
func OuterHTML(n *html.Node) string {
	var b strings.Builder
	html.Render(&b, n) //nolint:errcheck
	return b.String()
}
