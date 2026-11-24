package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	typeByte       = 1
	typeASCII      = 2
	typeShort      = 3
	typeLong       = 4
	typeRational   = 5
	typeSByte      = 6
	typeUndefined  = 7
	typeSShort     = 8
	typeSLong      = 9
	typeSRational  = 10
	typeFloat      = 11
	typeDouble     = 12
	typeIFD        = 13
	tagUserComment = 0x9286
	maxValues      = 16
)

var typeSizes = map[uint16]uint32{
	typeByte: 1, typeASCII: 1, typeShort: 2, typeLong: 4, typeRational: 8,
	typeSByte: 1, typeUndefined: 1, typeSShort: 2, typeSLong: 4, typeSRational: 8,
	typeFloat: 4, typeDouble: 8, typeIFD: 4,
}

type ifdEntry struct {
	tag   uint16
	typ   uint16
	count uint32
	value []byte
}

type tiffReader struct {
	data  []byte
	order binary.ByteOrder
	seen  map[uint32]bool
}

func parseEXIF(data []byte) []section {
	r, first, ok := newTIFFReader(data)
	if !ok {
		return []section{{name: "EXIF", fields: []field{{"Error", "invalid TIFF header"}}}}
	}
	sections, next := r.walk("EXIF IFD0", first, tiffTags)
	thumbnail, _ := r.walk("EXIF IFD1 (thumbnail)", next, tiffTags)
	return append(sections, thumbnail...)
}

func newTIFFReader(data []byte) (*tiffReader, uint32, bool) {
	if len(data) < 8 {
		return nil, 0, false
	}
	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return nil, 0, false
	}
	if order.Uint16(data[2:]) != 42 {
		return nil, 0, false
	}
	return &tiffReader{data: data, order: order, seen: map[uint32]bool{}}, order.Uint32(data[4:]), true
}

func (r *tiffReader) walk(name string, offset uint32, tags map[uint16]string) ([]section, uint32) {
	if offset == 0 || r.seen[offset] {
		return nil, 0
	}
	r.seen[offset] = true
	entries, next, ok := r.readIFD(offset)
	if !ok {
		return nil, 0
	}
	current := section{name: name}
	var children []section
	for _, e := range entries {
		if sub, isPointer := subIFDs[e.tag]; isPointer {
			child, _ := r.walk(sub.name, r.pointer(e), sub.tags)
			children = append(children, child...)
			continue
		}
		current.fields = append(current.fields, field{tagName(tags, e.tag), r.formatValue(e)})
	}
	return append([]section{current}, children...), next
}

func (r *tiffReader) readIFD(offset uint32) ([]ifdEntry, uint32, bool) {
	head, ok := r.slice(offset, 2)
	if !ok {
		return nil, 0, false
	}
	count := uint32(r.order.Uint16(head))
	var entries []ifdEntry
	for i := uint32(0); i < count; i++ {
		if e, ok := r.readEntry(offset + 2 + i*12); ok {
			entries = append(entries, e)
		}
	}
	tail, ok := r.slice(offset+2+count*12, 4)
	if !ok {
		return entries, 0, true
	}
	return entries, r.order.Uint32(tail), true
}

func (r *tiffReader) readEntry(pos uint32) (ifdEntry, bool) {
	raw, ok := r.slice(pos, 12)
	if !ok {
		return ifdEntry{}, false
	}
	e := ifdEntry{tag: r.order.Uint16(raw), typ: r.order.Uint16(raw[2:]), count: r.order.Uint32(raw[4:])}
	size, known := typeSizes[e.typ]
	if !known {
		return ifdEntry{}, false
	}
	total := uint64(size) * uint64(e.count)
	if total <= 4 {
		e.value = raw[8 : 8+total]
		return e, true
	}
	if total > uint64(len(r.data)) {
		return ifdEntry{}, false
	}
	e.value, ok = r.slice(r.order.Uint32(raw[8:]), uint32(total))
	return e, ok
}

func (r *tiffReader) slice(offset, length uint32) ([]byte, bool) {
	end := uint64(offset) + uint64(length)
	if end > uint64(len(r.data)) {
		return nil, false
	}
	return r.data[offset:end], true
}

func (r *tiffReader) pointer(e ifdEntry) uint32 {
	if (e.typ != typeLong && e.typ != typeIFD) || len(e.value) < 4 {
		return 0
	}
	return r.order.Uint32(e.value)
}

func (r *tiffReader) formatValue(e ifdEntry) string {
	switch {
	case e.tag == tagUserComment:
		return userComment(e.value)
	case e.typ == typeASCII:
		return sanitize(string(e.value))
	case e.typ == typeUndefined:
		return formatBytes(e.value)
	}
	return joinValues(r.decodeNumbers(e), e.count)
}

func (r *tiffReader) decodeNumbers(e ifdEntry) []string {
	size := int(typeSizes[e.typ])
	n := min(int(e.count), maxValues)
	values := make([]string, 0, n)
	for i := 0; i < n; i++ {
		values = append(values, r.decodeNumber(e.typ, e.value[i*size:]))
	}
	return values
}

func (r *tiffReader) decodeNumber(typ uint16, b []byte) string {
	o := r.order
	switch typ {
	case typeByte:
		return strconv.Itoa(int(b[0]))
	case typeSByte:
		return strconv.Itoa(int(int8(b[0])))
	case typeShort:
		return strconv.Itoa(int(o.Uint16(b)))
	case typeSShort:
		return strconv.Itoa(int(int16(o.Uint16(b))))
	case typeLong, typeIFD:
		return strconv.FormatUint(uint64(o.Uint32(b)), 10)
	case typeSLong:
		return strconv.Itoa(int(int32(o.Uint32(b))))
	case typeRational:
		return formatRational(int64(o.Uint32(b)), int64(o.Uint32(b[4:])))
	case typeSRational:
		return formatRational(int64(int32(o.Uint32(b))), int64(int32(o.Uint32(b[4:]))))
	case typeFloat:
		return strconv.FormatFloat(float64(math.Float32frombits(o.Uint32(b))), 'g', -1, 32)
	case typeDouble:
		return strconv.FormatFloat(math.Float64frombits(o.Uint64(b)), 'g', -1, 64)
	}
	return "?"
}

func formatRational(num, den int64) string {
	if den != 0 && num%den == 0 {
		return strconv.FormatInt(num/den, 10)
	}
	return fmt.Sprintf("%d/%d", num, den)
}

func joinValues(values []string, count uint32) string {
	s := strings.Join(values, ", ")
	if int(count) > len(values) {
		s += fmt.Sprintf(" ... (%d values)", count)
	}
	return s
}

func userComment(b []byte) string {
	if len(b) >= 8 && bytes.HasPrefix(b, []byte("ASCII")) {
		return sanitize(string(b[8:]))
	}
	return formatBytes(b)
}

func tagName(tags map[uint16]string, tag uint16) string {
	if name, ok := tags[tag]; ok {
		return name
	}
	return fmt.Sprintf("Tag 0x%04X", tag)
}
