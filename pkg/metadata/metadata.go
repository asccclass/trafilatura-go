// Package metadata extracts title, author, date, description and other
// metadata from HTML documents using Open Graph, Twitter Card, JSON-LD,
// Dublin Core and standard HTML meta tags.
package metadata

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"

	"example.com/trafilatura-go/pkg/langdetect"
	"example.com/trafilatura-go/pkg/utils"
)

// Metadata holds all extracted metadata for a page.
type Metadata struct {
	Title       string    `json:"title,omitempty"`
	Author      string    `json:"author,omitempty"`
	URL         string    `json:"url,omitempty"`
	Hostname    string    `json:"hostname,omitempty"`
	Description string    `json:"description,omitempty"`
	SiteName    string    `json:"sitename,omitempty"`
	Date        time.Time `json:"date,omitempty"`
	DateStr     string    `json:"date_str,omitempty"`
	Categories  []string  `json:"categories,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Language    string    `json:"language,omitempty"`
	Image       string    `json:"image,omitempty"`
	PageType    string    `json:"pagetype,omitempty"`
	License     string    `json:"license,omitempty"`
}

// knownDateFormats lists common date format strings used on the web.
var knownDateFormats = []string{
	time.RFC3339, time.RFC1123, time.RFC1123Z,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"January 2, 2006",
	"Jan 2, 2006",
	"02 January 2006",
	"02 Jan 2006",
	"Monday, January 2, 2006",
}

// parseDate attempts to parse a date string using known formats.
func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, fmt := range knownDateFormats {
		t, err := time.Parse(fmt, s)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// Extract pulls metadata from a parsed HTML document.
func Extract(doc *html.Node, pageURL string) *Metadata {
	m := &Metadata{}

	// Parse hostname
	if u, err := url.Parse(pageURL); err == nil {
		m.URL = pageURL
		m.Hostname = u.Hostname()
	}

	head := utils.FindFirst(doc, func(n *html.Node) bool {
		return utils.IsElement(n, "head")
	})

	if head != nil {
		extractFromHead(head, m)
	}

	// JSON-LD extraction
	scripts := utils.FindNodes(doc, func(n *html.Node) bool {
		return utils.IsElement(n, "script") &&
			strings.EqualFold(utils.GetAttr(n, "type"), "application/ld+json")
	})
	for _, s := range scripts {
		extractJSONLD(utils.TextContent(s), m)
	}

	// Fallback title from <title> tag
	if m.Title == "" {
		titleNode := utils.FindFirst(doc, func(n *html.Node) bool {
			return utils.IsElement(n, "title")
		})
		if titleNode != nil {
			m.Title = cleanTitle(utils.TextContent(titleNode))
		}
	}

	// Fallback: look for author in byline patterns
	if m.Author == "" {
		extractBylineAuthor(doc, m)
	}

	// Language from html lang attribute
	if m.Language == "" {
		htmlNode := utils.FindFirst(doc, func(n *html.Node) bool {
			return utils.IsElement(n, "html")
		})
		if htmlNode != nil {
			m.Language = utils.GetAttr(htmlNode, "lang")
		}
	}

	// Language from meta http-equiv
	if m.Language == "" {
		metaLang := utils.FindFirst(doc, func(n *html.Node) bool {
			return utils.IsElement(n, "meta") && strings.EqualFold(utils.GetAttr(n, "http-equiv"), "content-language")
		})
		if metaLang != nil {
			m.Language = utils.GetAttr(metaLang, "content")
		}
	}

	// Text-based language detection fallback: when all HTML/meta sources
	// failed to provide a language, detect from the body text content.
	if m.Language == "" {
		bodyNode := utils.FindFirst(doc, func(n *html.Node) bool {
			return utils.IsElement(n, "body")
		})
		if bodyNode != nil {
			if detected := langdetect.Detect(utils.TextContent(bodyNode)); detected != "" {
				m.Language = detected
			}
		}
	}

	return m
}

// extractFromHead pulls Open Graph, Twitter Card, and standard meta tags.
func extractFromHead(head *html.Node, m *Metadata) {
	for c := head.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		tag := strings.ToLower(c.Data)

		switch tag {
		case "meta":
			prop := strings.ToLower(utils.GetAttr(c, "property"))
			name := strings.ToLower(utils.GetAttr(c, "name"))
			content := utils.GetAttr(c, "content")

			switch prop {
			case "og:title":
				if m.Title == "" {
					m.Title = cleanTitle(content)
				}
			case "og:description":
				if m.Description == "" {
					m.Description = content
				}
			case "og:site_name":
				if m.SiteName == "" {
					m.SiteName = content
				}
			case "og:image":
				if m.Image == "" {
					m.Image = content
				}
			case "og:type":
				if m.PageType == "" {
					m.PageType = content
				}
			case "og:locale":
				if m.Language == "" {
					// og:locale format is usually language_TERRITORY, e.g., en_US. Take the first part.
					parts := strings.SplitN(content, "_", 2)
					m.Language = parts[0]
				}
			case "article:author":
				if m.Author == "" {
					m.Author = content
				}
			case "article:published_time", "article:modified_time":
				if m.DateStr == "" {
					m.DateStr = content
					if t, ok := parseDate(content); ok {
						m.Date = t
					}
				}
			case "article:tag":
				m.Tags = append(m.Tags, content)
			case "article:section":
				m.Categories = append(m.Categories, content)
			}

			switch name {
			case "author":
				if m.Author == "" {
					m.Author = content
				}
			case "description":
				if m.Description == "" {
					m.Description = content
				}
			case "twitter:title":
				if m.Title == "" {
					m.Title = cleanTitle(content)
				}
			case "twitter:description":
				if m.Description == "" {
					m.Description = content
				}
			case "twitter:image":
				if m.Image == "" {
					m.Image = content
				}
			case "dcterms.creator", "dc.creator":
				if m.Author == "" {
					m.Author = content
				}
			case "dcterms.date", "dc.date":
				if m.DateStr == "" {
					m.DateStr = content
					if t, ok := parseDate(content); ok {
						m.Date = t
					}
				}
			case "dcterms.language", "dc.language":
				if m.Language == "" {
					m.Language = content
				}
			case "keywords":
				for _, kw := range strings.Split(content, ",") {
					kw = strings.TrimSpace(kw)
					if kw != "" {
						m.Tags = append(m.Tags, kw)
					}
				}
			case "license":
				m.License = content
			}

		case "link":
			rel := strings.ToLower(utils.GetAttr(c, "rel"))
			if rel == "canonical" {
				if href := utils.GetAttr(c, "href"); href != "" {
					m.URL = href
				}
			}
			if rel == "license" {
				if href := utils.GetAttr(c, "href"); href != "" {
					m.License = href
				}
			}
		}
	}
}

// jsonLDSchema represents a partial JSON-LD schema object.
type jsonLDSchema struct {
	Type        interface{} `json:"@type"`
	Name        string      `json:"name"`
	Headline    string      `json:"headline"`
	Author      interface{} `json:"author"` // can be string or object
	Publisher   interface{} `json:"publisher"`
	DatePub     interface{} `json:"datePublished"` // can be string or other types
	DateMod     interface{} `json:"dateModified"` // can be string or other types
	Description string      `json:"description"`
	Image       interface{} `json:"image"`
	InLanguage  interface{} `json:"inLanguage"` // can be string or object
	Keywords    interface{} `json:"keywords"`
}

// jsonLDString safely extracts a string from an interface{} value.
// Returns empty string for non-string types (float64, map, slice, nil, etc.).
func jsonLDString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// extractJSONLD parses JSON-LD script content for metadata.
func extractJSONLD(raw string, m *Metadata) {
	// Safety net: recover from any unexpected panic in malformed JSON-LD
	defer func() { recover() }() //nolint:errcheck

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}

	// Handle both single objects and arrays
	var schemas []json.RawMessage
	if raw[0] == '[' {
		json.Unmarshal([]byte(raw), &schemas) //nolint:errcheck
	} else {
		// Try to detect @graph wrapper
		var wrapper map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &wrapper); err == nil {
			if graph, ok := wrapper["@graph"]; ok {
				var graphSchemas []json.RawMessage
				if json.Unmarshal(graph, &graphSchemas) == nil {
					schemas = graphSchemas
				}
			}
		}
		// If no @graph found (or parse failed), treat as single schema
		if len(schemas) == 0 {
			schemas = []json.RawMessage{json.RawMessage(raw)}
		}
	}

	for _, s := range schemas {
		var schema jsonLDSchema
		if err := json.Unmarshal(s, &schema); err != nil {
			continue
		}

		// Title / Headline
		if m.Title == "" {
			if schema.Headline != "" {
				m.Title = cleanTitle(schema.Headline)
			} else if schema.Name != "" {
				m.Title = cleanTitle(schema.Name)
			}
		}

		// Author
		if m.Author == "" {
			switch v := schema.Author.(type) {
			case string:
				m.Author = v
			case map[string]interface{}:
				if name, ok := v["name"].(string); ok {
					m.Author = name
				}
			case []interface{}:
				if len(v) > 0 {
					switch first := v[0].(type) {
					case string:
						m.Author = first
					case map[string]interface{}:
						if name, ok := first["name"].(string); ok {
							m.Author = name
						}
					}
				}
			}
		}

		// Publisher / SiteName
		if m.SiteName == "" {
			switch v := schema.Publisher.(type) {
			case string:
				m.SiteName = v
			case map[string]interface{}:
				if name, ok := v["name"].(string); ok {
					m.SiteName = name
				}
			case []interface{}:
				if len(v) > 0 {
					switch first := v[0].(type) {
					case string:
						m.SiteName = first
					case map[string]interface{}:
						if name, ok := first["name"].(string); ok {
							m.SiteName = name
						}
					}
				}
			}
		}

		// Date (safely handle non-string values)
		if m.DateStr == "" {
			dateStr := jsonLDString(schema.DatePub)
			if dateStr == "" {
				dateStr = jsonLDString(schema.DateMod)
			}
			if dateStr != "" {
				m.DateStr = dateStr
				if t, ok := parseDate(dateStr); ok {
					m.Date = t
				}
			}
		}

		// Description
		if m.Description == "" && schema.Description != "" {
			m.Description = schema.Description
		}

		// Language (safely handle non-string values)
		if m.Language == "" {
			if lang := jsonLDString(schema.InLanguage); lang != "" {
				m.Language = lang
			}
		}

		// Image
		if m.Image == "" && schema.Image != nil {
			switch v := schema.Image.(type) {
			case string:
				m.Image = v
			case map[string]interface{}:
				if url, ok := v["url"].(string); ok {
					m.Image = url
				}
			case []interface{}:
				if len(v) > 0 {
					switch first := v[0].(type) {
					case string:
						m.Image = first
					case map[string]interface{}:
						if url, ok := first["url"].(string); ok {
							m.Image = url
						}
					}
				}
			}
		}

		// Keywords
		switch v := schema.Keywords.(type) {
		case string:
			for _, kw := range strings.Split(v, ",") {
				kw = strings.TrimSpace(kw)
				if kw != "" {
					m.Tags = append(m.Tags, kw)
				}
			}
		case []interface{}:
			for _, item := range v {
				if kw, ok := item.(string); ok {
					kw = strings.TrimSpace(kw)
					if kw != "" {
						m.Tags = append(m.Tags, kw)
					}
				}
			}
		}
	}
}

// extractBylineAuthor scans visible content for common byline patterns.
func extractBylineAuthor(doc *html.Node, m *Metadata) {
	bylineSelectors := []struct{ attr, value string }{
		{"class", "author"},
		{"class", "byline"},
		{"class", "article-author"},
		{"itemprop", "author"},
		{"rel", "author"},
	}

	for _, sel := range bylineSelectors {
		node := utils.FindFirst(doc, func(n *html.Node) bool {
			return n.Type == html.ElementNode &&
				utils.GetAttrContains(n, sel.attr, sel.value)
		})
		if node != nil {
			text := strings.TrimSpace(utils.TextContent(node))
			if text != "" && utf8.RuneCountInString(text) < 100 {
				// Strip common prefixes like "By " or "Written by "
				for _, prefix := range []string{"by ", "written by ", "from "} {
					if strings.HasPrefix(strings.ToLower(text), prefix) {
						text = text[len(prefix):]
					}
				}
				m.Author = strings.TrimSpace(text)
				return
			}
		}
	}
}

// cleanTitle removes site name suffixes from titles (e.g. "Article | Site Name").
func cleanTitle(s string) string {
	s = strings.TrimSpace(s)
	for _, sep := range []string{" | ", " - ", " – ", " — ", " :: "} {
		if idx := strings.LastIndex(s, sep); idx > 0 {
			// Keep the part before the separator if it's longer
			left := strings.TrimSpace(s[:idx])
			if len(left) > 20 {
				return left
			}
		}
	}
	return s
}
