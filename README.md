
# 🔗📙 Sitemapper

A sitemap crawler/parser written in go with support for `sitemapindex` and `urlset`.

## Examples

### Get sitemap tree

Fetches the sitemap hierarchy and counts URLs at each level — without processing individual URLs.

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

### Crawl and process URLs

Traverses the full sitemap and invokes a callback for every URL found.

```go
package main

import (
	"fmt"

	sitemapper "github.com/voke/sitemapper"
)

func main() {
	total := 0

	handler := func(url string) {
		total++
		fmt.Printf("Process URL: %s\n", url)
	}

	sitemapper.CrawlSitemap("www.example.com", "https://www.example.com/sitemap.xml", handler)

	fmt.Println("Total URLs:", total)
}
```