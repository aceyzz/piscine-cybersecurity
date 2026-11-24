package main

import "fmt"

var (
	pngSignature  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	pngColorTypes = map[byte]string{0: "Grayscale", 2: "RGB", 3: "Indexed", 4: "Grayscale + alpha", 6: "RGBA"}
)

type chunk struct {
	kind string
	data []byte
}

func parsePNG(data []byte) []section {
	info := section{name: "PNG"}
	text := section{name: "PNG text"}
	var exif []section
	for _, c := range pngChunks(data) {
		switch c.kind {
		case "IHDR":
			info.fields = append(info.fields, ihdrFields(c.data)...)
		case "pHYs":
			info.fields = append(info.fields, physFields(c.data)...)
		case "tIME":
			info.fields = append(info.fields, timeFields(c.data)...)
		case "tEXt", "zTXt", "iTXt":
			text.fields = append(text.fields, textFields(c)...)
		case "eXIf":
			exif = append(exif, parseEXIF(c.data)...)
		}
	}
	sections := []section{info}
	if len(text.fields) > 0 {
		sections = append(sections, text)
	}
	return append(sections, exif...)
}

func pngChunks(data []byte) []chunk {
	var chunks []chunk
	pos := len(pngSignature)
	for pos+12 <= len(data) {
		length := int(be.Uint32(data[pos:]))
		end := pos + 12 + length
		if end > len(data) {
			break
		}
		c := chunk{kind: string(data[pos+4 : pos+8]), data: data[pos+8 : pos+8+length]}
		chunks = append(chunks, c)
		if c.kind == "IEND" {
			break
		}
		pos = end
	}
	return chunks
}

func ihdrFields(d []byte) []field {
	if len(d) < 13 {
		return nil
	}
	color, ok := pngColorTypes[d[9]]
	if !ok {
		color = "unknown"
	}
	interlace := "none"
	if d[12] == 1 {
		interlace = "Adam7"
	}
	return []field{
		{"Width", fmt.Sprint(be.Uint32(d))},
		{"Height", fmt.Sprint(be.Uint32(d[4:]))},
		{"Bit depth", fmt.Sprint(d[8])},
		{"Color type", color},
		{"Interlace", interlace},
	}
}

func physFields(d []byte) []field {
	if len(d) < 9 {
		return nil
	}
	unit := "aspect ratio"
	if d[8] == 1 {
		unit = "px/m"
	}
	return []field{{"Pixel density", fmt.Sprintf("%d x %d %s", be.Uint32(d), be.Uint32(d[4:]), unit)}}
}

func timeFields(d []byte) []field {
	if len(d) < 7 {
		return nil
	}
	stamp := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", be.Uint16(d), d[2], d[3], d[4], d[5], d[6])
	return []field{{"Last modified (tIME)", stamp}}
}

func textFields(c chunk) []field {
	key, rest, ok := splitNull(c.data)
	if !ok {
		return nil
	}
	value, err := pngTextValue(c.kind, rest)
	if err != nil {
		value = "(invalid: " + err.Error() + ")"
	}
	return []field{{sanitize(latin1(key)), sanitize(value)}}
}

func pngTextValue(kind string, rest []byte) (string, error) {
	switch kind {
	case "tEXt":
		return latin1(rest), nil
	case "zTXt":
		if len(rest) < 1 {
			return "", errTruncated
		}
		out, err := inflate(rest[1:])
		return latin1(out), err
	}
	return itxtValue(rest)
}

func itxtValue(rest []byte) (string, error) {
	if len(rest) < 2 {
		return "", errTruncated
	}
	compressed := rest[0] == 1
	_, rest, ok := splitNull(rest[2:])
	if !ok {
		return "", errTruncated
	}
	_, text, ok := splitNull(rest)
	if !ok {
		return "", errTruncated
	}
	if !compressed {
		return string(text), nil
	}
	out, err := inflate(text)
	return string(out), err
}
