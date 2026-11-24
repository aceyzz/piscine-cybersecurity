package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

const timeLayout = "2006-01-02 15:04:05"

type field struct {
	key   string
	value string
}

type section struct {
	name   string
	fields []field
}

type report struct {
	path     string
	sections []section
}

type format struct {
	name  string
	magic []byte
	parse func([]byte) []section
}

var formats = []format{
	{"JPEG", []byte{0xFF, 0xD8, 0xFF}, parseJPEG},
	{"PNG", pngSignature, parsePNG},
	{"GIF", []byte("GIF87a"), parseGIF},
	{"GIF", []byte("GIF89a"), parseGIF},
	{"BMP", []byte("BM"), parseBMP},
}

func buildReport(path string) (report, error) {
	info, err := os.Stat(path)
	if err != nil {
		return report{}, err
	}
	if !info.Mode().IsRegular() {
		return report{}, fmt.Errorf("%s: not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return report{}, err
	}
	f, ok := detectFormat(data)
	if !ok {
		return report{}, fmt.Errorf("%s: unsupported format", path)
	}
	sections := append([]section{fileSection(path, info, f.name)}, f.parse(data)...)
	return report{path: path, sections: sections}, nil
}

func detectFormat(data []byte) (format, bool) {
	for _, f := range formats {
		if bytes.HasPrefix(data, f.magic) {
			return f, true
		}
	}
	return format{}, false
}

func fileSection(path string, info os.FileInfo, formatName string) section {
	return section{name: "File", fields: []field{
		{"Name", filepath.Base(path)},
		{"Format", formatName},
		{"Size", fmt.Sprintf("%d bytes", info.Size())},
		{"Permissions", info.Mode().Perm().String()},
		{"Created", formatBirthTime(path)},
		{"Modified", info.ModTime().Format(timeLayout)},
	}}
}

func formatBirthTime(path string) string {
	t, ok := birthTime(path)
	if !ok {
		return "unavailable"
	}
	return t.Format(timeLayout)
}

func printReport(r report) {
	fmt.Printf("==> %s\n", r.path)
	for _, s := range r.sections {
		printSection(s)
	}
}

func printSection(s section) {
	fmt.Printf("[%s]\n", s.name)
	if len(s.fields) == 0 {
		fmt.Println("  (empty)")
		return
	}
	for _, f := range s.fields {
		fmt.Printf("  %-28s %s\n", f.key, truncate(f.value))
	}
}
