// Package output provides formatters that convert extracted content and
// metadata to TXT, Markdown, JSON, CSV, HTML, and XML.
package output

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"strings"
	"time"

	"example.com/trafilatura-go/pkg/extract"
	"example.com/trafilatura-go/pkg/metadata"
	"example.com/trafilatura-go/pkg/settings"
)

// Result combines extracted document and metadata.
type Result struct {
	URL      string
	Meta     *metadata.Metadata
	Doc      *extract.Document
	BodyText string // pre-rendered plain text body
}

// Format converts a Result to the requested output format.
func Format(r *Result, cfg *settings.Config) (string, error) {
	switch cfg.Format {
	case settings.FormatTXT:
		return formatText(r), nil
	case settings.FormatMarkdown:
		return formatMarkdown(r), nil
	case settings.FormatJSON:
		return formatJSON(r, cfg.PrettyPrint)
	case settings.FormatCSV:
		return formatCSV(r)
	case settings.FormatHTML:
		return formatHTML(r), nil
	case settings.FormatXML:
		return formatXML(r)
	default:
		return formatText(r), nil
	}
}

// ---- Plain Text ----

func formatText(r *Result) string {
	var b strings.Builder
	if r.Meta != nil && r.Meta.Title != "" {
		b.WriteString(r.Meta.Title)
		b.WriteString("\n")
		b.WriteString(strings.Repeat("=", len(r.Meta.Title)))
		b.WriteString("\n\n")
	}
	b.WriteString(r.BodyText)
	return b.String()
}

// ---- Markdown ----

func formatMarkdown(r *Result) string {
	var b strings.Builder
	if r.Meta != nil {
		if r.Meta.Title != "" {
			b.WriteString("# " + r.Meta.Title + "\n\n")
		}
		if r.Meta.Author != "" {
			b.WriteString(fmt.Sprintf("**Author:** %s  \n", r.Meta.Author))
		}
		if !r.Meta.Date.IsZero() {
			b.WriteString(fmt.Sprintf("**Date:** %s  \n", r.Meta.Date.Format("2006-01-02")))
		}
		if r.Meta.Author != "" || !r.Meta.Date.IsZero() {
			b.WriteString("\n---\n\n")
		}
	}
	if r.Doc != nil {
		b.WriteString(r.Doc.ToMarkdown())
	} else {
		b.WriteString(r.BodyText)
	}
	return b.String()
}

// ---- JSON ----

type jsonOutput struct {
	Title       string   `json:"title,omitempty"`
	Author      string   `json:"author,omitempty"`
	URL         string   `json:"url,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
	Description string   `json:"description,omitempty"`
	SiteName    string   `json:"sitename,omitempty"`
	Date        string   `json:"date,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Language    string   `json:"language,omitempty"`
	Image       string   `json:"image,omitempty"`
	PageType    string   `json:"pagetype,omitempty"`
	Text        string   `json:"text"`
	Excerpt     string   `json:"excerpt,omitempty"`
}

func formatJSON(r *Result, pretty bool) (string, error) {
	out := jsonOutput{Text: r.BodyText}
	if r.Meta != nil {
		out.Title = r.Meta.Title
		out.Author = r.Meta.Author
		out.URL = r.Meta.URL
		out.Hostname = r.Meta.Hostname
		out.Description = r.Meta.Description
		out.SiteName = r.Meta.SiteName
		out.Categories = r.Meta.Categories
		out.Tags = r.Meta.Tags
		out.Language = r.Meta.Language
		out.Image = r.Meta.Image
		out.PageType = r.Meta.PageType
		if !r.Meta.Date.IsZero() {
			out.Date = r.Meta.Date.Format(time.RFC3339)
		}
	}
	// First 200 runes as excerpt
	runes := []rune(r.BodyText)
	if len(runes) > 200 {
		out.Excerpt = string(runes[:200]) + "…"
	} else if len(runes) > 0 {
		out.Excerpt = string(runes)
	}

	var data []byte
	var err error
	if pretty {
		data, err = json.MarshalIndent(out, "", "  ")
	} else {
		data, err = json.Marshal(out)
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ---- CSV ----

func formatCSV(r *Result) (string, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)
	header := []string{"url", "title", "author", "date", "hostname", "description",
		"sitename", "categories", "tags", "language", "pagetype", "text"}
	if err := w.Write(header); err != nil {
		return "", err
	}

	row := make([]string, len(header))
	if r.Meta != nil {
		row[0] = r.Meta.URL
		row[1] = r.Meta.Title
		row[2] = r.Meta.Author
		if !r.Meta.Date.IsZero() {
			row[3] = r.Meta.Date.Format("2006-01-02")
		}
		row[4] = r.Meta.Hostname
		row[5] = r.Meta.Description
		row[6] = r.Meta.SiteName
		row[7] = strings.Join(r.Meta.Categories, "; ")
		row[8] = strings.Join(r.Meta.Tags, "; ")
		row[9] = r.Meta.Language
		row[10] = r.Meta.PageType
	}
	row[11] = r.BodyText
	if err := w.Write(row); err != nil {
		return "", err
	}
	w.Flush()
	return b.String(), w.Error()
}

// ---- HTML ----

// tableRowToHTML converts a pipe-separated row string to an HTML <tr> element.
func tableRowToHTML(rowText string) string {
	cells := strings.Split(rowText, " | ")
	var sb strings.Builder
	sb.WriteString("<tr>")
	for _, cell := range cells {
		sb.WriteString("<td>" + html.EscapeString(strings.TrimSpace(cell)) + "</td>")
	}
	sb.WriteString("</tr>")
	return sb.String()
}

// formatHTML renders a Result as a self-contained HTML document.
func formatHTML(r *Result) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	if r.Meta != nil {
		b.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(r.Meta.Title)))
		if r.Meta.Description != "" {
			b.WriteString(fmt.Sprintf(`  <meta name="description" content="%s">`+"\n", html.EscapeString(r.Meta.Description)))
		}
	}
	b.WriteString("</head>\n<body>\n")
	if r.Meta != nil {
		if r.Meta.Title != "" {
			b.WriteString("<h1>" + html.EscapeString(r.Meta.Title) + "</h1>\n")
		}
		meta := []string{}
		if r.Meta.Author != "" {
			meta = append(meta, "By "+html.EscapeString(r.Meta.Author))
		}
		if !r.Meta.Date.IsZero() {
			meta = append(meta, r.Meta.Date.Format("January 2, 2006"))
		}
		if len(meta) > 0 {
			b.WriteString("<p class=\"meta\">" + strings.Join(meta, " | ") + "</p>\n")
		}
	}

	if r.Doc != nil {
		b.WriteString("<div class=\"content\">\n")
		i := 0
		for i < len(r.Doc.Blocks) {
			block := r.Doc.Blocks[i]
			switch {
			case block.Level > 0:
				tag := fmt.Sprintf("h%d", block.Level)
				b.WriteString(fmt.Sprintf("<%s>%s</%s>\n", tag, html.EscapeString(block.Text), tag))
			case block.IsCode:
				b.WriteString("<pre><code>" + html.EscapeString(block.Text) + "</code></pre>\n")
			case block.IsQuote:
				b.WriteString("<blockquote><p>" + html.EscapeString(block.Text) + "</p></blockquote>\n")
			case block.IsTable:
				// Collect consecutive table rows and wrap them in a <table>
				b.WriteString("<table>\n")
				for i < len(r.Doc.Blocks) && r.Doc.Blocks[i].IsTable {
					b.WriteString(tableRowToHTML(r.Doc.Blocks[i].Text) + "\n")
					i++
				}
				b.WriteString("</table>\n")
				continue // i already advanced
			case block.IsImage:
				b.WriteString("<p>" + block.Text + "</p>\n") // already Markdown image syntax
			default:
				b.WriteString("<p>" + html.EscapeString(block.Text) + "</p>\n")
			}
			i++
		}
		b.WriteString("</div>\n")
	} else {
		b.WriteString("<div class=\"content\">\n")
		for _, para := range strings.Split(r.BodyText, "\n\n") {
			para = strings.TrimSpace(para)
			if para != "" {
				b.WriteString("<p>" + html.EscapeString(para) + "</p>\n")
			}
		}
		b.WriteString("</div>\n")
	}

	b.WriteString("</body>\n</html>")
	return b.String()
}

// ---- XML ----

type xmlDoc struct {
	XMLName     xml.Name `xml:"doc"`
	URL         string   `xml:"url,attr,omitempty"`
	Title       xmlText  `xml:"head>title,omitempty"`
	Author      xmlText  `xml:"head>author,omitempty"`
	Date        string   `xml:"head>date,omitempty"`
	Description xmlText  `xml:"head>description,omitempty"`
	SiteName    string   `xml:"head>sitename,omitempty"`
	Categories  []string `xml:"head>categories>category,omitempty"`
	Tags        []string `xml:"head>tags>tag,omitempty"`
	Language    string   `xml:"head>language,omitempty"`
	Body        xmlBody  `xml:"body"`
}

type xmlText struct {
	Value string `xml:",chardata"`
}

type xmlBody struct {
	Paragraphs []xmlParagraph `xml:"p,omitempty"`
	Headings   []xmlHeading   `xml:"head,omitempty"`
	Lists      []xmlParagraph `xml:"list>item,omitempty"`
	Code       []xmlParagraph `xml:"code,omitempty"`
	Tables     []xmlParagraph `xml:"table>row,omitempty"`
}

type xmlParagraph struct {
	Text string `xml:",chardata"`
}

type xmlHeading struct {
	Level int    `xml:"rend,attr,omitempty"`
	Text  string `xml:",chardata"`
}

func formatXML(r *Result) (string, error) {
	d := xmlDoc{}
	if r.Meta != nil {
		d.URL = r.Meta.URL
		d.Title = xmlText{r.Meta.Title}
		d.Author = xmlText{r.Meta.Author}
		d.Description = xmlText{r.Meta.Description}
		d.SiteName = r.Meta.SiteName
		d.Categories = r.Meta.Categories
		d.Tags = r.Meta.Tags
		d.Language = r.Meta.Language
		if !r.Meta.Date.IsZero() {
			d.Date = r.Meta.Date.Format("2006-01-02")
		}
	}

	if r.Doc != nil {
		for _, b := range r.Doc.Blocks {
			switch {
			case b.Level > 0:
				d.Body.Headings = append(d.Body.Headings, xmlHeading{Level: b.Level, Text: b.Text})
			case b.IsList:
				d.Body.Lists = append(d.Body.Lists, xmlParagraph{b.Text})
			case b.IsCode:
				d.Body.Code = append(d.Body.Code, xmlParagraph{b.Text})
			case b.IsTable:
				d.Body.Tables = append(d.Body.Tables, xmlParagraph{b.Text})
			default:
				d.Body.Paragraphs = append(d.Body.Paragraphs, xmlParagraph{b.Text})
			}
		}
	} else {
		for _, para := range strings.Split(r.BodyText, "\n\n") {
			para = strings.TrimSpace(para)
			if para != "" {
				d.Body.Paragraphs = append(d.Body.Paragraphs, xmlParagraph{para})
			}
		}
	}

	data, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return xml.Header + string(data), nil
}
