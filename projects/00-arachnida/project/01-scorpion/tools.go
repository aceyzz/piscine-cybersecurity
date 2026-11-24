package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

const (
	maxInflate    = 1 << 20
	maxInlineText = 64
	maxDisplay    = 120
	maxRawBytes   = 8
)

var (
	be           = binary.BigEndian
	le           = binary.LittleEndian
	errTruncated = errors.New("truncated data")
)

func warn(err error) {
	fmt.Fprintln(os.Stderr, "scorpion:", err)
}

func sanitize(s string) string {
	s = strings.ToValidUTF8(strings.TrimRight(s, "\x00"), "?")
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return '.'
	}, s)
}

func latin1(b []byte) string {
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return string(runes)
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7E {
			return false
		}
	}
	return true
}

func formatBytes(b []byte) string {
	text := strings.TrimRight(string(b), "\x00")
	switch {
	case text == "":
		return ""
	case len(text) <= maxInlineText && isPrintableASCII(text):
		return text
	case len(b) <= maxRawBytes:
		return strings.Trim(fmt.Sprint(b), "[]")
	}
	return fmt.Sprintf("%d bytes", len(b))
}

func splitNull(b []byte) ([]byte, []byte, bool) {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return nil, nil, false
	}
	return b[:i], b[i+1:], true
}

func inflate(b []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, maxInflate))
}

func truncatedSection(name string) []section {
	return []section{{name: name, fields: []field{{"Error", "truncated header"}}}}
}

func truncate(s string) string {
	runes := []rune(s)
	if len(runes) <= maxDisplay {
		return s
	}
	return fmt.Sprintf("%s... (%d chars)", string(runes[:maxDisplay]), len(runes))
}
