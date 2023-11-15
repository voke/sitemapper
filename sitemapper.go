package sitemapper

import (
	"strings"

	"github.com/gocolly/colly"
)

const UserAgent = "Mozilla/5.0 (compatible; SiteMapper/1.0)"

type CallbackHandler func(string)

func CrawlSitemap(domain string, sitemapURL string, handler CallbackHandler) {

	// Create a Collector
	c := colly.NewCollector(colly.UserAgent(UserAgent))

	// Create a callback on the XPath query searching for the URLs
	c.OnXML("//urlset/url/loc", func(e *colly.XMLElement) {
		handler(strings.TrimSpace(e.Text))
	})

	// Enqueue additional crawls if sitemapindex
	c.OnXML("//sitemapindex/sitemap/loc", func(e *colly.XMLElement) {
		c.Visit(e.Text)
	})

	// Start the collector
	c.Visit(sitemapURL)

}
