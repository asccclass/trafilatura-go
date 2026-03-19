package metadata

import (
	"bytes"
	"testing"

	"golang.org/x/net/html"
)

func TestExtract(t *testing.T) {
	htmlStr := `
	<html lang="en">
	<head>
		<title>Page Title</title>
		<meta name="author" content="John Doe">
		<meta property="og:description" content="A test page.">
		<script type="application/ld+json">
		{
			"@type": "Article",
			"headline": "JSON-LD Title",
			"datePublished": "2023-10-01"
		}
		</script>
	</head>
	<body></body>
	</html>`

	doc, _ := html.Parse(bytes.NewReader([]byte(htmlStr)))
	m := Extract(doc, "https://example.com/test")

	// JSON-LD is preferred over <title> if present
	if m.Title != "JSON-LD Title" {
		t.Errorf("expected title 'JSON-LD Title', got %q", m.Title)
	}
	if m.Author != "John Doe" {
		t.Errorf("expected author 'John Doe', got %q", m.Author)
	}
	if m.Description != "A test page." {
		t.Errorf("expected description 'A test page.', got %q", m.Description)
	}
	if m.Language != "en" {
		t.Errorf("expected language 'en', got %q", m.Language)
	}
	if m.DateStr != "2023-10-01" {
		t.Errorf("expected DateStr '2023-10-01', got %q", m.DateStr)
	}
}
