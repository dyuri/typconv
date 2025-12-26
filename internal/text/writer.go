package text

import (
	"fmt"
	"io"

	"github.com/dyuri/typconv/internal/model"
)

// Writer handles writing TYP data to mkgmap text format
type Writer struct {
	w io.Writer
}

// NewWriter creates a new text format writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Write outputs the TYP data in mkgmap text format
func (w *Writer) Write(typ *model.TYPFile) error {
	// Write header section
	if err := w.writeHeader(typ.Header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write draw order (if present)
	if err := w.writeDrawOrder(typ.DrawOrder); err != nil {
		return fmt.Errorf("write draw order: %w", err)
	}

	// Write point types
	for _, pt := range typ.Points {
		if err := w.writePointType(pt); err != nil {
			return fmt.Errorf("write point type: %w", err)
		}
	}

	// Write line types
	for _, lt := range typ.Lines {
		if err := w.writeLineType(lt); err != nil {
			return fmt.Errorf("write line type: %w", err)
		}
	}

	// Write polygon types
	for _, poly := range typ.Polygons {
		if err := w.writePolygonType(poly); err != nil {
			return fmt.Errorf("write polygon type: %w", err)
		}
	}

	return nil
}

// writeHeader writes the [_id] section
func (w *Writer) writeHeader(h model.Header) error {
	// Format:
	// [_id]
	// CodePage=1252
	// FID=3511
	// ProductCode=1
	// [end]

	_, err := fmt.Fprintf(w.w, "[_id]\n")
	if err != nil {
		return err
	}

	if h.CodePage != 0 {
		fmt.Fprintf(w.w, "CodePage=%d\n", h.CodePage)
	}

	if h.FID != 0 {
		fmt.Fprintf(w.w, "FID=%d\n", h.FID)
	}

	if h.PID != 0 {
		fmt.Fprintf(w.w, "ProductCode=%d\n", h.PID)
	}

	_, err = fmt.Fprintf(w.w, "[end]\n\n")
	return err
}

// writeDrawOrder writes the draw order section (if not empty)
func (w *Writer) writeDrawOrder(order model.DrawOrder) error {
	// TODO: Implement draw order writing
	// Format needs investigation - likely comma-separated type lists

	return nil // Draw order is optional
}

// writePointType writes a [_point] section
func (w *Writer) writePointType(pt model.PointType) error {
	fmt.Fprintf(w.w, "[_point]\n")

	// Type code
	if pt.SubType != 0 {
		fmt.Fprintf(w.w, "Type=0x%x\nSubType=0x%x\n", pt.Type, pt.SubType)
	} else {
		fmt.Fprintf(w.w, "Type=0x%x\n", pt.Type)
	}

	// Labels
	for langCode, text := range pt.Labels {
		// Format: String1=0x04,Trail Junction
		fmt.Fprintf(w.w, "String1=0x%s,%s\n", langCode, text)
	}

	// Colors
	if !pt.DayColor.IsZero() {
		fmt.Fprintf(w.w, "DayColor=#%02x%02x%02x\n",
			pt.DayColor.R, pt.DayColor.G, pt.DayColor.B)
	}

	if !pt.NightColor.IsZero() {
		fmt.Fprintf(w.w, "NightColor=#%02x%02x%02x\n",
			pt.NightColor.R, pt.NightColor.G, pt.NightColor.B)
	}

	// Icon bitmaps
	if pt.DayIcon != nil {
		if err := w.writeXPM(pt.DayIcon, "DayXpm"); err != nil {
			return err
		}
	}

	if pt.NightIcon != nil && pt.NightIcon != pt.DayIcon {
		if err := w.writeXPM(pt.NightIcon, "NightXpm"); err != nil {
			return err
		}
	}

	// Font style
	// TODO: Map FontStyle to mkgmap format

	fmt.Fprintf(w.w, "[end]\n\n")
	return nil
}

// writeLineType writes a [_line] section
func (w *Writer) writeLineType(lt model.LineType) error {
	fmt.Fprintf(w.w, "[_line]\n")

	// Type code
	if lt.SubType != 0 {
		fmt.Fprintf(w.w, "Type=0x%x\nSubType=0x%x\n", lt.Type, lt.SubType)
	} else {
		fmt.Fprintf(w.w, "Type=0x%x\n", lt.Type)
	}

	// Labels
	for langCode, text := range lt.Labels {
		fmt.Fprintf(w.w, "String1=0x%s,%s\n", langCode, text)
	}

	// Line width
	if lt.LineWidth > 0 {
		fmt.Fprintf(w.w, "LineWidth=%d\n", lt.LineWidth)
	}

	// Border width
	if lt.BorderWidth > 0 {
		fmt.Fprintf(w.w, "BorderWidth=%d\n", lt.BorderWidth)
	}

	// Colors
	if !lt.DayColor.IsZero() {
		fmt.Fprintf(w.w, "DayColor=#%02x%02x%02x\n",
			lt.DayColor.R, lt.DayColor.G, lt.DayColor.B)
	}

	if !lt.NightColor.IsZero() {
		fmt.Fprintf(w.w, "NightColor=#%02x%02x%02x\n",
			lt.NightColor.R, lt.NightColor.G, lt.NightColor.B)
	}

	if !lt.DayBorderColor.IsZero() {
		fmt.Fprintf(w.w, "DayBorderColor=#%02x%02x%02x\n",
			lt.DayBorderColor.R, lt.DayBorderColor.G, lt.DayBorderColor.B)
	}

	if !lt.NightBorderColor.IsZero() {
		fmt.Fprintf(w.w, "NightBorderColor=#%02x%02x%02x\n",
			lt.NightBorderColor.R, lt.NightBorderColor.G, lt.NightBorderColor.B)
	}

	// Line pattern bitmaps
	if lt.DayPattern != nil {
		if err := w.writeXPM(lt.DayPattern, "DayXpm"); err != nil {
			return err
		}
	}

	if lt.NightPattern != nil && lt.NightPattern != lt.DayPattern {
		if err := w.writeXPM(lt.NightPattern, "NightXpm"); err != nil {
			return err
		}
	}

	fmt.Fprintf(w.w, "[end]\n\n")
	return nil
}

// writePolygonType writes a [_polygon] section
func (w *Writer) writePolygonType(poly model.PolygonType) error {
	fmt.Fprintf(w.w, "[_polygon]\n")

	// Type code
	if poly.SubType != 0 {
		fmt.Fprintf(w.w, "Type=0x%x\nSubType=0x%x\n", poly.Type, poly.SubType)
	} else {
		fmt.Fprintf(w.w, "Type=0x%x\n", poly.Type)
	}

	// Labels
	for langCode, text := range poly.Labels {
		fmt.Fprintf(w.w, "String1=0x%s,%s\n", langCode, text)
	}

	// Colors
	if !poly.DayColor.IsZero() {
		fmt.Fprintf(w.w, "DayColor=#%02x%02x%02x\n",
			poly.DayColor.R, poly.DayColor.G, poly.DayColor.B)
	}

	if !poly.NightColor.IsZero() {
		fmt.Fprintf(w.w, "NightColor=#%02x%02x%02x\n",
			poly.NightColor.R, poly.NightColor.G, poly.NightColor.B)
	}

	// Polygon pattern bitmaps
	if poly.DayPattern != nil {
		if err := w.writeXPM(poly.DayPattern, "DayXpm"); err != nil {
			return err
		}
	}

	if poly.NightPattern != nil && poly.NightPattern != poly.DayPattern {
		if err := w.writeXPM(poly.NightPattern, "NightXpm"); err != nil {
			return err
		}
	}

	fmt.Fprintf(w.w, "[end]\n\n")
	return nil
}

// writeXPM writes a bitmap in XPM format
func (w *Writer) writeXPM(bmp *model.Bitmap, tag string) error {
	// XPM format:
	// IconXpm="8 8 2 1"
	// "! c #ff0000"
	// "  c none"
	// "!!!!!!!!"
	// "!      !"
	// ...

	// Palette - use all printable ASCII characters (excluding space and quote)
	// This gives us 94 single-char codes. For more colors, we'd need multi-char codes.
	chars := "!#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"

	// If we need more than 94 colors, use two-character combinations
	if len(bmp.Palette) > len(chars) {
		// Generate two-character codes
		var extendedChars []string
		for _, c1 := range chars {
			for _, c2 := range chars {
				extendedChars = append(extendedChars, string([]byte{byte(c1), byte(c2)}))
				if len(extendedChars) >= 255 {
					break
				}
			}
			if len(extendedChars) >= 255 {
				break
			}
		}

		if len(bmp.Palette) > 255 {
			return fmt.Errorf("too many colors for XPM encoding: %d (max 255)", len(bmp.Palette))
		}

		// Write header with chars-per-pixel=2
		fmt.Fprintf(w.w, "%s=\"%d %d %d 2\"\n",
			tag, bmp.Width, bmp.Height, len(bmp.Palette))

		// Write palette with multi-char codes
		for i, color := range bmp.Palette {
			code := extendedChars[i]
			if color.R == 0 && color.G == 0 && color.B == 0 && color.Alpha == 0 {
				fmt.Fprintf(w.w, "\"%s c none\"\n", code)
			} else {
				fmt.Fprintf(w.w, "\"%s c #%02x%02x%02x\"\n",
					code, color.R, color.G, color.B)
			}
		}

		// Pixel data with two-char codes
		for y := 0; y < bmp.Height; y++ {
			fmt.Fprintf(w.w, "\"")
			for x := 0; x < bmp.Width; x++ {
				idx := y*bmp.Width + x
				if idx >= len(bmp.Data) {
					return fmt.Errorf("bitmap data too short")
				}
				pixelIdx := bmp.Data[idx]
				if int(pixelIdx) >= len(extendedChars) {
					return fmt.Errorf("pixel index out of range: %d", pixelIdx)
				}
				fmt.Fprintf(w.w, "%s", extendedChars[pixelIdx])
			}
			fmt.Fprintf(w.w, "\"\n")
		}

		return nil
	}

	// Single-character codes (original code path)
	// Write header with chars-per-pixel=1
	fmt.Fprintf(w.w, "%s=\"%d %d %d 1\"\n",
		tag, bmp.Width, bmp.Height, len(bmp.Palette))

	for i, color := range bmp.Palette {
		if i >= len(chars) {
			return fmt.Errorf("too many colors for XPM encoding: %d", len(bmp.Palette))
		}

		char := chars[i]
		if color.R == 0 && color.G == 0 && color.B == 0 && color.Alpha == 0 {
			// Transparent
			fmt.Fprintf(w.w, "\"%c c none\"\n", char)
		} else {
			fmt.Fprintf(w.w, "\"%c c #%02x%02x%02x\"\n",
				char, color.R, color.G, color.B)
		}
	}

	// Pixel data
	for y := 0; y < bmp.Height; y++ {
		fmt.Fprintf(w.w, "\"")
		for x := 0; x < bmp.Width; x++ {
			idx := y*bmp.Width + x
			if idx >= len(bmp.Data) {
				return fmt.Errorf("bitmap data too short")
			}
			pixelIdx := bmp.Data[idx]
			if int(pixelIdx) >= len(chars) {
				return fmt.Errorf("pixel index out of range")
			}
			fmt.Fprintf(w.w, "%c", chars[pixelIdx])
		}
		fmt.Fprintf(w.w, "\"\n")
	}

	return nil
}
