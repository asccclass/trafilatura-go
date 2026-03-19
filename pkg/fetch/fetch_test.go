package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/trafilatura-go/pkg/settings"
)

func TestFetchMany(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, world!"))
	}))
	defer server.Close()

	cfg := settings.DefaultConfig()
	cfg.Concurrency = 2
	cfg.Delay = 10 * time.Millisecond
	f := New(cfg)

	urls := []string{server.URL, server.URL, server.URL}
	results := f.FetchMany(urls)

	count := 0
	for res := range results {
		if res.Err != nil {
			t.Errorf("unexpected error: %v", res.Err)
		}
		if res.Response.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK")
		}
		count++
	}

	if count != len(urls) {
		t.Errorf("expected %d results, got %d", len(urls), count)
	}
}
