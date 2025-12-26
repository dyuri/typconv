# typconv - Project Context for AI Assistants

## Project Overview

**typconv** is a command-line tool and Go library for working with Garmin TYP (custom map type) files. It enables conversion between binary TYP format and text format (mkgmap-compatible), and can extract TYP files from Garmin .img container files.

**Primary Goal**: Provide Linux users with a native tool to work with binary TYP files without requiring Wine or Windows-only tools like img2typ.

**Why This Matters**:
- Binary TYP files are embedded in all Garmin .img maps
- Currently, only Windows tools can convert binary → text
- This is the first open-source implementation of the binary TYP format
- Enables the companion project `typtui` to work with real-world maps

## Current Project Status

**Phase**: Initial setup and infrastructure
**Implementation Plan**: See `typ-parser-implementation-plan.md` for detailed roadmap

### Completed
- [x] Project structure created
- [x] Go module initialized
- [x] Planning and specification document

### In Progress
- [ ] Initial file setup (README, LICENSE, docs)
- [ ] Basic CLI framework

### Upcoming (Phase 1 - MVP)
- [ ] Binary TYP header parsing
- [ ] Basic metadata extraction (FID, PID, CodePage)
- [ ] Point/Line/Polygon type parsing
- [ ] Text format writer (mkgmap format)
- [ ] Basic CLI commands

## Project Architecture

### Directory Structure

```
typconv/
├── cmd/typconv/          # CLI entry point
├── internal/             # Internal packages (not exported)
│   ├── binary/          # Binary TYP reader/writer
│   ├── text/            # Text format reader/writer
│   ├── model/           # Unified internal data model
│   ├── img/             # .img container parser
│   └── converter/       # Conversion orchestration
├── pkg/typconv/         # Public API for library use
├── testdata/            # Test TYP/img files
├── docs/                # Documentation
└── scripts/             # Build and test scripts
```

### Core Data Flow

```
Binary TYP → Reader → Internal Model → Writer → Text Format
     ↑                                             ↓
     └─────────────── Round-trip ─────────────────┘
```

### Key Packages

1. **internal/model**: Unified representation of TYP data
   - `TYPFile`: Complete TYP structure
   - `PointType`, `LineType`, `PolygonType`: Map element definitions
   - `Bitmap`: Icon/pattern image data
   - Language-agnostic, format-agnostic

2. **internal/binary**: Binary format parsing/writing
   - Little-endian binary format
   - Section-based structure
   - XPM-like bitmap encoding
   - Must handle various format versions

3. **internal/text**: mkgmap text format
   - INI-like format with `[_point]`, `[_line]`, `[_polygon]` sections
   - XPM bitmap data in text form
   - Must be compatible with mkgmap compiler

4. **internal/img**: .img container support
   - FAT-based container format
   - Multiple subfiles (map data + TYP)
   - TYP extraction

5. **pkg/typconv**: Public API
   - Simple functions for common operations
   - Can be imported by other Go projects (like typtui)

## Technical Concepts

### Binary TYP Format

The binary format consists of:
1. **File Header**: Identification, version, CodePage, FID/PID
2. **Section Directory**: Table of contents pointing to data sections
3. **Type Sections**: Point, Line, Polygon definitions
4. **Draw Order**: Rendering priority
5. **Bitmap Data**: Embedded icons/patterns

**Encoding**: Little-endian, variable-length records, null-terminated strings

**Character Sets**: CodePage field specifies encoding (1252=Western, 1250=Central European, etc.)

### Text Format (mkgmap)

Example:
```
[_id]
CodePage=1252
FID=3511
ProductCode=1
[end]

[_point]
Type=0x2f06
String1=0x04,Trail Junction
DayColor=#ff0000
IconXpm="8 8 2 1"
"! c #ff0000"
"  c none"
"!!!!!!"
"!    !"
...
[end]
```

### Data Model Design Principles

- **Format-agnostic**: Model doesn't know about binary vs text
- **Lossless**: Can represent all features of both formats
- **Type-safe**: Use Go types to prevent errors
- **Complete**: Includes metadata, all element types, bitmaps

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Error handling: return errors, don't panic (except for programmer errors)
- Comments: Document exported functions and non-obvious logic

### Testing Strategy

1. **Unit Tests**: Test individual parsers/writers
2. **Round-trip Tests**: Binary → Text → Binary should preserve data
3. **Real-world Files**: Test with actual OpenHiking, OpenMTBMap files
4. **Validation Tests**: Compare output with img2typ reference

### Error Handling

- Parse errors should be descriptive (include offset, expected vs actual)
- Invalid files: fail gracefully with clear messages
- Partial parsing: consider recovering when possible

### Dependencies

Keep dependencies minimal:
- Standard library for most operations
- `github.com/spf13/cobra` for CLI (industry standard)
- Consider `golang.org/x/text/encoding` for character set conversion
- No UI dependencies (pure CLI/library)

## Key Implementation Notes

### Binary Parsing Challenges

1. **Variable-length records**: Types have different sizes based on flags
2. **String encoding**: Must respect CodePage setting
3. **Bitmap format**: Custom format similar to XPM but binary
4. **Unknown fields**: Some header fields are not fully documented
5. **Format versions**: May need to handle variations

### XPM/Bitmap Handling

- Binary: Color palette + indexed pixel data
- Text: XPM format with character mapping
- Conversion: Map binary palette indices to XPM characters
- Transparency: Handle "none" color

### Performance Considerations

- Files are typically small (<100KB)
- Use `io.ReaderAt` for random access (avoid loading entire file)
- Binary parsing should be fast (<100ms typical)
- Memory efficiency: Don't duplicate large bitmaps

## Common Tasks for AI Assistants

### When Adding Binary Format Support

1. Check `typ-parser-implementation-plan.md` for format specification
2. Update `internal/model` if new fields needed
3. Implement in `internal/binary/reader.go` or `writer.go`
4. Add corresponding test in `*_test.go`
5. Document findings in `docs/BINARY_FORMAT.md`

### When Adding Text Format Support

1. Check mkgmap documentation for syntax
2. Implement in `internal/text/reader.go` or `writer.go`
3. Ensure round-trip compatibility
4. Test with real mkgmap files

### When Adding CLI Commands

1. Add command to `cmd/typconv/main.go`
2. Follow existing command structure
3. Add flags with `cobra`
4. Update `README.md` usage section

## References

### Documentation
- Implementation Plan: `typ-parser-implementation-plan.md`
- Binary Format: `docs/BINARY_FORMAT.md` (to be created)
- Usage Guide: `docs/USAGE.md` (to be created)

### External Resources
- mkgmap TYP compiler: https://www.mkgmap.org.uk/doc/typ-compiler
- OSM Wiki: https://wiki.openstreetmap.org/wiki/OSM_Map_On_Garmin
- TYP format guide: https://www.cferrero.net/maps/guide_to_TYPs.html

### Related Projects
- typtui: TUI editor for TYP files (companion project)
- mkgmap: Creates Garmin maps from OSM data
- img2typ: Windows tool for TYP extraction (reference implementation)

## Success Criteria

### Technical
- Parse 95%+ of real-world TYP files
- Round-trip accuracy >99%
- Performance: <100ms for typical files
- Zero crashes on malformed input

### User Experience
- Single command converts files
- Clear error messages
- Works with popular maps (OpenHiking, OpenMTBMap)
- Can be used as library by typtui

## Current Development Focus

**Immediate priorities:**
1. Set up basic project infrastructure (README, LICENSE, docs)
2. Implement basic CLI framework with cobra
3. Create data model in `internal/model`
4. Implement binary header parsing
5. Test with real TYP files

**Next phase:**
- Full binary reader implementation
- Text format writer
- Round-trip testing

## Notes for AI Assistants

- **Always check the implementation plan** before making architectural decisions
- **Test with real files** from OpenHiking/OpenMTBMap when possible
- **Document reverse engineering findings** in `docs/BINARY_FORMAT.md`
- **Maintain compatibility** with mkgmap text format
- **Keep it simple**: This is a conversion tool, not a full map editor
- **Performance matters**: Binary parsing should be fast
- **The goal is Linux native**: No Wine, no Windows dependencies

## Questions/Uncertainties

Track unknowns here as development progresses:

- [ ] Exact binary format for extended types (subtypes >0x1F)
- [ ] Handling of NT map format variations
- [ ] All CodePage values and their encodings
- [ ] Meaning of reserved/unknown header fields
- [ ] Draw order section exact format

Update this file as you discover answers!
