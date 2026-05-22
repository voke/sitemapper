package sitemapper

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/gobwas/glob"
	"github.com/gocolly/colly/v2"
)

const UserAgent = "Mozilla/5.0 (compatible; SiteMapper/1.0)"

type CallbackHandler func(string)

type Options struct {
	SitemapWhitelist string
	URLWhitelist     string
	BaseURL          string
}

// Stats holds counters collected during a CrawlSitemap run.
type Stats struct {
	URLsProcessed     int `json:"urls_processed"`
	URLsIgnored       int `json:"urls_ignored"`
	SitemapsProcessed int `json:"sitemaps_processed"`
	SitemapsIgnored   int `json:"sitemaps_ignored"`
}

type compiledOptions struct {
	sitemapWhitelist glob.Glob
	urlWhitelist     glob.Glob
	baseURL          *url.URL
}

func compileOptions(opts Options) (compiledOptions, error) {
	var co compiledOptions
	if opts.SitemapWhitelist != "" {
		g, err := glob.Compile(opts.SitemapWhitelist)
		if err != nil {
			return co, fmt.Errorf("invalid SitemapWhitelist pattern: %w", err)
		}
		co.sitemapWhitelist = g
	}
	if opts.URLWhitelist != "" {
		g, err := glob.Compile(opts.URLWhitelist)
		if err != nil {
			return co, fmt.Errorf("invalid URLWhitelist pattern: %w", err)
		}
		co.urlWhitelist = g
	}
	if opts.BaseURL != "" {
		base, err := url.Parse(opts.BaseURL)
		if err != nil {
			return co, fmt.Errorf("invalid BaseURL: %w", err)
		}
		co.baseURL = base
	}
	return co, nil
}

func (co compiledOptions) resolveURL(loc string) string {
	loc = strings.TrimSpace(loc)
	if co.baseURL == nil {
		return loc
	}
	ref, err := url.Parse(loc)
	if err != nil || ref.IsAbs() {
		return loc
	}
	return co.baseURL.ResolveReference(ref).String()
}

func (co compiledOptions) matchesSitemapWhitelist(loc string) bool {
	if co.sitemapWhitelist == nil {
		return true
	}
	return co.sitemapWhitelist.Match(strings.TrimSpace(loc))
}

func (co compiledOptions) matchesURLWhitelist(url string) bool {
	if co.urlWhitelist == nil {
		return true
	}
	return co.urlWhitelist.Match(url)
}

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

// SitemapNode represents a sitemap entry in the tree structure.
type SitemapNode struct {
	Sitemap   string        `json:"sitemap"`
	URLs      int           `json:"urls"`
	TotalURLs int           `json:"total_urls"`
	Children  []SitemapNode `json:"children,omitempty"`
}

func fetchSitemapBytes(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func decompressIfGzipped(url string, data []byte) ([]byte, error) {
	if strings.HasSuffix(strings.ToLower(url), ".gz") {
		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		return io.ReadAll(gr)
	}
	return data, nil
}

func buildSitemapNode(sitemapURL string, co compiledOptions) (SitemapNode, error) {
	node := SitemapNode{Sitemap: sitemapURL}

	raw, err := fetchSitemapBytes(sitemapURL)
	if err != nil {
		return node, fmt.Errorf("fetching %s: %w", sitemapURL, err)
	}

	xmlData, err := decompressIfGzipped(sitemapURL, raw)
	if err != nil {
		return node, fmt.Errorf("decompressing %s: %w", sitemapURL, err)
	}

	var si sitemapIndex
	if xml.Unmarshal(xmlData, &si) == nil && len(si.Sitemaps) > 0 {
		for _, s := range si.Sitemaps {
			loc := co.resolveURL(s.Loc)
			if !co.matchesSitemapWhitelist(loc) {
				continue
			}
			child, err := buildSitemapNode(loc, co)
			if err != nil {
				return node, err
			}
			node.TotalURLs += child.TotalURLs
			node.Children = append(node.Children, child)
		}
		return node, nil
	}

	var us sitemapURLSet
	if xml.Unmarshal(xmlData, &us) == nil {
		node.URLs = len(us.URLs)
		node.TotalURLs = node.URLs
	}

	return node, nil
}

// GetSitemapTree fetches and builds a tree structure of the sitemap,
// counting URLs at each level without invoking a callback per URL.
// The full sitemap hierarchy is always traversed; no filtering is applied.
func GetSitemapTree(sitemapURL string) ([]SitemapNode, error) {
	node, err := buildSitemapNode(sitemapURL, compiledOptions{})
	if err != nil {
		return nil, err
	}
	return []SitemapNode{node}, nil
}

func parseGzippedSitemap(data []byte, c *colly.Collector, handler CallbackHandler, co compiledOptions, processed, urlsIgnored, sitemapsProcessed, sitemapsIgnored *int64) {
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
			loc := co.resolveURL(u.Loc)
			if !co.matchesURLWhitelist(loc) {
				atomic.AddInt64(urlsIgnored, 1)
				continue
			}
			atomic.AddInt64(processed, 1)
			handler(loc)
		}
		return
	}

	var si sitemapIndex
	if xml.Unmarshal(xmlData, &si) == nil {
		for _, s := range si.Sitemaps {
			if !co.matchesSitemapWhitelist(s.Loc) {
				atomic.AddInt64(sitemapsIgnored, 1)
				continue
			}
			loc := co.resolveURL(s.Loc)
			atomic.AddInt64(sitemapsProcessed, 1)
			fmt.Println("Found sitemap:", loc)
			c.Visit(loc)
		}
	}
}

// CrawlSitemap fetches and crawls a sitemap, calling handler for each URL found.
// An optional Options value can be passed to filter which nested sitemaps or
// individual URLs are followed. Returns Stats with counts of processed and
// ignored URLs.
func CrawlSitemap(sitemapURL string, handler CallbackHandler, opts ...Options) (Stats, error) {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}

	co, err := compileOptions(o)
	if err != nil {
		return Stats{}, err
	}

	var processed, urlsIgnored, sitemapsProcessed, sitemapsIgnored int64

	fmt.Println("Begin crawling sitemap at:", sitemapURL)

	// Create a Collector
	c := colly.NewCollector(colly.UserAgent(UserAgent), colly.MaxBodySize(50*1024*1024))

	// Handle gzip-compressed sitemaps (.xml.gz)
	c.OnResponse(func(r *colly.Response) {
		if strings.HasSuffix(r.Request.URL.Path, ".gz") {
			parseGzippedSitemap(r.Body, c, handler, co, &processed, &urlsIgnored, &sitemapsProcessed, &sitemapsIgnored)
		}
	})

	// Create a callback on the XPath query searching for the URLs
	c.OnXML("//urlset/url/loc", func(e *colly.XMLElement) {
		loc := co.resolveURL(e.Text)
		if !co.matchesURLWhitelist(loc) {
			atomic.AddInt64(&urlsIgnored, 1)
			return
		}
		atomic.AddInt64(&processed, 1)
		handler(loc)
	})

	// Enqueue additional crawls if sitemapindex
	c.OnXML("//sitemapindex/sitemap/loc", func(e *colly.XMLElement) {
		loc := co.resolveURL(e.Text)
		if !co.matchesSitemapWhitelist(loc) {
			atomic.AddInt64(&sitemapsIgnored, 1)
			return
		}
		atomic.AddInt64(&sitemapsProcessed, 1)
		fmt.Println("Found sitemap:", loc)
		c.Visit(loc)
	})

	// Start the collector
	c.Visit(sitemapURL)

	return Stats{
		URLsProcessed:     int(atomic.LoadInt64(&processed)),
		URLsIgnored:       int(atomic.LoadInt64(&urlsIgnored)),
		SitemapsProcessed: int(atomic.LoadInt64(&sitemapsProcessed)),
		SitemapsIgnored:   int(atomic.LoadInt64(&sitemapsIgnored)),
	}, nil
}
