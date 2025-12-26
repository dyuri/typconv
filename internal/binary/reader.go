package binary

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dyuri/typconv/internal/model"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// Reader handles parsing of binary TYP files
type Reader struct {
	r         io.ReaderAt
	size      int64
	endian    binary.ByteOrder    // Garmin uses little-endian
	typHeader *TYPHeader          // Parsed header with section pointers
	decoder   *encoding.Decoder   // Text decoder for strings (based on codepage)
}

// NewReader creates a new binary TYP reader
func NewReader(r io.ReaderAt, size int64) *Reader {
	return &Reader{
		r:      r,
		size:   size,
		endian: binary.LittleEndian,
	}
}

// Parse reads the entire TYP file and returns the internal model
func (r *Reader) Parse() (*model.TYPFile, error) {
	typ := model.NewTYPFile()

	// Read header
	header, err := r.ReadHeader()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	typ.Header = *header

	// Parse POI (Point) types using array structure
	if r.typHeader.Points.ArraySize > 0 {
		points, err := r.ReadPointTypes(r.typHeader.Points)
		if err != nil {
			return nil, fmt.Errorf("read point types: %w", err)
		}
		typ.Points = points
	}

	// Parse Polyline (Line) types using array structure
	if r.typHeader.Polylines.ArraySize > 0 {
		lines, err := r.ReadLineTypes(r.typHeader.Polylines)
		if err != nil {
			return nil, fmt.Errorf("read line types: %w", err)
		}
		typ.Lines = lines
	}

	// Parse Polygon types using array structure
	if r.typHeader.Polygons.ArraySize > 0 {
		polygons, err := r.ReadPolygonTypes(r.typHeader.Polygons)
		if err != nil {
			return nil, fmt.Errorf("read polygon types: %w", err)
		}
		typ.Polygons = polygons
	}

	return typ, nil
}

// findSectionDirectory attempts to locate the section directory
// Returns the offset, or -1 if not found
func (r *Reader) findSectionDirectory() int64 {
	// First, try reading offset from header
	// Some formats store section dir offset at specific locations
	headerBuf := make([]byte, 256)
	if _, err := r.r.ReadAt(headerBuf, 0); err == nil {
		// Try offset 0x15 (sometimes stores section offset)
		candidateOffset := int64(r.endian.Uint32(headerBuf[0x15:0x19]))
		if candidateOffset > 0 && candidateOffset < r.size && r.isSectionDirectoryAt(candidateOffset) {
			return candidateOffset
		}
	}

	// Try common fixed locations
	candidates := []int64{
		0x15, 0x19, 0x20, 0x25, 0x30, 0x40,
		0x50, 0x60, 0x70, 0x7A, 0x7C, 0x80, 0x90, 0xA0,
	}

	for _, offset := range candidates {
		if r.isSectionDirectoryAt(offset) {
			return offset
		}
	}

	// Scan through first 512 bytes
	for offset := int64(0); offset < 512 && offset < r.size-20; offset += 2 {
		if r.isSectionDirectoryAt(offset) {
			return offset
		}
	}

	return -1
}

// isSectionDirectoryAt checks if a section directory exists at the given offset
func (r *Reader) isSectionDirectoryAt(offset int64) bool {
	// Need at least 14 bytes (2 for count + 12 for one entry)
	if offset+14 > r.size {
		return false
	}

	buf := make([]byte, 128)
	if _, err := r.r.ReadAt(buf, offset); err != nil {
		return false
	}

	count := int(r.endian.Uint16(buf[0:2]))
	// Section count should be reasonable (1-10 typically)
	if count < 1 || count > 10 {
		return false
	}

	// Check if first section entry looks valid
	entryOffset := 2
	if entryOffset+12 > len(buf) {
		return false
	}

	secType := buf[entryOffset]
	secOffset := r.endian.Uint32(buf[entryOffset+1 : entryOffset+5])
	secLength := r.endian.Uint32(buf[entryOffset+5 : entryOffset+9])

	// Valid section types are 0x01-0x04
	if secType < 0x01 || secType > 0x04 {
		return false
	}

	// Section offset should be after the directory and within file
	if int64(secOffset) <= offset || int64(secOffset) >= r.size {
		return false
	}

	// Section length should be reasonable
	if secLength == 0 || int64(secLength) > r.size {
		return false
	}

	return true
}

// SectionInfo contains metadata for a TYP section (points, lines, polygons)
type SectionInfo struct {
	DataOffset  uint32 // Offset to data section
	DataLength  uint32 // Length of data section
	ArrayOffset uint32 // Offset to index array
	ArrayModulo uint16 // Size of each array entry (3, 4, or 5 bytes)
	ArraySize   uint32 // Total size of array in bytes
}

// TYPHeader represents the parsed header with section pointers
type TYPHeader struct {
	Descriptor uint16 // First field, often equals header length
	Version    uint16
	Year       uint16
	Month      uint8
	Day        uint8
	Hour       uint8
	Minutes    uint8
	Seconds    uint8
	CodePage   uint16
	PID        uint16 // Product ID
	FID        uint16 // Family ID

	// Section information
	Points    SectionInfo
	Polylines SectionInfo
	Polygons  SectionInfo
	Order     SectionInfo
}

// ReadHeader reads and parses the TYP file header
// Format based on QMapShack implementation
func (r *Reader) ReadHeader() (*model.Header, error) {
	// Allocate buffer for header (minimum 0x5B bytes)
	buf := make([]byte, 256)
	if _, err := r.r.ReadAt(buf, 0); err != nil {
		return nil, fmt.Errorf("read header bytes: %w", err)
	}

	// Offset 0x00-0x01: Descriptor (uint16)
	descriptor := r.endian.Uint16(buf[0x00:0x02])

	// Offset 0x02-0x0B: "GARMIN TYP" signature
	if string(buf[0x02:0x0C]) != "GARMIN TYP" {
		return nil, fmt.Errorf("unrecognized TYP file format - missing GARMIN TYP signature")
	}

	// Offset 0x0C: Version (uint16)
	version := r.endian.Uint16(buf[0x0C:0x0E])

	// Offset 0x0E: Year (uint16) - add 1900
	year := r.endian.Uint16(buf[0x0E:0x10])

	// Offset 0x10-0x14: Date/time fields
	month := buf[0x10] // 0-based!
	day := buf[0x11]
	hour := buf[0x12]
	minutes := buf[0x13]
	seconds := buf[0x14]

	// Offset 0x15-0x16: CodePage (uint16)
	codePage := r.endian.Uint16(buf[0x15:0x17])

	// Section data pointers
	// Points
	pointsDataOffset := r.endian.Uint32(buf[0x17:0x1B])
	pointsDataLength := r.endian.Uint32(buf[0x1B:0x1F])

	// Polylines
	polylinesDataOffset := r.endian.Uint32(buf[0x1F:0x23])
	polylinesDataLength := r.endian.Uint32(buf[0x23:0x27])

	// Polygons
	polygonsDataOffset := r.endian.Uint32(buf[0x27:0x2B])
	polygonsDataLength := r.endian.Uint32(buf[0x2B:0x2F])

	// Offset 0x2F-0x30: PID (uint16)
	pid := r.endian.Uint16(buf[0x2F:0x31])

	// Offset 0x31-0x32: FID (uint16)
	fid := r.endian.Uint16(buf[0x31:0x33])

	// Array metadata for each section
	// Points array
	pointsArrayOffset := r.endian.Uint32(buf[0x33:0x37])
	pointsArrayModulo := r.endian.Uint16(buf[0x37:0x39])
	pointsArraySize := r.endian.Uint32(buf[0x39:0x3D])

	// Polylines array
	polylinesArrayOffset := r.endian.Uint32(buf[0x3D:0x41])
	polylinesArrayModulo := r.endian.Uint16(buf[0x41:0x43])
	polylinesArraySize := r.endian.Uint32(buf[0x43:0x47])

	// Polygons array
	polygonsArrayOffset := r.endian.Uint32(buf[0x47:0x4B])
	polygonsArrayModulo := r.endian.Uint16(buf[0x4B:0x4D])
	polygonsArraySize := r.endian.Uint32(buf[0x4D:0x51])

	// Draw order array
	orderArrayOffset := r.endian.Uint32(buf[0x51:0x55])
	orderArrayModulo := r.endian.Uint16(buf[0x55:0x57])
	orderArraySize := r.endian.Uint32(buf[0x57:0x5B])

	// Store section information for parsing
	r.typHeader = &TYPHeader{
		Descriptor: descriptor,
		Version:    version,
		Year:       year,
		Month:      month,
		Day:        day,
		Hour:       hour,
		Minutes:    minutes,
		Seconds:    seconds,
		CodePage:   codePage,
		PID:        pid,
		FID:        fid,
		Points: SectionInfo{
			DataOffset:  pointsDataOffset,
			DataLength:  pointsDataLength,
			ArrayOffset: pointsArrayOffset,
			ArrayModulo: pointsArrayModulo,
			ArraySize:   pointsArraySize,
		},
		Polylines: SectionInfo{
			DataOffset:  polylinesDataOffset,
			DataLength:  polylinesDataLength,
			ArrayOffset: polylinesArrayOffset,
			ArrayModulo: polylinesArrayModulo,
			ArraySize:   polylinesArraySize,
		},
		Polygons: SectionInfo{
			DataOffset:  polygonsDataOffset,
			DataLength:  polygonsDataLength,
			ArrayOffset: polygonsArrayOffset,
			ArrayModulo: polygonsArrayModulo,
			ArraySize:   polygonsArraySize,
		},
		Order: SectionInfo{
			ArrayOffset: orderArrayOffset,
			ArrayModulo: orderArrayModulo,
			ArraySize:   orderArraySize,
		},
	}

	// Set up text decoder based on codepage
	switch codePage {
	case 1252: // Windows-1252 (Western European)
		r.decoder = charmap.Windows1252.NewDecoder()
	case 1250: // Windows-1250 (Central European, includes Hungarian)
		r.decoder = charmap.Windows1250.NewDecoder()
	case 65001: // UTF-8
		r.decoder = nil // Use UTF-8 directly
	default:
		// Default to Windows-1252
		r.decoder = charmap.Windows1252.NewDecoder()
	}

	header := &model.Header{
		Version:  int(version),
		CodePage: int(codePage),
		FID:      int(fid),
		PID:      int(pid),
	}

	return header, nil
}

// Section represents a section in the TYP file
type Section struct {
	Type   byte   // Section type (1=points, 2=lines, 3=polygons, etc.)
	Offset uint32 // Offset from file start
	Length uint32 // Section length in bytes
}

// ReadSectionDirectory reads the section directory table of contents
func (r *Reader) ReadSectionDirectory(offset int64) ([]Section, error) {
	// Read section count (uint16 at directory start)
	buf := make([]byte, 2)
	if _, err := r.r.ReadAt(buf, offset); err != nil {
		return nil, fmt.Errorf("read section count: %w", err)
	}
	count := int(r.endian.Uint16(buf))

	if count == 0 || count > 100 { // Sanity check
		return nil, fmt.Errorf("invalid section count: %d", count)
	}

	// Read section entries (12 bytes each based on spec)
	sections := make([]Section, count)
	entrySize := int64(12)

	for i := 0; i < count; i++ {
		entryOffset := offset + 2 + int64(i)*entrySize
		entryBuf := make([]byte, entrySize)

		if _, err := r.r.ReadAt(entryBuf, entryOffset); err != nil {
			return nil, fmt.Errorf("read section entry %d: %w", i, err)
		}

		sections[i] = Section{
			Type:   entryBuf[0],
			Offset: r.endian.Uint32(entryBuf[1:5]),
			Length: r.endian.Uint32(entryBuf[5:9]),
			// Bytes 9-11 are reserved
		}
	}

	return sections, nil
}

// ReadPointTypes reads all point type definitions using the index array
func (r *Reader) ReadPointTypes(section SectionInfo) ([]model.PointType, error) {
	// Calculate number of entries in the index array
	if section.ArrayModulo == 0 || (section.ArraySize%uint32(section.ArrayModulo)) != 0 {
		return nil, nil // Empty or invalid array
	}

	numEntries := int(section.ArraySize / uint32(section.ArrayModulo))
	points := make([]model.PointType, 0, numEntries)

	for i := 0; i < numEntries; i++ {
		// Read array entry
		arrayPos := int64(section.ArrayOffset) + int64(i)*int64(section.ArrayModulo)
		typCode, dataOffset, err := r.readArrayEntry(arrayPos, section.ArrayModulo)
		if err != nil {
			return nil, fmt.Errorf("read array entry %d: %w", i, err)
		}

		// Decode type/subtype
		typ, subtyp := r.decodeTypeSubtype(typCode)

		// Read point data
		pt, err := r.readPointData(int64(section.DataOffset)+int64(dataOffset), typ, subtyp)
		if err != nil {
			return nil, fmt.Errorf("read point data at offset 0x%x: %w", section.DataOffset+dataOffset, err)
		}

		points = append(points, pt)
	}

	return points, nil
}

// readArrayEntry reads an index array entry
// Returns the type code and data offset
func (r *Reader) readArrayEntry(offset int64, modulo uint16) (uint16, uint32, error) {
	buf := make([]byte, 8)
	if _, err := r.r.ReadAt(buf, offset); err != nil && err != io.EOF {
		return 0, 0, err
	}

	// Type/subtype is always first 2 bytes
	typeCode := r.endian.Uint16(buf[0:2])

	// Data offset size depends on modulo
	var dataOffset uint32
	switch modulo {
	case 5:
		// 24-bit offset (3 bytes)
		dataOffset = uint32(buf[2]) | (uint32(buf[3]) << 8) | (uint32(buf[4]) << 16)
	case 4:
		// 16-bit offset (2 bytes)
		dataOffset = uint32(r.endian.Uint16(buf[2:4]))
	case 3:
		// 8-bit offset (1 byte)
		dataOffset = uint32(buf[2])
	default:
		return 0, 0, fmt.Errorf("unsupported array modulo: %d", modulo)
	}

	return typeCode, dataOffset, nil
}

// decodeTypeSubtype decodes the bit-packed type/subtype field
// Based on QMapShack implementation
func (r *Reader) decodeTypeSubtype(t16 uint16) (uint32, uint32) {
	// Unpack the 16-bit field
	t16_2 := (t16 >> 5) | ((t16 & 0x1f) << 11)
	typ := uint32(t16_2 & 0x7FF)    // 11 bits
	subtyp := uint32(t16 & 0x01F)   // 5 bits

	// Check for extended type
	if t16&0x2000 != 0 {
		typ = 0x10000 | (typ << 8) | subtyp
	} else {
		typ = (typ << 8) + subtyp
	}

	return typ, subtyp
}

// readPointData reads a single point type definition from the data section
func (r *Reader) readPointData(offset int64, typ, subtyp uint32) (model.PointType, error) {
	// Read first 5 bytes: flags, width, height, ncolors, ctype
	buf := make([]byte, 4096)
	n, err := r.r.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return model.PointType{}, err
	}
	buf = buf[:n]

	if len(buf) < 5 {
		return model.PointType{}, fmt.Errorf("buffer too small: %d bytes", len(buf))
	}

	flags := buf[0]
	width := int(buf[1])
	height := int(buf[2])
	ncolors := int(buf[3])
	_ = buf[4] // ctype - TODO: use for color table reading

	hasLabels := (flags & 0x04) != 0
	hasTextColors := (flags & 0x08) != 0
	dayNightMode := flags & 0x03

	pt := model.PointType{
		Type:    int(typ),
		SubType: int(subtyp),
		Labels:  make(map[string]string),
	}

	pos := 5

	// Read color table
	// TODO: Implement color table reading based on ncolors and ctype

	// Read bitmap
	// TODO: Implement bitmap reading based on width, height, and bpp

	// For now, skip to labels by estimating bitmap size
	// This is a placeholder - we need proper bitmap parsing
	bpp := r.calculateBPP(ncolors)
	paletteSize := ncolors * 3
	bitmapSize := (width * height * bpp) / 8
	if (width*height*bpp)%8 != 0 {
		bitmapSize++
	}

	pos += paletteSize + bitmapSize

	// Handle day/night modes
	if dayNightMode == 0x03 {
		// Separate night bitmap - skip it too
		pos += 2 // ncolors, ctype for night
		// Would need to read night palette and bitmap here
	}

	// Read labels if present
	if hasLabels && pos < len(buf) {
		labels, bytesRead, err := r.readLabels(buf[pos:])
		if err == nil {
			pt.Labels = labels
			pos += bytesRead
		}
	}

	// Read text colors if present
	if hasTextColors && pos < len(buf) {
		// TODO: Implement text color reading
	}

	return pt, nil
}

// calculateBPP calculates bits per pixel from number of colors
func (r *Reader) calculateBPP(ncolors int) int {
	if ncolors <= 2 {
		return 1
	} else if ncolors <= 4 {
		return 2
	} else if ncolors <= 16 {
		return 4
	}
	return 8
}

// readLabels reads the label section
// Returns labels map, bytes read, and error
func (r *Reader) readLabels(buf []byte) (map[string]string, int, error) {
	if len(buf) < 1 {
		return nil, 0, fmt.Errorf("buffer too small for labels")
	}

	labels := make(map[string]string)
	pos := 0

	// Read length (1 or 2 bytes)
	length := int(buf[pos])
	n := 1 // number of bytes used for length field

	if (length & 0x01) == 0 {
		// 2-byte length
		if len(buf) < 2 {
			return nil, 0, fmt.Errorf("buffer too small for 2-byte length")
		}
		n = 2
		length = int(buf[pos]) | (int(buf[pos+1]) << 8)
	}

	pos += n
	length -= n

	// Read label entries
	for length > 0 && pos < len(buf) {
		// Read language code
		if pos >= len(buf) {
			break
		}
		langCode := buf[pos]
		pos++
		length -= 2 * n

		// Read null-terminated string
		strStart := pos
		for pos < len(buf) && buf[pos] != 0 {
			pos++
			length -= 2 * n
		}

		if pos >= len(buf) {
			break
		}

		// Decode string
		labelText, _ := r.decodeString(buf[strStart:pos])
		labels[fmt.Sprintf("%02x", langCode)] = labelText

		pos++ // Skip null terminator
	}

	return labels, pos, nil
}

// readPointType reads a single point type entry (OLD FUNCTION - DEPRECATED)
// Returns the point type, number of bytes read, and any error
func (r *Reader) readPointType(offset int64) (model.PointType, int, error) {
	// Allocate buffer for reading (max reasonable size)
	bufSize := 4096 // Increase buffer size
	buf := make([]byte, bufSize)
	n, err := r.r.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return model.PointType{}, 0, err
	}
	buf = buf[:n] // Trim to actual bytes read

	pos := 0

	// Need at least 4 bytes for type code, subtype, flags
	if len(buf) < 4 {
		return model.PointType{}, 0, fmt.Errorf("buffer too small: %d bytes", len(buf))
	}

	// Bytes 0-1: Type code (uint16)
	typeCode := r.endian.Uint16(buf[pos : pos+2])
	pos += 2

	// Byte 2: SubType
	subType := buf[pos]
	pos++

	// Byte 3: Flags
	flags := buf[pos]
	pos++

	pt := model.PointType{
		Type:    int(typeCode),
		SubType: int(subType),
		Labels:  make(map[string]string),
	}

	// Check if has icon (bit 0 of flags)
	if flags&0x01 != 0 {
		bitmap, size, err := r.readBitmap(offset + int64(pos))
		if err != nil {
			return model.PointType{}, 0, fmt.Errorf("read icon bitmap: %w", err)
		}
		pt.Icon = bitmap
		pos += size
	}

	// Check bounds before reading label count
	if pos >= len(buf) {
		return model.PointType{}, 0, fmt.Errorf("unexpected end of data at label count")
	}

	// Read number of labels
	labelCount := int(buf[pos])
	pos++

	// Read each label
	for i := 0; i < labelCount; i++ {
		if pos >= len(buf) {
			return model.PointType{}, 0, fmt.Errorf("unexpected end of data in label %d", i)
		}

		langCode := buf[pos]
		pos++

		// Read null-terminated string
		strEnd := pos
		for strEnd < len(buf) && buf[strEnd] != 0 {
			strEnd++
		}

		if strEnd >= len(buf) {
			return model.PointType{}, 0, fmt.Errorf("unterminated label string")
		}

		labelText, _ := r.decodeString(buf[pos:strEnd])
		pt.Labels[fmt.Sprintf("%02x", langCode)] = labelText
		pos = strEnd + 1 // Skip null terminator
	}

	// Check if has day color (bit 1 of flags)
	if flags&0x02 != 0 {
		if pos+3 > len(buf) {
			return model.PointType{}, 0, fmt.Errorf("unexpected end of data at day color")
		}
		pt.DayColor = model.Color{
			R:     buf[pos],
			G:     buf[pos+1],
			B:     buf[pos+2],
			Alpha: 255, // Assume opaque
		}
		pos += 3
	}

	// Check if has night color (bit 2 of flags)
	if flags&0x04 != 0 {
		if pos+3 > len(buf) {
			return model.PointType{}, 0, fmt.Errorf("unexpected end of data at night color")
		}
		pt.NightColor = model.Color{
			R:     buf[pos],
			G:     buf[pos+1],
			B:     buf[pos+2],
			Alpha: 255, // Assume opaque
		}
		pos += 3
	}

	// TODO: Parse font style if present (need to determine flag bit)

	return pt, pos, nil
}

// ReadLineTypes reads all line type definitions using the index array
func (r *Reader) ReadLineTypes(section SectionInfo) ([]model.LineType, error) {
	if section.ArrayModulo == 0 || (section.ArraySize%uint32(section.ArrayModulo)) != 0 {
		return nil, nil // Empty or invalid array
	}

	numEntries := int(section.ArraySize / uint32(section.ArrayModulo))
	lines := make([]model.LineType, 0, numEntries)

	for i := 0; i < numEntries; i++ {
		// Read array entry
		arrayPos := int64(section.ArrayOffset) + int64(i)*int64(section.ArrayModulo)
		typCode, _, err := r.readArrayEntry(arrayPos, section.ArrayModulo)
		if err != nil {
			return nil, fmt.Errorf("read array entry %d: %w", i, err)
		}

		// Decode type/subtype
		typ, subtyp := r.decodeTypeSubtype(typCode)

		// Read line data (simplified for now - TODO: read actual data using dataOffset)
		lt := model.LineType{
			Type:    int(typ),
			SubType: int(subtyp),
			Labels:  make(map[string]string),
		}

		lines = append(lines, lt)
	}

	return lines, nil
}

// readLineType reads a single line type entry
func (r *Reader) readLineType(offset int64) (model.LineType, int, error) {
	// Allocate buffer for reading (max reasonable size)
	bufSize := 4096
	buf := make([]byte, bufSize)
	n, err := r.r.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return model.LineType{}, 0, err
	}
	buf = buf[:n] // Trim to actual bytes read

	pos := 0

	// Need at least 4 bytes for type code, subtype, flags
	if len(buf) < 4 {
		return model.LineType{}, 0, fmt.Errorf("buffer too small: %d bytes", len(buf))
	}

	// Similar structure to point types
	typeCode := r.endian.Uint16(buf[pos : pos+2])
	pos += 2

	subType := buf[pos]
	pos++

	flags := buf[pos]
	pos++

	lt := model.LineType{
		Type:    int(typeCode),
		SubType: int(subType),
		Labels:  make(map[string]string),
	}

	// TODO: Parse line-specific fields (width, border, line style)
	// For now, just parse labels and colors similar to points

	// Skip pattern if present (bit 0)
	if flags&0x01 != 0 {
		_, size, err := r.readBitmap(offset + int64(pos))
		if err != nil {
			// Continue anyway
		} else {
			pos += size
		}
	}

	// Check bounds before reading label count
	if pos >= len(buf) {
		return model.LineType{}, 0, fmt.Errorf("unexpected end of data at label count")
	}

	// Read labels
	labelCount := int(buf[pos])
	pos++

	for i := 0; i < labelCount; i++ {
		if pos >= len(buf) {
			return model.LineType{}, 0, fmt.Errorf("unexpected end of data in label %d", i)
		}

		langCode := buf[pos]
		pos++

		// Read null-terminated string with bounds check
		strEnd := pos
		for strEnd < len(buf) && buf[strEnd] != 0 {
			strEnd++
		}

		if strEnd >= len(buf) {
			return model.LineType{}, 0, fmt.Errorf("unterminated label string")
		}

		labelText, _ := r.decodeString(buf[pos:strEnd])
		lt.Labels[fmt.Sprintf("%02x", langCode)] = labelText
		pos = strEnd + 1 // Skip null terminator
	}

	// Colors (if present)
	if flags&0x02 != 0 {
		if pos+3 > len(buf) {
			return model.LineType{}, 0, fmt.Errorf("unexpected end of data at day color")
		}
		lt.DayColor = model.Color{R: buf[pos], G: buf[pos+1], B: buf[pos+2], Alpha: 255}
		pos += 3
	}
	if flags&0x04 != 0 {
		if pos+3 > len(buf) {
			return model.LineType{}, 0, fmt.Errorf("unexpected end of data at night color")
		}
		lt.NightColor = model.Color{R: buf[pos], G: buf[pos+1], B: buf[pos+2], Alpha: 255}
		pos += 3
	}

	return lt, pos, nil
}

// ReadPolygonTypes reads all polygon type definitions using the index array
func (r *Reader) ReadPolygonTypes(section SectionInfo) ([]model.PolygonType, error) {
	if section.ArrayModulo == 0 || (section.ArraySize%uint32(section.ArrayModulo)) != 0 {
		return nil, nil // Empty or invalid array
	}

	numEntries := int(section.ArraySize / uint32(section.ArrayModulo))
	polygons := make([]model.PolygonType, 0, numEntries)

	for i := 0; i < numEntries; i++ {
		// Read array entry
		arrayPos := int64(section.ArrayOffset) + int64(i)*int64(section.ArrayModulo)
		typCode, _, err := r.readArrayEntry(arrayPos, section.ArrayModulo)
		if err != nil {
			return nil, fmt.Errorf("read array entry %d: %w", i, err)
		}

		// Decode type/subtype
		typ, subtyp := r.decodeTypeSubtype(typCode)

		// Read polygon data (simplified for now - TODO: read actual data using dataOffset)
		poly := model.PolygonType{
			Type:    int(typ),
			SubType: int(subtyp),
			Labels:  make(map[string]string),
		}

		polygons = append(polygons, poly)
	}

	return polygons, nil
}

// readPolygonType reads a single polygon type entry
func (r *Reader) readPolygonType(offset int64) (model.PolygonType, int, error) {
	// Allocate buffer for reading (max reasonable size)
	bufSize := 4096
	buf := make([]byte, bufSize)
	n, err := r.r.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return model.PolygonType{}, 0, err
	}
	buf = buf[:n] // Trim to actual bytes read

	pos := 0

	// Need at least 4 bytes for type code, subtype, flags
	if len(buf) < 4 {
		return model.PolygonType{}, 0, fmt.Errorf("buffer too small: %d bytes", len(buf))
	}

	// Similar structure to point types
	typeCode := r.endian.Uint16(buf[pos : pos+2])
	pos += 2

	subType := buf[pos]
	pos++

	flags := buf[pos]
	pos++

	poly := model.PolygonType{
		Type:    int(typeCode),
		SubType: int(subType),
		Labels:  make(map[string]string),
	}

	// Skip pattern if present (bit 0)
	if flags&0x01 != 0 {
		_, size, err := r.readBitmap(offset + int64(pos))
		if err != nil {
			// Continue anyway
		} else {
			pos += size
		}
	}

	// Check bounds before reading label count
	if pos >= len(buf) {
		return model.PolygonType{}, 0, fmt.Errorf("unexpected end of data at label count")
	}

	// Read labels
	labelCount := int(buf[pos])
	pos++

	for i := 0; i < labelCount; i++ {
		if pos >= len(buf) {
			return model.PolygonType{}, 0, fmt.Errorf("unexpected end of data in label %d", i)
		}

		langCode := buf[pos]
		pos++

		// Read null-terminated string with bounds check
		strEnd := pos
		for strEnd < len(buf) && buf[strEnd] != 0 {
			strEnd++
		}

		if strEnd >= len(buf) {
			return model.PolygonType{}, 0, fmt.Errorf("unterminated label string")
		}

		labelText, _ := r.decodeString(buf[pos:strEnd])
		poly.Labels[fmt.Sprintf("%02x", langCode)] = labelText
		pos = strEnd + 1 // Skip null terminator
	}

	// Colors (if present)
	if flags&0x02 != 0 {
		if pos+3 > len(buf) {
			return model.PolygonType{}, 0, fmt.Errorf("unexpected end of data at day color")
		}
		poly.DayColor = model.Color{R: buf[pos], G: buf[pos+1], B: buf[pos+2], Alpha: 255}
		pos += 3
	}
	if flags&0x04 != 0 {
		if pos+3 > len(buf) {
			return model.PolygonType{}, 0, fmt.Errorf("unexpected end of data at night color")
		}
		poly.NightColor = model.Color{R: buf[pos], G: buf[pos+1], B: buf[pos+2], Alpha: 255}
		pos += 3
	}

	return poly, pos, nil
}

// readBitmap reads bitmap data at the specified offset
// Returns the bitmap, number of bytes read, and any error
func (r *Reader) readBitmap(offset int64) (*model.Bitmap, int, error) {
	buf := make([]byte, 4096) // Max reasonable bitmap size
	n, err := r.r.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return nil, 0, err
	}
	buf = buf[:n]

	pos := 0

	// Need at least 4 bytes
	if len(buf) < 4 {
		return nil, 0, fmt.Errorf("bitmap buffer too small: %d bytes", len(buf))
	}

	// Byte 0: Width
	width := int(buf[pos])
	pos++

	// Byte 1: Height
	height := int(buf[pos])
	pos++

	// Byte 2: Color mode
	colorMode := buf[pos]
	pos++

	// Byte 3: Number of colors in palette
	numColors := int(buf[pos])
	pos++

	// Sanity check
	if width == 0 || height == 0 || width > 256 || height > 256 {
		return nil, 0, fmt.Errorf("invalid bitmap dimensions: %dx%d", width, height)
	}
	if numColors > 256 {
		return nil, 0, fmt.Errorf("invalid color count: %d", numColors)
	}

	bmp := &model.Bitmap{
		Width:     width,
		Height:    height,
		ColorMode: mapColorMode(colorMode),
		Palette:   make([]model.Color, numColors),
	}

	// Check we have enough data for palette
	if pos+numColors*3 > len(buf) {
		return nil, 0, fmt.Errorf("insufficient data for palette: need %d bytes, have %d", numColors*3, len(buf)-pos)
	}

	// Read palette (RGB triples)
	for i := 0; i < numColors; i++ {
		bmp.Palette[i] = model.Color{
			R:     buf[pos],
			G:     buf[pos+1],
			B:     buf[pos+2],
			Alpha: 255, // Assume opaque unless R=G=B=0
		}
		// Check for transparency marker
		if bmp.Palette[i].R == 0 && bmp.Palette[i].G == 0 && bmp.Palette[i].B == 0 {
			bmp.Palette[i].Alpha = 0
		}
		pos += 3
	}

	// Calculate pixel data size
	pixelDataSize := width * height
	if colorMode == 4 { // 4-bit mode (2 pixels per byte)
		pixelDataSize = (width*height + 1) / 2
	}

	// Check we have enough data for pixels
	if pos+pixelDataSize > len(buf) {
		return nil, 0, fmt.Errorf("insufficient data for pixels: need %d bytes, have %d", pixelDataSize, len(buf)-pos)
	}

	bmp.Data = make([]byte, width*height)

	if colorMode == 4 {
		// 4-bit mode: unpack 2 pixels per byte
		for i := 0; i < width*height; i += 2 {
			b := buf[pos]
			pos++
			bmp.Data[i] = (b >> 4) & 0x0F
			if i+1 < width*height {
				bmp.Data[i+1] = b & 0x0F
			}
		}
	} else if colorMode == 8 || colorMode == 1 {
		// 8-bit mode or monochrome: one byte per pixel
		copy(bmp.Data, buf[pos:pos+pixelDataSize])
		pos += pixelDataSize
	} else {
		// True color or unknown mode
		copy(bmp.Data, buf[pos:pos+pixelDataSize])
		pos += pixelDataSize
	}

	return bmp, pos, nil
}

// mapColorMode maps the binary color mode value to our enum
func mapColorMode(mode byte) model.ColorMode {
	switch mode {
	case 1:
		return model.Monochrome
	case 4:
		return model.Color16
	case 8:
		return model.Color256
	case 32:
		return model.TrueColor
	default:
		return model.Color256 // Default to 8-bit
	}
}

// decodeString decodes a byte slice using the configured codepage decoder
func (r *Reader) decodeString(data []byte) (string, error) {
	if r.decoder == nil {
		// No decoder set, return as-is (shouldn't happen after ReadHeader)
		return string(data), nil
	}
	decoded, err := r.decoder.Bytes(data)
	if err != nil {
		return string(data), err // Fall back to raw string on error
	}
	return string(decoded), nil
}

// readString reads a null-terminated string at the specified offset
func (r *Reader) readString(offset int64, maxLen int) (string, int, error) {
	buf := make([]byte, maxLen)
	if _, err := r.r.ReadAt(buf, offset); err != nil {
		return "", 0, err
	}

	// Find null terminator
	for i, b := range buf {
		if b == 0 {
			decoded, _ := r.decodeString(buf[:i])
			return decoded, i + 1, nil
		}
	}

	// No null terminator found within maxLen
	decoded, _ := r.decodeString(buf)
	return decoded, maxLen, nil
}
