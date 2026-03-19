package spider

import (
	"testing"
)

// ---- ParseSitemap tests ----

func TestParseSitemap_URLSet(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/page1</loc></url>
  <url><loc>https://example.com/page2</loc></url>
  <url><loc>  https://example.com/page3  </loc></url>
</urlset>`

	urls, err := ParseSitemap([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 3 {
		t.Errorf("expected 3 URLs, got %d: %v", len(urls), urls)
	}
	if urls[0] != "https://example.com/page1" {
		t.Errorf("expected first URL to be 'https://example.com/page1', got %q", urls[0])
	}
	// Whitespace should be trimmed (page3)
	if urls[2] != "https://example.com/page3" {
		t.Errorf("expected trimmed URL, got %q", urls[2])
	}
}

func TestParseSitemap_SitemapIndex(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://example.com/sitemap1.xml</loc></sitemap>
  <sitemap><loc>https://example.com/sitemap2.xml</loc></sitemap>
</sitemapindex>`

	urls, err := ParseSitemap([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 2 {
		t.Errorf("expected 2 sitemap URLs, got %d", len(urls))
	}
}

func TestParseSitemap_PlainText(t *testing.T) {
	text := "https://example.com/a\nhttps://example.com/b\nnot-a-url\n  https://example.com/c"
	urls, err := ParseSitemap([]byte(text))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 3 {
		t.Errorf("expected 3 URLs from plain text, got %d: %v", len(urls), urls)
	}
}

// ---- ParseFeed tests ----

func TestParseFeed_RSS(t *testing.T) {
	rss := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Article One</title>
      <link>https://example.com/article-1</link>
      <pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Article Two</title>
      <link>https://example.com/article-2</link>
    </item>
  </channel>
</rss>`

	entries, err := ParseFeed([]byte(rss))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Title != "Article One" {
		t.Errorf("expected title 'Article One', got %q", entries[0].Title)
	}
	if entries[0].URL != "https://example.com/article-1" {
		t.Errorf("expected URL 'https://example.com/article-1', got %q", entries[0].URL)
	}
}

func TestParseFeed_Atom(t *testing.T) {
	atom := `<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Atom Entry</title>
    <link rel="alternate" href="https://example.com/atom-1"/>
    <updated>2024-01-01T00:00:00Z</updated>
  </entry>
</feed>`

	entries, err := ParseFeed([]byte(atom))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 atom entry, got %d", len(entries))
	}
	if entries[0].Title != "Atom Entry" {
		t.Errorf("expected title 'Atom Entry', got %q", entries[0].Title)
	}
	if entries[0].URL != "https://example.com/atom-1" {
		t.Errorf("expected Atom URL, got %q", entries[0].URL)
	}
}

func TestParseFeed_Invalid(t *testing.T) {
	_, err := ParseFeed([]byte("<notafeed/>"))
	if err == nil {
		t.Error("expected error for invalid feed, got nil")
	}
}

// ---- FilterURL tests ----

func TestFilterURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		baseHost string
		sameHost bool
		want     bool
	}{
		{"valid same host", "https://example.com/article", "example.com", true, true},
		{"different host when sameHost=true", "https://other.com/article", "example.com", true, false},
		{"different host when sameHost=false", "https://other.com/article", "example.com", false, true},
		{"skip pdf", "https://example.com/doc.pdf", "example.com", false, false},
		{"skip jpg", "https://example.com/img.jpg", "example.com", false, false},
		{"skip /tag/ path", "https://example.com/tag/golang", "example.com", false, false},
		{"skip /category/ path", "https://example.com/category/tech", "example.com", false, false},
		{"skip /wp-content/ path", "https://example.com/wp-content/upload.zip", "example.com", false, false},
		{"ftp scheme", "ftp://example.com/file", "example.com", false, false},
		{"invalid url", "://bad url", "example.com", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterURL(tt.rawURL, tt.baseHost, tt.sameHost)
			if got != tt.want {
				t.Errorf("FilterURL(%q, %q, %v) = %v, want %v", tt.rawURL, tt.baseHost, tt.sameHost, got, tt.want)
			}
		})
	}
}

// ---- URLStore tests ----

func TestURLStore_AddAndSeen(t *testing.T) {
	s := NewURLStore()

	if s.Seen("https://example.com") {
		t.Error("expected URL to not be seen initially")
	}

	added := s.Add("https://example.com")
	if !added {
		t.Error("expected Add to return true for new URL")
	}
	if !s.Seen("https://example.com") {
		t.Error("expected URL to be seen after Add")
	}

	// Adding same URL again should return false
	added2 := s.Add("https://example.com")
	if added2 {
		t.Error("expected Add to return false for duplicate URL")
	}
}

func TestURLStore_Normalisation(t *testing.T) {
	s := NewURLStore()
	// Trailing slash and fragment should be treated as same URL
	s.Add("https://example.com/page/")
	if !s.Seen("https://example.com/page") {
		t.Error("expected URL with trailing slash to match normalised URL")
	}
	s.Add("https://example.com/article#section1")
	if !s.Seen("https://example.com/article") {
		t.Error("expected URL with fragment to match normalised URL")
	}
}

func TestURLStore_Len(t *testing.T) {
	s := NewURLStore()
	if s.Len() != 0 {
		t.Errorf("expected Len() = 0, got %d", s.Len())
	}
	s.Add("https://example.com/a")
	s.Add("https://example.com/b")
	s.Add("https://example.com/a") // duplicate
	if s.Len() != 2 {
		t.Errorf("expected Len() = 2, got %d", s.Len())
	}
}

// ---- normaliseURL tests ----

func TestNormaliseURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/page/", "https://example.com/page"},
		{"https://example.com/page", "https://example.com/page"},
		{"https://example.com/page#anchor", "https://example.com/page"},
		{"https://example.com/page/?q=1#top", "https://example.com/page/?q=1"},
		{"://invalid", "://invalid"}, // invalid URLs returned as-is
	}
	for _, tt := range tests {
		got := normaliseURL(tt.input)
		if got != tt.want {
			t.Errorf("normaliseURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---- DiscoverFeeds tests ----

func TestDiscoverFeeds(t *testing.T) {
	html := `<html><head>
  <link rel="alternate" type="application/rss+xml" href="/feed.rss" title="RSS Feed">
  <link rel="alternate" type="application/atom+xml" href="https://example.com/atom.xml" title="Atom Feed">
  <link rel="stylesheet" href="/style.css">
</head><body></body></html>`

	feeds := DiscoverFeeds([]byte(html), "https://example.com")
	if len(feeds) != 2 {
		t.Errorf("expected 2 feeds, got %d: %v", len(feeds), feeds)
	}
	// Relative URL should be resolved
	found := false
	for _, f := range feeds {
		if f == "https://example.com/feed.rss" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected resolved relative feed URL, got: %v", feeds)
	}
}
