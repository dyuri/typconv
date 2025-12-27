package text

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dyuri/typconv/internal/model"
)

// xpmBuilder builds a bitmap from XPM data
type xpmBuilder struct {
	width    int
	height   int
	ncolors  int
	cpp      int // chars per pixel
	palette  map[string]model.Color
	lines    []string
	inHeader bool
}

// newXPMBuilder creates a new XPM builder from a header line
// Header format: "width height ncolors cpp"
func newXPMBuilder(header string) *xpmBuilder {
	// Remove quotes
	header = strings.Trim(header, "\"")
	parts := strings.Fields(header)

	if len(parts) < 4 {
		return &xpmBuilder{inHeader: true}
	}

	width, _ := strconv.Atoi(parts[0])
	height, _ := strconv.Atoi(parts[1])
	ncolors, _ := strconv.Atoi(parts[2])
	cpp, _ := strconv.Atoi(parts[3])

	return &xpmBuilder{
		width:    width,
		height:   height,
		ncolors:  ncolors,
		cpp:      cpp,
		palette:  make(map[string]model.Color),
		lines:    make([]string, 0),
		inHeader: true,
	}
}

// addLine adds a line of XPM data
func (x *xpmBuilder) addLine(line string) {
	// Remove quotes
	line = strings.Trim(line, "\"")
	x.lines = append(x.lines, line)
}

// build constructs the bitmap from accumulated XPM data
func (x *xpmBuilder) build() (*model.Bitmap, error) {
	if len(x.lines) == 0 {
		return nil, fmt.Errorf("no XPM data")
	}

	// Parse palette (first ncolors lines)
	charToPaletteIdx := make(map[string]int)
	palette := make([]model.Color, 0, x.ncolors)

	for i := 0; i < x.ncolors && i < len(x.lines); i++ {
		line := x.lines[i]

		// XPM color line format: "char c color"
		// For multi-char: "chars c color"
		if len(line) < x.cpp+3 {
			continue
		}

		charCode := line[0:x.cpp]
		rest := strings.TrimSpace(line[x.cpp:])

		// Parse color part: "c #rrggbb" or "c none"
		parts := strings.Fields(rest)
		if len(parts) < 2 {
			continue
		}

		var color model.Color
		if strings.ToLower(parts[1]) == "none" {
			// Transparent color
			color = model.Color{R: 0, G: 0, B: 0, Alpha: 0}
		} else if strings.HasPrefix(parts[1], "#") {
			// RGB color
			colorStr := parts[1][1:]
			if len(colorStr) == 6 {
				r, _ := strconv.ParseUint(colorStr[0:2], 16, 8)
				g, _ := strconv.ParseUint(colorStr[2:4], 16, 8)
				b, _ := strconv.ParseUint(colorStr[4:6], 16, 8)
				color = model.Color{R: byte(r), G: byte(g), B: byte(b), Alpha: 255}
			}
		}

		charToPaletteIdx[charCode] = len(palette)
		palette = append(palette, color)
	}

	// Parse pixel data (remaining lines after palette)
	pixelLines := x.lines[x.ncolors:]
	if len(pixelLines) != x.height {
		return nil, fmt.Errorf("expected %d pixel lines, got %d", x.height, len(pixelLines))
	}

	// Build pixel data
	pixelData := make([]byte, x.width*x.height)
	for y, line := range pixelLines {
		if len(line) < x.width*x.cpp {
			return nil, fmt.Errorf("line %d too short: expected %d chars, got %d", y, x.width*x.cpp, len(line))
		}

		for col := 0; col < x.width; col++ {
			charCode := line[col*x.cpp : col*x.cpp+x.cpp]
			if idx, ok := charToPaletteIdx[charCode]; ok {
				pixelData[y*x.width+col] = byte(idx)
			}
		}
	}

	// Determine color mode based on palette size
	var colorMode model.ColorMode
	switch {
	case len(palette) <= 2:
		colorMode = model.Monochrome
	case len(palette) <= 16:
		colorMode = model.Color16
	default:
		colorMode = model.Color256
	}

	return &model.Bitmap{
		Width:     x.width,
		Height:    x.height,
		ColorMode: colorMode,
		Palette:   palette,
		Data:      pixelData,
	}, nil
}
