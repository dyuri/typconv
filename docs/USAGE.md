# typconv Usage Guide

Comprehensive guide to using typconv for working with Garmin TYP files.

## Table of Contents

1. [Installation](#installation)
2. [Basic Usage](#basic-usage)
3. [Converting Files](#converting-files)
4. [Extracting from .img](#extracting-from-img)
5. [File Information](#file-information)
6. [Validation](#validation)
7. [Advanced Usage](#advanced-usage)
8. [Using as a Library](#using-as-a-library)
9. [Troubleshooting](#troubleshooting)

## Installation

### From Source

```bash
git clone https://github.com/dyuri/typconv
cd typconv
make install
```

This installs `typconv` to `$GOPATH/bin`.

### Using Go

```bash
go install github.com/dyuri/typconv/cmd/typconv@latest
```

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
typconv extract --help
```

### Command Structure

```
typconv [global-flags] <command> [command-flags] <arguments>
```

## Converting Files

### Binary to Text (bin2txt)

Convert a binary TYP file to mkgmap text format:

```bash
# Output to stdout
typconv bin2txt map.typ

# Save to file
typconv bin2txt map.typ -o map.txt

# Verbose output
typconv bin2txt map.typ -o map.txt -v

# Skip bitmap data (faster, smaller output)
typconv bin2txt map.typ -o map.txt --no-xpm

# Output as JSON
typconv bin2txt map.typ -o map.json --format json
```

### Text to Binary (txt2bin)

Convert mkgmap text format to binary TYP:

```bash
# Basic conversion
typconv txt2bin map.txt -o map.typ

# Override FID and PID
typconv txt2bin map.txt -o map.typ --fid 3511 --pid 1

# Specify character encoding
typconv txt2bin map.txt -o map.typ --codepage 1250

# Verbose output
typconv txt2bin map.txt -o map.typ -v
```

#### CodePage Values

| CodePage | Encoding           | Use For              |
|----------|--------------------|----------------------|
| 1252     | Windows-1252       | Western Europe       |
| 1250     | Windows-1250       | Central Europe       |
| 1251     | Windows-1251       | Cyrillic             |
| 65001    | UTF-8              | Unicode              |

## Extracting from .img

Garmin .img files are containers that can include TYP files along with map data.

### List TYP Files

```bash
# See what TYP files are in the .img
typconv extract gmapsupp.img --list
```

### Extract Single TYP

```bash
# Extract the first TYP file found
typconv extract gmapsupp.img -o output/
```

### Extract All TYP Files

```bash
# Extract all TYP files
typconv extract gmapsupp.img -o output/ --all
```

### Extract and Convert

```bash
# Extract TYP and immediately convert to text
typconv extract gmapsupp.img -o output/
typconv bin2txt output/maptype.typ -o maptype.txt
```

## File Information

Display metadata about TYP files:

```bash
# Human-readable summary
typconv info map.typ

# Detailed information
typconv info map.typ -v

# JSON output (for scripting)
typconv info map.typ --json

# Brief summary only
typconv info map.typ --brief
```

Example output:
```
TYP File: map.typ
Family ID: 3511
Product ID: 1
CodePage: 1252
Version: 1

Points:   15 types
Lines:    42 types
Polygons: 28 types

Total size: 45,823 bytes
```

## Validation

Validate TYP file structure and contents:

```bash
# Basic validation
typconv validate map.typ

# Verbose validation (show all checks)
typconv validate map.typ -v

# Strict mode (fail on warnings)
typconv validate map.typ --strict
```

### What Gets Validated

- File header integrity
- Type code ranges
- FID/PID validity
- Bitmap dimensions
- Section structure
- String encoding
- Cross-references

## Advanced Usage

### Round-Trip Testing

Test that conversion preserves data:

```bash
# Convert binary → text → binary
typconv bin2txt original.typ -o temp.txt
typconv txt2bin temp.txt -o recreated.typ

# Compare
typconv diff original.typ recreated.typ
```

### Batch Processing

```bash
# Convert all TYP files in a directory
for f in *.typ; do
    typconv bin2txt "$f" -o "${f%.typ}.txt"
done

# Extract from multiple .img files
for img in *.img; do
    typconv extract "$img" -o "extracted/${img%.img}/" --all
done
```

### Piping

```bash
# Chain commands
typconv bin2txt map.typ | grep "Type=0x2f06" -A 10

# Convert and compare in one line
diff <(typconv bin2txt map1.typ) <(typconv bin2txt map2.typ)
```

### Working with Real Maps

#### OpenHiking Example

```bash
# Download OpenHiking map
wget https://openhiking.eu/downloads/openhiking-CE.img

# Extract TYP
typconv extract openhiking-CE.img -o extracted/

# Convert to text for editing
typconv bin2txt extracted/openhiking.typ -o openhiking.txt

# Edit openhiking.txt with your editor
vim openhiking.txt

# Convert back to binary
typconv txt2bin openhiking.txt -o custom.typ

# Validate your changes
typconv validate custom.typ
```

#### Custom Map Creation

```bash
# Start with a template
typconv bin2txt template.typ -o mymap.txt

# Edit mymap.txt to add custom point types
cat >> mymap.txt << 'EOF'

[_point]
Type=0x2f10
String1=0x04,Custom Marker
DayColor=#00ff00
IconXpm="8 8 2 1"
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

# Validate
typconv validate mymap.typ
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
    fmt.Printf("Point types: %d\n", len(typ.Points))

    // Write to text format
    out, _ := os.Create("output.txt")
    defer out.Close()

    typconv.WriteTextTYP(out, typ)
}
```

### Advanced Library Usage

```go
package main

import (
    "github.com/dyuri/typconv/pkg/typconv"
    "github.com/dyuri/typconv/internal/model"
)

func customProcessing() {
    // Parse TYP file
    typ, _ := typconv.ParseBinaryTYP(reader, size)

    // Modify point types
    for i := range typ.Points {
        // Change all red markers to green
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

    // Write back
    typconv.WriteBinaryTYP(writer, typ)
}
```

### Integration with typtui

```go
// In typtui project
import "github.com/dyuri/typconv/pkg/typconv"

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

## Troubleshooting

### Common Issues

#### "Invalid TYP header"

The file might not be a valid TYP file:
```bash
# Check file type
file map.typ

# Try extracting from .img first
typconv extract map.img -o extracted/
```

#### "Character encoding error"

The CodePage might be incorrect:
```bash
# Try different CodePage
typconv txt2bin map.txt -o map.typ --codepage 1250

# Use UTF-8
typconv txt2bin map.txt -o map.typ --codepage 65001
```

#### "Bitmap parsing failed"

Bitmap data might be corrupted:
```bash
# Skip bitmaps
typconv bin2txt map.typ -o map.txt --no-xpm

# Validate the file
typconv validate map.typ -v
```

#### "Round-trip produces different output"

This might be expected due to:
- Padding bytes in binary format
- Order of sections
- Comment stripping

```bash
# Use semantic comparison
typconv diff original.typ recreated.typ
```

### Debugging

Enable verbose output:
```bash
typconv -v bin2txt problematic.typ
```

Get detailed error information:
```bash
typconv bin2txt problematic.typ 2>&1 | tee debug.log
```

### Getting Help

If you encounter issues:

1. Check the [FAQ](../README.md#faq)
2. Search [existing issues](https://github.com/dyuri/typconv/issues)
3. Create a [new issue](https://github.com/dyuri/typconv/issues/new) with:
   - Command used
   - Error message
   - Sample file (if possible)
   - Output of `typconv version`

## Performance Tips

### Large .img Files

For large .img files, use `--list` first:
```bash
# List before extracting
typconv extract large.img --list

# Then extract only what you need
```

### Batch Conversion

Use parallel processing:
```bash
# Process files in parallel (requires GNU parallel)
parallel typconv bin2txt {} -o {.}.txt ::: *.typ
```

### Memory Usage

typconv loads entire files into memory. For very large files:
- Use streaming mode (if available)
- Process files individually
- Increase system memory limits

## Best Practices

### 1. Always Validate

```bash
# Before using converted files
typconv validate output.typ
```

### 2. Keep Backups

```bash
# Before modifying
cp original.typ original.typ.backup
```

### 3. Test Round-trips

```bash
# Ensure no data loss
typconv bin2txt original.typ -o temp.txt
typconv txt2bin temp.txt -o test.typ
typconv diff original.typ test.typ
```

### 4. Version Control

```bash
# Track text versions in git
git add map.txt
git commit -m "Update point colors"

# Convert to binary for use
typconv txt2bin map.txt -o map.typ
```

## Next Steps

- See [BINARY_FORMAT.md](BINARY_FORMAT.md) for format details
- Check [README.md](../README.md) for project overview
- Visit the [project repository](https://github.com/dyuri/typconv) for updates

---

**Last Updated**: 2025-12-26
