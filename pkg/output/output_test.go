package output

import (
	"strings"
	"testing"
	"time"

	"example.com/trafilatura-go/pkg/extract"
	"example.com/trafilatura-go/pkg/metadata"
	"example.com/trafilatura-go/pkg/settings"
)

func TestFormat(t *testing.T) {
	res := &Result{
		Meta: &metadata.Metadata{
			Title:       "Test <Title>", // Check HTML escaping
			Author:      "John Doe",
			Description: "Test & descriptions",
		},
		Doc: &extract.Document{
			Blocks: []*extract.Block{
				{Text: "Test paragraph.", Tag: "p"},
				{Text: "alert('xss')", Tag: "code", IsCode: true},
			},
		},
	}

	cfg := settings.DefaultConfig()
	cfg.Format = settings.FormatHTML

	htmlStr, err := Format(res, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify standard library html escaping
	if strings.Contains(htmlStr, "<Title>") {
		t.Errorf("expected HTML escaping for title")
	}
	if !strings.Contains(htmlStr, "&lt;Title&gt;") {
		t.Errorf("expected escaped title to be present")
	}
	if !strings.Contains(htmlStr, "&amp; descriptions") {
		t.Errorf("expected escaped description to be present")
	}
}

// helper: build a minimal Result for testing
func makeResult(title, author, bodyText string, blocks []*extract.Block) *Result {
	meta := &metadata.Metadata{
		Title:  title,
		Author: author,
		URL:    "https://example.com/test",
	}
	doc := &extract.Document{Blocks: blocks}
	return &Result{
		URL:      "https://example.com/test",
		Meta:     meta,
		Doc:      doc,
		BodyText: bodyText,
	}
}

func TestFormatText(t *testing.T) {
	r := makeResult("Hello World", "", "Some body text.", nil)
	r.Doc = nil // force plain-text path
	got := formatText(r)
	if !strings.Contains(got, "Hello World") {
		t.Errorf("expected title in output, got: %s", got)
	}
	if !strings.Contains(got, "Some body text.") {
		t.Errorf("expected body text in output, got: %s", got)
	}
}

func TestFormatMarkdown(t *testing.T) {
	meta := &metadata.Metadata{
		Title:  "My Article",
		Author: "Jane Doe",
		Date:   time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
	}
	r := &Result{
		Meta:     meta,
		BodyText: "Content here.",
		Doc:      &extract.Document{Blocks: []*extract.Block{{Text: "Content here."}}},
	}
	got := formatMarkdown(r)
	if !strings.HasPrefix(got, "# My Article") {
		t.Errorf("expected markdown h1 title, got: %s", got)
	}
	if !strings.Contains(got, "**Author:** Jane Doe") {
		t.Errorf("expected author line, got: %s", got)
	}
	if !strings.Contains(got, "**Date:** 2024-06-15") {
		t.Errorf("expected date line, got: %s", got)
	}
}

func TestFormatJSON(t *testing.T) {
	r := makeResult("JSON Test", "Alice", strings.Repeat("word ", 250), nil)
	r.Doc = nil
	cfg := settings.DefaultConfig()
	cfg.Format = settings.FormatJSON
	got, err := Format(r, cfg)
	if err != nil {
		t.Fatalf("formatJSON error: %v", err)
	}
	if !strings.Contains(got, `"title":"JSON Test"`) {
		t.Errorf("expected title in json, got: %s", got)
	}
	if !strings.Contains(got, `"excerpt"`) {
		t.Errorf("expected excerpt field in json, got: %s", got)
	}
	// Excerpt should end with ellipsis since text is long
	if !strings.Contains(got, "…") {
		t.Errorf("expected ellipsis in excerpt for long text, got: %s", got)
	}
}

func TestFormatJSON_Pretty(t *testing.T) {
	r := makeResult("Pretty", "", "Short.", nil)
	r.Doc = nil
	cfg := settings.DefaultConfig()
	cfg.Format = settings.FormatJSON
	cfg.PrettyPrint = true
	got, err := Format(r, cfg)
	if err != nil {
		t.Fatalf("formatJSON pretty error: %v", err)
	}
	// Pretty JSON should have newlines and indentation
	if !strings.Contains(got, "\n") {
		t.Errorf("expected newlines in pretty JSON, got: %s", got)
	}
}

func TestFormatHTML_TableElement(t *testing.T) {
	blocks := []*extract.Block{
		{Text: "Header 1 | Header 2 | Header 3", IsTable: true, Tag: "tr"},
		{Text: "Cell A | Cell B | Cell C", IsTable: true, Tag: "tr"},
	}
	r := makeResult("Table Test", "", "", blocks)
	cfg := settings.DefaultConfig()
	cfg.Format = settings.FormatHTML
	got, err := Format(r, cfg)
	if err != nil {
		t.Fatalf("formatHTML error: %v", err)
	}
	// Should produce real <table> not <p class="table-row">
	if !strings.Contains(got, "<table>") {
		t.Errorf("expected <table> in HTML output, got: %s", got)
	}
	if !strings.Contains(got, "<tr>") {
		t.Errorf("expected <tr> in HTML output, got: %s", got)
	}
	if !strings.Contains(got, "<td>Header 1</td>") {
		t.Errorf("expected <td>Header 1</td> in HTML output, got: %s", got)
	}
	if strings.Contains(got, "table-row") {
		t.Errorf("should NOT have old <p class=\"table-row\">, got: %s", got)
	}
}

func TestFormatHTML_Heading(t *testing.T) {
	blocks := []*extract.Block{
		{Text: "Section Title", Level: 2, Tag: "h2"},
		{Text: "Body paragraph.", Tag: "p"},
	}
	r := makeResult("Test", "", "", blocks)
	cfg := settings.DefaultConfig()
	cfg.Format = settings.FormatHTML
	got, err := Format(r, cfg)
	if err != nil {
		t.Fatalf("formatHTML error: %v", err)
	}
	if !strings.Contains(got, "<h2>Section Title</h2>") {
		t.Errorf("expected <h2>, got: %s", got)
	}
	if !strings.Contains(got, "<p>Body paragraph.</p>") {
		t.Errorf("expected <p>, got: %s", got)
	}
}

func TestFormatCSV(t *testing.T) {
	meta := &metadata.Metadata{
		Title:    "CSV Test",
		Author:   "Bob",
		URL:      "https://example.com/csv",
		Hostname: "example.com",
		Language: "en",
	}
	r := &Result{
		URL:      "https://example.com/csv",
		Meta:     meta,
		BodyText: "The body text.",
	}
	cfg := settings.DefaultConfig()
	cfg.Format = settings.FormatCSV
	got, err := Format(r, cfg)
	if err != nil {
		t.Fatalf("formatCSV error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) < 2 {
		t.Errorf("expected at least header + data row, got: %q", got)
	}
	if !strings.Contains(lines[0], "url") {
		t.Errorf("expected 'url' in CSV header, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "CSV Test") {
		t.Errorf("expected title in CSV data, got: %s", lines[1])
	}
}

func TestFormatXML_AllBlockTypes(t *testing.T) {
	blocks := []*extract.Block{
		{Text: "Main Heading", Level: 1, Tag: "h1"},
		{Text: "A paragraph of text.", Tag: "p"},
		{Text: "- List item one", IsList: true, Tag: "li"},
		{Text: "code block content", IsCode: true, Tag: "code"},
		{Text: "Col1 | Col2", IsTable: true, Tag: "tr"},
	}
	r := makeResult("XML Test", "", "", blocks)
	cfg := settings.DefaultConfig()
	cfg.Format = settings.FormatXML
	got, err := Format(r, cfg)
	if err != nil {
		t.Fatalf("formatXML error: %v", err)
	}
	if !strings.Contains(got, "<p>A paragraph") {
		t.Errorf("expected paragraph in XML <p>, got: %s", got)
	}
	if !strings.Contains(got, "<item>") {
		t.Errorf("expected list <item> in XML, got: %s", got)
	}
	if !strings.Contains(got, "<code>") {
		t.Errorf("expected <code> block in XML, got: %s", got)
	}
	if !strings.Contains(got, "<row>") {
		t.Errorf("expected table <row> in XML, got: %s", got)
	}
}

func TestTableRowToHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"A | B | C", "<tr><td>A</td><td>B</td><td>C</td></tr>"},
		{"Only One", "<tr><td>Only One</td></tr>"},
		{" Padded | Cells ", "<tr><td>Padded</td><td>Cells</td></tr>"},
	}
	for _, tt := range tests {
		got := tableRowToHTML(tt.input)
		if got != tt.expected {
			t.Errorf("tableRowToHTML(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
