package main

import "fmt"

func parseGIF(data []byte) []section {
	if len(data) < 13 {
		return truncatedSection("GIF")
	}
	info := section{name: "GIF", fields: []field{
		{"Version", string(data[3:6])},
		{"Width", fmt.Sprint(le.Uint16(data[6:]))},
		{"Height", fmt.Sprint(le.Uint16(data[8:]))},
	}}
	pos := 13
	if data[10]&0x80 != 0 {
		colors := 1 << (int(data[10]&0x07) + 1)
		info.fields = append(info.fields, field{"Global color table", fmt.Sprintf("%d colors", colors)})
		pos += 3 * colors
	}
	frames, comments := gifBlocks(data, pos)
	info.fields = append(info.fields, field{"Frames", fmt.Sprint(frames)})
	for _, c := range comments {
		info.fields = append(info.fields, field{"Comment", c})
	}
	return []section{info}
}

func gifBlocks(data []byte, pos int) (int, []string) {
	frames := 0
	var comments []string
	for pos < len(data) {
		switch data[pos] {
		case 0x21:
			if pos+1 >= len(data) {
				return frames, comments
			}
			body, next := gifSubBlocks(data, pos+2)
			if data[pos+1] == 0xFE {
				comments = append(comments, sanitize(latin1(body)))
			}
			pos = next
		case 0x2C:
			frames++
			pos = skipGIFImage(data, pos)
		default:
			return frames, comments
		}
	}
	return frames, comments
}

func gifSubBlocks(data []byte, pos int) ([]byte, int) {
	var body []byte
	for pos < len(data) {
		size := int(data[pos])
		pos++
		if size == 0 {
			return body, pos
		}
		if pos+size > len(data) {
			return body, len(data)
		}
		body = append(body, data[pos:pos+size]...)
		pos += size
	}
	return body, pos
}

func skipGIFImage(data []byte, pos int) int {
	if pos+10 > len(data) {
		return len(data)
	}
	flags := data[pos+9]
	pos += 10
	if flags&0x80 != 0 {
		pos += 3 * (1 << (int(flags&0x07) + 1))
	}
	_, next := gifSubBlocks(data, pos+1)
	return next
}
