// Package headless provides a Playwright-based headless browser fetcher.
// It renders JavaScript-heavy pages (SPA/React/Vue) before returning the HTML,
// unlike the standard fetch package which only retrieves static HTML.
//
// First-time setup – install Chromium (run once):
//
//	go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
package headless

import (
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"

	"example.com/trafilatura-go/pkg/settings"
)

// Client holds a running Playwright instance and Chromium browser.
// A single Client should be reused across multiple Fetch calls to avoid
// the overhead of restarting the browser each time.
type Client struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	cfg     *settings.Config
}

// New launches a headless Chromium browser using Playwright.
// Call Close() when done to release all resources.
func New(cfg *settings.Config) (*Client, error) {
	if cfg == nil {
		cfg = settings.DefaultConfig()
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("launching playwright: %w", err)
	}

	headless := true
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: &headless,
	})
	if err != nil {
		pw.Stop() //nolint:errcheck
		return nil, fmt.Errorf("launching chromium: %w", err)
	}

	return &Client{pw: pw, browser: browser, cfg: cfg}, nil
}

// Fetch navigates to rawURL in a new browser page, waits for the network to
// become idle (JavaScript rendering complete), and returns the final HTML.
// The page is closed after each call; the browser remains open for reuse.
func (c *Client) Fetch(rawURL string) ([]byte, error) {
	timeout := float64(c.cfg.Timeout / time.Millisecond)
	if timeout <= 0 {
		timeout = 30_000 // 30 s default
	}

	ctx, err := c.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(c.cfg.UserAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("creating browser context: %w", err)
	}
	defer ctx.Close()

	page, err := ctx.NewPage()
	if err != nil {
		return nil, fmt.Errorf("creating page: %w", err)
	}
	defer page.Close()

	// Navigate and wait until network is idle (JS rendered)
	_, err = page.Goto(rawURL, playwright.PageGotoOptions{
		Timeout:   playwright.Float(timeout),
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		return nil, fmt.Errorf("navigating to %q: %w", rawURL, err)
	}

	// Get the fully rendered HTML
	html, err := page.Content()
	if err != nil {
		return nil, fmt.Errorf("getting page content: %w", err)
	}

	return []byte(html), nil
}

// Close stops the browser and Playwright process, releasing all resources.
func (c *Client) Close() {
	c.browser.Close() //nolint:errcheck
	c.pw.Stop()       //nolint:errcheck
}
