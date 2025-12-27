# typconv Usage Guide

Comprehensive guide to using typconv for working with Garmin TYP files.

## Table of Contents

1. [Installation](#installation)
2. [Basic Usage](#basic-usage)
3. [Converting Files](#converting-files)
4. [Character Encoding](#character-encoding)
5. [Advanced Usage](#advanced-usage)
6. [Using as a Library](#using-as-a-library)
7. [Troubleshooting](#troubleshooting)

## Installation

### From Source

```bash
git clone https://github.com/dyuri/typconv
cd typconv
go build -o build/typconv ./cmd/typconv
```

This creates the `typconv` binary in the `build/` directory.

### Using Go

```bash
go install github.com/dyuri/typconv/cmd/typconv@latest
```

This installs `typconv` to `$GOPATH/bin`.

### Verify Installation

```bash
typconv version
```

## Basic Usage

### Getting Help

```bash
# General help
typconv --help

# Command-specific help
typconv bin2txt --help
typconv txt2bin --help
```

### Command Structure

```
typconv <command> [command-flags] <arguments>
```

### Available Commands

```
typconv [command]

Available Commands:
  bin2txt      Convert binary TYP to text format
  txt2bin      Convert text format to binary TYP
  version      Show version information
  help         Show help for any command
```

## Converting Files

### Binary to Text (bin2txt)

Convert a binary TYP file to mkgmap text format:

```bash
# Output to stdout
typconv bin2txt map.typ

# Save to file
typconv bin2txt map.typ -o map.txt

# Skip bitmap data (faster, smaller output)
typconv bin2txt map.typ -o map.txt --no-xpm

# Skip label strings
typconv bin2txt map.typ -o map.txt --no-labels

# Skip both bitmaps and labels
typconv bin2txt map.typ -o map.txt --no-xpm --no-labels
```

**Available Flags:**
- `-o, --output FILE` - Output file path (default: stdout)
- `--no-xpm` - Skip XPM bitmap data
- `--no-labels` - Skip label strings

### Text to Binary (txt2bin)

Convert mkgmap text format to binary TYP:

```bash
# Basic conversion (CodePage auto-detected from file)
typconv txt2bin map.txt -o map.typ

# Override FID and PID
typconv txt2bin map.txt -o map.typ --fid 3511 --pid 1

# Force specific CodePage (usually not needed)
typconv txt2bin map.txt -o map.typ --codepage 1250
```

**Available Flags:**
- `-o, --output FILE` - Output file path (required)
- `--fid NUMBER` - Override Family ID from file
- `--pid NUMBER` - Override Product ID from file
- `--codepage NUMBER` - Override character encoding (auto-detected by default)

**Important Note**: The `--codepage` flag is typically not needed. typconv automatically reads the CodePage from the `[_id]` section in your text file (e.g., `CodePage=1250`). Only use `--codepage` if you need to force a different encoding than what's in the file.

## Character Encoding

typconv handles character encoding automatically:

### Automatic CodePage Detection

When converting text → binary, typconv reads the CodePage from your text file:

```
[_id]
CodePage=1250
FID=1
ProductCode=3690
[end]
```

This CodePage value determines how text strings are encoded in the binary file.

### Supported CodePages

| CodePage | Encoding           | Languages                              |
|----------|--------------------|----------------------------------------|
| 1252     | Windows-1252       | Western European (English, French, Spanish) |
| 1250     | Windows-1250       | Central European (Hungarian, Polish, Czech) |
| 1251     | Windows-1251       | Cyrillic (Russian, Ukrainian, Bulgarian)    |
| 1254     | Windows-1254       | Turkish                                |
| 437      | CP437              | Original IBM PC                        |

### Text File Encoding

- Text files (output from `bin2txt`) are always **UTF-8** encoded
- typconv automatically converts between UTF-8 (text files) and the specified CodePage (binary files)
- Special characters like ő, ű, á, é are preserved in round-trip conversion

### Example with Hungarian Characters

```bash
# Original binary TYP with CodePage 1250
typconv bin2txt hungarian.typ -o hungarian.txt

# hungarian.txt contains UTF-8 text with characters like:
# String1=0x14,Főváros
# String1=0x14,Cukrászda

# Convert back - CodePage 1250 is auto-detected from file
typconv txt2bin hungarian.txt -o hungarian_new.typ

# Verify round-trip preserves characters
typconv bin2txt hungarian_new.typ -o verify.txt
diff hungarian.txt verify.txt  # Should be identical
```

## Advanced Usage

### Round-Trip Testing

Test that conversion preserves all data:

```bash
# Convert binary → text → binary
typconv bin2txt original.typ -o temp.txt
typconv txt2bin temp.txt -o recreated.typ

# Verify by converting back to text
typconv bin2txt recreated.typ -o verify.txt

# Compare text files (should be identical or nearly identical)
diff temp.txt verify.txt
```

**Note**: Some cosmetic differences may appear (label ordering, transparent pixel characters in XPM), but all functional data is preserved.

### Batch Processing

```bash
# Convert all TYP files in a directory to text
for f in *.typ; do
    typconv bin2txt "$f" -o "${f%.typ}.txt"
done

# Convert all text files back to binary
for f in *.txt; do
    typconv txt2bin "$f" -o "${f%.txt}.typ"
done
```

### Piping

```bash
# Display point types
typconv bin2txt map.typ | grep "Type=0x2f06" -A 10

# Compare two TYP files
diff <(typconv bin2txt map1.typ) <(typconv bin2txt map2.typ)

# Count feature types
typconv bin2txt map.typ | grep -c "^\[_point\]"
typconv bin2txt map.typ | grep -c "^\[_line\]"
typconv bin2txt map.typ | grep -c "^\[_polygon\]"
```

### Working with Real Maps

#### OpenHiking Example

```bash
# If you have an OpenHiking TYP file extracted from a .img map
typconv bin2txt openhiking.typ -o openhiking.txt

# Edit openhiking.txt with your editor
vim openhiking.txt

# Convert back to binary
typconv txt2bin openhiking.txt -o custom.typ

# Test that it works
typconv bin2txt custom.typ > test.txt
```

#### Custom Map Creation

```bash
# Start with an existing TYP as a template
typconv bin2txt template.typ -o mymap.txt

# Edit mymap.txt to add custom point types
cat >> mymap.txt << 'EOF'

[_point]
Type=0x2f10
String1=0x04,Custom Marker
DayColor=#00ff00
DayXpm="8 8 2 1"
"! c #00ff00"
"  c none"
"  !!!!  "
" !!!!!! "
"!!!!!!!!"
"!!!!!!!!"
"!!!!!!!!"
" !!!!!! "
"  !!!!  "
"   !!   "
[end]
EOF

# Convert to binary
typconv txt2bin mymap.txt -o mymap.typ --fid 9999 --pid 1
```

## Using as a Library

### Basic Example

```go
package main

import (
    "fmt"
    "os"
    "github.com/dyuri/typconv/pkg/typconv"
)

func main() {
    // Open binary TYP file
    f, err := os.Open("map.typ")
    if err != nil {
        panic(err)
    }
    defer f.Close()

    // Get file size
    stat, _ := f.Stat()

    // Parse binary TYP
    typ, err := typconv.ParseBinaryTYP(f, stat.Size())
    if err != nil {
        panic(err)
    }

    // Access data
    fmt.Printf("FID: %d, PID: %d\n", typ.Header.FID, typ.Header.PID)
    fmt.Printf("CodePage: %d\n", typ.Header.CodePage)
    fmt.Printf("Point types: %d\n", len(typ.Points))
    fmt.Printf("Line types: %d\n", len(typ.Lines))
    fmt.Printf("Polygon types: %d\n", len(typ.Polygons))

    // Write to text format
    out, _ := os.Create("output.txt")
    defer out.Close()
    typconv.WriteTextTYP(out, typ)
}
```

### Round-Trip Example

```go
package main

import (
    "os"
    "github.com/dyuri/typconv/pkg/typconv"
)

func main() {
    // Read binary
    f, _ := os.Open("input.typ")
    defer f.Close()
    stat, _ := f.Stat()
    typ, _ := typconv.ParseBinaryTYP(f, stat.Size())

    // Write to text
    txt, _ := os.Create("output.txt")
    defer txt.Close()
    typconv.WriteTextTYP(txt, typ)
    txt.Close()

    // Read text back
    txt2, _ := os.Open("output.txt")
    defer txt2.Close()
    typ2, _ := typconv.ParseTextTYP(txt2)

    // Write to binary
    bin, _ := os.Create("output.typ")
    defer bin.Close()
    typconv.WriteBinaryTYP(bin, typ2)
}
```

### Programmatic Modification

```go
package main

import (
    "github.com/dyuri/typconv/pkg/typconv"
    "github.com/dyuri/typconv/internal/model"
    "os"
)

func main() {
    // Parse TYP file
    f, _ := os.Open("input.typ")
    defer f.Close()
    stat, _ := f.Stat()
    typ, _ := typconv.ParseBinaryTYP(f, stat.Size())

    // Modify point types - change all red markers to green
    for i := range typ.Points {
        if typ.Points[i].DayColor.R == 0xFF &&
           typ.Points[i].DayColor.G == 0x00 {
            typ.Points[i].DayColor.G = 0xFF
            typ.Points[i].DayColor.R = 0x00
        }
    }

    // Add a new point type
    newPoint := model.PointType{
        Type:    0x2f20,
        SubType: 0x00,
        Labels: map[string]string{
            "04": "Custom Point",
        },
        DayColor: model.Color{R: 0xFF, G: 0x00, B: 0xFF},
    }
    typ.Points = append(typ.Points, newPoint)

    // Write back to binary
    out, _ := os.Create("output.typ")
    defer out.Close()
    typconv.WriteBinaryTYP(out, typ)
}
```

### Integration with typtui

```go
// In typtui project
import "github.com/dyuri/typconv/pkg/typconv"

func (m *Model) LoadBinaryTYP(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()

    stat, _ := f.Stat()
    typ, err := typconv.ParseBinaryTYP(f, stat.Size())
    if err != nil {
        return err
    }

    m.typFile = typ
    return nil
}

func (m *Model) SaveBinaryTYP(path string) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()

    return typconv.WriteBinaryTYP(f, m.typFile)
}
```

## Troubleshooting

### Common Issues

#### "Invalid TYP header" or "parse TYP file: ..."

The file might not be a valid TYP file or might be corrupted:

```bash
# Check file type
file map.typ

# Check file size (should be > 100 bytes)
ls -lh map.typ

# Try to get basic info by examining header
xxd map.typ | head -10
```

If the file is inside a .img container, you'll need to extract it first (currently requires external tools like `img2typ` on Windows).

#### "Character encoding error" or garbled text

```bash
# Check what CodePage the file uses
typconv bin2txt problematic.typ | head -20

# The output should show:
# [_id]
# CodePage=XXXX
# ...
```

If characters appear garbled in the text output, the file may have an unusual or incorrectly set CodePage.

#### Labels truncated or corrupted

This should not happen with the current version. If you experience this:

```bash
# Convert to text and check
typconv bin2txt map.typ -o test.txt

# Look for short labels that should be longer
grep "String1=" test.txt
```

If labels are truncated, please file an issue with a sample file.

#### Round-trip produces different file size

This is normal and expected:

- Binary files may have different sizes due to:
  - Normalized XPM format (transparent pixel characters)
  - Removal of optional redundant color fields
  - Different timestamp in header
  - Optimized data layout

As long as the feature counts match and a second round-trip produces identical text, the conversion is correct.

```bash
# First conversion
typconv bin2txt original.typ -o step1.txt
typconv txt2bin step1.txt -o step2.typ
typconv bin2txt step2.typ -o step3.txt

# Second conversion
typconv txt2bin step3.txt -o step4.typ
typconv bin2txt step4.typ -o step5.txt

# step3.txt and step5.txt should be identical
diff step3.txt step5.txt
```

### Debugging

Get detailed information:

```bash
# The conversion output shows statistics
typconv txt2bin map.txt -o map.typ
# Output:
#   Successfully converted map.txt to map.typ
#   CodePage: 1250, FID: 1, PID: 3690
#   Points: 402, Lines: 126, Polygons: 73
```

### Getting Help

If you encounter issues:

1. Check that you're using the latest version: `typconv version`
2. Try converting with `--no-xpm` to isolate bitmap issues
3. Search [existing issues](https://github.com/dyuri/typconv/issues)
4. Create a [new issue](https://github.com/dyuri/typconv/issues/new) with:
   - Command used
   - Error message
   - Sample file (if possible to share)
   - Output of `typconv version`

## Performance

### Speed

Typical conversion times on modern hardware:

- Small files (<10KB): <10ms
- Medium files (50KB): ~50ms
- Large files (>100KB): ~100ms

### Memory Usage

typconv loads entire files into memory. For very large files:
- Binary parsing: ~2-3x file size
- Text output: ~5-10x file size (due to verbose XPM format)

### Optimization Tips

```bash
# Skip bitmaps if you don't need them (much faster)
typconv bin2txt large.typ -o output.txt --no-xpm

# Process multiple files in parallel
parallel typconv bin2txt {} -o {.}.txt ::: *.typ
```

## Best Practices

### 1. Keep Backups

```bash
# Before modifying
cp original.typ original.typ.backup
```

### 2. Test Round-trips

```bash
# Ensure no data loss
typconv bin2txt original.typ -o temp.txt
typconv txt2bin temp.txt -o test.typ
typconv bin2txt test.typ -o verify.txt
diff temp.txt verify.txt
```

### 3. Version Control

```bash
# Track text versions in git (more readable than binary)
git add map.txt
git commit -m "Change point colors to green"

# Convert to binary for use
typconv txt2bin map.txt -o map.typ
```

### 4. Validate Your Edits

After manually editing text files:

```bash
# Try to convert back to catch syntax errors
typconv txt2bin edited.txt -o test.typ
```

If the conversion fails, you have a syntax error in your text file.

## Next Steps

- See [BINARY_FORMAT.md](BINARY_FORMAT.md) for format details
- Check [README.md](../README.md) for project overview
- Visit the [project repository](https://github.com/dyuri/typconv) for updates

---

**Last Updated**: 2025-12-27
