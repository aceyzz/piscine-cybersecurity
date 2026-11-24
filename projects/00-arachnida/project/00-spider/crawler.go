package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"golang.org/x/net/html"
)

const (
	requestTimeout = 15 * time.Second
	maxPageSize    = 10 << 20
	userAgent      = "Mozilla/5.0 (compatible; spider/1.0)"
)

type target struct {
	url   *url.URL
	depth int
}

type crawler struct {
	cfg        config
	client     *http.Client
	seenPages  map[string]bool
	seenImages map[string]bool
	saved      int
}

func newCrawler(cfg config) *crawler {
	return &crawler{
		cfg:        cfg,
		client:     &http.Client{Timeout: requestTimeout},
		seenPages:  map[string]bool{cfg.start.String(): true},
		seenImages: map[string]bool{},
	}
}

func (c *crawler) run() int {
	queue := []target{{url: c.cfg.start, depth: 0}}
	for len(queue) > 0 {
		current := queue[0]
		queue = append(queue[1:], c.visit(current)...)
	}
	return c.saved
}

func (c *crawler) visit(t target) []target {
	images, pages, err := c.scrape(t.url)
	if err != nil {
		warn(err)
		return nil
	}
	for _, img := range images {
		c.saveImage(img)
	}
	if t.depth >= c.cfg.depth {
		return nil
	}
	return c.nextTargets(pages, t.depth+1)
}

func (c *crawler) nextTargets(pages []*url.URL, depth int) []target {
	var next []target
	for _, p := range pages {
		key := p.String()
		if c.seenPages[key] || !sameHost(p, c.cfg.start) {
			continue
		}
		c.seenPages[key] = true
		next = append(next, target{url: p, depth: depth})
	}
	return next
}

func (c *crawler) scrape(page *url.URL) ([]*url.URL, []*url.URL, error) {
	resp, err := c.fetch(page)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if !hasContentType(resp, "text/html") {
		return nil, nil, nil
	}
	images, pages := extractLinks(io.LimitReader(resp.Body, maxPageSize), resp.Request.URL)
	return images, pages, nil
}

func (c *crawler) saveImage(u *url.URL) {
	key := u.String()
	if c.seenImages[key] {
		return
	}
	c.seenImages[key] = true
	dest, err := c.download(u)
	if err != nil {
		warn(err)
		return
	}
	c.saved++
	fmt.Println(dest)
}

func (c *crawler) download(u *url.URL) (string, error) {
	resp, err := c.fetch(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if !hasContentType(resp, "image/") {
		return "", fmt.Errorf("%s: not an image", u)
	}
	dest := uniquePath(c.cfg.output, path.Base(u.Path))
	return dest, writeFile(dest, resp.Body)
}

func (c *crawler) fetch(u *url.URL) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %s", u, resp.Status)
	}
	return resp, nil
}

func extractLinks(r io.Reader, base *url.URL) ([]*url.URL, []*url.URL) {
	var images, pages []*url.URL
	z := html.NewTokenizer(r)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return images, pages
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		tok := z.Token()
		link := resolveURL(base, linkAttr(tok))
		switch {
		case link == nil:
		case isImage(link):
			images = append(images, link)
		case tok.Data == "a":
			pages = append(pages, link)
		}
	}
}

func linkAttr(tok html.Token) string {
	switch tok.Data {
	case "img":
		return attrValue(tok, "src")
	case "a":
		return attrValue(tok, "href")
	}
	return ""
}

func attrValue(tok html.Token, key string) string {
	for _, a := range tok.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
