package main

import (
	"bytes"
	"fmt"
)

var (
	jfifHeader   = []byte("JFIF\x00")
	exifHeader   = []byte("Exif\x00\x00")
	xmpHeader    = []byte("http://ns.adobe.com/xap/1.0/\x00")
	sofNames     = map[byte]string{0xC0: "Baseline DCT", 0xC1: "Extended sequential DCT", 0xC2: "Progressive DCT", 0xC3: "Lossless"}
	densityUnits = map[byte]string{0: "aspect ratio", 1: "dpi", 2: "dpcm"}
)

type segment struct {
	marker byte
	data   []byte
}

func parseJPEG(data []byte) []section {
	info := section{name: "JPEG"}
	var exif []section
	for _, seg := range jpegSegments(data) {
		switch {
		case seg.marker == 0xE0 && bytes.HasPrefix(seg.data, jfifHeader):
			info.fields = append(info.fields, jfifFields(seg.data)...)
		case seg.marker == 0xE1 && bytes.HasPrefix(seg.data, exifHeader):
			exif = append(exif, parseEXIF(seg.data[len(exifHeader):])...)
		case seg.marker == 0xE1 && bytes.HasPrefix(seg.data, xmpHeader):
			info.fields = append(info.fields, field{"XMP", fmt.Sprintf("%d bytes", len(seg.data)-len(xmpHeader))})
		case seg.marker == 0xFE:
			info.fields = append(info.fields, field{"Comment", sanitize(string(seg.data))})
		case isSOF(seg.marker):
			info.fields = append(info.fields, sofFields(seg)...)
		}
	}
	return append([]section{info}, exif...)
}

func jpegSegments(data []byte) []segment {
	var segments []segment
	pos := 2
	for pos+4 <= len(data) && data[pos] == 0xFF {
		marker := data[pos+1]
		if marker == 0xFF {
			pos++
			continue
		}
		if marker == 0xD9 {
			break
		}
		if isStandalone(marker) {
			pos += 2
			continue
		}
		length := int(be.Uint16(data[pos+2:]))
		end := pos + 2 + length
		if length < 2 || end > len(data) {
			break
		}
		segments = append(segments, segment{marker: marker, data: data[pos+4 : end]})
		if marker == 0xDA {
			break
		}
		pos = end
	}
	return segments
}

func isStandalone(marker byte) bool {
	return marker == 0x01 || marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7)
}

func isSOF(marker byte) bool {
	return marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC
}

func sofFields(seg segment) []field {
	if len(seg.data) < 6 {
		return nil
	}
	name, ok := sofNames[seg.marker]
	if !ok {
		name = fmt.Sprintf("SOF 0x%02X", seg.marker)
	}
	return []field{
		{"Encoding", name},
		{"Width", fmt.Sprint(be.Uint16(seg.data[3:]))},
		{"Height", fmt.Sprint(be.Uint16(seg.data[1:]))},
		{"Bits per sample", fmt.Sprint(seg.data[0])},
		{"Components", fmt.Sprint(seg.data[5])},
	}
}

func jfifFields(data []byte) []field {
	if len(data) < 12 {
		return nil
	}
	unit, ok := densityUnits[data[7]]
	if !ok {
		unit = "unknown unit"
	}
	return []field{
		{"JFIF version", fmt.Sprintf("%d.%02d", data[5], data[6])},
		{"Density", fmt.Sprintf("%d x %d %s", be.Uint16(data[8:]), be.Uint16(data[10:]), unit)},
	}
}
