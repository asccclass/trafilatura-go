// Package trafilatura is the top-level API for trafilatura-go.
// It mirrors the Python library's main public functions:
//   - Fetch   – download a URL (HTTP or headless browser)
//   - Extract – extract text from raw HTML bytes
//   - Process – fetch + extract + format in one call
package trafilatura

import (
	"bytes"
	"fmt"

	"golang.org/x/net/html"

	"example.com/trafilatura-go/pkg/extract"
	"example.com/trafilatura-go/pkg/fetch"
	"example.com/trafilatura-go/pkg/headless"
	"example.com/trafilatura-go/pkg/langdetect"
	"example.com/trafilatura-go/pkg/metadata"
	"example.com/trafilatura-go/pkg/output"
	"example.com/trafilatura-go/pkg/settings"
)

// Fetch downloads a URL and returns its body bytes.
// When cfg.Headless is true, it uses a Playwright headless Chromium browser to
// render the page (required for JavaScript-heavy SPA pages). Otherwise it uses
// a standard HTTP client.
func Fetch(rawURL string, cfg *settings.Config) ([]byte, error) {
	if cfg == nil {
		cfg = settings.DefaultConfig()
	}

	if cfg.Headless {
		client, err := headless.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("headless init: %w", err)
		}
		defer client.Close()
		return client.Fetch(rawURL)
	}

	f := fetch.New(cfg)
	resp, err := f.Fetch(rawURL)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// Extract takes raw HTML bytes (and optional source URL) and returns
// the extracted text in the format specified by cfg.
// If no language was detected from metadata, the extracted body text is used
// for automatic language detection via the langdetect package.
func Extract(data []byte, pageURL string, cfg *settings.Config) (string, error) {
	if cfg == nil {
		cfg = settings.DefaultConfig()
	}

	// Parse HTML
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("HTML parse error: %w", err)
	}

	// Extract metadata
	meta := metadata.Extract(doc, pageURL)

	// Extract main content
	ex := extract.New(cfg)
	document := ex.FromNode(doc)

	if len(document.Blocks) == 0 {
		return "", fmt.Errorf("no content extracted from %q", pageURL)
	}

	// Render body text
	var bodyText string
	if cfg.Format == settings.FormatMarkdown {
		bodyText = document.ToMarkdown()
	} else {
		bodyText = document.ToText()
	}

	// Check minimum size
	if len([]rune(bodyText)) < cfg.MinOutputSize {
		return "", fmt.Errorf("extracted content too short (%d chars)", len([]rune(bodyText)))
	}

	// Language fallback: if metadata didn't provide a language, detect from text
	if meta.Language == "" {
		if detected := langdetect.Detect(bodyText); detected != "" {
			meta.Language = detected
		}
	}

	result := &output.Result{
		URL:      pageURL,
		Meta:     meta,
		Doc:      document,
		BodyText: bodyText,
	}

	return output.Format(result, cfg)
}

// Process fetches a URL and extracts its content in one call.
func Process(rawURL string, cfg *settings.Config) (string, error) {
	if cfg == nil {
		cfg = settings.DefaultConfig()
	}
	data, err := Fetch(rawURL, cfg)
	if err != nil {
		return "", err
	}
	return Extract(data, rawURL, cfg)
}

// ExtractMetadata returns only the metadata for a given HTML document,
// without extracting the main text body.
func ExtractMetadata(data []byte, pageURL string) (*metadata.Metadata, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("HTML parse error: %w", err)
	}
	return metadata.Extract(doc, pageURL), nil
}
