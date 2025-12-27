# typconv - Garmin TYP Converter

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/dyuri/typconv)](https://golang.org/)

Native Linux tool for converting Garmin TYP (custom map type) files between binary and text formats.

## Features

- ✅ **Binary TYP → Text**: Convert binary TYP files to mkgmap-compatible text format
- ✅ **Text → Binary TYP**: Create binary TYP files from text definitions
- ✅ **Round-trip Conversion**: Full bidirectional conversion with data preservation
- ✅ **Character Encoding**: Automatic CodePage detection and support for all Windows codepages
- ✅ **No Wine required**: Pure Go implementation, works natively on Linux
- ✅ **Library support**: Can be used as a Go library in other projects

## Why typconv?

Currently, the only tool that can convert binary TYP files to text is `img2typ`, a Windows-only application. Linux users must either:
- Use Wine to run Windows tools
- Manually hex-edit binary files
- Give up on customizing maps from .img files

**typconv** changes this by providing the first open-source, native Linux implementation of the binary TYP format.

## Installation

### From Source

```bash
git clone https://github.com/dyuri/typconv
cd typconv
go build -o build/typconv ./cmd/typconv
```

### Using Go

```bash
go install github.com/dyuri/typconv/cmd/typconv@latest
```

## Quick Start

### Convert Binary to Text

```bash
# Convert a binary TYP file to text format
typconv bin2txt map.typ -o map.txt

# Convert and display to stdout
typconv bin2txt map.typ
```

### Convert Text to Binary

```bash
# Convert text format to binary TYP
# CodePage is automatically detected from the text file header
typconv txt2bin custom.txt -o custom.typ

# Override FID and PID if needed
typconv txt2bin custom.txt -o custom.typ --fid 3511 --pid 1

# Override CodePage (only if you need to force a specific encoding)
typconv txt2bin custom.txt -o custom.typ --codepage 1250
```

### Round-Trip Conversion

```bash
# Binary → Text → Binary preserves all data
typconv bin2txt original.typ -o temp.txt
typconv txt2bin temp.txt -o recreated.typ
typconv bin2txt recreated.typ -o verify.txt

# temp.txt and verify.txt should be identical
```

## Usage

### Commands

```
typconv [command] [flags] [arguments]

Available Commands:
  bin2txt      Convert binary TYP to text format
  txt2bin      Convert text format to binary TYP
  version      Show version information
  help         Show help for any command
```

### bin2txt Flags

```
  -o, --output FILE   Output file path (default: stdout)
  --no-xpm            Skip XPM bitmap data
  --no-labels         Skip label strings
```

### txt2bin Flags

```
  -o, --output FILE      Output file path (required)
  --fid NUMBER          Override Family ID
  --pid NUMBER          Override Product ID
  --codepage NUMBER     Override character encoding (auto-detected by default)
```

**Note**: The `--codepage` flag is optional. If not specified, typconv automatically reads the CodePage from the `[_id]` section of your text file.

### Character Encoding

typconv automatically detects and uses the correct character encoding:

| CodePage | Encoding           | Languages                    |
|----------|--------------------|------------------------------|
| 1252     | Windows-1252       | Western European             |
| 1250     | Windows-1250       | Central European (Hungarian) |
| 1251     | Windows-1251       | Cyrillic                     |
| 437      | CP437              | Original IBM PC              |

The text files are always UTF-8 encoded, and typconv handles the conversion automatically.

## Use as a Library

```go
package main

import (
    "os"
    "github.com/dyuri/typconv/pkg/typconv"
)

func main() {
    // Parse binary TYP file
    f, _ := os.Open("map.typ")
    defer f.Close()

    stat, _ := f.Stat()
    typ, err := typconv.ParseBinaryTYP(f, stat.Size())
    if err != nil {
        panic(err)
    }

    // Write to text format
    out, _ := os.Create("map.txt")
    defer out.Close()
    typconv.WriteTextTYP(out, typ)

    // Write back to binary
    outBin, _ := os.Create("map_new.typ")
    defer outBin.Close()
    typconv.WriteBinaryTYP(outBin, typ)
}
```

## Examples

### Working with Real Maps

```bash
# Convert OpenHiking TYP to text for editing
typconv bin2txt openhiking.typ -o editable.txt

# Edit editable.txt with your favorite text editor
vim editable.txt

# Convert back to binary
typconv txt2bin editable.txt -o custom.typ
```

### Batch Processing

```bash
# Convert all TYP files in a directory to text
for f in *.typ; do
    typconv bin2txt "$f" -o "${f%.typ}.txt"
done
```

## Text Format

typconv uses the mkgmap text format, which is compatible with the mkgmap compiler:

```
[_id]
CodePage=1252
FID=3511
ProductCode=1
[end]

[_point]
Type=0x2f06
SubType=0x00
String1=0x04,Trail Junction
String1=0x14,Főváros
DayColor=#ff0000
NightColor=#ff0000
DayXpm="8 8 2 1"
"! c #ff0000"
"  c none"
"!!!!!!!!"
"!      !"
"!      !"
"!      !"
"!      !"
"!      !"
"!      !"
"!!!!!!!!"
[end]
```

## Project Status

**Status**: Core functionality complete and production-ready ✅

### Completed Features
- ✅ Binary TYP reader (parser)
- ✅ Text format writer (mkgmap-compatible)
- ✅ Text format reader
- ✅ Binary TYP writer
- ✅ Full round-trip conversion (binary ↔ text)
- ✅ CLI framework with cobra
- ✅ Character encoding support (all Windows codepages)
- ✅ Automatic CodePage detection
- ✅ XPM bitmap handling (day/night patterns)
- ✅ Multi-language label support
- ✅ Comprehensive test suite

### Not Yet Implemented
- ⏳ `extract` command - Extract TYP from .img container files
- ⏳ `info` command - Display TYP file metadata
- ⏳ `validate` command - Validate TYP file structure
- ⏳ JSON output format

**The core mission is complete**: typconv can successfully convert binary TYP files to text and back, preserving all data including non-ASCII characters. This is the primary functionality needed for editing Garmin TYP files on Linux.

## Testing

Tested with real-world TYP files:

- **OpenHiking**: European hiking maps (402 points, 126 lines, 73 polygons)
- **OpenMTBMap**: Mountain bike maps
- **Various CodePages**: 1250 (Hungarian), 1252 (Western European), 437 (IBM PC)

All test files successfully complete round-trip conversion with 100% data preservation.

```bash
# Run test suite
go test ./...
```

## Related Projects

- **[typtui](https://github.com/dyuri/typtui)**: Terminal UI editor for TYP files (companion project)
- **[mkgmap](https://www.mkgmap.org.uk/)**: Compile OSM data into Garmin maps
- **img2typ**: Reference Windows implementation

## Documentation

- [Usage Guide](docs/USAGE.md): Comprehensive usage examples
- [Binary Format](docs/BINARY_FORMAT.md): Reverse engineering notes
- [AI Context](CLAUDE.md): Project context for AI assistants

## Resources

- [mkgmap TYP Documentation](https://www.mkgmap.org.uk/doc/typ-compiler)
- [OSM Wiki - Garmin Maps](https://wiki.openstreetmap.org/wiki/OSM_Map_On_Garmin)
- [TYP Format Guide](https://www.cferrero.net/maps/guide_to_TYPs.html)

## Contributing

Contributions are welcome! This is an open-source implementation of a previously undocumented format.

### Areas Where Help is Needed

- Testing with diverse TYP files from different map sources
- Implementation of .img container extraction
- Additional validation features
- Cross-platform testing (macOS, Windows)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- OpenStreetMap community for documentation and sample files
- mkgmap developers for the text format specification
- OpenHiking and OpenMTBMap projects for test files
- QMapShack project for binary format reference implementation

## Author

Created by dyuri

---

**Note**: This project implements the complete binary TYP format through reverse engineering and analysis of real files. Round-trip conversion is tested and verified with real-world maps.
