package binary

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/dyuri/typconv/internal/model"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// Writer handles writing TYP files to binary format
type Writer struct {
	w        io.Writer
	endian   binary.ByteOrder
	encoding encoding.Encoding // Text encoding for strings (based on codepage)

	// Accumulated sections during write
	pointsData    *bytes.Buffer
	polylinesData *bytes.Buffer
	polygonsData  *bytes.Buffer

	pointsArray    *bytes.Buffer
	polylinesArray *bytes.Buffer
	polygonsArray  *bytes.Buffer
	orderArray     *bytes.Buffer
}

// NewWriter creates a new binary TYP writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:              w,
		endian:         binary.LittleEndian,
		pointsData:     &bytes.Buffer{},
		polylinesData:  &bytes.Buffer{},
		polygonsData:   &bytes.Buffer{},
		pointsArray:    &bytes.Buffer{},
		polylinesArray: &bytes.Buffer{},
		polygonsArray:  &bytes.Buffer{},
		orderArray:     &bytes.Buffer{},
	}
}

// Write writes a complete TYP file to binary format
func (w *Writer) Write(typ *model.TYPFile) error {
	// Set up text encoder based on CodePage
	if err := w.setupEncoder(typ.Header.CodePage); err != nil {
		return fmt.Errorf("setup encoder: %w", err)
	}

	// Write point types
	if err := w.writePointTypes(typ.Points); err != nil {
		return fmt.Errorf("write point types: %w", err)
	}

	// Write line types
	if err := w.writeLineTypes(typ.Lines); err != nil {
		return fmt.Errorf("write line types: %w", err)
	}

	// Write polygon types
	if err := w.writePolygonTypes(typ.Polygons); err != nil {
		return fmt.Errorf("write polygon types: %w", err)
	}

	// Write draw order
	if err := w.writeDrawOrder(typ); err != nil {
		return fmt.Errorf("write draw order: %w", err)
	}

	// Calculate all offsets
	headerSize := uint32(0x5B)

	pointsArrayOffset := headerSize
	pointsArraySize := uint32(w.pointsArray.Len())

	polylinesArrayOffset := pointsArrayOffset + pointsArraySize
	polylinesArraySize := uint32(w.polylinesArray.Len())

	polygonsArrayOffset := polylinesArrayOffset + polylinesArraySize
	polygonsArraySize := uint32(w.polygonsArray.Len())

	orderArrayOffset := polygonsArrayOffset + polygonsArraySize
	orderArraySize := uint32(w.orderArray.Len())

	pointsDataOffset := orderArrayOffset + orderArraySize
	pointsDataSize := uint32(w.pointsData.Len())

	polylinesDataOffset := pointsDataOffset + pointsDataSize
	polylinesDataSize := uint32(w.polylinesData.Len())

	polygonsDataOffset := polylinesDataOffset + polylinesDataSize
	polygonsDataSize := uint32(w.polygonsData.Len())

	// Determine array modulo (size of each array entry)
	// Use 5 bytes if any offset is > 65535 (3-byte offset), otherwise 4 bytes (2-byte offset)
	pointsModulo := uint16(4)
	if pointsDataSize > 65535 {
		pointsModulo = 5
	}

	polylinesModulo := uint16(4)
	if polylinesDataSize > 65535 {
		polylinesModulo = 5
	}

	polygonsModulo := uint16(4)
	if polygonsDataSize > 65535 {
		polygonsModulo = 5
	}

	orderModulo := uint16(3) // Draw order typically uses 3-byte entries

	// Write header
	if err := w.writeHeader(&typ.Header, headerInfo{
		pointsDataOffset:     pointsDataOffset,
		pointsDataSize:       pointsDataSize,
		polylinesDataOffset:  polylinesDataOffset,
		polylinesDataSize:    polylinesDataSize,
		polygonsDataOffset:   polygonsDataOffset,
		polygonsDataSize:     polygonsDataSize,
		pointsArrayOffset:    pointsArrayOffset,
		pointsArrayModulo:    pointsModulo,
		pointsArraySize:      pointsArraySize,
		polylinesArrayOffset: polylinesArrayOffset,
		polylinesArrayModulo: polylinesModulo,
		polylinesArraySize:   polylinesArraySize,
		polygonsArrayOffset:  polygonsArrayOffset,
		polygonsArrayModulo:  polygonsModulo,
		polygonsArraySize:    polygonsArraySize,
		orderArrayOffset:     orderArrayOffset,
		orderArrayModulo:     orderModulo,
		orderArraySize:       orderArraySize,
	}); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write arrays and data sections in order
	if _, err := w.pointsArray.WriteTo(w.w); err != nil {
		return fmt.Errorf("write points array: %w", err)
	}
	if _, err := w.polylinesArray.WriteTo(w.w); err != nil {
		return fmt.Errorf("write polylines array: %w", err)
	}
	if _, err := w.polygonsArray.WriteTo(w.w); err != nil {
		return fmt.Errorf("write polygons array: %w", err)
	}
	if _, err := w.orderArray.WriteTo(w.w); err != nil {
		return fmt.Errorf("write order array: %w", err)
	}
	if _, err := w.pointsData.WriteTo(w.w); err != nil {
		return fmt.Errorf("write points data: %w", err)
	}
	if _, err := w.polylinesData.WriteTo(w.w); err != nil {
		return fmt.Errorf("write polylines data: %w", err)
	}
	if _, err := w.polygonsData.WriteTo(w.w); err != nil {
		return fmt.Errorf("write polygons data: %w", err)
	}

	return nil
}

// headerInfo contains calculated offsets for the header
type headerInfo struct {
	pointsDataOffset     uint32
	pointsDataSize       uint32
	polylinesDataOffset  uint32
	polylinesDataSize    uint32
	polygonsDataOffset   uint32
	polygonsDataSize     uint32
	pointsArrayOffset    uint32
	pointsArrayModulo    uint16
	pointsArraySize      uint32
	polylinesArrayOffset uint32
	polylinesArrayModulo uint16
	polylinesArraySize   uint32
	polygonsArrayOffset  uint32
	polygonsArrayModulo  uint16
	polygonsArraySize    uint32
	orderArrayOffset     uint32
	orderArrayModulo     uint16
	orderArraySize       uint32
}

// setupEncoder sets up the text encoder based on CodePage
func (w *Writer) setupEncoder(codePage int) error {
	switch codePage {
	case 1252:
		w.encoding = charmap.Windows1252
	case 1250:
		w.encoding = charmap.Windows1250
	case 65001:
		// UTF-8 - no encoding needed
		w.encoding = nil
	default:
		// Default to Windows-1252
		w.encoding = charmap.Windows1252
	}

	return nil
}

// encodeString encodes a string using the configured CodePage
// Unsupported characters are replaced with '?' instead of causing errors
func (w *Writer) encodeString(s string) ([]byte, error) {
	if w.encoding == nil {
		// UTF-8 - no encoding needed
		return []byte(s), nil
	}

	// Encode character by character to handle unsupported runes gracefully
	result := make([]byte, 0, len(s))
	for _, r := range s {
		// Create a fresh encoder for each character to avoid state issues
		encoder := w.encoding.NewEncoder()
		b, err := encoder.Bytes([]byte(string(r)))
		if err != nil {
			// Character can't be encoded, use '?'
			result = append(result, '?')
		} else {
			result = append(result, b...)
		}
	}
	return result, nil
}

// writeHeader writes the TYP file header
func (w *Writer) writeHeader(header *model.Header, info headerInfo) error {
	buf := make([]byte, 0x5B)

	// Offset 0x00-0x01: Descriptor (header size)
	w.endian.PutUint16(buf[0x00:0x02], 0x5B)

	// Offset 0x02-0x0B: "GARMIN TYP" signature
	copy(buf[0x02:0x0C], "GARMIN TYP")

	// Offset 0x0C-0x0D: Version
	version := uint16(1)
	if header.Version > 0 {
		version = uint16(header.Version)
	}
	w.endian.PutUint16(buf[0x0C:0x0E], version)

	// Offset 0x0E-0x14: Date/time (use current time)
	now := time.Now()
	year := now.Year() - 1900
	month := int(now.Month()) - 1 // 0-based
	day := now.Day()
	hour := now.Hour()
	minutes := now.Minute()
	seconds := now.Second()

	w.endian.PutUint16(buf[0x0E:0x10], uint16(year))
	buf[0x10] = byte(month)
	buf[0x11] = byte(day)
	buf[0x12] = byte(hour)
	buf[0x13] = byte(minutes)
	buf[0x14] = byte(seconds)

	// Offset 0x15-0x16: CodePage
	codePage := header.CodePage
	if codePage == 0 {
		codePage = 1252 // Default to Windows-1252
	}
	w.endian.PutUint16(buf[0x15:0x17], uint16(codePage))

	// Section data pointers
	w.endian.PutUint32(buf[0x17:0x1B], info.pointsDataOffset)
	w.endian.PutUint32(buf[0x1B:0x1F], info.pointsDataSize)
	w.endian.PutUint32(buf[0x1F:0x23], info.polylinesDataOffset)
	w.endian.PutUint32(buf[0x23:0x27], info.polylinesDataSize)
	w.endian.PutUint32(buf[0x27:0x2B], info.polygonsDataOffset)
	w.endian.PutUint32(buf[0x2B:0x2F], info.polygonsDataSize)

	// Offset 0x2F-0x30: PID
	w.endian.PutUint16(buf[0x2F:0x31], uint16(header.PID))

	// Offset 0x31-0x32: FID
	w.endian.PutUint16(buf[0x31:0x33], uint16(header.FID))

	// Array metadata
	w.endian.PutUint32(buf[0x33:0x37], info.pointsArrayOffset)
	w.endian.PutUint16(buf[0x37:0x39], info.pointsArrayModulo)
	w.endian.PutUint32(buf[0x39:0x3D], info.pointsArraySize)

	w.endian.PutUint32(buf[0x3D:0x41], info.polylinesArrayOffset)
	w.endian.PutUint16(buf[0x41:0x43], info.polylinesArrayModulo)
	w.endian.PutUint32(buf[0x43:0x47], info.polylinesArraySize)

	w.endian.PutUint32(buf[0x47:0x4B], info.polygonsArrayOffset)
	w.endian.PutUint16(buf[0x4B:0x4D], info.polygonsArrayModulo)
	w.endian.PutUint32(buf[0x4D:0x51], info.polygonsArraySize)

	w.endian.PutUint32(buf[0x51:0x55], info.orderArrayOffset)
	w.endian.PutUint16(buf[0x55:0x57], info.orderArrayModulo)
	w.endian.PutUint32(buf[0x57:0x5B], info.orderArraySize)

	// Write header
	if _, err := w.w.Write(buf); err != nil {
		return err
	}

	return nil
}

// encodeTypeSubtype encodes type and subtype into the bit-packed format
func (w *Writer) encodeTypeSubtype(typ, subtyp uint32) uint16 {
	// Reverse of decodeTypeSubtype
	var t16 uint16

	// Check if this is an extended type
	if typ >= 0x10000 {
		// Extended type: has bit 13 set
		t16 = 0x2000
		// Extract original type and subtype
		subtyp = typ & 0xFF
		typ = (typ >> 8) & 0x7FF
	} else {
		// Normal type: extract type and subtype
		subtyp = typ & 0xFF
		typ = typ >> 8
	}

	// Pack: bottom 11 bits are type, top 5 bits are subtype
	t16_2 := (uint16(typ) & 0x7FF) | (uint16(subtyp) << 11)

	// Reverse the bit shuffling from decodeTypeSubtype
	t16 |= (t16_2 << 5) & 0xFFE0
	t16 |= (t16_2 >> 11) & 0x001F

	return t16
}

// writePointTypes writes all point type definitions
func (w *Writer) writePointTypes(points []model.PointType) error {
	for i, pt := range points {
		// Get data offset before writing
		dataOffset := w.pointsData.Len()

		// Write point data to buffer
		if err := w.writePointData(&pt); err != nil {
			return fmt.Errorf("write point %d: %w", i, err)
		}

		// Write array entry
		typeCode := w.encodeTypeSubtype(uint32(pt.Type), uint32(pt.SubType))
		if err := w.writeArrayEntry(w.pointsArray, typeCode, uint32(dataOffset)); err != nil {
			return fmt.Errorf("write point array entry %d: %w", i, err)
		}
	}
	return nil
}

// writePointData writes a single point type definition to the data buffer
func (w *Writer) writePointData(pt *model.PointType) error {
	buf := &bytes.Buffer{}

	// Determine flags
	hasLabels := len(pt.Labels) > 0
	hasTextColors := false // TODO: Implement text color support
	dayNightMode := uint8(0)

	if pt.DayIcon != nil && pt.NightIcon != nil {
		dayNightMode = 0x03 // Separate night bitmap
	} else if pt.NightIcon != nil {
		dayNightMode = 0x02 // Night mode only
	} else if pt.DayIcon != nil {
		dayNightMode = 0x01 // Day mode only
	}

	flags := dayNightMode
	if hasLabels {
		flags |= 0x04
	}
	if hasTextColors {
		flags |= 0x08
	}

	// Get icon properties (from day icon if available)
	width, height, ncolors, ctype := byte(0), byte(0), byte(0), byte(0)
	if pt.DayIcon != nil {
		width = byte(pt.DayIcon.Width)
		height = byte(pt.DayIcon.Height)
		ncolors = byte(len(pt.DayIcon.Palette))
		ctype = 0x10 // Default color type
	}

	// Write header (5 bytes)
	buf.WriteByte(flags)
	buf.WriteByte(width)
	buf.WriteByte(height)
	buf.WriteByte(ncolors)
	buf.WriteByte(ctype)

	// Write day color table
	if pt.DayIcon != nil && len(pt.DayIcon.Palette) > 0 {
		if err := w.writeColorTable(buf, pt.DayIcon.Palette); err != nil {
			return fmt.Errorf("write day color table: %w", err)
		}
	}

	// Write day bitmap
	if pt.DayIcon != nil {
		bpp := w.calculateBPP(len(pt.DayIcon.Palette))
		if err := w.writeBitmap(buf, pt.DayIcon.Data, width, height, bpp); err != nil {
			return fmt.Errorf("write day bitmap: %w", err)
		}
	}

	// Write night bitmap if separate
	if dayNightMode == 0x03 && pt.NightIcon != nil {
		nightNcolors := byte(len(pt.NightIcon.Palette))
		nightCtype := byte(0x10)
		buf.WriteByte(nightNcolors)
		buf.WriteByte(nightCtype)

		// Write night color table
		if err := w.writeColorTable(buf, pt.NightIcon.Palette); err != nil {
			return fmt.Errorf("write night color table: %w", err)
		}

		// Write night bitmap
		nightBpp := w.calculateBPP(len(pt.NightIcon.Palette))
		if err := w.writeBitmap(buf, pt.NightIcon.Data, byte(pt.NightIcon.Width), byte(pt.NightIcon.Height), nightBpp); err != nil {
			return fmt.Errorf("write night bitmap: %w", err)
		}
	}

	// Write labels
	if hasLabels {
		if err := w.writeLabels(buf, pt.Labels); err != nil {
			return fmt.Errorf("write labels: %w", err)
		}
	}

	// Write to points data buffer
	if _, err := buf.WriteTo(w.pointsData); err != nil {
		return err
	}

	return nil
}

// calculateBPP determines bits per pixel based on palette size
func (w *Writer) calculateBPP(ncolors int) int {
	switch {
	case ncolors <= 2:
		return 1
	case ncolors <= 4:
		return 2
	case ncolors <= 16:
		return 4
	default:
		return 8
	}
}

// writeColorTable writes a color palette in BGR format
func (w *Writer) writeColorTable(buf *bytes.Buffer, palette []model.Color) error {
	for _, color := range palette {
		// Colors are stored as BGR (not RGB!)
		buf.WriteByte(color.B)
		buf.WriteByte(color.G)
		buf.WriteByte(color.R)
	}
	return nil
}

// writeBitmap writes bit-packed pixel data
func (w *Writer) writeBitmap(buf *bytes.Buffer, pixelData []byte, width, height byte, bpp int) error {
	totalPixels := int(width) * int(height)
	if len(pixelData) != totalPixels {
		return fmt.Errorf("pixel data size mismatch: expected %d, got %d", totalPixels, len(pixelData))
	}

	// Calculate bitmap size in bytes (bit-packed)
	bitsTotal := totalPixels * bpp
	bytesNeeded := bitsTotal / 8
	if bitsTotal%8 != 0 {
		bytesNeeded++
	}

	// Pack pixels based on bits per pixel
	packedData := make([]byte, bytesNeeded)

	switch bpp {
	case 1:
		// 1 bpp: 8 pixels per byte
		for i := 0; i < totalPixels; i++ {
			byteIdx := i / 8
			bitIdx := 7 - (i % 8) // MSB first
			if pixelData[i] > 0 {
				packedData[byteIdx] |= 1 << bitIdx
			}
		}
	case 2:
		// 2 bpp: 4 pixels per byte
		for i := 0; i < totalPixels; i++ {
			byteIdx := i / 4
			pixelInByte := 3 - (i % 4) // MSB first
			packedData[byteIdx] |= (pixelData[i] & 0x03) << (pixelInByte * 2)
		}
	case 4:
		// 4 bpp: 2 pixels per byte
		for i := 0; i < totalPixels; i++ {
			byteIdx := i / 2
			if i%2 == 0 {
				// High nibble
				packedData[byteIdx] |= (pixelData[i] & 0x0F) << 4
			} else {
				// Low nibble
				packedData[byteIdx] |= pixelData[i] & 0x0F
			}
		}
	case 8:
		// 8 bpp: 1 pixel per byte
		copy(packedData, pixelData)
	default:
		return fmt.Errorf("unsupported bpp: %d", bpp)
	}

	buf.Write(packedData)
	return nil
}

// writeLabels writes the label section with special length counting
func (w *Writer) writeLabels(buf *bytes.Buffer, labels map[string]string) error {
	// Build labels data first to calculate length
	labelsBuf := &bytes.Buffer{}

	for langCodeStr, text := range labels {
		// Parse language code
		var langCode byte
		if _, err := fmt.Sscanf(langCodeStr, "%x", &langCode); err != nil {
			return fmt.Errorf("invalid language code %q: %w", langCodeStr, err)
		}

		// Encode label text
		encoded, err := w.encodeString(text)
		if err != nil {
			return fmt.Errorf("encode label: %w", err)
		}

		// Write language code
		labelsBuf.WriteByte(langCode)

		// Write null-terminated string
		labelsBuf.Write(encoded)
		labelsBuf.WriteByte(0)
	}

	// Calculate length using special counting
	// In QMapShack's algorithm: each byte in the data costs 2*n in the length counter
	// where n is the number of bytes in the length field itself
	labelsData := labelsBuf.Bytes()
	actualLength := len(labelsData)

	// Try n=1 first (1-byte length field)
	n := 1
	// length_value = actualLength * 2*n + n  = actualLength * 2 + 1
	length := actualLength*2 + n

	if length > 255 {
		// Need 2-byte length field
		n = 2
		// length_value = actualLength * 2*n + n = actualLength * 4 + 2
		length = actualLength*4 + n

		// Write 2-byte length (bit 0 clear indicates 2-byte field)
		length16 := uint16(length) & 0xFFFE // Clear bit 0
		buf.WriteByte(byte(length16 & 0xFF))
		buf.WriteByte(byte(length16 >> 8))
	} else {
		// Write 1-byte length (bit 0 set indicates 1-byte field)
		buf.WriteByte(byte(length | 0x01))
	}

	// Write label data
	buf.Write(labelsData)

	return nil
}

// writeArrayEntry writes an array entry (type code + data offset)
func (w *Writer) writeArrayEntry(arrayBuf *bytes.Buffer, typeCode uint16, dataOffset uint32) error {
	// Write type code (2 bytes)
	typeBuf := make([]byte, 2)
	w.endian.PutUint16(typeBuf, typeCode)
	arrayBuf.Write(typeBuf)

	// Write offset (2 bytes for now, will adjust if needed)
	offsetBuf := make([]byte, 2)
	w.endian.PutUint16(offsetBuf, uint16(dataOffset))
	arrayBuf.Write(offsetBuf)

	return nil
}

// writeLineTypes writes all line type definitions
func (w *Writer) writeLineTypes(lines []model.LineType) error {
	for i, lt := range lines {
		dataOffset := w.polylinesData.Len()

		if err := w.writeLineData(&lt); err != nil {
			return fmt.Errorf("write line %d: %w", i, err)
		}

		typeCode := w.encodeTypeSubtype(uint32(lt.Type), uint32(lt.SubType))
		if err := w.writeArrayEntry(w.polylinesArray, typeCode, uint32(dataOffset)); err != nil {
			return fmt.Errorf("write line array entry %d: %w", i, err)
		}
	}
	return nil
}

// writeLineData writes a single line type definition
func (w *Writer) writeLineData(lt *model.LineType) error {
	buf := &bytes.Buffer{}

	// Determine color type and pattern height
	ctyp := w.determineLineColorType(lt)
	rows := 0
	if lt.DayPattern != nil {
		rows = lt.DayPattern.Height
	}

	ctypRows := byte(ctyp | (rows << 3))

	// Determine flags
	hasLabels := len(lt.Labels) > 0
	hasTextColors := false

	flags := byte(0)
	if hasLabels {
		flags |= 0x01
	}
	if hasTextColors {
		flags |= 0x04
	}

	// Write header (2 bytes)
	buf.WriteByte(ctypRows)
	buf.WriteByte(flags)

	// Write color/pattern data based on ctyp
	if err := w.writeLineColorData(buf, lt, ctyp, rows); err != nil {
		return fmt.Errorf("write line color data: %w", err)
	}

	// Write labels
	if hasLabels {
		if err := w.writeLabels(buf, lt.Labels); err != nil {
			return fmt.Errorf("write labels: %w", err)
		}
	}

	// Write to polylines data buffer
	if _, err := buf.WriteTo(w.polylinesData); err != nil {
		return err
	}

	return nil
}

// determineLineColorType determines the color type for a line
func (w *Writer) determineLineColorType(lt *model.LineType) int {
	hasDayPattern := lt.DayPattern != nil
	hasNightPattern := lt.NightPattern != nil

	if !hasDayPattern && !hasNightPattern {
		// Solid colors
		if lt.DayColor == lt.NightColor && lt.DayBorderColor == lt.NightBorderColor {
			return 0x00 // Same day/night
		}
		return 0x01 // Separate day/night
	}

	// Pattern mode
	// If only day pattern exists (no night), treat as same day/night
	if hasDayPattern && !hasNightPattern {
		return 0x00 // Same pattern for day/night
	}

	// If only night pattern exists (unusual), treat as same day/night
	if !hasDayPattern && hasNightPattern {
		return 0x00 // Same pattern for day/night
	}

	// Both patterns exist - check for transparency modes
	dayTransparent := len(lt.DayPattern.Palette) > 0 && lt.DayPattern.Palette[0].Alpha == 0
	nightTransparent := len(lt.NightPattern.Palette) > 0 && lt.NightPattern.Palette[0].Alpha == 0

	if dayTransparent && nightTransparent {
		return 0x07 // Both transparent
	} else if dayTransparent {
		return 0x03 // Day transparent, night solid
	} else if nightTransparent {
		return 0x04 // Day solid, night transparent
	}

	// Check if palettes are the same
	if w.palettesEqual(lt.DayPattern.Palette, lt.NightPattern.Palette) {
		return 0x00 // Same day/night
	}

	return 0x01 // Separate day/night
}

// palettesEqual checks if two color palettes are equal
func (w *Writer) palettesEqual(p1, p2 []model.Color) bool {
	if len(p1) != len(p2) {
		return false
	}
	for i := range p1 {
		if p1[i] != p2[i] {
			return false
		}
	}
	return true
}

// writeLineColorData writes color/pattern data for a line type
func (w *Writer) writeLineColorData(buf *bytes.Buffer, lt *model.LineType, ctyp, rows int) error {
	switch ctyp {
	case 0x00:
		// Single day/night mode
		if rows > 0 {
			// Pattern bitmap
			if lt.DayPattern == nil || len(lt.DayPattern.Palette) < 2 {
				return fmt.Errorf("day pattern missing or invalid")
			}
			// Write 2-color palette (BGR format)
			buf.WriteByte(lt.DayPattern.Palette[1].B)
			buf.WriteByte(lt.DayPattern.Palette[1].G)
			buf.WriteByte(lt.DayPattern.Palette[1].R)
			buf.WriteByte(lt.DayPattern.Palette[0].B)
			buf.WriteByte(lt.DayPattern.Palette[0].G)
			buf.WriteByte(lt.DayPattern.Palette[0].R)

			// Write pattern bitmap
			if err := w.writeBitmap(buf, lt.DayPattern.Data, 32, byte(rows), 1); err != nil {
				return err
			}
		} else {
			// Solid colors
			buf.WriteByte(lt.DayColor.B)
			buf.WriteByte(lt.DayColor.G)
			buf.WriteByte(lt.DayColor.R)
			buf.WriteByte(lt.DayBorderColor.B)
			buf.WriteByte(lt.DayBorderColor.G)
			buf.WriteByte(lt.DayBorderColor.R)
			buf.WriteByte(byte(lt.LineWidth))
			buf.WriteByte(byte(lt.BorderWidth))
		}

	case 0x01:
		// Separate day/night (both must exist)
		if rows > 0 {
			// Day and night pattern bitmaps
			if lt.DayPattern == nil || len(lt.DayPattern.Palette) < 2 {
				return fmt.Errorf("day pattern missing or invalid for color type 0x01")
			}
			if lt.NightPattern == nil || len(lt.NightPattern.Palette) < 2 {
				return fmt.Errorf("night pattern missing or invalid for color type 0x01")
			}

			// Day palette
			buf.WriteByte(lt.DayPattern.Palette[1].B)
			buf.WriteByte(lt.DayPattern.Palette[1].G)
			buf.WriteByte(lt.DayPattern.Palette[1].R)
			buf.WriteByte(lt.DayPattern.Palette[0].B)
			buf.WriteByte(lt.DayPattern.Palette[0].G)
			buf.WriteByte(lt.DayPattern.Palette[0].R)

			// Night palette
			buf.WriteByte(lt.NightPattern.Palette[1].B)
			buf.WriteByte(lt.NightPattern.Palette[1].G)
			buf.WriteByte(lt.NightPattern.Palette[1].R)
			buf.WriteByte(lt.NightPattern.Palette[0].B)
			buf.WriteByte(lt.NightPattern.Palette[0].G)
			buf.WriteByte(lt.NightPattern.Palette[0].R)

			// Write pattern bitmap (use day pattern data)
			if err := w.writeBitmap(buf, lt.DayPattern.Data, 32, byte(rows), 1); err != nil {
				return err
			}
		} else {
			// Day and night solid colors
			buf.WriteByte(lt.DayColor.B)
			buf.WriteByte(lt.DayColor.G)
			buf.WriteByte(lt.DayColor.R)
			buf.WriteByte(lt.DayBorderColor.B)
			buf.WriteByte(lt.DayBorderColor.G)
			buf.WriteByte(lt.DayBorderColor.R)
			buf.WriteByte(lt.NightColor.B)
			buf.WriteByte(lt.NightColor.G)
			buf.WriteByte(lt.NightColor.R)
			buf.WriteByte(lt.NightBorderColor.B)
			buf.WriteByte(lt.NightBorderColor.G)
			buf.WriteByte(lt.NightBorderColor.R)
			buf.WriteByte(byte(lt.LineWidth))
			buf.WriteByte(byte(lt.BorderWidth))
		}

	case 0x03:
		// Day with transparency, night solid
		if rows > 0 {
			if lt.DayPattern == nil || len(lt.DayPattern.Palette) < 2 {
				return fmt.Errorf("day pattern missing or invalid")
			}
			if lt.NightPattern == nil || len(lt.NightPattern.Palette) < 2 {
				return fmt.Errorf("night pattern missing or invalid")
			}

			// Day color (palette[1])
			buf.WriteByte(lt.DayPattern.Palette[1].B)
			buf.WriteByte(lt.DayPattern.Palette[1].G)
			buf.WriteByte(lt.DayPattern.Palette[1].R)

			// Night palette
			buf.WriteByte(lt.NightPattern.Palette[1].B)
			buf.WriteByte(lt.NightPattern.Palette[1].G)
			buf.WriteByte(lt.NightPattern.Palette[1].R)
			buf.WriteByte(lt.NightPattern.Palette[0].B)
			buf.WriteByte(lt.NightPattern.Palette[0].G)
			buf.WriteByte(lt.NightPattern.Palette[0].R)

			// Write pattern bitmap
			if err := w.writeBitmap(buf, lt.DayPattern.Data, 32, byte(rows), 1); err != nil {
				return err
			}
		}
	}

	return nil
}

// writePolygonTypes writes all polygon type definitions
func (w *Writer) writePolygonTypes(polygons []model.PolygonType) error {
	for i, poly := range polygons {
		dataOffset := w.polygonsData.Len()

		if err := w.writePolygonData(&poly); err != nil {
			return fmt.Errorf("write polygon %d: %w", i, err)
		}

		typeCode := w.encodeTypeSubtype(uint32(poly.Type), uint32(poly.SubType))
		if err := w.writeArrayEntry(w.polygonsArray, typeCode, uint32(dataOffset)); err != nil {
			return fmt.Errorf("write polygon array entry %d: %w", i, err)
		}
	}
	return nil
}

// writePolygonData writes a single polygon type definition
func (w *Writer) writePolygonData(poly *model.PolygonType) error {
	buf := &bytes.Buffer{}

	// Determine color type
	ctyp := w.determinePolygonColorType(poly)

	// Determine flags
	hasLabels := len(poly.Labels) > 0
	hasTextColors := false

	flags := byte(ctyp)
	if hasLabels {
		flags |= 0x10
	}
	if hasTextColors {
		flags |= 0x20
	}

	// Write flags (1 byte)
	buf.WriteByte(flags)

	// Write color/pattern data
	if err := w.writePolygonColorData(buf, poly, ctyp); err != nil {
		return fmt.Errorf("write polygon color data: %w", err)
	}

	// Write labels
	if hasLabels {
		if err := w.writeLabels(buf, poly.Labels); err != nil {
			return fmt.Errorf("write labels: %w", err)
		}
	}

	// Write to polygons data buffer
	if _, err := buf.WriteTo(w.polygonsData); err != nil {
		return err
	}

	return nil
}

// determinePolygonColorType determines the color type for a polygon
// Polygon color types:
// 0x01: Different day/night colors with border
// 0x06: Same day/night color, no border
// 0x07: Different day/night colors, no border
// 0x08: Same day/night pattern
// 0x09: Different day/night patterns
func (w *Writer) determinePolygonColorType(poly *model.PolygonType) int {
	hasDayPattern := poly.DayPattern != nil
	hasNightPattern := poly.NightPattern != nil

	if !hasDayPattern && !hasNightPattern {
		// Solid colors
		if poly.DayColor == poly.NightColor {
			return 0x06 // Same day/night, no border
		}
		return 0x07 // Different day/night, no border
	}

	// Pattern mode
	// If only one pattern exists, treat as same day/night
	if hasDayPattern && !hasNightPattern {
		return 0x08 // Same day/night pattern
	}

	if !hasDayPattern && hasNightPattern {
		return 0x08 // Same day/night pattern (unusual case)
	}

	// Both patterns exist - check if they're the same
	if w.palettesEqual(poly.DayPattern.Palette, poly.NightPattern.Palette) {
		return 0x08 // Same day/night pattern
	}

	return 0x09 // Different day/night patterns
}

// writePolygonColorData writes color/pattern data for a polygon type
func (w *Writer) writePolygonColorData(buf *bytes.Buffer, poly *model.PolygonType, ctyp int) error {
	switch ctyp {
	case 0x06:
		// Same fill for day/night, no border
		buf.WriteByte(poly.DayColor.B)
		buf.WriteByte(poly.DayColor.G)
		buf.WriteByte(poly.DayColor.R)

	case 0x07:
		// Different fill for day/night, no border
		buf.WriteByte(poly.DayColor.B)
		buf.WriteByte(poly.DayColor.G)
		buf.WriteByte(poly.DayColor.R)
		buf.WriteByte(poly.NightColor.B)
		buf.WriteByte(poly.NightColor.G)
		buf.WriteByte(poly.NightColor.R)

	case 0x08:
		// Day & night same pattern (2 colors)
		if poly.DayPattern == nil || len(poly.DayPattern.Palette) < 2 {
			return fmt.Errorf("day pattern missing or invalid")
		}

		// Write 2-color palette
		buf.WriteByte(poly.DayPattern.Palette[1].B)
		buf.WriteByte(poly.DayPattern.Palette[1].G)
		buf.WriteByte(poly.DayPattern.Palette[1].R)
		buf.WriteByte(poly.DayPattern.Palette[0].B)
		buf.WriteByte(poly.DayPattern.Palette[0].G)
		buf.WriteByte(poly.DayPattern.Palette[0].R)

		// Write pattern bitmap (polygons are always 32×32, 1 bpp)
		if err := w.writeBitmap(buf, poly.DayPattern.Data, 32, 32, 1); err != nil {
			return err
		}

	case 0x09:
		// Day & night different patterns (both must exist)
		if poly.DayPattern == nil || len(poly.DayPattern.Palette) < 2 {
			return fmt.Errorf("day pattern missing or invalid for color type 0x09")
		}
		if poly.NightPattern == nil || len(poly.NightPattern.Palette) < 2 {
			return fmt.Errorf("night pattern missing or invalid for color type 0x09")
		}

		// Write day palette
		buf.WriteByte(poly.DayPattern.Palette[1].B)
		buf.WriteByte(poly.DayPattern.Palette[1].G)
		buf.WriteByte(poly.DayPattern.Palette[1].R)
		buf.WriteByte(poly.DayPattern.Palette[0].B)
		buf.WriteByte(poly.DayPattern.Palette[0].G)
		buf.WriteByte(poly.DayPattern.Palette[0].R)

		// Write night palette
		buf.WriteByte(poly.NightPattern.Palette[1].B)
		buf.WriteByte(poly.NightPattern.Palette[1].G)
		buf.WriteByte(poly.NightPattern.Palette[1].R)
		buf.WriteByte(poly.NightPattern.Palette[0].B)
		buf.WriteByte(poly.NightPattern.Palette[0].G)
		buf.WriteByte(poly.NightPattern.Palette[0].R)

		// Write pattern bitmap (same data for both, different palettes)
		if err := w.writeBitmap(buf, poly.DayPattern.Data, 32, 32, 1); err != nil {
			return err
		}
	}

	return nil
}

// writeDrawOrder writes the draw order array
func (w *Writer) writeDrawOrder(typ *model.TYPFile) error {
	// Draw order is typically empty or auto-generated
	// For now, just write an empty array
	return nil
}
