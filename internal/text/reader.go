package text

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/dyuri/typconv/internal/model"
)

// Reader handles reading TYP data from mkgmap text format
type Reader struct {
	scanner *bufio.Scanner
	line    int
}

// NewReader creates a new text format reader
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
		line:    0,
	}
}

// Read parses the entire text file and returns the internal model
func (r *Reader) Read() (*model.TYPFile, error) {
	typ := model.NewTYPFile()

	for r.scanner.Scan() {
		r.line++
		line := strings.TrimSpace(r.scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Parse section headers
		if strings.HasPrefix(line, "[") {
			section := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")

			switch section {
			case "_id":
				if err := r.readHeader(&typ.Header); err != nil {
					return nil, fmt.Errorf("line %d: read header: %w", r.line, err)
				}

			case "_point":
				pt, err := r.readPointType()
				if err != nil {
					return nil, fmt.Errorf("line %d: read point type: %w", r.line, err)
				}
				typ.Points = append(typ.Points, pt)

			case "_line":
				lt, err := r.readLineType()
				if err != nil {
					return nil, fmt.Errorf("line %d: read line type: %w", r.line, err)
				}
				typ.Lines = append(typ.Lines, lt)

			case "_polygon":
				poly, err := r.readPolygonType()
				if err != nil {
					return nil, fmt.Errorf("line %d: read polygon type: %w", r.line, err)
				}
				typ.Polygons = append(typ.Polygons, poly)

			case "end":
				// End of section marker
				continue

			default:
				// Unknown section - skip until [end]
				if err := r.skipToEnd(); err != nil {
					return nil, fmt.Errorf("line %d: skip unknown section: %w", r.line, err)
				}
			}
		}
	}

	if err := r.scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return typ, nil
}

// readHeader reads the [_id] section
func (r *Reader) readHeader(header *model.Header) error {
	for r.scanner.Scan() {
		r.line++
		line := strings.TrimSpace(r.scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[end]") {
			return nil
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "CodePage":
			if v, err := strconv.Atoi(value); err == nil {
				header.CodePage = v
			}
		case "FID":
			if v, err := strconv.Atoi(value); err == nil {
				header.FID = v
			}
		case "ProductCode":
			if v, err := strconv.Atoi(value); err == nil {
				header.PID = v
			}
		}
	}

	return nil
}

// readPointType reads a [_point] section
func (r *Reader) readPointType() (model.PointType, error) {
	pt := model.PointType{
		Labels: make(map[string]string),
	}

	var currentXPM *xpmBuilder
	var xpmTarget string // "DayXpm" or "NightXpm"

	for r.scanner.Scan() {
		r.line++
		line := strings.TrimSpace(r.scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[end]") {
			// Finalize any pending XPM
			if currentXPM != nil {
				bmp, err := currentXPM.build()
				if err != nil {
					return pt, fmt.Errorf("build XPM: %w", err)
				}
				if xpmTarget == "DayXpm" {
					pt.DayIcon = bmp
				} else if xpmTarget == "NightXpm" {
					pt.NightIcon = bmp
				}
			}
			return pt, nil
		}

		// Handle XPM data lines
		if currentXPM != nil {
			if strings.HasPrefix(line, "\"") {
				currentXPM.addLine(line)
				continue
			} else {
				// XPM finished, build it
				bmp, err := currentXPM.build()
				if err != nil {
					return pt, fmt.Errorf("build XPM: %w", err)
				}
				if xpmTarget == "DayXpm" {
					pt.DayIcon = bmp
				} else if xpmTarget == "NightXpm" {
					pt.NightIcon = bmp
				}
				currentXPM = nil
			}
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Type":
			pt.Type = parseHexInt(value)
		case "SubType":
			pt.SubType = parseHexInt(value)
		case "String1", "String2", "String3":
			// Format: String1=0x04,Label text
			if langCode, text, ok := parseLabel(value); ok {
				pt.Labels[langCode] = text
			}
		case "DayColor":
			pt.DayColor = parseColor(value)
		case "NightColor":
			pt.NightColor = parseColor(value)
		case "DayXpm", "IconXpm":
			xpmTarget = "DayXpm"
			currentXPM = newXPMBuilder(value)
		case "NightXpm":
			xpmTarget = "NightXpm"
			currentXPM = newXPMBuilder(value)
		}
	}

	return pt, nil
}

// readLineType reads a [_line] section
func (r *Reader) readLineType() (model.LineType, error) {
	lt := model.LineType{
		Labels: make(map[string]string),
	}

	var currentXPM *xpmBuilder
	var xpmTarget string

	for r.scanner.Scan() {
		r.line++
		line := strings.TrimSpace(r.scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[end]") {
			if currentXPM != nil {
				bmp, err := currentXPM.build()
				if err != nil {
					return lt, fmt.Errorf("build XPM: %w", err)
				}
				if xpmTarget == "DayXpm" {
					lt.DayPattern = bmp
				} else if xpmTarget == "NightXpm" {
					lt.NightPattern = bmp
				}
			}
			return lt, nil
		}

		// Handle XPM data
		if currentXPM != nil {
			if strings.HasPrefix(line, "\"") {
				currentXPM.addLine(line)
				continue
			} else {
				bmp, err := currentXPM.build()
				if err != nil {
					return lt, fmt.Errorf("build XPM: %w", err)
				}
				if xpmTarget == "DayXpm" {
					lt.DayPattern = bmp
				} else if xpmTarget == "NightXpm" {
					lt.NightPattern = bmp
				}
				currentXPM = nil
			}
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Type":
			lt.Type = parseHexInt(value)
		case "SubType":
			lt.SubType = parseHexInt(value)
		case "String1", "String2", "String3":
			if langCode, text, ok := parseLabel(value); ok {
				lt.Labels[langCode] = text
			}
		case "LineWidth":
			if v, err := strconv.Atoi(value); err == nil {
				lt.LineWidth = v
			}
		case "BorderWidth":
			if v, err := strconv.Atoi(value); err == nil {
				lt.BorderWidth = v
			}
		case "DayColor":
			lt.DayColor = parseColor(value)
		case "NightColor":
			lt.NightColor = parseColor(value)
		case "DayBorderColor":
			lt.DayBorderColor = parseColor(value)
		case "NightBorderColor":
			lt.NightBorderColor = parseColor(value)
		case "DayXpm":
			xpmTarget = "DayXpm"
			currentXPM = newXPMBuilder(value)
		case "NightXpm":
			xpmTarget = "NightXpm"
			currentXPM = newXPMBuilder(value)
		}
	}

	return lt, nil
}

// readPolygonType reads a [_polygon] section
func (r *Reader) readPolygonType() (model.PolygonType, error) {
	poly := model.PolygonType{
		Labels: make(map[string]string),
	}

	var currentXPM *xpmBuilder
	var xpmTarget string

	for r.scanner.Scan() {
		r.line++
		line := strings.TrimSpace(r.scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[end]") {
			if currentXPM != nil {
				bmp, err := currentXPM.build()
				if err != nil {
					return poly, fmt.Errorf("build XPM: %w", err)
				}
				if xpmTarget == "DayXpm" {
					poly.DayPattern = bmp
				} else if xpmTarget == "NightXpm" {
					poly.NightPattern = bmp
				}
			}
			return poly, nil
		}

		// Handle XPM data
		if currentXPM != nil {
			if strings.HasPrefix(line, "\"") {
				currentXPM.addLine(line)
				continue
			} else {
				bmp, err := currentXPM.build()
				if err != nil {
					return poly, fmt.Errorf("build XPM: %w", err)
				}
				if xpmTarget == "DayXpm" {
					poly.DayPattern = bmp
				} else if xpmTarget == "NightXpm" {
					poly.NightPattern = bmp
				}
				currentXPM = nil
			}
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Type":
			poly.Type = parseHexInt(value)
		case "SubType":
			poly.SubType = parseHexInt(value)
		case "String1", "String2", "String3":
			if langCode, text, ok := parseLabel(value); ok {
				poly.Labels[langCode] = text
			}
		case "DayColor":
			poly.DayColor = parseColor(value)
		case "NightColor":
			poly.NightColor = parseColor(value)
		case "DayXpm":
			xpmTarget = "DayXpm"
			currentXPM = newXPMBuilder(value)
		case "NightXpm":
			xpmTarget = "NightXpm"
			currentXPM = newXPMBuilder(value)
		}
	}

	return poly, nil
}

// skipToEnd skips lines until [end] is found
func (r *Reader) skipToEnd() error {
	for r.scanner.Scan() {
		r.line++
		line := strings.TrimSpace(r.scanner.Text())
		if strings.HasPrefix(line, "[end]") {
			return nil
		}
	}
	return fmt.Errorf("unexpected EOF looking for [end]")
}

// parseHexInt parses a hex string like "0x2f06" or decimal
func parseHexInt(s string) int {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		if v, err := strconv.ParseInt(s[2:], 16, 64); err == nil {
			return int(v)
		}
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return 0
}

// parseColor parses a color string like "#ff0000"
func parseColor(s string) model.Color {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return model.Color{}
	}

	s = s[1:] // Remove #
	if len(s) != 6 {
		return model.Color{}
	}

	r, _ := strconv.ParseUint(s[0:2], 16, 8)
	g, _ := strconv.ParseUint(s[2:4], 16, 8)
	b, _ := strconv.ParseUint(s[4:6], 16, 8)

	return model.Color{
		R:     byte(r),
		G:     byte(g),
		B:     byte(b),
		Alpha: 255,
	}
}

// parseLabel parses a label string like "0x04,Trail Junction"
func parseLabel(s string) (langCode string, text string, ok bool) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	langCode = strings.TrimSpace(parts[0])
	if strings.HasPrefix(langCode, "0x") || strings.HasPrefix(langCode, "0X") {
		langCode = strings.ToLower(langCode[2:])
	}

	text = strings.TrimSpace(parts[1])
	return langCode, text, true
}
