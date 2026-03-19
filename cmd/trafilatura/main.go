// trafilatura-go: a Go reimplementation of the trafilatura web content extractor.
//
// Usage:
//
//	trafilatura [flags] [URL | -f file.html]
//
// Examples:
//
//	trafilatura https://example.com/article
//	trafilatura -format json https://example.com/article
//	trafilatura -crawl -max-urls 50 https://example.com
//	cat page.html | trafilatura -stdin -url https://example.com/page
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	trafilatura "example.com/trafilatura-go"
	"example.com/trafilatura-go/pkg/fetch"
	"example.com/trafilatura-go/pkg/output"
	"example.com/trafilatura-go/pkg/settings"
	"example.com/trafilatura-go/pkg/spider"
)

func main() {
	// --- CLI Flags ---
	var (
		fFormat          = flag.String("format", "txt", "Output format: txt, markdown, json, csv, html, xml")
		fIncludeComments = flag.Bool("comments", false, "Include comment sections")
		fIncludeTables   = flag.Bool("tables", true, "Include tables")
		fIncludeImages   = flag.Bool("images", false, "Include images (Markdown syntax)")
		fIncludeLinks    = flag.Bool("links", false, "Include hyperlinks")
		fNoFallback      = flag.Bool("no-fallback", false, "Disable fallback extraction algorithms")
		fPrecision       = flag.Bool("precision", false, "Favour precision over recall")
		fRecall          = flag.Bool("recall", false, "Favour recall over precision")
		fPretty          = flag.Bool("pretty", false, "Pretty-print JSON output")
		fTimeout         = flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
		fUserAgent       = flag.String("user-agent", "trafilatura-go/1.0", "HTTP User-Agent header")
		fDelay           = flag.Duration("delay", 500*time.Millisecond, "Delay between requests when crawling")
		fConcurrency     = flag.Int("concurrency", 4, "Number of concurrent download workers")
		fMaxURLs         = flag.Int("max-urls", 1000, "Maximum URLs to crawl")
		fCrawl           = flag.Bool("crawl", false, "Crawl the site from the given seed URL")
		fSitemap         = flag.Bool("sitemap", false, "Discover and use sitemap instead of crawling")
		fSameHost        = flag.Bool("same-host", true, "Only follow links to the same hostname")
		fHeadless        = flag.Bool("headless", false, "Use headless Chromium (Playwright) to render SPA/JS pages")
		fStdin           = flag.Bool("stdin", false, "Read HTML from stdin instead of fetching")
		fURL             = flag.String("url", "", "Source URL (used for metadata when reading from stdin/file)")
		fFile            = flag.String("f", "", "Read HTML from a file")
		fOutput          = flag.String("o", "", "Write output to a file (default: stdout)")
		fVersion         = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *fVersion {
		fmt.Println("trafilatura-go v1.0.0")
		os.Exit(0)
	}

	// --- Build configuration ---
	cfg := &settings.Config{
		IncludeComments:  *fIncludeComments,
		IncludeTables:    *fIncludeTables,
		IncludeImages:    *fIncludeImages,
		IncludeLinks:     *fIncludeLinks,
		NoFallback:       *fNoFallback,
		FavorPrecision:   *fPrecision,
		FavorRecall:      *fRecall,
		MinExtractedSize: 250,
		MinOutputSize:    10,
		Timeout:          *fTimeout,
		MaxRedirect:      10,
		Headless:         *fHeadless,
		UserAgent:        *fUserAgent,
		Headers:          map[string]string{},
		Format:           settings.OutputFormat(*fFormat),
		PrettyPrint:      *fPretty,
		MaxDepth:         5,
		MaxURLs:          *fMaxURLs,
		Delay:            *fDelay,
		SameHost:         *fSameHost,
		Concurrency:      *fConcurrency,
	}

	// --- Determine output writer ---
	var outWriter io.Writer = os.Stdout
	if *fOutput != "" {
		f, err := os.Create(*fOutput)
		if err != nil {
			log.Fatalf("Cannot open output file: %v", err)
		}
		defer f.Close()
		outWriter = f
	}

	// --- Dispatch based on mode ---

	// 1. Stdin mode
	if *fStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Reading stdin: %v", err)
		}
		result, err := trafilatura.Extract(data, *fURL, cfg)
		if err != nil {
			log.Fatalf("Extraction failed: %v", err)
		}
		fmt.Fprintln(outWriter, result)
		return
	}

	// 2. File mode
	if *fFile != "" {
		data, err := os.ReadFile(*fFile)
		if err != nil {
			log.Fatalf("Reading file: %v", err)
		}
		pageURL := *fURL
		if pageURL == "" {
			pageURL = "file://" + *fFile
		}
		result, err := trafilatura.Extract(data, pageURL, cfg)
		if err != nil {
			log.Fatalf("Extraction failed: %v", err)
		}
		fmt.Fprintln(outWriter, result)
		return
	}

	// 3. URL mode
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	seedURL := args[0]
	fetcher := fetch.New(cfg)

	// 4. Sitemap mode
	if *fSitemap {
		urls, err := spider.DiscoverSitemap(seedURL, fetcher)
		if err != nil {
			log.Fatalf("Sitemap discovery: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Found %d URLs in sitemap\n", len(urls))
		processURLList(urls, fetcher, cfg, outWriter)
		return
	}

	// 5. Crawl mode
	if *fCrawl {
		crawler := spider.NewCrawler(fetcher, cfg)
		results := crawler.Crawl(seedURL)
		for res := range results {
			if res.Error != nil {
				fmt.Fprintf(os.Stderr, "Error crawling %s: %v\n", res.URL, res.Error)
				continue
			}
			extracted, err := trafilatura.Extract(res.Body, res.URL, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Extraction error for %s: %v\n", res.URL, err)
				continue
			}
			fmt.Fprintf(outWriter, "=== %s ===\n%s\n\n", res.URL, extracted)
		}
		return
	}

	// 6. Single URL mode
	extracted, err := trafilatura.Process(seedURL, cfg)
	if err != nil {
		log.Fatalf("Processing %s: %v", seedURL, err)
	}
	fmt.Fprintln(outWriter, extracted)
}

// processURLList downloads and extracts a list of URLs sequentially.
func processURLList(urls []string, f *fetch.Fetcher, cfg *settings.Config, w io.Writer) {
	for _, u := range urls {
		if u = strings.TrimSpace(u); u == "" {
			continue
		}
		resp, err := f.Fetch(u)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fetch error %s: %v\n", u, err)
			continue
		}
		extracted, err := trafilatura.Extract(resp.Body, u, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Extract error %s: %v\n", u, err)
			continue
		}
		meta, _ := trafilatura.ExtractMetadata(resp.Body, u)
		_ = output.Result{Meta: meta}
		fmt.Fprintf(w, "=== %s ===\n%s\n\n", u, extracted)
		time.Sleep(cfg.Delay)
	}
}
