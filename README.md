
# 🔗📙 Sitemapper

A sitemap crawler/parser written in Go with support for `sitemapindex` and `urlset`.

## Installation

```bash
go get github.com/voke/sitemapper
```

## Examples

### Crawl and process all URLs

```go
package main

import (
	"fmt"
	"log"

	sitemapper "github.com/voke/sitemapper"
)

func main() {
	handler := func(url string) {
		fmt.Println(url)
	}

	stats, err := sitemapper.CrawlSitemap("https://www.example.com/sitemap.xml", handler)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("URLs processed: %d\n", stats.URLsProcessed)
}
```

### Crawl with whitelist filters

Only follow sitemaps matching `*-product.xml` and only process URLs containing `/products/`.

```go
stats, err := sitemapper.CrawlSitemap(
    "https://www.example.com/sitemap.xml",
    handler,
    sitemapper.Options{
        SitemapWhitelist: "*-product.xml",
        URLWhitelist:     "*/products/*",
    },
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Processed:         %d\n", stats.URLsProcessed)
fmt.Printf("URLs ignored:      %d\n", stats.URLsIgnored)
fmt.Printf("Sitemaps followed: %d\n", stats.SitemapsProcessed)
fmt.Printf("Sitemaps ignored:  %d\n", stats.SitemapsIgnored)
```

### Get sitemap tree

Fetches the sitemap hierarchy and counts URLs at each level.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	sitemapper "github.com/voke/sitemapper"
)

func main() {
	tree, err := sitemapper.GetSitemapTree("https://www.example.com/sitemap.xml")
	if err != nil {
		log.Fatal(err)
	}

	out, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(out))
}
```

Example output:

```json
[
  {
    "sitemap": "https://www.example.com/sitemap.xml",
    "urls": 0,
    "total_urls": 8500,
    "children": [
      {
        "sitemap": "https://www.example.com/sitemap/products.xml.gz",
        "urls": 5000,
        "total_urls": 5000
      },
      {
        "sitemap": "https://www.example.com/sitemap/categories.xml.gz",
        "urls": 3500,
        "total_urls": 3500
      }
    ]
  }
]
```

---

## CLI

A command-line tool is included under `cmd/sitemapper`.

```bash
go install github.com/voke/sitemapper/cmd/sitemapper@latest
```

```
Usage: sitemapper [flags] <sitemap-url>
```

| Flag | Type | Description |
|---|---|---|
| `-tree` | bool | Print sitemap hierarchy as JSON instead of crawling URLs |
| `-sitemap-whitelist` | string | Glob pattern to filter nested sitemaps (e.g. `*-product.xml`) |
| `-url-whitelist` | string | Glob pattern to filter URLs passed to output (e.g. `*/products/*`) |
| `-quiet` | bool | Suppress URL output; only print stats |
| `-json-stats` | bool | Print stats as JSON instead of plain text |

**Examples:**

```bash
# Print sitemap hierarchy as JSON
sitemapper -tree https://www.example.com/sitemap.xml

# Crawl all URLs
sitemapper https://www.example.com/sitemap.xml

# Only product sitemaps and product URLs, stats as JSON
sitemapper \
  -sitemap-whitelist "*-product.xml" \
  -url-whitelist "*/products/*" \
  -json-stats \
  https://www.example.com/sitemap.xml

# Quiet mode — just the stats
sitemapper -quiet https://www.example.com/sitemap.xml
```