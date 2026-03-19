package extract

import (
	"strings"
	"testing"

	"example.com/trafilatura-go/pkg/settings"
)

func TestExtract(t *testing.T) {
	htmlStr := `<html><body>
	<header>Site Header Noise</header>
	<main>
		<h1>Test Title</h1>
		<p>This is a test paragraph that has enough words to be scored as content. It needs to be relatively long to pass the threshold.</p>
		<p>Another test paragraph with more content signal to be extracted properly. Let's add more sentences to increase the word count and punctuation density.</p>
	</main>
	<footer>Site Footer Noise</footer>
	</body></html>`

	cfg := settings.DefaultConfig()
	cfg.MinExtractedSize = 10
	cfg.FavorRecall = true
	ex := New(cfg)
	doc, err := ex.FromBytes([]byte(htmlStr))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := doc.ToText()
	if strings.Contains(text, "Site Header") {
		t.Errorf("expected header to be omitted")
	}
	if strings.Contains(text, "Site Footer") {
		t.Errorf("expected footer to be omitted")
	}
	if !strings.Contains(text, "Test Title") {
		t.Errorf("expected title to be present")
	}
	if !strings.Contains(text, "Another test paragraph") {
		t.Errorf("expected paragraph to be present, got: %s", text)
	}
}
