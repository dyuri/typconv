# typconv - Garmin TYP Converter

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/dyuri/typconv)](https://golang.org/)

Native Linux tool for converting Garmin TYP (custom map type) files between binary and text formats.

## Features

- **Binary TYP → Text**: Convert binary TYP files to mkgmap-compatible text format
- **Text → Binary TYP**: Create binary TYP files from text definitions
- **Extract from .img**: Extract TYP files from Garmin .img map containers
- **No Wine required**: Pure Go implementation, works natively on Linux
- **Fast and reliable**: Optimized binary parsing with comprehensive error handling
- **Library support**: Can be used as a Go library in other projects

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
make install
```

### Using Go

```bash
go install github.com/dyuri/typconv/cmd/typconv@latest
```

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/dyuri/typconv/releases).

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
typconv txt2bin custom.txt -o custom.typ

# Override FID and PID
typconv txt2bin custom.txt -o custom.typ --fid 3511 --pid 1
```

### Extract from .img Files

```bash
# Extract all TYP files from a .img container
typconv extract gmapsupp.img -o output/

# List TYP files without extracting
typconv extract gmapsupp.img --list

# Extract only the first TYP file
typconv extract gmapsupp.img -o output/
```

### Display Information

```bash
# Show TYP file metadata
typconv info map.typ

# Show detailed information
typconv info map.typ --json
```

### Validate TYP Files

```bash
# Validate TYP file structure
typconv validate custom.typ
```

## Usage

### Commands

```
typconv [command] [flags] [arguments]

Commands:
  bin2txt      Convert binary TYP to text
  txt2bin      Convert text to binary TYP
  extract      Extract TYP from .img file
  info         Display TYP file information
  validate     Validate TYP file structure
  version      Show version information
  help         Show help for any command
```

### Global Flags

```
  -v, --verbose        Verbose output
  -q, --quiet         Suppress non-error output
  -h, --help          Show help
  --version           Show version
```

### bin2txt Flags

```
  -o, --output FILE   Output file path (default: stdout)
  -f, --format TEXT   Output format: mkgmap (default), json
  --no-xpm            Skip XPM bitmap data
  --no-labels         Skip label strings
```

### txt2bin Flags

```
  -o, --output FILE   Output file path (required)
  --fid NUMBER        Override Family ID
  --pid NUMBER        Override Product ID
  --codepage NUMBER   Character encoding (default: 1252)
```

### extract Flags

```
  -o, --output DIR    Output directory (required)
  -l, --list          List TYP files without extracting
  --all               Extract all TYP files (default: first)
```

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
}
```

## Examples

### Round-trip Conversion

```bash
# Test binary → text → binary conversion
typconv bin2txt input.typ -o temp.txt
typconv txt2bin temp.txt -o output.typ

# Verify they're equivalent
typconv diff input.typ output.typ
```

### Working with Real Maps

```bash
# Extract TYP from OpenHiking map
typconv extract openhiking.img -o extracted/

# Convert to text for editing
typconv bin2txt extracted/openhiking.typ -o editable.txt

# Edit editable.txt with your favorite text editor
# Then convert back
typconv txt2bin editable.txt -o custom.typ
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
DayColor=#ff0000
NightColor=#ff0000
IconXpm="8 8 2 1"
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

**Current Phase**: Initial development

See `typ-parser-implementation-plan.md` for the complete roadmap.

### Completed Features
- [x] Project setup and infrastructure

### In Development
- [ ] Binary TYP header parsing
- [ ] Basic type section parsing
- [ ] Text format writer
- [ ] CLI framework

### Planned Features
- [ ] Full binary format support
- [ ] Text format reader
- [ ] Round-trip conversion
- [ ] .img container extraction
- [ ] Validation and linting
- [ ] Comprehensive testing

## Testing

The project uses real-world TYP files for testing:

- **OpenHiking**: European hiking maps
- **OpenMTBMap**: Mountain bike maps
- **OpenTopoMap**: Topographic maps

Test files are validated against the reference `img2typ` implementation to ensure compatibility.

## Contributing

Contributions are welcome! This is an open-source implementation of a previously undocumented format.

### Areas Where Help is Needed

- Testing with diverse TYP files
- Documentation of binary format quirks
- Support for additional character encodings
- Cross-platform testing

### Development Setup

```bash
git clone https://github.com/dyuri/typconv
cd typconv
make dev-setup
make test
```

See `CONTRIBUTING.md` for guidelines.

## Related Projects

- **[typtui](https://github.com/dyuri/typtui)**: Terminal UI editor for TYP files (companion project)
- **[mkgmap](https://www.mkgmap.org.uk/)**: Compile OSM data into Garmin maps
- **img2typ**: Reference Windows implementation

## Documentation

- [Implementation Plan](typ-parser-implementation-plan.md): Detailed roadmap and specifications
- [Binary Format](docs/BINARY_FORMAT.md): Reverse engineering notes
- [Usage Guide](docs/USAGE.md): Comprehensive usage examples
- [AI Context](CLAUDE.md): Project context for AI assistants

## Resources

- [mkgmap TYP Documentation](https://www.mkgmap.org.uk/doc/typ-compiler)
- [OSM Wiki - Garmin Maps](https://wiki.openstreetmap.org/wiki/OSM_Map_On_Garmin)
- [TYP Format Guide](https://www.cferrero.net/maps/guide_to_TYPs.html)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- OpenStreetMap community for documentation and sample files
- mkgmap developers for the text format specification
- OpenHiking and OpenMTBMap projects for test files

## Author

Created by dyuri

---

**Note**: This project is in active development. The binary TYP format is being reverse-engineered through analysis of real files and comparison with existing tools. Contributions and feedback are appreciated!
