package langdetect

import (
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string // expected ISO 639-1 code, or "" if unknown
	}{
		{
			name: "english",
			text: "The quick brown fox jumps over the lazy dog. " +
				"This is a standard English sentence used for testing purposes. " +
				"Language detection should return 'en' for this input.",
			want: "en",
		},
		{
			name: "chinese",
			// Using string concatenation to avoid any encoding issues
			text: "\u9019\u662f\u4e00\u6bb5\u4e2d\u6587\u6587\u5b57\uff0c\u7528\u4f86\u6e2c\u8a66\u8a9e\u8a00\u5075\u6e2c\u529f\u80fd\u3002" +
				"\u672c\u5de5\u5177\u652f\u63f4\u591a\u7a2e\u8a9e\u8a00\u7684\u81ea\u52d5\u5075\u6e2c\uff0c\u5305\u542b\u7e41\u9ad4\u4e2d\u6587\u8207\u7c21\u9ad4\u4e2d\u6587\u3002" +
				"\u8acb\u6e2c\u8a66\u81ea\u52d5\u8a9e\u8a00\u5075\u6e2c\u529f\u80fd\u662f\u5426\u6b63\u5e38\u904b\u4f5c\u3002",
			want: "zh",
		},
		{
			name: "japanese",
			text: "\u3053\u308c\u306f\u65e5\u672c\u8a9e\u306e\u30c6\u30ad\u30b9\u30c8\u3067\u3059\u3002" +
				"\u8a00\u8a9e\u691c\u51fa\u6a5f\u80fd\u306e\u30c6\u30b9\u30c8\u306b\u4f7f\u7528\u3055\u308c\u307e\u3059\u3002" +
				"\u8907\u6570\u306e\u8a00\u8a9e\u3092\u81ea\u52d5\u7684\u306b\u691c\u51fa\u3067\u304d\u307e\u3059\u3002",
			want: "ja",
		},
		{
			name: "german",
			text: "Dies ist ein deutscher Text. " +
				"Er wird verwendet, um die Spracherkennungsfunktion zu testen. " +
				"Die Erkennung mehrerer Sprachen ist moeglich.",
			want: "de",
		},
		{
			name: "too short returns empty",
			text: "Hi",
			want: "",
		},
		{
			name: "empty string returns empty",
			text: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.text)
			if tt.want == "" {
				if got != "" {
					t.Errorf("Detect(%q) = %q, want empty string", tt.text, got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("Detect(%q...) = %q, want %q", tt.text[:min(20, len(tt.text))], got, tt.want)
			}
		})
	}
}

func TestDetectWithConfidence(t *testing.T) {
	// Long English text that should pass the minTextRunes threshold easily
	text := "The quick brown fox jumps over the lazy dog. " +
		"Testing language detection confidence scoring with a long English sentence. " +
		"This should be detected reliably as English."
	lang, conf := DetectWithConfidence(text)
	if lang != "en" {
		t.Errorf("expected language 'en', got %q (confidence: %.3f)", lang, conf)
	}
	if conf <= 0 || conf > 1 {
		t.Errorf("expected confidence in (0, 1], got %f", conf)
	}
}

func TestDetect_LowConfidenceReturnsEmpty(t *testing.T) {
	// A very short text that shouldn't have high confidence
	text := "aa"
	result := Detect(text)
	if result != "" {
		t.Errorf("expected empty string for very short text, got %q", result)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
