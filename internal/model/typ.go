package model

// TYPFile represents the complete TYP data in a format-agnostic way.
// This is the unified internal representation used for conversion between
// binary and text formats.
type TYPFile struct {
	Header    Header
	Points    []PointType
	Lines     []LineType
	Polygons  []PolygonType
	DrawOrder DrawOrder
	Icons     map[string]*Bitmap // Key format: "point_0x2f06", "line_0x01", etc.
}

// Header contains TYP file metadata
type Header struct {
	Version  int // Format version
	CodePage int // Character encoding (1252, 1250, 65001, etc.)
	FID      int // Family ID
	PID      int // Product ID
	MapID    int // Map ID (if present)
}

// PointType represents a POI (Point of Interest) type definition
type PointType struct {
	Type       int               // Type code (e.g., 0x2f06)
	SubType    int               // SubType (0x00-0x1F, or extended)
	Labels     map[string]string // Language code -> label text (e.g., "04" -> "Trail Junction")
	Icon       *Bitmap           // Icon bitmap (optional)
	DayColor   Color             // Day display color
	NightColor Color             // Night display color
	FontStyle  FontStyle         // Label font style
}

// LineType represents a linear feature (road, path, boundary, etc.)
type LineType struct {
	Type             int               // Type code
	SubType          int               // SubType
	Labels           map[string]string // Language-specific labels
	LineWidth        int               // Line width in pixels
	BorderWidth      int               // Border width in pixels
	DayColor         Color             // Day line color
	NightColor       Color             // Night line color
	DayBorderColor   Color             // Day border color
	NightBorderColor Color             // Night border color
	UseOrientation   bool              // Whether line has direction
	LineStyle        LineStyle         // Solid, dashed, dotted, etc.
	Pattern          *Bitmap           // Line pattern bitmap (optional)
}

// PolygonType represents an area feature (forest, water, building, etc.)
type PolygonType struct {
	Type           int               // Type code
	SubType        int               // SubType
	Labels         map[string]string // Language-specific labels
	Pattern        *Bitmap           // Fill pattern bitmap (optional)
	DayColor       Color             // Day fill color
	NightColor     Color             // Night fill color
	FontStyle      FontStyle         // Label font style
	ExtendedLabels bool              // Extended label format flag
}

// DrawOrder defines rendering priority for map elements
type DrawOrder struct {
	Points   []int // Point type codes in rendering order
	Lines    []int // Line type codes in rendering order
	Polygons []int // Polygon type codes in rendering order
}

// Color represents an RGBA color
type Color struct {
	R     byte // Red (0-255)
	G     byte // Green (0-255)
	B     byte // Blue (0-255)
	Alpha byte // Alpha/transparency (0=transparent, 255=opaque)
}

// IsZero returns true if the color is uninitialized (all zeros)
func (c Color) IsZero() bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.Alpha == 0
}

// FontStyle defines how labels are rendered
type FontStyle int

const (
	FontNormal  FontStyle = iota // Normal size font
	FontSmall                    // Small font
	FontLarge                    // Large font
	FontNoLabel                  // Don't show label
)

// LineStyle defines line rendering style
type LineStyle int

const (
	LineSolid  LineStyle = iota // Solid line
	LineDashed                  // Dashed line
	LineDotted                  // Dotted line
)

// Bitmap represents image data (icons, patterns, etc.)
type Bitmap struct {
	Width     int       // Width in pixels
	Height    int       // Height in pixels
	ColorMode ColorMode // Color depth/mode
	Palette   []Color   // Color palette (for indexed modes)
	Data      []byte    // Pixel data (format depends on ColorMode)
}

// ColorMode defines bitmap color encoding
type ColorMode int

const (
	Monochrome ColorMode = iota // 1-bit monochrome
	Color16                     // 4-bit indexed (16 colors)
	Color256                    // 8-bit indexed (256 colors)
	TrueColor                   // 24-bit RGB + 8-bit alpha
)

// LanguageCode represents ISO language codes used in TYP files
// Common codes seen in Garmin TYP files
const (
	LangUnspecified = "00"
	LangFrench      = "01"
	LangGerman      = "02"
	LangDutch       = "03"
	LangEnglish     = "04"
	LangItalian     = "05"
	LangFinnish     = "06"
	LangSwedish     = "07"
	LangSpanish     = "08"
	LangBasque      = "09"
	LangCatalan     = "0a"
	LangGalician    = "0b"
	LangWelsh       = "0c"
	LangGaelic      = "0d"
	LangDanish      = "0e"
	LangNorwegian   = "0f"
	LangPolish      = "10"
	LangCzech       = "11"
	LangSlovak      = "12"
	LangHungarian   = "13"
	LangCroatian    = "14"
	LangTurkish     = "15"
	LangGreek       = "16"
	LangRussian     = "17"
)

// NewTYPFile creates a new empty TYP file structure
func NewTYPFile() *TYPFile {
	return &TYPFile{
		Points:   make([]PointType, 0),
		Lines:    make([]LineType, 0),
		Polygons: make([]PolygonType, 0),
		Icons:    make(map[string]*Bitmap),
	}
}
