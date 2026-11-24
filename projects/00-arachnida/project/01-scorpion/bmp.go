package main

import "fmt"

var (
	dibNames = map[uint32]string{
		12:  "BITMAPCOREHEADER",
		40:  "BITMAPINFOHEADER",
		52:  "BITMAPV2INFOHEADER",
		56:  "BITMAPV3INFOHEADER",
		108: "BITMAPV4HEADER",
		124: "BITMAPV5HEADER",
	}
	bmpCompressions = map[uint32]string{
		0: "none (BI_RGB)",
		1: "RLE8",
		2: "RLE4",
		3: "BITFIELDS",
		4: "JPEG",
		5: "PNG",
		6: "ALPHABITFIELDS",
	}
)

func parseBMP(data []byte) []section {
	if len(data) < 26 {
		return truncatedSection("BMP")
	}
	header := le.Uint32(data[14:])
	name, ok := dibNames[header]
	if !ok {
		name = fmt.Sprintf("unknown (%d bytes)", header)
	}
	info := section{name: "BMP", fields: []field{
		{"Declared file size", fmt.Sprintf("%d bytes", le.Uint32(data[2:]))},
		{"Pixel data offset", fmt.Sprint(le.Uint32(data[10:]))},
		{"DIB header", name},
	}}
	switch {
	case header == 12:
		info.fields = append(info.fields, coreFields(data)...)
	case header >= 40 && len(data) >= 54:
		info.fields = append(info.fields, infoFields(data)...)
	}
	return []section{info}
}

func coreFields(d []byte) []field {
	return []field{
		{"Width", fmt.Sprint(le.Uint16(d[18:]))},
		{"Height", fmt.Sprint(le.Uint16(d[20:]))},
		{"Bits per pixel", fmt.Sprint(le.Uint16(d[24:]))},
	}
}

func infoFields(d []byte) []field {
	height := int64(int32(le.Uint32(d[22:])))
	order := "bottom-up"
	if height < 0 {
		order = "top-down"
		height = -height
	}
	compression, ok := bmpCompressions[le.Uint32(d[30:])]
	if !ok {
		compression = "unknown"
	}
	return []field{
		{"Width", fmt.Sprint(int32(le.Uint32(d[18:])))},
		{"Height", fmt.Sprint(height)},
		{"Row order", order},
		{"Bits per pixel", fmt.Sprint(le.Uint16(d[28:]))},
		{"Compression", compression},
		{"Image size", fmt.Sprintf("%d bytes", le.Uint32(d[34:]))},
		{"Resolution", fmt.Sprintf("%d x %d px/m", int32(le.Uint32(d[38:])), int32(le.Uint32(d[42:])))},
		{"Colors used", fmt.Sprint(le.Uint32(d[46:]))},
	}
}
