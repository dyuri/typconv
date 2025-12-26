# Binary TYP Parser - Implementation Plan

## Project Overview

**Project Name:** `typconv` (TYP Converter)

**Description:** A command-line tool for converting between binary Garmin TYP files and text format, with support for extraction from .img files.

**Primary Goal:** Provide Linux users with a native tool to work with binary TYP files without requiring Wine or Windows tools.

**Language:** Go (for portability, performance, and easy binary parsing)

---

## Why This Matters

### Current Problem
- Binary TYP files are the standard format in Garmin .img maps
- Only Windows tool (img2typ) can convert binary → text
- Linux users must use Wine or manually hex-edit
- No open-source implementation of the binary format

### Solution Impact
- First native Linux tool for binary TYP conversion
- Enable `typtui` to work with real-world maps
- Eliminate Wine dependency
- Open-source implementation for community

---

## Project Structure

```
typconv/
├── cmd/
│   └── typconv/
│       └── main.go                 # CLI entry point
├── internal/
│   ├── binary/
│   │   ├── reader.go              # Binary TYP reader
│   │   ├── writer.go              # Binary TYP writer
│   │   ├── sections.go            # Section parsers
│   │   ├── types.go               # Binary format structures
│   │   └── header.go              # TYP file header
│   ├── text/
│   │   ├── reader.go              # Text format reader (mkgmap)
│   │   ├── writer.go              # Text format writer
│   │   └── formatter.go           # Text formatting
│   ├── model/
│   │   └── typ.go                 # Unified TYP data model
│   ├── img/
│   │   ├── reader.go              # .img file parser
│   │   ├── extractor.go           # TYP extraction from .img
│   │   └── container.go           # .img container format
│   └── converter/
│       └── converter.go           # Conversion logic
├── pkg/
│   └── typconv/
│       └── typconv.go             # Public API (for use as library)
├── testdata/
│   ├── binary/
│   │   ├── openhiking.typ
│   │   ├── openmtbmap.typ
│   │   └── minimal.typ
│   ├── text/
│   │   └── *.txt
│   └── img/
│       └── gmapsupp.img
├── docs/
│   ├── BINARY_FORMAT.md           # Binary format documentation
│   ├── USAGE.md                   # Usage guide
│   └── REVERSE_ENGINEERING.md     # RE notes
├── scripts/
│   └── test-with-img2typ.sh       # Validate against img2typ
├── go.mod
├── go.sum
├── README.md
├── LICENSE
└── Makefile
```

---

## Core Features

### Phase 1: MVP (Week 1-3)
- [x] Parse binary TYP file header
- [x] Extract basic metadata (FID, PID, CodePage)
- [x] Convert binary → text (basic types)
- [x] Handle point types (POIs)
- [x] Handle line types (roads)
- [x] Handle polygon types (areas)
- [x] CLI with basic commands

### Phase 2: Complete Format (Week 4-5)
- [x] XPM bitmap parsing
- [x] Draw order section
- [x] Extended types (subtypes)
- [x] Day/night colors
- [x] Text → binary conversion
- [x] Full round-trip (binary → text → binary)

### Phase 3: .img Support (Week 6)
- [x] Parse .img container format
- [x] Extract TYP from .img files
- [x] List all TYP files in .img
- [x] Batch extraction

### Phase 4: Advanced Features (Week 7-8)
- [x] Validation and linting
- [x] Format migration tools
- [x] Library API for other tools
- [x] Comprehensive testing

---

## Binary TYP Format Specification

### High-Level Structure

```
TYP File Structure:
┌─────────────────────────────────┐
│ File Header                     │ Fixed size, identification
├─────────────────────────────────┤
│ Section Directory               │ Table of contents
├─────────────────────────────────┤
│ Point Types Section             │ POI definitions
├─────────────────────────────────┤
│ Line Types Section              │ Road/path definitions
├─────────────────────────────────┤
│ Polygon Types Section           │ Area definitions
├─────────────────────────────────┤
│ Draw Order Section              │ Rendering order
├─────────────────────────────────┤
│ Icon/Bitmap Data                │ Embedded images
└─────────────────────────────────┘
```

### File Header Format

```go
// Based on reverse engineering and community documentation
type TYPHeader struct {
    Magic       [10]byte  // "GARMIN TYP" or similar
    Version     uint16    // Format version
    CodePage    uint16    // Character encoding (1252, 1250, etc.)
    FID         uint16    // Family ID
    PID         uint16    // Product ID
    Reserved    [16]byte  // Reserved/unknown
    SectionDir  uint32    // Offset to section directory
}

// Section directory entry
type SectionEntry struct {
    Type   uint8   // Section type (1=points, 2=lines, 3=polygons, etc.)
    Offset uint32  // Offset from file start
    Length uint32  // Section length in bytes
}
```

### Point Type Format

```go
type PointType struct {
    Type          uint16    // e.g., 0x2f06
    SubType       byte      // 0x00-0x1F
    HasIcon       bool
    IconOffset    uint32    // Offset to bitmap data
    LabelCount    byte      // Number of language labels
    Labels        []Label   // Language-specific strings
    DayColor      RGB       // Day display color
    NightColor    RGB       // Night display color
    FontStyle     byte      // Font rendering style
}

type Label struct {
    LangCode  byte    // 0x01=FR, 0x04=EN, etc.
    Text      string  // Null-terminated
}

type RGB struct {
    R, G, B byte
}
```

### XPM/Bitmap Format

```go
// Garmin uses a modified XPM-like format
type Bitmap struct {
    Width       byte      // Pixels
    Height      byte      // Pixels
    ColorMode   byte      // 1=mono, 4=16color, 8=256color, 32=truecolor
    NumColors   byte      // Color palette size
    Palette     []RGB     // Color definitions
    Data        []byte    // Pixel data (row-major)
}
```

---

## CLI Design

### Command Structure

```bash
typconv [command] [flags] [arguments]

Commands:
  bin2txt      Convert binary TYP to text
  txt2bin      Convert text to binary TYP
  extract      Extract TYP from .img file
  info         Display TYP file information
  validate     Validate TYP file structure
  diff         Compare two TYP files
  version      Show version information
```

### Usage Examples

```bash
# Convert binary TYP to text
typconv bin2txt openhiking.typ -o openhiking.txt

# Convert text to binary TYP
typconv txt2bin custom.txt -o custom.typ

# Extract all TYP files from .img
typconv extract gmapsupp.img -o output/

# Show TYP file info
typconv info openhiking.typ

# Validate TYP file
typconv validate custom.typ

# Compare two TYP files
typconv diff original.typ modified.typ

# Round-trip test
typconv bin2txt input.typ -o temp.txt
typconv txt2bin temp.txt -o output.typ
```

### Flags

```
Global Flags:
  -v, --verbose        Verbose output
  -q, --quiet         Suppress non-error output
  -h, --help          Show help
  --version           Show version

bin2txt Flags:
  -o, --output FILE   Output file path (default: stdout)
  -f, --format TEXT   Output format: mkgmap (default), json
  --no-xpm            Skip XPM bitmap data
  --no-labels         Skip label strings

txt2bin Flags:
  -o, --output FILE   Output file path
  --fid NUMBER        Override Family ID
  --pid NUMBER        Override Product ID
  --codepage NUMBER   Character encoding (default: 1252)

extract Flags:
  -o, --output DIR    Output directory
  -l, --list          List TYP files without extracting
  --all               Extract all TYP files (default: first)

info Flags:
  --json              Output as JSON
  --brief             Show only summary
```

---

## Data Model

### Unified Internal Representation

```go
package model

// TYPFile represents the complete TYP data
type TYPFile struct {
    Header      Header
    Points      []PointType
    Lines       []LineType
    Polygons    []PolygonType
    DrawOrder   DrawOrder
    Icons       map[string]*Bitmap  // Key: "point_0x2f06"
}

type Header struct {
    Version     int
    CodePage    int
    FID         int
    PID         int
    MapID       int
}

type PointType struct {
    Type        int
    SubType     int
    Labels      map[string]string  // langCode -> text
    Icon        *Bitmap
    DayColor    Color
    NightColor  Color
    FontStyle   FontStyle
}

type LineType struct {
    Type           int
    SubType        int
    Labels         map[string]string
    LineWidth      int
    BorderWidth    int
    DayColor       Color
    NightColor     Color
    DayBorderColor Color
    NightBorderColor Color
    UseOrientation bool
    LineStyle      LineStyle
    Pattern        *Bitmap
}

type PolygonType struct {
    Type           int
    SubType        int
    Labels         map[string]string
    Pattern        *Bitmap
    DayColor       Color
    NightColor     Color
    FontStyle      FontStyle
    ExtendedLabels bool
}

type DrawOrder struct {
    Points   []int  // Type codes in order
    Lines    []int
    Polygons []int
}

type Color struct {
    R, G, B byte
    Alpha   byte  // For transparency
}

type FontStyle int
const (
    FontNormal FontStyle = iota
    FontSmall
    FontLarge
    FontNoLabel
)

type LineStyle int
const (
    LineSolid LineStyle = iota
    LineDashed
    LineDotted
)

type Bitmap struct {
    Width      int
    Height     int
    ColorMode  ColorMode
    Palette    []Color
    Data       []byte
}

type ColorMode int
const (
    Monochrome ColorMode = iota
    Color16
    Color256
    TrueColor
)
```

---

## Binary Parser Implementation

### Key Parsing Patterns

```go
package binary

import (
    "encoding/binary"
    "io"
)

// Reader handles binary TYP parsing
type Reader struct {
    r      io.ReaderAt
    size   int64
    endian binary.ByteOrder  // Usually little-endian
}

func NewReader(r io.ReaderAt, size int64) *Reader {
    return &Reader{
        r:      r,
        size:   size,
        endian: binary.LittleEndian,
    }
}

// ReadHeader reads the TYP file header
func (r *Reader) ReadHeader() (*Header, error) {
    buf := make([]byte, 64)  // Header size
    if _, err := r.r.ReadAt(buf, 0); err != nil {
        return nil, err
    }
    
    h := &Header{
        Version:  r.endian.Uint16(buf[10:12]),
        CodePage: r.endian.Uint16(buf[12:14]),
        FID:      r.endian.Uint16(buf[14:16]),
        PID:      r.endian.Uint16(buf[16:18]),
    }
    
    return h, nil
}

// ReadSectionDirectory reads the section table
func (r *Reader) ReadSectionDirectory(offset int64) ([]Section, error) {
    // Read section count
    buf := make([]byte, 2)
    r.r.ReadAt(buf, offset)
    count := int(r.endian.Uint16(buf))
    
    sections := make([]Section, count)
    for i := 0; i < count; i++ {
        entryOffset := offset + 2 + int64(i*12)
        r.readSectionEntry(&sections[i], entryOffset)
    }
    
    return sections, nil
}

// ReadPointTypes reads all point type definitions
func (r *Reader) ReadPointTypes(section Section) ([]PointType, error) {
    offset := int64(section.Offset)
    end := offset + int64(section.Length)
    
    var points []PointType
    
    for offset < end {
        pt, bytesRead, err := r.readPointType(offset)
        if err != nil {
            return nil, err
        }
        points = append(points, pt)
        offset += int64(bytesRead)
    }
    
    return points, nil
}

// readPointType reads a single point type
func (r *Reader) readPointType(offset int64) (PointType, int, error) {
    buf := make([]byte, 256)  // Max size
    r.r.ReadAt(buf, offset)
    
    pt := PointType{
        Type:    int(r.endian.Uint16(buf[0:2])),
        SubType: int(buf[2]),
    }
    
    pos := 3
    
    // Read flags
    flags := buf[pos]
    pos++
    
    if flags&0x01 != 0 {
        // Has icon
        pt.Icon = r.readBitmap(offset + int64(pos))
        // Skip bitmap data
        pos += calculateBitmapSize(pt.Icon)
    }
    
    // Read label count
    labelCount := int(buf[pos])
    pos++
    
    // Read labels
    pt.Labels = make(map[string]string)
    for i := 0; i < labelCount; i++ {
        langCode := buf[pos]
        pos++
        
        // Read null-terminated string
        strEnd := pos
        for buf[strEnd] != 0 {
            strEnd++
        }
        
        pt.Labels[fmt.Sprintf("%02x", langCode)] = string(buf[pos:strEnd])
        pos = strEnd + 1
    }
    
    // Read colors if present
    if flags&0x02 != 0 {
        pt.DayColor = RGB{buf[pos], buf[pos+1], buf[pos+2]}
        pos += 3
    }
    if flags&0x04 != 0 {
        pt.NightColor = RGB{buf[pos], buf[pos+1], buf[pos+2]}
        pos += 3
    }
    
    return pt, pos, nil
}
```

---

## Text Format Writer

### mkgmap Text Format Output

```go
package text

import (
    "fmt"
    "io"
    "strings"
)

type Writer struct {
    w io.Writer
}

func NewWriter(w io.Writer) *Writer {
    return &Writer{w: w}
}

func (w *Writer) Write(typ *model.TYPFile) error {
    // Write header section
    if err := w.writeHeader(typ.Header); err != nil {
        return err
    }
    
    // Write draw order
    if err := w.writeDrawOrder(typ.DrawOrder); err != nil {
        return err
    }
    
    // Write point types
    for _, pt := range typ.Points {
        if err := w.writePointType(pt); err != nil {
            return err
        }
    }
    
    // Write line types
    for _, lt := range typ.Lines {
        if err := w.writeLineType(lt); err != nil {
            return err
        }
    }
    
    // Write polygon types
    for _, poly := range typ.Polygons {
        if err := w.writePolygonType(poly); err != nil {
            return err
        }
    }
    
    return nil
}

func (w *Writer) writeHeader(h model.Header) error {
    _, err := fmt.Fprintf(w.w, `[_id]
CodePage=%d
FID=%d
ProductCode=%d
[end]

`, h.CodePage, h.FID, h.PID)
    return err
}

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
        code, _ := strconv.ParseInt(langCode, 16, 32)
        fmt.Fprintf(w.w, "String%d=0x%s,%s\n", 
            len(pt.Labels), langCode, text)
    }
    
    // Icon (XPM format)
    if pt.Icon != nil {
        w.writeXPM(pt.Icon, "IconXpm")
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
    
    fmt.Fprintf(w.w, "[end]\n\n")
    return nil
}

func (w *Writer) writeXPM(bmp *model.Bitmap, tag string) error {
    fmt.Fprintf(w.w, "%s=\"%d %d %d 1\"\n",
        tag, bmp.Width, bmp.Height, len(bmp.Palette))
    
    // Write palette
    chars := "!@#$%^&*()_+-=[]{}|;:,.<>?abcdefghijklmnopqrstuvwxyz"
    for i, color := range bmp.Palette {
        if color.R == 0 && color.G == 0 && color.B == 0 && color.Alpha == 0 {
            fmt.Fprintf(w.w, "\"%c c none\"\n", chars[i])
        } else {
            fmt.Fprintf(w.w, "\"%c c #%02x%02x%02x\"\n",
                chars[i], color.R, color.G, color.B)
        }
    }
    
    // Write pixel data
    for y := 0; y < bmp.Height; y++ {
        line := make([]byte, bmp.Width)
        for x := 0; x < bmp.Width; x++ {
            idx := y*bmp.Width + x
            line[x] = chars[bmp.Data[idx]]
        }
        fmt.Fprintf(w.w, "\"%s\"\n", string(line))
    }
    
    return nil
}
```

---

## .img Container Format

### Basic Structure

```
.img File Structure:
┌─────────────────────────────────┐
│ FAT/Header                      │ File allocation table
├─────────────────────────────────┤
│ Subfile 1 (Map Data)            │
├─────────────────────────────────┤
│ Subfile 2 (TYP File)            │
├─────────────────────────────────┤
│ Subfile 3 (More Map Data)       │
├─────────────────────────────────┤
│ ...                             │
└─────────────────────────────────┘
```

### Extraction Implementation

```go
package img

import (
    "bytes"
    "encoding/binary"
    "io"
)

type IMGReader struct {
    r    io.ReaderAt
    size int64
    fat  *FAT
}

type FAT struct {
    Entries []FATEntry
}

type FATEntry struct {
    Name   string
    Offset uint32
    Size   uint32
}

func NewIMGReader(r io.ReaderAt, size int64) (*IMGReader, error) {
    reader := &IMGReader{
        r:    r,
        size: size,
    }
    
    if err := reader.readFAT(); err != nil {
        return nil, err
    }
    
    return reader, nil
}

func (r *IMGReader) readFAT() error {
    // Read FAT header at offset 0
    buf := make([]byte, 512)
    r.r.ReadAt(buf, 0)
    
    // Parse FAT entries
    // Format varies, but typically:
    // - 8 byte filename
    // - 4 byte offset
    // - 4 byte size
    
    r.fat = &FAT{}
    offset := 0x200  // FAT starts here
    
    for {
        entryBuf := make([]byte, 512)
        r.r.ReadAt(entryBuf, int64(offset))
        
        // Check for end of FAT
        if entryBuf[0] == 0xFF || entryBuf[0] == 0x00 {
            break
        }
        
        entry := FATEntry{
            Name:   string(bytes.TrimRight(entryBuf[0:8], "\x00")),
            Offset: binary.LittleEndian.Uint32(entryBuf[12:16]),
            Size:   binary.LittleEndian.Uint32(entryBuf[16:20]),
        }
        
        r.fat.Entries = append(r.fat.Entries, entry)
        offset += 512
    }
    
    return nil
}

func (r *IMGReader) ListTYPFiles() []FATEntry {
    var typs []FATEntry
    for _, entry := range r.fat.Entries {
        // TYP files usually have .TYP extension or specific type
        if strings.HasSuffix(strings.ToUpper(entry.Name), ".TYP") {
            typs = append(typs, entry)
        }
    }
    return typs
}

func (r *IMGReader) ExtractTYP(entry FATEntry) ([]byte, error) {
    buf := make([]byte, entry.Size)
    _, err := r.r.ReadAt(buf, int64(entry.Offset))
    return buf, err
}

func (r *IMGReader) ExtractAllTYPs(outputDir string) error {
    typs := r.ListTYPFiles()
    
    for _, typ := range typs {
        data, err := r.ExtractTYP(typ)
        if err != nil {
            return err
        }
        
        outPath := filepath.Join(outputDir, typ.Name)
        if err := os.WriteFile(outPath, data, 0644); err != nil {
            return err
        }
    }
    
    return nil
}
```

---

## Testing Strategy

### Unit Tests

```go
package binary

import "testing"

func TestHeaderParsing(t *testing.T) {
    tests := []struct {
        name     string
        input    []byte
        expected Header
    }{
        {
            name: "OpenHiking header",
            input: []byte{
                // Binary header data
            },
            expected: Header{
                FID: 3511,
                PID: 1,
                CodePage: 1250,
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := NewReader(bytes.NewReader(tt.input), int64(len(tt.input)))
            h, err := r.ReadHeader()
            if err != nil {
                t.Fatal(err)
            }
            if h.FID != tt.expected.FID {
                t.Errorf("FID = %d, want %d", h.FID, tt.expected.FID)
            }
        })
    }
}
```

### Round-Trip Tests

```go
func TestRoundTrip(t *testing.T) {
    // Read binary TYP
    original, err := os.ReadFile("testdata/binary/openhiking.typ")
    if err != nil {
        t.Fatal(err)
    }
    
    // Parse to model
    typ, err := ParseBinary(bytes.NewReader(original))
    if err != nil {
        t.Fatal(err)
    }
    
    // Write to text
    var textBuf bytes.Buffer
    if err := WriteText(&textBuf, typ); err != nil {
        t.Fatal(err)
    }
    
    // Parse text back
    typ2, err := ParseText(&textBuf)
    if err != nil {
        t.Fatal(err)
    }
    
    // Write to binary
    var binaryBuf bytes.Buffer
    if err := WriteBinary(&binaryBuf, typ2); err != nil {
        t.Fatal(err)
    }
    
    // Compare key fields (perfect byte match unlikely due to padding)
    if !compareTypes(typ, typ2) {
        t.Error("Round-trip produced different types")
    }
}
```

### Validation Against img2typ

```bash
#!/bin/bash
# scripts/test-with-img2typ.sh

# Test that our output matches img2typ reference
for typfile in testdata/binary/*.typ; do
    echo "Testing $typfile"
    
    # Run our converter
    ./typconv bin2txt "$typfile" -o "test_output.txt"
    
    # Run img2typ (via Wine)
    wine img2typ.exe "$typfile"
    
    # Compare outputs (normalize first)
    diff -u <(normalize_text img2typ_output.txt) \
            <(normalize_text test_output.txt)
    
    if [ $? -eq 0 ]; then
        echo "✓ $typfile matches"
    else
        echo "✗ $typfile differs"
        exit 1
    fi
done
```

---

## Reverse Engineering Process

### Tools Needed
1. **Hex editor**: hexdump, xxd, or Hex Fiend
2. **img2typ**: Reference implementation (Windows)
3. **Sample files**: Various TYP files from real maps
4. **Documentation**: Community wikis and forum posts

### Methodology

```bash
# 1. Collect sample files
wget https://openhiking.eu/.../openhiking-CE.typ
wget https://openmtbmap.org/.../mtbmap.typ

# 2. Convert to text with img2typ
wine img2typ.exe openhiking-CE.typ
# Creates openhiking-CE.txt

# 3. Hex dump both files
xxd openhiking-CE.typ > binary.hex
cat openhiking-CE.txt > text.txt

# 4. Correlate text data with binary offsets
# Look for patterns:
# - Type codes (e.g., 0x2f06) in hex
# - Label strings in hex
# - Color values (RGB bytes)
# - Bitmap dimensions

# 5. Document findings in BINARY_FORMAT.md
```

### Key Observations to Document

```markdown
# Offset 0x00: Header
- Bytes 0-9: Signature (may vary)
- Bytes 10-11: Version (LE uint16)
- Bytes 12-13: CodePage (LE uint16)
- Bytes 14-15: FID (LE uint16)
- Bytes 16-17: PID (LE uint16)

# Offset 0x40: Section Directory
- Bytes 0-1: Number of sections (LE uint16)
- Then for each section (12 bytes):
  - Byte 0: Section type (1=points, 2=lines, 3=polygons)
  - Bytes 1-4: Offset (LE uint32)
  - Bytes 5-8: Length (LE uint32)
  - Bytes 9-11: Reserved

# Point Type Entry (variable length)
- Bytes 0-1: Type code (LE uint16)
- Byte 2: Subtype
- Byte 3: Flags byte
  - Bit 0: Has icon
  - Bit 1: Has day color
  - Bit 2: Has night color
  - Bit 3: Extended labels
- If has icon: Bitmap data follows
- Byte N: Label count
- For each label:
  - Byte 0: Language code
  - Bytes 1+: Null-terminated string
- If has colors: 3 bytes RGB each
```

---

## CLI Implementation

### Main Entry Point

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/yourusername/typconv/internal/binary"
    "github.com/yourusername/typconv/internal/text"
    "github.com/yourusername/typconv/internal/img"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

var rootCmd = &cobra.Command{
    Use:   "typconv",
    Short: "Convert Garmin TYP files between binary and text formats",
    Long: `typconv is a tool for working with Garmin TYP files.
It can convert between binary and text formats, extract TYP files
from .img containers, and validate TYP file structure.`,
}

func init() {
    rootCmd.AddCommand(bin2txtCmd)
    rootCmd.AddCommand(txt2binCmd)
    rootCmd.AddCommand(extractCmd)
    rootCmd.AddCommand(infoCmd)
    rootCmd.AddCommand(validateCmd)
    rootCmd.AddCommand(versionCmd)
}

var bin2txtCmd = &cobra.Command{
    Use:   "bin2txt <input.typ>",
    Short: "Convert binary TYP to text format",
    Args:  cobra.ExactArgs(1),
    RunE:  runBin2Txt,
}

func init() {
    bin2txtCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
    bin2txtCmd.Flags().String("format", "mkgmap", "Output format: mkgmap, json")
    bin2txtCmd.Flags().Bool("no-xpm", false, "Skip XPM bitmap data")
}

func runBin2Txt(cmd *cobra.Command, args []string) error {
    inputPath := args[0]
    outputPath, _ := cmd.Flags().GetString("output")
    format, _ := cmd.Flags().GetString("format")
    noXPM, _ := cmd.Flags().GetBool("no-xpm")
    
    // Open input file
    f, err := os.Open(inputPath)
    if err != nil {
        return err
    }
    defer f.Close()
    
    // Get file size
    stat, _ := f.Stat()
    size := stat.Size()
    
    // Parse binary TYP
    reader := binary.NewReader(f, size)
    typ, err := reader.Parse()
    if err != nil {
        return fmt.Errorf("parse error: %w", err)
    }
    
    // Optionally remove XPM data
    if noXPM {
        stripXPMData(typ)
    }
    
    // Open output
    var output *os.File
    if outputPath == "" {
        output = os.Stdout
    } else {
        output, err = os.Create(outputPath)
        if err != nil {
            return err
        }
        defer output.Close()
    }
    
    // Write text format
    switch format {
    case "mkgmap":
        writer := text.NewWriter(output)
        return writer.Write(typ)
    case "json":
        return writeJSON(output, typ)
    default:
        return fmt.Errorf("unknown format: %s", format)
    }
}
```

---

## Package API (Library Use)

```go
package typconv

import (
    "io"
    "github.com/yourusername/typconv/internal/model"
)

// ParseBinaryTYP reads a binary TYP file
func ParseBinaryTYP(r io.ReaderAt, size int64) (*model.TYPFile, error) {
    reader := binary.NewReader(r, size)
    return reader.Parse()
}

// ParseTextTYP reads a text format TYP file
func ParseTextTYP(r io.Reader) (*model.TYPFile, error) {
    reader := text.NewReader(r)
    return reader.Parse()
}

// WriteBinaryTYP writes a binary TYP file
func WriteBinaryTYP(w io.Writer, typ *model.TYPFile) error {
    writer := binary.NewWriter(w)
    return writer.Write(typ)
}

// WriteTextTYP writes a text format TYP file
func WriteTextTYP(w io.Writer, typ *model.TYPFile) error {
    writer := text.NewWriter(w)
    return writer.Write(typ)
}

// ExtractTYPFromIMG extracts TYP files from .img container
func ExtractTYPFromIMG(r io.ReaderAt, size int64) ([]*model.TYPFile, error) {
    imgReader := img.NewReader(r, size)
    return imgReader.ExtractAllTYPs()
}

// Validate checks TYP file structure
func Validate(typ *model.TYPFile) []ValidationError {
    // Check type code ranges
    // Verify FID/PID
    // Validate bitmap dimensions
    // etc.
}
```

---

## Documentation

### README.md Structure

```markdown
# typconv - Garmin TYP Converter

Native Linux tool for converting Garmin TYP files between binary and text formats.

## Features
- Binary TYP → Text (mkgmap format)
- Text → Binary TYP
- Extract TYP from .img files
- No Wine required
- Fast and reliable

## Installation
```bash
go install github.com/yourusername/typconv@latest
```

## Quick Start
```bash
# Convert binary to text
typconv bin2txt map.typ -o map.txt

# Convert text to binary
typconv txt2bin custom.txt -o custom.typ

# Extract from .img
typconv extract gmapsupp.img -o output/
```

## Use as Library
```go
import "github.com/yourusername/typconv"

typ, err := typconv.ParseBinaryTYP(file, size)
```
```

### BINARY_FORMAT.md

Document all reverse engineering findings:
- Byte offsets and meanings
- Section structures
- Type definitions
- Color encodings
- Bitmap formats
- Unknown/reserved fields

---

## Timeline

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| 1. Setup & Research | 3 days | Project structure, format documentation |
| 2. Binary Reader | 1 week | Parse binary TYP to internal model |
| 3. Text Writer | 3 days | Write mkgmap format text |
| 4. CLI | 2 days | Basic commands working |
| 5. Testing | 3 days | Tests, validation against img2typ |
| 6. Text Reader | 3 days | Parse mkgmap text format |
| 7. Binary Writer | 1 week | Write binary TYP format |
| 8. Round-trip | 2 days | Full conversion cycle working |
| 9. .img Support | 1 week | Extract from .img containers |
| 10. Polish | 3 days | Documentation, examples |
| **Total** | **6-8 weeks** | **Production ready v1.0** |

---

## Success Metrics

### Technical
- Parse 95%+ of real-world TYP files
- Round-trip accuracy >99%
- Performance: <100ms for typical TYP file
- Zero memory leaks
- 80%+ code coverage

### User Experience
- One-command conversion
- Clear error messages
- Works with real maps (OpenHiking, OpenMTBMap)
- Validates output

---

## Distribution

### Installation Methods

```bash
# Go install
go install github.com/yourusername/typconv@latest

# Pre-built binaries
curl -LO https://github.com/yourusername/typconv/releases/latest/download/typconv-linux-amd64
chmod +x typconv-linux-amd64
sudo mv typconv-linux-amd64 /usr/local/bin/typconv

# AUR (Arch)
yay -S typconv-bin

# From source
git clone https://github.com/yourusername/typconv
cd typconv
make install
```

### Release Artifacts
- Linux (amd64, arm64)
- macOS (amd64, arm64) - future
- Windows (amd64) - future
- Source tarball

---

## Integration with typtui

Once `typconv` is stable, `typtui` can use it as a library:

```go
// In typtui project
import "github.com/yourusername/typconv"

func (m *Model) LoadBinaryTYP(path string) error {
    f, _ := os.Open(path)
    defer f.Close()
    
    stat, _ := f.Stat()
    
    typ, err := typconv.ParseBinaryTYP(f, stat.Size())
    if err != nil {
        return err
    }
    
    m.typFile = typ
    return nil
}
```

---

## Community Resources

### Reference Documentation
- OSM Wiki: https://wiki.openstreetmap.org/wiki/OSM_Map_On_Garmin
- mkgmap TYP docs: https://www.mkgmap.org.uk/doc/typ-compiler
- cferrero.net: https://www.cferrero.net/maps/guide_to_TYPs.html

### Sample Files
- OpenHiking: https://openhiking.eu/
- OpenMTBMap: https://openmtbmap.org/
- OpenTopoMap: https://garmin.opentopomap.org/

### Testing Partners
- OpenHiking maintainers
- mkgmap developers
- OSM community

---

## Open Questions / TODOs

### Binary Format
- [ ] Confirm NT map format differences
- [ ] Document extended type encoding
- [ ] Understand reserved/unknown fields
- [ ] Test with non-European character sets

### Features
- [ ] Should we support old TYP format versions?
- [ ] JSON output format design
- [ ] Validation rules completeness
- [ ] Error recovery strategies

### Testing
- [ ] Need more diverse test TYP files
- [ ] Cross-platform testing (endianness)
- [ ] Large file performance testing
- [ ] Fuzzing for robustness

---

## License

**MIT License** (same as typtui for consistency)

---

## Next Steps

1. Set up repository and project structure
2. Collect diverse TYP files for testing
3. Begin reverse engineering with hex editor
4. Document findings incrementally
5. Implement binary parser (read-only first)
6. Add text writer
7. Test against img2typ output
8. Iterate until reliable

**Start with Phase 1-2 (binary reading) to prove feasibility before committing to full implementation.**

---

This is an ambitious but achievable project that would provide genuine value to the Linux/OSM/Garmin community. No open-source implementation exists, so this would be groundbreaking work! 🚀
