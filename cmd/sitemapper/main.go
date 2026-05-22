package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	sitemapper "github.com/voke/sitemapper"
)

func main() {
	sitemapWhitelist := flag.String("sitemap-whitelist", "", "Glob pattern to filter nested sitemaps (e.g. \"*-product.xml\")")
	urlWhitelist := flag.String("url-whitelist", "", "Glob pattern to filter URLs passed to output (e.g. \"*/products/*\")")
	quiet := flag.Bool("quiet", false, "Suppress URL output; only print stats")
	jsonStats := flag.Bool("json-stats", false, "Print stats as JSON instead of plain text")
	tree := flag.Bool("tree", false, "Print sitemap hierarchy as JSON instead of crawling URLs")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sitemapper [flags] <sitemap-url>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	target := flag.Arg(0)

	if *tree {
		nodes, err := sitemapper.GetSitemapTree(target)
		if err != nil {
			log.Fatal(err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(nodes); err != nil {
			log.Fatal(err)
		}
		return
	}

	opts := sitemapper.Options{
		SitemapWhitelist: *sitemapWhitelist,
		URLWhitelist:     *urlWhitelist,
	}

	handler := func(u string) {
		if !*quiet {
			fmt.Println(u)
		}
	}

	stats, err := sitemapper.CrawlSitemap(target, handler, opts)
	if err != nil {
		log.Fatal(err)
	}

	if *jsonStats {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(stats); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("---")
		fmt.Printf("URLs processed:      %d\n", stats.URLsProcessed)
		fmt.Printf("URLs ignored:        %d\n", stats.URLsIgnored)
		fmt.Printf("Sitemaps processed:  %d\n", stats.SitemapsProcessed)
		fmt.Printf("Sitemaps ignored:    %d\n", stats.SitemapsIgnored)
	}
}
