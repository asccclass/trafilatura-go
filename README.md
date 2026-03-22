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

## 🔍 專案分析與改進建議 (Project Analysis & Improvements)

這是本 Go 版本專案的原始碼分析，我們發現了幾個架構上的優點，但也存在一些開發上的技術債與可以改進的方向：

### 🚨 目前存在的問題 (Current Issues)

1. **缺乏自動化測試 (Missing Tests)**
   - **問題**：整個專案資料夾內（包含 `pkg/extract`、`pkg/metadata`、`pkg/fetch` 等核心演算法模組）目前完全沒有任何 `_test.go` 單元測試檔案。
   - **影響**：網頁擷取邏輯高度依賴啟發式規則（Heuristics），在面對各式各樣混亂的 HTML 結構時容易產生非預期錯誤。沒有 Regression Tests 與邊界值測試，將難以確保程式碼重構或新增功能時不會破壞原有的提取準確度。
2. **多語系偵測過於簡化 (Language Detection Flaws)**
   - **問題**：在 `pkg/metadata/metadata.go` 中，對文章語言的捕捉僅依賴 `<html lang="xx">` 屬性。
   - **影響**：原版 Python Trafilatura 能夠選擇使用 `fasttext` 或 `cld2` 等強大的機器學習模組進行真實語系偵測。目前 Go 版本的實作若遇到未標示或錯誤標示 HTML `lang` 的網站，將無法得知文章真實的語言內容。
3. **字串清理與 HTML 轉譯存在安全盲區 (String Escaping Quirks)**
   - **問題**：~~在 `pkg/output/output.go` 裡的 `htmlEscape` 函式採用開發者手動實作的字串替換 (`strings.ReplaceAll(...)`)~~ (✅ 已替換為標準庫 `html.EscapeString`)；另外在 `formatJSON` 中擷取 Excerpt 首段時，直接使用 `runes[:200]` 切斷陣列。
   - **影響**：~~手動設計的 HTML Escape 容易遺漏邊界情況（例如未處理單引號或其他特殊 Entities），建議改用 Go 標準庫 `html.EscapeString` 來避免 XSS 與注入風險。~~直接使用 `runes[:200]` 截斷字串在遇到複雜的 Unicode 組合字元（例如 Emoji 或特定語言的連字組合）時，有可能會造成字元截斷錯誤或呈現亂碼。
4. **爬蟲與下載的併發模型設計 (Concurrency Model in Fetch)**
   - **問題**：`pkg/fetch/fetch.go` 中的 `FetchMany` 雖然使用了 Goroutine + Channel 所構成的 Semaphore 來控制最大併發數量，但其迴圈仍會為「每一個給定的 URL」生成一個獨立的 Goroutine，然後再讓其在 Semaphore 上排隊等待 (Block)。
   - **影響**：如果使用者傳入了十萬個甚至百萬個 URL 列表，短時間內將導致記憶體中產生大量處於休眠狀態的 Goroutines，引發不必要的記憶體與排程開銷。

### 💡 後續需要改進的地方 (Areas for Improvement)

1. **引入完善的測試框架與語料庫 (Test Corpus)**：
   - 強烈建議使用 Go 內建的 `testing` 工具來為 `extract` 與 `metadata` 邏輯撰寫詳盡的單元測試。
   - 建立「語料庫 (Corpus)」測試資料夾，抓取數個目標網站（例如新聞網站、部落格、維基百科）作為 Golden Tests 對照組，並在 CI 流程中持續比對輸出品質。
2. **重構併發模型為固定 Worker Pool**：
   - 替換 `pkg/fetch` 與 `pkg/spider` 中的併發實作。建立固定數量的 Worker Pool 模式，並利用單一 Task Channel 來分發網址任務，如此可大幅降低系統資源開銷並提高爬取穩定性。或考慮直接整合如 `gocolly/colly` 等成熟的爬蟲框架作為 Fetch 層。
3. **提升內文提取 (Extraction) 的精準度與容錯能力**：
   - 深入優化 `pkg/extract/extract.go` 中的 `densityBestNode` 評分演算法。現在的版本過於單純。
   - 針對單頁應用程式 (SPA) 或動態載入的元件增加例外處理。建立去雜訊（Boilerplate Removal）的標籤、常見導覽列 / 側邊欄 Class 名稱的黑/白名單判斷系統。
4. **優化標準庫的使用與增強 JSON-LD 解析容錯 (✅ 已解決)**：
   - 全面將自製的 `htmlEscape` 替換為 `html` 標準庫的 `EscapeString` 防護。
   - `pkg/metadata/metadata.go` 的 `extractJSONLD` 已加入更嚴謹且深層的 Type Assertion 檢查（新增支援 `@graph` 巢狀結構、安全型別轉換防護、以及 `recover` 機制），避免在不合規範結構的網頁上觸發 Runtime Panic。
