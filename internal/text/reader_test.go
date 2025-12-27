package text

import (
	"strings"
	"testing"
)

func TestReadHeader(t *testing.T) {
	input := `[_id]
CodePage=1252
FID=3511
ProductCode=1
[end]
`
	reader := NewReader(strings.NewReader(input))
	typ, err := reader.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if typ.Header.CodePage != 1252 {
		t.Errorf("CodePage = %d, want 1252", typ.Header.CodePage)
	}
	if typ.Header.FID != 3511 {
		t.Errorf("FID = %d, want 3511", typ.Header.FID)
	}
	if typ.Header.PID != 1 {
		t.Errorf("PID = %d, want 1", typ.Header.PID)
	}
}

func TestReadPointType(t *testing.T) {
	input := `[_point]
Type=0x2f06
SubType=0x00
String1=0x04,Trail Junction
DayColor=#ff0000
[end]
`
	reader := NewReader(strings.NewReader(input))
	typ, err := reader.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(typ.Points) != 1 {
		t.Fatalf("Got %d points, want 1", len(typ.Points))
	}

	pt := typ.Points[0]
	if pt.Type != 0x2f06 {
		t.Errorf("Type = 0x%x, want 0x2f06", pt.Type)
	}
	if pt.SubType != 0x00 {
		t.Errorf("SubType = 0x%x, want 0x00", pt.SubType)
	}
	if pt.Labels["04"] != "Trail Junction" {
		t.Errorf("Label = %q, want %q", pt.Labels["04"], "Trail Junction")
	}
	if pt.DayColor.R != 255 || pt.DayColor.G != 0 || pt.DayColor.B != 0 {
		t.Errorf("DayColor = RGB(%d,%d,%d), want RGB(255,0,0)",
			pt.DayColor.R, pt.DayColor.G, pt.DayColor.B)
	}
}

func TestReadPointWithXPM(t *testing.T) {
	input := `[_point]
Type=0x100
DayXpm="8 8 2 1"
"! c #ff0000"
"  c none"
"!!!!!!!!"
"!      !"
"! !!!! !"
"! !!!! !"
"! !!!! !"
"! !!!! !"
"!      !"
"!!!!!!!!"
[end]
`
	reader := NewReader(strings.NewReader(input))
	typ, err := reader.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(typ.Points) != 1 {
		t.Fatalf("Got %d points, want 1", len(typ.Points))
	}

	pt := typ.Points[0]
	if pt.DayIcon == nil {
		t.Fatal("DayIcon is nil")
	}

	if pt.DayIcon.Width != 8 || pt.DayIcon.Height != 8 {
		t.Errorf("Icon size = %dx%d, want 8x8", pt.DayIcon.Width, pt.DayIcon.Height)
	}

	if len(pt.DayIcon.Palette) != 2 {
		t.Errorf("Palette size = %d, want 2", len(pt.DayIcon.Palette))
	}

	// Check first color (! = red)
	if pt.DayIcon.Palette[0].R != 255 {
		t.Errorf("Palette[0] red = %d, want 255", pt.DayIcon.Palette[0].R)
	}

	// Check second color (space = transparent)
	if pt.DayIcon.Palette[1].Alpha != 0 {
		t.Errorf("Palette[1] alpha = %d, want 0 (transparent)", pt.DayIcon.Palette[1].Alpha)
	}
}

func TestReadLineType(t *testing.T) {
	input := `[_line]
Type=0x100
LineWidth=4
BorderWidth=2
DayColor=#dd7755
NightColor=#dd7755
[end]
`
	reader := NewReader(strings.NewReader(input))
	typ, err := reader.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(typ.Lines) != 1 {
		t.Fatalf("Got %d lines, want 1", len(typ.Lines))
	}

	lt := typ.Lines[0]
	if lt.Type != 0x100 {
		t.Errorf("Type = 0x%x, want 0x100", lt.Type)
	}
	if lt.LineWidth != 4 {
		t.Errorf("LineWidth = %d, want 4", lt.LineWidth)
	}
	if lt.BorderWidth != 2 {
		t.Errorf("BorderWidth = %d, want 2", lt.BorderWidth)
	}
}

func TestReadPolygonType(t *testing.T) {
	input := `[_polygon]
Type=0x200
DayColor=#262626
NightColor=#262626
[end]
`
	reader := NewReader(strings.NewReader(input))
	typ, err := reader.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(typ.Polygons) != 1 {
		t.Fatalf("Got %d polygons, want 1", len(typ.Polygons))
	}

	poly := typ.Polygons[0]
	if poly.Type != 0x200 {
		t.Errorf("Type = 0x%x, want 0x200", poly.Type)
	}
}

func TestParseHexInt(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0x100", 0x100},
		{"0X200", 0x200},
		{"256", 256},
		{"0x2f06", 0x2f06},
	}

	for _, tt := range tests {
		got := parseHexInt(tt.input)
		if got != tt.want {
			t.Errorf("parseHexInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		input string
		r, g, b byte
	}{
		{"#ff0000", 255, 0, 0},
		{"#00ff00", 0, 255, 0},
		{"#0000ff", 0, 0, 255},
		{"#dd7755", 0xdd, 0x77, 0x55},
	}

	for _, tt := range tests {
		color := parseColor(tt.input)
		if color.R != tt.r || color.G != tt.g || color.B != tt.b {
			t.Errorf("parseColor(%q) = RGB(%d,%d,%d), want RGB(%d,%d,%d)",
				tt.input, color.R, color.G, color.B, tt.r, tt.g, tt.b)
		}
	}
}

func TestParseLabel(t *testing.T) {
	tests := []struct {
		input    string
		wantLang string
		wantText string
		wantOK   bool
	}{
		{"0x04,Trail Junction", "04", "Trail Junction", true},
		{"0x14,Autópálya", "14", "Autópálya", true},
		{"invalid", "", "", false},
	}

	for _, tt := range tests {
		lang, text, ok := parseLabel(tt.input)
		if ok != tt.wantOK {
			t.Errorf("parseLabel(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
		}
		if lang != tt.wantLang {
			t.Errorf("parseLabel(%q) lang = %q, want %q", tt.input, lang, tt.wantLang)
		}
		if text != tt.wantText {
			t.Errorf("parseLabel(%q) text = %q, want %q", tt.input, text, tt.wantText)
		}
	}
}
