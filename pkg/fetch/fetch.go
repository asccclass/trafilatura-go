// Package fetch handles downloading web pages with proper HTTP handling.
package fetch

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"example.com/trafilatura-go/pkg/settings"
)

// Response holds the downloaded content and metadata.
type Response struct {
	URL        string
	Body       []byte
	StatusCode int
	Headers    http.Header
	FinalURL   string // after redirects
}

// Fetcher handles HTTP downloading.
type Fetcher struct {
	client *http.Client
	cfg    *settings.Config
}

// New creates a Fetcher with the given configuration.
func New(cfg *settings.Config) *Fetcher {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= cfg.MaxRedirect {
				return fmt.Errorf("stopped after %d redirects", cfg.MaxRedirect)
			}
			return nil
		},
	}

	return &Fetcher{client: client, cfg: cfg}
}

// Fetch downloads a URL and returns the Response.
func (f *Fetcher) Fetch(rawURL string) (*Response, error) {
	return f.FetchWithContext(context.Background(), rawURL)
}

// FetchWithContext downloads a URL respecting the given context.
func (f *Fetcher) FetchWithContext(ctx context.Context, rawURL string) (*Response, error) {
	// Validate URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", f.cfg.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	for k, v := range f.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %q: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %q", resp.StatusCode, rawURL)
	}

	// Handle gzip
	var reader io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gr.Close()
		reader = gr
	}

	// Read body with configurable size limit
	maxSize := f.cfg.MaxBodySize
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // default 10 MB
	}
	body, err := io.ReadAll(io.LimitReader(reader, maxSize))
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	return &Response{
		URL:        rawURL,
		Body:       body,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		FinalURL:   resp.Request.URL.String(),
	}, nil
}

// FetchMany downloads multiple URLs concurrently, honouring the Delay setting.
func (f *Fetcher) FetchMany(urls []string) <-chan Result {
	out := make(chan Result, len(urls))
	if len(urls) == 0 {
		close(out)
		return out
	}

	workQueue := make(chan string, len(urls))
	for _, u := range urls {
		workQueue <- u
	}
	close(workQueue)

	workers := f.cfg.Concurrency
	if workers <= 0 {
		workers = 1
	}
	if workers > len(urls) {
		workers = len(urls)
	}

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for u := range workQueue {
				resp, err := f.Fetch(u)
				out <- Result{Response: resp, Err: err, URL: u}
				if f.cfg.Delay > 0 {
					time.Sleep(f.cfg.Delay)
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Result wraps a Response and potential error for concurrent operations.
type Result struct {
	URL      string
	Response *Response
	Err      error
}

// IsHTML returns true when Content-Type indicates an HTML response.
func (r *Response) IsHTML() bool {
	ct := r.Headers.Get("Content-Type")
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
}
