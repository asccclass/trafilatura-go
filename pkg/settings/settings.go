// Package settings provides configuration structures for trafilatura-go.
package settings

import "time"

// OutputFormat defines the output format for extracted content.
type OutputFormat string

const (
	FormatTXT      OutputFormat = "txt"
	FormatMarkdown OutputFormat = "markdown"
	FormatJSON     OutputFormat = "json"
	FormatCSV      OutputFormat = "csv"
	FormatHTML     OutputFormat = "html"
	FormatXML      OutputFormat = "xml"
)

// Config holds all configuration for extraction.
type Config struct {
	// Extraction options
	IncludeComments  bool
	IncludeTables    bool
	IncludeImages    bool
	IncludeLinks     bool
	NoFallback       bool
	FavorPrecision   bool
	FavorRecall      bool
	MinExtractedSize int // minimum character count
	MinOutputSize    int // minimum output characters

	// Fetch options
	Timeout     time.Duration
	MaxRedirect int
	MaxBodySize int64 // maximum response body size in bytes (0 = use default 10 MB)
	Headless    bool  // use headless Chromium (Playwright) to render JS-heavy SPA pages
	UserAgent   string
	Headers     map[string]string

	// Output options
	Format      OutputFormat
	PrettyPrint bool

	// Spider options
	MaxDepth    int
	MaxURLs     int
	Delay       time.Duration
	SameHost    bool
	Concurrency int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		IncludeComments:  false,
		IncludeTables:    true,
		IncludeImages:    false,
		IncludeLinks:     false,
		NoFallback:       false,
		FavorPrecision:   false,
		FavorRecall:      false,
		MinExtractedSize: 250,
		MinOutputSize:    10,
		Timeout:          30 * time.Second,
		MaxRedirect:      10,
		UserAgent:        "trafilatura-go/1.0 (+https://github.com/trafilatura-go)",
		Headers:          map[string]string{},
		Format:           FormatTXT,
		PrettyPrint:      false,
		MaxDepth:         5,
		MaxURLs:          1000,
		Delay:            1 * time.Second,
		SameHost:         true,
		Concurrency:      4,
	}
}
