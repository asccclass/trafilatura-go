// Package langdetect provides text-based language detection as a fallback
// when HTML metadata (lang attribute, meta tags) does not specify the language.
// It wraps the whatlanggo library and returns standard ISO 639-1 codes.
package langdetect

import (
	"strings"
	"unicode/utf8"

	"github.com/abadojack/whatlanggo"
)

// minTextRunes is the minimum character count needed for reliable detection.
const minTextRunes = 20

// isoCode maps whatlanggo language names to ISO 639-1 two-letter codes.
// whatlanggo returns language names like "Mandarin" or "English"; we normalise
// them here. Only languages commonly found on the web are listed; others fall
// back to the whatlanggo three-letter code.
var isoCode = map[string]string{
	"Afrikaans":   "af",
	"Albanian":    "sq",
	"Arabic":      "ar",
	"Azerbaijani": "az",
	"Belarusian":  "be",
	"Bengali":     "bn",
	"Bosnian":     "bs",
	"Bulgarian":   "bg",
	"Catalan":     "ca",
	"Croatian":    "hr",
	"Czech":       "cs",
	"Danish":      "da",
	"Dutch":       "nl",
	"English":     "en",
	"Estonian":    "et",
	"Finnish":     "fi",
	"French":      "fr",
	"Georgian":    "ka",
	"German":      "de",
	"Greek":       "el",
	"Gujarati":    "gu",
	"Hebrew":      "he",
	"Hindi":       "hi",
	"Hungarian":   "hu",
	"Icelandic":   "is",
	"Indonesian":  "id",
	"Italian":     "it",
	"Japanese":    "ja",
	"Kannada":     "kn",
	"Kazakh":      "kk",
	"Korean":      "ko",
	"Latvian":     "lv",
	"Lithuanian":  "lt",
	"Macedonian":  "mk",
	"Malay":       "ms",
	"Mandarin":    "zh",
	"Marathi":     "mr",
	"Nepali":      "ne",
	"Norwegian":   "no",
	"Persian":     "fa",
	"Polish":      "pl",
	"Portuguese":  "pt",
	"Punjabi":     "pa",
	"Romanian":    "ro",
	"Russian":     "ru",
	"Serbian":     "sr",
	"Sinhala":     "si",
	"Slovak":      "sk",
	"Slovenian":   "sl",
	"Somali":      "so",
	"Spanish":     "es",
	"Swedish":     "sv",
	"Tagalog":     "tl",
	"Tamil":       "ta",
	"Telugu":      "te",
	"Thai":        "th",
	"Turkish":     "tr",
	"Ukrainian":   "uk",
	"Urdu":        "ur",
	"Uzbek":       "uz",
	"Vietnamese":  "vi",
	"Welsh":       "cy",
	"Yoruba":      "yo",
}

// Detect returns the ISO 639-1 language code of the given text, or an empty
// string if the text is too short or detection confidence is too low.
//
// Examples:
//
//	Detect("Hello, how are you? This is an English sentence.")  // "en"
//	Detect("這是一段中文文字，用來測試語言偵測功能。")                         // "zh"
//	Detect("Hi")                                                // ""  (too short)
func Detect(text string) string {
	lang, _ := DetectWithConfidence(text)
	return lang
}

// DetectWithConfidence returns the ISO 639-1 code and a confidence score (0-1).
// Useful when callers want to decide their own threshold.
// Returns ("", 0) when text is too short or confidence is low.
func DetectWithConfidence(text string) (lang string, confidence float64) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) < minTextRunes {
		return "", 0
	}

	info := whatlanggo.Detect(text)

	// Require minimum confidence to avoid false positives on short/ambiguous texts
	if info.Confidence < 0.5 {
		return "", info.Confidence
	}

	langName := info.Lang.String()
	if code, ok := isoCode[langName]; ok {
		return code, info.Confidence
	}

	// Fallback: LangToStringShort returns the ISO 639-1 code
	short := strings.ToLower(whatlanggo.LangToStringShort(info.Lang))
	if short == "" || short == "  " {
		return "", info.Confidence
	}
	return short, info.Confidence
}
