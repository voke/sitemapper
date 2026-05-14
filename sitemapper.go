package sitemapper

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/gocolly/colly"
)

const UserAgent = "Mozilla/5.0 (compatible; SiteMapper/1.0)"

type CallbackHandler func(string)

type sitemapURLSet struct {
	URLs []struct {
		Loc string `xml:"loc"`
	} `xml:"url"`
}

type sitemapIndex struct {
	Sitemaps []struct {
		Loc string `xml:"loc"`
	} `xml:"sitemap"`
}

func parseGzippedSitemap(data []byte, c *colly.Collector, handler CallbackHandler) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		fmt.Println("Error creating gzip reader:", err)
		return
	}
	defer gr.Close()

	xmlData, err := io.ReadAll(gr)
	if err != nil {
		fmt.Println("Error decompressing gzip:", err)
		return
	}

	var us sitemapURLSet
	if xml.Unmarshal(xmlData, &us) == nil && len(us.URLs) > 0 {
		for _, u := range us.URLs {
			handler(strings.TrimSpace(u.Loc))
		}
		return
	}

	var si sitemapIndex
	if xml.Unmarshal(xmlData, &si) == nil {
		for _, s := range si.Sitemaps {
			fmt.Println("Found sitemap:", s.Loc)
			c.Visit(s.Loc)
		}
	}
}

func CrawlSitemap(domain string, sitemapURL string, handler CallbackHandler) {

	fmt.Println("Begin crawling sitemap at:", sitemapURL)

	// Create a Collector
	c := colly.NewCollector(colly.UserAgent(UserAgent), colly.MaxBodySize(50*1024*1024))

	// Handle gzip-compressed sitemaps (.xml.gz)
	c.OnResponse(func(r *colly.Response) {
		if strings.HasSuffix(r.Request.URL.Path, ".gz") {
			parseGzippedSitemap(r.Body, c, handler)
		}
	})

	// Create a callback on the XPath query searching for the URLs
	c.OnXML("//urlset/url/loc", func(e *colly.XMLElement) {
		handler(strings.TrimSpace(e.Text))
	})

	// Enqueue additional crawls if sitemapindex
	c.OnXML("//sitemapindex/sitemap/loc", func(e *colly.XMLElement) {
		fmt.Println("Found sitemap:", e.Text)
		c.Visit(e.Text)
	})

	// Start the collector
	c.Visit(sitemapURL)

}
