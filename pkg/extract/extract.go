// Package extract implements web content extraction algorithms.
// It combines a readability-style scoring approach with fallback heuristics
// (similar to jusText) to robustly identify the main article text.
package extract

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"golang.org/x/net/html"

	"example.com/trafilatura-go/pkg/settings"
	"example.com/trafilatura-go/pkg/utils"
)

// --------- Document representation ---------

// Block represents a text block extracted from the DOM.
type Block struct {
	Text     string
	Tag      string
	Level    int // heading level (1-6) or 0
	IsCode   bool
	IsQuote  bool
	IsTable  bool
	IsList   bool
	IsImage  bool
	Score    float64
	LinkText float64 // proportion of text that is link text
}

// Document holds all extracted blocks.
type Document struct {
	Blocks   []*Block
	RawHTML  string
	Comments []*Block
}

// --------- Extractor ---------

// Extractor performs the main content extraction.
type Extractor struct {
	cfg *settings.Config
}

// New creates an Extractor with the given configuration.
func New(cfg *settings.Config) *Extractor {
	return &Extractor{cfg: cfg}
}

// FromBytes parses raw HTML bytes and extracts the main content.
func (e *Extractor) FromBytes(data []byte) (*Document, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	return e.FromNode(doc), nil
}

// FromNode extracts content from an already-parsed HTML node.
func (e *Extractor) FromNode(doc *html.Node) *Document {
	// Step 1: pre-clean the tree
	cleaned := e.preclean(doc)

	// Step 2: try to find a main content container
	main := e.findMainContent(cleaned)
	if main == nil {
		main = cleaned
	}

	// Step 3: convert to text blocks
	blocks := e.toBlocks(main, 0)

	// Step 4: score and filter blocks
	filtered := e.scoreAndFilter(blocks)

	// Step 5: optionally extract comments
	var comments []*Block
	if e.cfg.IncludeComments {
		comments = e.extractComments(doc)
	}

	return &Document{
		Blocks:   filtered,
		Comments: comments,
	}
}

// preclean removes clearly boilerplate elements from the tree clone.
func (e *Extractor) preclean(doc *html.Node) *html.Node {
	// Remove noise nodes
	noiseNodes := utils.FindNodes(doc, func(n *html.Node) bool {
		return utils.IsNoiseElement(n)
	})
	for _, n := range noiseNodes {
		utils.RemoveNode(n)
	}
	return doc
}

// findMainContent uses multiple strategies to identify the main content container.
func (e *Extractor) findMainContent(doc *html.Node) *html.Node {
	// Strategy 1: semantic tags
	for _, tag := range []string{"main", "article"} {
		node := utils.FindFirst(doc, func(n *html.Node) bool {
			return utils.IsElement(n, tag)
		})
		if node != nil && len(utils.TextContent(node)) > e.cfg.MinExtractedSize {
			return node
		}
	}

	// Strategy 2: role="main"
	node := utils.FindFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode &&
			strings.EqualFold(utils.GetAttr(n, "role"), "main")
	})
	if node != nil && len(utils.TextContent(node)) > e.cfg.MinExtractedSize {
		return node
	}

	// Strategy 3: id/class heuristics
	for _, pattern := range []string{"content", "article", "post", "entry", "story", "text"} {
		node = utils.FindFirst(doc, func(n *html.Node) bool {
			if n.Type != html.ElementNode {
				return false
			}
			id := strings.ToLower(utils.GetAttr(n, "id"))
			cls := strings.ToLower(utils.GetAttr(n, "class"))
			return strings.Contains(id, pattern) || strings.Contains(cls, pattern)
		})
		if node != nil && len(utils.TextContent(node)) > e.cfg.MinExtractedSize {
			return node
		}
	}

	// Strategy 4: density-based selection — find the div/section with the
	// highest text-to-markup ratio
	return e.densityBestNode(doc)
}

// densityBestNode finds the block element with the highest text density.
func (e *Extractor) densityBestNode(doc *html.Node) *html.Node {
	type candidate struct {
		node  *html.Node
		score float64
	}

	var best candidate

	candidates := utils.FindNodes(doc, func(n *html.Node) bool {
		tag := strings.ToLower(n.Data)
		return n.Type == html.ElementNode &&
			(tag == "div" || tag == "section" || tag == "td" || tag == "table" || tag == "form")
	})

	for _, n := range candidates {
		text := utils.TextContent(n)
		score := utils.ContentSignal(text)
		if score > best.score {
			best = candidate{node: n, score: score}
		}
	}

	return best.node
}

// toBlocks recursively converts DOM nodes to text blocks.
func (e *Extractor) toBlocks(n *html.Node, depth int) []*Block {
	var blocks []*Block

	if n == nil {
		return blocks
	}

	switch n.Type {
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text != "" && depth > 0 {
			blocks = append(blocks, &Block{Text: text})
		}

	case html.ElementNode:
		tag := strings.ToLower(n.Data)

		// Code blocks
		if tag == "pre" || tag == "code" {
			text := utils.TextContent(n)
			if strings.TrimSpace(text) != "" {
				blocks = append(blocks, &Block{
					Text:   text,
					Tag:    tag,
					IsCode: true,
				})
			}
			return blocks
		}

		// Blockquotes
		if tag == "blockquote" || tag == "q" {
			text := utils.CleanText(utils.TextContent(n))
			if text != "" {
				blocks = append(blocks, &Block{
					Text:    text,
					Tag:     tag,
					IsQuote: true,
				})
			}
			return blocks
		}

		// Headings
		if utils.HeadingTags[tag] {
			text := strings.TrimSpace(utils.TextContent(n))
			if text != "" {
				level := int(tag[1] - '0')
				blocks = append(blocks, &Block{
					Text:  text,
					Tag:   tag,
					Level: level,
				})
			}
			return blocks
		}

		// Tables
		if tag == "table" && e.cfg.IncludeTables {
			tableBlocks := e.extractTable(n)
			blocks = append(blocks, tableBlocks...)
			return blocks
		}

		// Images
		if tag == "img" && e.cfg.IncludeImages {
			alt := utils.GetAttr(n, "alt")
			src := utils.GetAttr(n, "src")
			if src != "" {
				blocks = append(blocks, &Block{
					Text:    fmt.Sprintf("![%s](%s)", alt, src),
					Tag:     tag,
					IsImage: true,
				})
			}
			return blocks
		}

		// Lists
		if tag == "ul" || tag == "ol" {
			listBlocks := e.extractList(n, tag == "ol")
			blocks = append(blocks, listBlocks...)
			return blocks
		}

		// Paragraph / line break elements: collect children
		if tag == "p" || tag == "div" || tag == "section" || tag == "article" ||
			tag == "li" || tag == "td" || tag == "th" {
			var parts []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				childBlocks := e.toBlocks(c, depth+1)
				for _, b := range childBlocks {
					if b.Tag != "" {
						// structured child — emit as-is
						blocks = append(blocks, b)
					} else {
						parts = append(parts, b.Text)
					}
				}
			}
			combined := utils.CleanText(strings.Join(parts, " "))
			if combined != "" {
				linkDensity := e.linkDensity(n)
				blocks = append(blocks, &Block{
					Text:     combined,
					Tag:      tag,
					LinkText: linkDensity,
				})
			}
			return blocks
		}

		// Inline tags: pass through children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			blocks = append(blocks, e.toBlocks(c, depth+1)...)
		}
	}

	return blocks
}

// linkDensity returns the ratio of anchor text to total text in a node.
func (e *Extractor) linkDensity(n *html.Node) float64 {
	totalText := len(strings.Fields(utils.TextContent(n)))
	if totalText == 0 {
		return 0
	}
	var linkText int
	links := utils.FindNodes(n, func(c *html.Node) bool {
		return utils.IsElement(c, "a")
	})
	for _, l := range links {
		linkText += len(strings.Fields(utils.TextContent(l)))
	}
	return float64(linkText) / float64(totalText)
}

// extractList extracts ordered or unordered list items into blocks.
func (e *Extractor) extractList(n *html.Node, ordered bool) []*Block {
	var blocks []*Block
	counter := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !utils.IsElement(c, "li") {
			continue
		}
		text := utils.CleanText(utils.TextContent(c))
		if text == "" {
			continue
		}
		counter++
		prefix := "- "
		if ordered {
			prefix = fmt.Sprintf("%d. ", counter)
		}
		blocks = append(blocks, &Block{
			Text:   prefix + text,
			Tag:    "li",
			IsList: true,
		})
	}
	return blocks
}

// extractTable converts an HTML table into text blocks (one per row).
func (e *Extractor) extractTable(n *html.Node) []*Block {
	var blocks []*Block
	rows := utils.FindNodes(n, func(c *html.Node) bool {
		return utils.IsElement(c, "tr")
	})
	for _, row := range rows {
		var cells []string
		for c := row.FirstChild; c != nil; c = c.NextSibling {
			if utils.IsElement(c, "td") || utils.IsElement(c, "th") {
				cells = append(cells, strings.TrimSpace(utils.TextContent(c)))
			}
		}
		if len(cells) > 0 {
			blocks = append(blocks, &Block{
				Text:    strings.Join(cells, " | "),
				Tag:     "tr",
				IsTable: true,
			})
		}
	}
	return blocks
}

// scoreAndFilter assigns quality scores and removes low-quality blocks.
func (e *Extractor) scoreAndFilter(blocks []*Block) []*Block {
	if len(blocks) == 0 {
		return blocks
	}

	// Compute scores
	for _, b := range blocks {
		if b.IsCode || b.IsQuote || b.IsList || b.IsTable || b.IsImage {
			b.Score = 1.0 // always keep structured elements
			continue
		}
		if b.Level > 0 {
			b.Score = 0.8 // keep headings
			continue
		}
		words := float64(len(strings.Fields(b.Text)))
		if words < 3 {
			b.Score = 0
			continue
		}
		// Penalise high link density
		linkPenalty := math.Max(0, 1-b.LinkText*2)
		b.Score = utils.ContentSignal(b.Text) * linkPenalty / 1000
	}

	// Find score threshold
	var scores []float64
	for _, b := range blocks {
		if b.Score > 0 {
			scores = append(scores, b.Score)
		}
	}

	threshold := 0.0
	if len(scores) > 0 {
		var sum float64
		for _, s := range scores {
			sum += s
		}
		avg := sum / float64(len(scores))
		if e.cfg.FavorPrecision {
			threshold = avg * 0.5
		} else if e.cfg.FavorRecall {
			threshold = avg * 0.1
		} else {
			threshold = avg * 0.25
		}
	}

	var filtered []*Block
	for _, b := range blocks {
		if b.Score > threshold || b.IsCode || b.IsQuote || b.Level > 0 || b.IsList || b.IsImage || b.IsTable {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// extractComments looks for comment sections in the document.
func (e *Extractor) extractComments(doc *html.Node) []*Block {
	commentNode := utils.FindFirst(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		for _, kw := range []string{"comments", "comment-section", "disqus"} {
			if utils.GetAttrContains(n, "id", kw) || utils.GetAttrContains(n, "class", kw) {
				return true
			}
		}
		return false
	})

	if commentNode == nil {
		return nil
	}

	return e.toBlocks(commentNode, 1)
}

// --------- Text output ---------

// ToText converts a Document to plain text.
func (d *Document) ToText() string {
	var b strings.Builder
	for i, block := range d.Blocks {
		if i > 0 {
			// Use extra blank line before headings and tables for readability
			if block.Level > 0 || block.IsTable {
				b.WriteString("\n\n")
			} else {
				b.WriteString("\n\n")
			}
		}
		b.WriteString(block.Text)
	}
	if len(d.Comments) > 0 {
		b.WriteString("\n\n--- Comments ---\n\n")
		for _, c := range d.Comments {
			b.WriteString(c.Text)
			b.WriteString("\n")
		}
	}
	return utils.CleanText(b.String())
}

// ToMarkdown converts a Document to Markdown.
func (d *Document) ToMarkdown() string {
	var b strings.Builder
	for i, block := range d.Blocks {
		if i > 0 {
			b.WriteString("\n\n")
		}
		switch {
		case block.Level > 0:
			prefix := strings.Repeat("#", block.Level)
			b.WriteString(prefix + " " + block.Text)
		case block.IsCode:
			b.WriteString("```\n" + block.Text + "\n```")
		case block.IsQuote:
			lines := strings.Split(block.Text, "\n")
			for _, l := range lines {
				b.WriteString("> " + l + "\n")
			}
		case block.IsList:
			b.WriteString(block.Text)
		case block.IsTable:
			b.WriteString(block.Text)
		case block.IsImage:
			b.WriteString(block.Text)
		default:
			b.WriteString(block.Text)
		}
	}
	if len(d.Comments) > 0 {
		b.WriteString("\n\n---\n\n## Comments\n\n")
		for _, c := range d.Comments {
			b.WriteString(c.Text + "\n\n")
		}
	}
	return strings.TrimSpace(b.String())
}
