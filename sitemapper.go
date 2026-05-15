package sitemapper

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
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

func buildSitemapNode(sitemapURL string) (SitemapNode, error) {
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
			child, err := buildSitemapNode(strings.TrimSpace(s.Loc))
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
func GetSitemapTree(sitemapURL string) ([]SitemapNode, error) {
	node, err := buildSitemapNode(sitemapURL)
	if err != nil {
		return nil, err
	}
	return []SitemapNode{node}, nil
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
