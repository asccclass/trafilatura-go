// Package spider handles web crawling, sitemap parsing, feed discovery,
// and URL management (deduplication and filtering).
package spider

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"

	"example.com/trafilatura-go/pkg/fetch"
	"example.com/trafilatura-go/pkg/settings"
	"example.com/trafilatura-go/pkg/utils"
)

// ---- URL management ----

// URLStore tracks seen URLs with deduplication.
type URLStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

// NewURLStore creates an empty URLStore.
func NewURLStore() *URLStore {
	return &URLStore{seen: make(map[string]struct{})}
}

// Add records a URL as seen, returning true if it was new.
func (s *URLStore) Add(u string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	norm := normaliseURL(u)
	if _, exists := s.seen[norm]; exists {
		return false
	}
	s.seen[norm] = struct{}{}
	return true
}

// Seen returns true when the URL has already been recorded.
func (s *URLStore) Seen(u string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.seen[normaliseURL(u)]
	return ok
}

// Len returns the number of tracked URLs.
func (s *URLStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.seen)
}

// normaliseURL strips trailing slashes and fragments for deduplication.
func normaliseURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Fragment = ""
	result := u.String()
	return strings.TrimRight(result, "/")
}

// FilterURL returns false when a URL should be excluded.
func FilterURL(rawURL, baseHost string, sameHost bool) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	// Exclude common non-content paths
	path := strings.ToLower(u.Path)
	for _, skip := range []string{"/tag/", "/category/", "/page/", "/wp-content/",
		".pdf", ".jpg", ".png", ".gif", ".zip", ".mp4", ".mp3"} {
		if strings.Contains(path, skip) {
			return false
		}
	}
	if sameHost && u.Hostname() != baseHost {
		return false
	}
	return true
}

// ---- Sitemap parsing ----

// sitemapIndex is the top-level sitemap index element.
type sitemapIndex struct {
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// urlSet represents a standard sitemap urlset.
type urlSet struct {
	URLs []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod"`
	ChangeFreq string  `xml:"changefreq"`
	Priority   float64 `xml:"priority"`
}

// ParseSitemap parses sitemap XML (urlset or sitemapindex) and returns all URLs.
func ParseSitemap(data []byte) ([]string, error) {
	var urls []string

	// Try urlset first
	var us urlSet
	if err := xml.Unmarshal(data, &us); err == nil && len(us.URLs) > 0 {
		for _, u := range us.URLs {
			if u.Loc != "" {
				urls = append(urls, strings.TrimSpace(u.Loc))
			}
		}
		return urls, nil
	}

	// Try sitemapIndex
	var si sitemapIndex
	if err := xml.Unmarshal(data, &si); err == nil && len(si.Sitemaps) > 0 {
		for _, s := range si.Sitemaps {
			if s.Loc != "" {
				urls = append(urls, strings.TrimSpace(s.Loc))
			}
		}
		return urls, nil
	}

	// Plain text fallback (one URL per line)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "http") {
			urls = append(urls, line)
		}
	}
	return urls, nil
}

// DiscoverSitemap tries common sitemap paths for a base URL.
func DiscoverSitemap(baseURL string, f *fetch.Fetcher) ([]string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	base := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

	candidates := []string{
		base + "/sitemap.xml",
		base + "/sitemap_index.xml",
		base + "/sitemap/sitemap.xml",
		base + "/robots.txt",
	}

	for _, candidate := range candidates {
		resp, err := f.Fetch(candidate)
		if err != nil {
			continue
		}
		if strings.HasSuffix(candidate, "robots.txt") {
			// Extract sitemap directive from robots.txt
			for _, line := range strings.Split(string(resp.Body), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(strings.ToLower(line), "sitemap:") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						smURL := strings.TrimSpace(parts[1])
						smResp, err := f.Fetch(smURL)
						if err == nil {
							return ParseSitemap(smResp.Body)
						}
					}
				}
			}
			continue
		}
		urls, err := ParseSitemap(resp.Body)
		if err == nil && len(urls) > 0 {
			return urls, nil
		}
	}
	return nil, fmt.Errorf("no sitemap found for %s", baseURL)
}

// ---- Feed parsing ----

// feedEntry represents an entry in an RSS/Atom feed.
type feedEntry struct {
	Title   string
	URL     string
	PubDate string
}

// ParseFeed parses RSS or Atom XML and returns entries.
func ParseFeed(data []byte) ([]feedEntry, error) {
	// Try RSS
	type rssItem struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		PubDate string `xml:"pubDate"`
	}
	type rssChannel struct {
		Items []rssItem `xml:"item"`
	}
	type rss struct {
		Channel rssChannel `xml:"channel"`
	}

	var rssFeed rss
	if err := xml.Unmarshal(data, &rssFeed); err == nil && len(rssFeed.Channel.Items) > 0 {
		var entries []feedEntry
		for _, item := range rssFeed.Channel.Items {
			entries = append(entries, feedEntry{
				Title:   item.Title,
				URL:     item.Link,
				PubDate: item.PubDate,
			})
		}
		return entries, nil
	}

	// Try Atom
	type atomLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	}
	type atomEntry struct {
		Title   string     `xml:"title"`
		Links   []atomLink `xml:"link"`
		Updated string     `xml:"updated"`
	}
	type atom struct {
		Entries []atomEntry `xml:"entry"`
	}

	var atomFeed atom
	if err := xml.Unmarshal(data, &atomFeed); err == nil && len(atomFeed.Entries) > 0 {
		var entries []feedEntry
		for _, entry := range atomFeed.Entries {
			link := ""
			for _, l := range entry.Links {
				if l.Rel == "alternate" || l.Rel == "" {
					link = l.Href
					break
				}
			}
			entries = append(entries, feedEntry{
				Title:   entry.Title,
				URL:     link,
				PubDate: entry.Updated,
			})
		}
		return entries, nil
	}

	return nil, fmt.Errorf("could not parse feed")
}

// DiscoverFeeds looks for RSS/Atom feed links in an HTML page.
func DiscoverFeeds(data []byte, baseURL string) []string {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	base, _ := url.Parse(baseURL)
	var feeds []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "link") {
			rel := strings.ToLower(getAttr(n, "rel"))
			typ := strings.ToLower(getAttr(n, "type"))
			if strings.Contains(rel, "alternate") &&
				(strings.Contains(typ, "rss") || strings.Contains(typ, "atom") || strings.Contains(typ, "json")) {
				href := getAttr(n, "href")
				if href != "" {
					u, err := url.Parse(href)
					if err == nil {
						if !u.IsAbs() && base != nil {
							u = base.ResolveReference(u)
						}
						feeds = append(feeds, u.String())
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return feeds
}

// getAttr is an alias to utils.GetAttr for backward compatibility within this package.
func getAttr(n *html.Node, key string) string {
	return utils.GetAttr(n, key)
}

// ---- Crawler ----

// CrawlResult holds the result of crawling a single page.
type CrawlResult struct {
	URL   string
	Body  []byte
	Error error
}

// Crawler performs breadth-first web crawling from a seed URL.
type Crawler struct {
	fetcher *fetch.Fetcher
	cfg     *settings.Config
	store   *URLStore
}

// NewCrawler creates a Crawler.
func NewCrawler(f *fetch.Fetcher, cfg *settings.Config) *Crawler {
	return &Crawler{
		fetcher: f,
		cfg:     cfg,
		store:   NewURLStore(),
	}
}

// queueItem holds a URL along with its BFS crawl depth.
type queueItem struct {
	url   string
	depth int
}

// Crawl performs BFS crawling starting from seedURL, returning a channel of results.
// It respects cfg.MaxDepth to limit crawl depth and cfg.MaxURLs to cap total URLs.
func (c *Crawler) Crawl(seedURL string) <-chan CrawlResult {
	out := make(chan CrawlResult, 100)
	go func() {
		defer close(out)
		base, err := url.Parse(seedURL)
		if err != nil {
			out <- CrawlResult{URL: seedURL, Error: err}
			return
		}
		baseHost := base.Hostname()

		queue := []queueItem{{url: seedURL, depth: 0}}
		c.store.Add(seedURL)

		for len(queue) > 0 && c.store.Len() <= c.cfg.MaxURLs {
			item := queue[0]
			queue = queue[1:]

			resp, err := c.fetcher.Fetch(item.url)
			if err != nil {
				out <- CrawlResult{URL: item.url, Error: err}
				continue
			}

			out <- CrawlResult{URL: item.url, Body: resp.Body}

			// Discover links only if we haven't reached max depth
			if item.depth < c.cfg.MaxDepth && c.store.Len() < c.cfg.MaxURLs {
				links := extractLinks(resp.Body, item.url)
				for _, link := range links {
					if !c.store.Seen(link) && FilterURL(link, baseHost, c.cfg.SameHost) {
						c.store.Add(link)
						queue = append(queue, queueItem{url: link, depth: item.depth + 1})
					}
				}
			}
		}
	}()
	return out
}

// extractLinks finds all href links in HTML content.
func extractLinks(data []byte, baseURL string) []string {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	base, _ := url.Parse(baseURL)
	var links []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "a") {
			href := getAttr(n, "href")
			if href != "" && !strings.HasPrefix(href, "#") && !strings.HasPrefix(href, "javascript:") {
				u, err := url.Parse(href)
				if err == nil {
					if !u.IsAbs() && base != nil {
						u = base.ResolveReference(u)
					}
					links = append(links, u.String())
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return links
}
