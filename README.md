# trafilatura-go

A complete **Go reimplementation** of [trafilatura](https://github.com/adbar/trafilatura) — a high-performance web content extractor and crawler.

## 📋 Original Project Analysis

### What trafilatura does

| Feature | Description |
|---|---|
| **Web crawling** | Sitemap (TXT/XML), feed (RSS/Atom/JSON), and smart BFS crawling with MaxDepth |
| **Downloading** | Polite HTTP with retries, redirect handling, gzip, concurrency; configurable `MaxBodySize` |
| **Headless browser** | Playwright-powered Chromium for SPA/React/Vue pages (`-headless` flag) |
| **Text extraction** | Readability-style density scoring + jusText-style paragraph filtering |
| **Metadata** | Title, Author, Date, Description, Site Name via OG / JSON-LD / Dublin Core |
| **Language detection** | Auto-detects 84 languages from text content (whatlanggo) when HTML `lang` is absent |
| **Structure** | Headings, lists, tables, blockquotes, code blocks, images |
| **Output formats** | TXT, Markdown, JSON, CSV, HTML (real `<table>`), XML (list/code/table) |
| **URL management** | Deduplication, filtering, same-host enforcement |
| **CLI** | Full command-line interface with rich flag set |

### ✨ 本 Go 版本的重點優化特色 (Key Enhancements)

- **高效能併發爬蟲 (Concurrent Worker Pool)**：將原本的爬蟲架構重構為 `Coordinator` + 固定數量 `Worker Pool` 架構，大幅提升大量網頁抓取的效能與穩定性。
- **高精準度內文提取 (Enhanced Extraction)**：升級 `densityBestNode` 評分演算法，加入父子節點分數聚合、內容容器白名單加權、以及更嚴格的常見雜訊黑名單懲罰機制與過濾。
- **高容錯 JSON-LD 解析 (Robust Metadata)**：加入支援 `@graph` 巢狀結構、嚴謹的弱型別檢查與 `recover` 防護機制，避免在遇到不合規範的網頁時發生 Runtime Panic。
- **完善的語料庫測試 (Golden Tests Corpus)**：建立涵蓋新聞、中日韓部落格、百科、SPA 空殼及高雜訊頁面等代表性場景的自動化測試套件，確保程式碼重構不引發回歸錯誤 (Regression)。
- **安全的 HTML 轉譯 (Secure Escaping)**：全面採用 Go 標準庫 `html.EscapeString` 處理字串轉譯，預防 XSS 與注入攻擊風險。
- **內建智慧語系 fallback 偵測 (Language Detection Fallback)**：當網頁缺乏 HTML `lang` 屬性時，自動整合由 `whatlanggo` 提供的高準確度 text-based 語系分析，免除錯標或漏標導致的擷取缺陷。
- **Unicode 安全的字串處理 (Unicode-Safe Truncation)**：在擷取內容摘要 (Excerpt) 時，改採智慧長度截斷引擎，優先尋找空白與標點符號的斷詞邊界，完美避開 Emoji 組合與連字號等複雜 Unicode 因粗暴切斷 rune 陣列所造成的亂碼現象。

---

## 🗂️ Project Structure

```
trafilatura-go/
├── trafilatura.go              # Public API (Fetch, Extract, Process)
├── go.mod
├── cmd/
│   └── trafilatura/
│       └── main.go            # CLI entry point
└── pkg/
    ├── settings/
    │   └── settings.go        # Config struct & defaults
    ├── fetch/
    │   └── fetch.go           # HTTP downloader (concurrent, polite)
    ├── headless/
    │   └── headless.go        # Playwright headless browser (SPA support)
    ├── langdetect/
    │   └── langdetect.go      # Text-based language detection (84 languages)
    ├── metadata/
    │   └── metadata.go        # OG / JSON-LD / Dublin Core extractor
    ├── extract/
    │   └── extract.go         # Core content extraction algorithm
    ├── output/
    │   └── output.go          # TXT / MD / JSON / CSV / HTML / XML formatters
    ├── spider/
    │   └── spider.go          # Crawler, sitemap & feed parser, URL store
    └── utils/
        └── utils.go           # HTML helpers, text cleaning, scoring
```

---

## 🚀 Installation

```bash
go install github.com/trafilatura-go/cmd/trafilatura@latest
```

Or build from source:

```bash
git clone https://github.com/your-org/trafilatura-go
cd trafilatura-go
go build ./cmd/trafilatura
```

---

## Test

```bash
go test ./...
```

## 📖 CLI Usage

```bash
# Extract a single article (plain text)
trafilatura https://example.com/article

# Output as JSON
trafilatura -format json https://example.com/article

# Output as Markdown
trafilatura -format markdown https://example.com/article

# Include tables and comments
trafilatura -tables -comments https://example.com/article

# ★ Render SPA/React/Vue pages with headless Chromium
trafilatura -headless https://react.dev/

# ★ Headless + JSON output
trafilatura -headless -format json https://vue-app.example.com

# Read from file
trafilatura -f page.html -url https://example.com/page

# Read from stdin
cat page.html | trafilatura -stdin -url https://example.com/page

# Write to file
trafilatura -format json -o result.json https://example.com/article

# Crawl a website (BFS)
trafilatura -crawl -max-urls 100 https://example.com

# Use sitemap
trafilatura -sitemap https://example.com

# Favour precision (less noise, potentially missing some content)
trafilatura -precision https://example.com/article

# Favour recall (more content, potentially some noise)
trafilatura -recall https://example.com/article
```

### All flags

| Flag | Default | Description |
|---|---|---|
| `-format` | `txt` | Output format: `txt`, `markdown`, `json`, `csv`, `html`, `xml` |
| `-headless` | false | Use Playwright headless Chromium to render SPA/JS pages |
| `-comments` | false | Include comment sections |
| `-tables` | true | Include HTML tables |
| `-images` | false | Include images (as Markdown syntax) |
| `-links` | false | Preserve hyperlinks |
| `-no-fallback` | false | Disable fallback extraction |
| `-precision` | false | Favour precision over recall |
| `-recall` | false | Favour recall over precision |
| `-pretty` | false | Pretty-print JSON output |
| `-timeout` | 30s | HTTP request timeout |
| `-user-agent` | `trafilatura-go/1.0` | HTTP User-Agent |
| `-delay` | 500ms | Delay between crawl requests |
| `-concurrency` | 4 | Concurrent download workers |
| `-max-urls` | 1000 | Maximum URLs to crawl |
| `-crawl` | false | Crawl mode (BFS from seed) |
| `-sitemap` | false | Discover and use sitemap |
| `-same-host` | true | Restrict crawl to same hostname |
| `-stdin` | false | Read HTML from stdin |
| `-url` | — | Source URL (for stdin/file mode metadata) |
| `-f` | — | Read HTML from a file |
| `-o` | — | Write output to a file |
| `-version` | false | Print version |

---

## 📦 Go API

```go
import (
    trafilatura "github.com/trafilatura-go"
    "github.com/trafilatura-go/pkg/settings"
)

cfg := settings.DefaultConfig()
cfg.Format = settings.FormatJSON
cfg.PrettyPrint = true
cfg.IncludeTables = true

// Fetch + extract in one call
text, err := trafilatura.Process("https://example.com/article", cfg)

// Or step by step:
data, err := trafilatura.Fetch("https://example.com/article", cfg)
text, err  := trafilatura.Extract(data, "https://example.com/article", cfg)

// Metadata only
meta, err  := trafilatura.ExtractMetadata(data, "https://example.com/article")
fmt.Println(meta.Title, meta.Author, meta.Date)

// ★ Headless mode (SPA / React / Vue)
cfg.Headless = true
text, err = trafilatura.Process("https://react.dev/", cfg)
```

---

## 🌐 Headless Browser (SPA Support)

For JavaScript-heavy pages that require rendering before extraction:

```bash
# Install Chromium once (first time only)
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium

# Extract SPA page via CLI
trafilatura -headless -format json https://react.dev/

# Crawl SPA site
trafilatura -headless -crawl -max-urls 20 https://spa-site.example.com
```

## 🔤 Automatic Language Detection

Language is now detected from multiple sources in priority order:

1. `og:locale` (Open Graph)
2. `dc.language` (Dublin Core)
3. `<html lang>` attribute
4. `http-equiv content-language`
5. **Text-based detection** via [whatlanggo](https://github.com/abadojack/whatlanggo) (84 languages)

The fallback text-based detection activates automatically when all metadata sources are empty — no configuration needed.



---

## 🔑 Key Design Decisions (Go vs Python)

| Aspect | Python (trafilatura) | Go (trafilatura-go) |
|---|---|---|
| HTML parsing | `lxml` (C extension) | `golang.org/x/net/html` |
| Concurrency | `concurrent.futures` | Goroutines + channels |
| Date extraction | `htmldate` library | Built-in multi-format parser |
| Language detection | `py3langid` optional | Placeholder (pluggable) |
| Content scoring | jusText + readability | Unified density + signal scoring |
| Output | Multiple libs | Pure stdlib + encoding packages |

---

## 🏗️ Extraction Algorithm

1. **Pre-clean** — Remove `<script>`, `<style>`, `<nav>`, `<aside>`, `<footer>`, class/id noise
2. **Main element detection** — `<main>`, `<article>`, `role="main"`, class heuristics, density scoring
3. **Block extraction** — Recursive DOM walk → typed text blocks (paragraph, heading, list, quote, code, table, image)
4. **Scoring** — Content signal (word count × sentence density × link density) per block
5. **Filtering** — Threshold-based removal of low-signal blocks
6. **Output formatting** — Render to TXT / Markdown / JSON / CSV / HTML / XML

---

## 📜 License

Apache 2.0 — same as the original trafilatura project.

---
