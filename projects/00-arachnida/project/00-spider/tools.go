package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
}

func warn(err error) {
	fmt.Fprintln(os.Stderr, "spider:", err)
}

func isFlagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func parseHTTPURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || !isHTTP(u) || u.Host == "" {
		return nil, fmt.Errorf("invalid URL: %s", raw)
	}
	u.Fragment = ""
	return u, nil
}

func resolveURL(base *url.URL, raw string) *url.URL {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	u := base.ResolveReference(ref)
	if !isHTTP(u) {
		return nil
	}
	u.Fragment = ""
	return u
}

func isHTTP(u *url.URL) bool {
	return u.Scheme == "http" || u.Scheme == "https"
}

func sameHost(a, b *url.URL) bool {
	return strings.EqualFold(a.Hostname(), b.Hostname())
}

func isImage(u *url.URL) bool {
	return imageExtensions[strings.ToLower(path.Ext(u.Path))]
}

func hasContentType(resp *http.Response, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), prefix)
}

func uniquePath(dir, name string) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	candidate := filepath.Join(dir, name)
	for i := 1; fileExists(candidate); i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s_%d%s", stem, i, ext))
	}
	return candidate
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func writeFile(dest string, r io.Reader) error {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(dest)
		return err
	}
	return f.Close()
}
