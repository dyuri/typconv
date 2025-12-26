# TYP Parser Progress Report

## Current Status: Array-Based Parser Working! 🎉🎉

We've successfully implemented the array-based parser and all 3 test files parse correctly!

### Latest Implementation (2025-12-26 15:30 UTC)

**What's Working:**
- ✅ Complete header parsing with all array metadata (offsets, modulo, sizes)
- ✅ Index array reading (3/4/5 byte entry formats)
- ✅ Type/subtype bit-unpacking (QMapShack algorithm)
- ✅ Variable-length label parsing with language codes
- ✅ Codepage decoding (CP1250, CP1252, UTF-8)
- ✅ Point/line/polygon type enumeration via arrays
- ✅ All 3 test files parse successfully!

**What's Pending:**
- ⚠️ Color table parsing (currently skipping over)
- ⚠️ Bitmap data extraction (estimating size, not parsing pixels)
- ⚠️ Full polyline/polygon data reading (structure works, data stubbed)

## Previous Status: Format Fully Documented! 🎉

We've successfully reverse-engineered the complete TYP file format by studying the QMapShack open-source implementation!

## Major Breakthrough ✅

### The Real Format Structure

The TYP format uses an **index/data array structure**, NOT sequential records as we initially thought:

1. **Header**: Contains metadata and section pointers
2. **Index Arrays**: Small arrays (3-5 bytes per entry) containing type codes and offsets
3. **Data Sections**: Variable-length records accessed via the index arrays

This explains why our initial implementation failed - we were trying to read sequentially instead of using the index!

## What We Now Know

### ✅ Complete Header Format
Documented all fields from 0x00 to 0x5B+:
- Descriptor, signature, version, date
- CodePage for text encoding
- Section offsets and lengths (data AND arrays)
- PID/FID
- Array metadata (offset, modulo, size)

**See**: `TYP-FORMAT-SPEC.md` for complete byte-by-byte documentation

### ✅ Index Array Structure
- Each geometry section has an index array
- Array entries are 3, 4, or 5 bytes (specified by `arrayModulo`)
- Entries contain bit-packed type/subtype + offset
- Offsets are relative to section's `dataOffset`

### ✅ Point Record Format
- Flags byte (localization, text colors, day/night mode)
- Width, Height, Number of colors, Color type
- Color table (RGB palette)
- Bit-packed bitmap data
- Optional labels (multi-language with length prefix)
- Optional text colors (day/night, label size)

### ✅ Text Encoding
- Uses codepage from header (1250, 1252, 65001=UTF-8)
- Hungarian files use CP1250 (Central European)
- Labels are null-terminated strings in specified encoding

### ✅ Color Handling
- Colors stored as BGR (not RGB!)
- Multiple color modes (standard, with alpha, transparency)
- Separate day/night palettes supported

## Source of Truth

**QMapShack**: Open-source C++ project that successfully renders Garmin TYP files
- Repository: https://github.com/Maproom/qmapshack
- Key file: `src/qmapshack/map/garmin/CGarminTyp.cpp`
- License: GPL v3.0 (we can learn from it, reimplement in Go)

## Test Results (Current Implementation)

### With Old Implementation
```bash
$ ./build/typconv bin2txt testdata/binary/M00000.typ
# Parses header ✓
# But reads wrong data (no index arrays) ❌
# Labels garbled (wrong offsets) ❌

$ ./build/typconv bin2txt testdata/binary/M03690.typ
# Error: invalid bitmap dimensions ❌

$ ./build/typconv bin2txt testdata/binary/oh_3690.typ
# Error: invalid bitmap dimensions ❌
```

### With New Understanding
- Need to reimplement using index/data structure
- Read array entries to find type definitions
- Use offsets from array to access data
- Decode type/subtype bit-packing correctly

## Files Created/Updated

### Documentation
- ✅ **TYP-FORMAT-SPEC.md** - Complete format specification
- ✅ **qmapshack-typ-parsing-findings.md** - QMapShack analysis notes
- ✅ **PROGRESS.md** - This file
- ✅ **FINDINGS.md** - Historical reverse engineering attempts

### Code (needs rewrite)
- ⚠️ **internal/binary/reader.go** - Current implementation is wrong
- ⚠️ **internal/text/writer.go** - XPM encoding works
- ⚠️ **internal/model/types.go** - Model is correct

## What Works

### ✅ Infrastructure
- Go project structure
- CLI framework (cobra)
- Build system
- Test files

### ✅ Partial Implementations
- Header parsing (codepage, offsets, etc.)
- Codepage decoding (Windows-1250, 1252, UTF-8)
- XPM encoder with 255-color support
- Buffer safety (no more crashes)
- Text writer skeleton

### ✅ Knowledge
- Complete format specification
- Working reference (QMapShack)
- Understanding of bit-packing, indexing, encoding

## What Needs To Be Done

### High Priority - Core Parser Rewrite

1. **Update Header Parsing** ✓ (mostly correct, needs array field parsing)
   - Add array offset/modulo/size fields
   - Parse all section metadata

2. **Implement Array Reading** ❌ (new code)
   ```go
   // Read index array
   numEntries := arraySize / arrayModulo
   for i := 0; i < numEntries; i++ {
       entry := readArrayEntry(arrayOffset + i*arrayModulo, arrayModulo)
       typ, subtyp := decodeTypeSubtype(entry.typeField)
       offset := entry.dataOffset

       // Read actual data
       data := readPointData(dataOffset + offset)
   }
   ```

3. **Implement Type/Subtype Decoding** ❌
   - Bit-pack/unpack 16-bit field
   - Handle extended types (0x10000 flag)

4. **Implement Point Data Reader** ❌
   - Read flags, dimensions, colors
   - Parse color table
   - Decode bitmap with correct bit depth
   - Parse labels (variable-length)
   - Parse text colors (optional)

5. **Implement Line Data Reader** ❌
   - Similar to points + line properties
   - Width, style, border

6. **Implement Polygon Data Reader** ❌
   - Similar to points + polygon properties
   - Fill pattern, border

### Medium Priority

- [ ] Draw order array parsing
- [ ] Comprehensive error handling
- [ ] Validation of parsed data
- [ ] Support all color modes
- [ ] Support all day/night modes

### Low Priority

- [ ] Text → Binary conversion
- [ ] Round-trip testing
- [ ] Format version support
- [ ] Optimization

## Implementation Strategy

### Phase 1: Header (partially done)
```go
type TYPHeader struct {
    Descriptor uint16
    // ... existing fields ...

    // Add section structs
    Points    SectionInfo
    Polylines SectionInfo
    Polygons  SectionInfo
    Order     SectionInfo
}

type SectionInfo struct {
    DataOffset   uint32
    DataLength   uint32
    ArrayOffset  uint32
    ArrayModulo  uint16
    ArraySize    uint32
}
```

### Phase 2: Array Reading
```go
type ArrayEntry struct {
    TypeCode   uint16
    DataOffset uint32  // 8, 16, or 24 bit depending on modulo
}

func (r *Reader) ReadIndexArray(section SectionInfo) ([]ArrayEntry, error) {
    numEntries := section.ArraySize / section.ArrayModulo
    entries := make([]ArrayEntry, numEntries)

    for i := 0; i < numEntries; i++ {
        offset := section.ArrayOffset + uint32(i) * uint32(section.ArrayModulo)
        entries[i] = r.readArrayEntry(offset, section.ArrayModulo)
    }

    return entries, nil
}
```

### Phase 3: Data Parsing
```go
func (r *Reader) readPointData(offset uint32) (*model.PointType, error) {
    flags := r.readUint8(offset)
    width := r.readUint8(offset + 1)
    height := r.readUint8(offset + 2)
    ncolors := r.readUint8(offset + 3)
    ctype := r.readUint8(offset + 4)

    hasLabels := (flags & 0x04) != 0
    hasTextColors := (flags & 0x08) != 0

    // Read color table
    palette := r.readColorTable(offset + 5, ncolors)

    // Read bitmap
    bpp := calculateBPP(ncolors)
    bitmap := r.readBitmap(offset + 5 + ncolors*3, width, height, bpp)

    // Read labels if present
    if hasLabels {
        labels := r.readLabels(...)
    }

    // Read text colors if present
    if hasTextColors {
        colors := r.readTextColors(...)
    }

    return pointType, nil
}
```

## Success Criteria

- [x] Understand format structure (index/data arrays)
- [x] Document complete format specification
- [x] Have working reference (QMapShack)
- [x] Parse header with array metadata
- [x] Read and decode index arrays
- [x] Parse point records correctly (labels working, bitmaps need work)
- [x] Parse line records correctly (structure working, data parsing stubbed)
- [x] Parse polygon records correctly (structure working, data parsing stubbed)
- [x] Extract labels with correct encoding
- [x] Successfully convert all 3 test files to text
- [ ] Match structure with QMapShack parsing (partially - need bitmap/color parsing)

**Current completion**: ~75% (format understood ✓, specification complete ✓, array parsing working ✓, bitmap parsing pending)

## Resources

### Documentation
- **Format Spec**: `TYP-FORMAT-SPEC.md` (complete)
- **QMapShack Findings**: `qmapshack-typ-parsing-findings.md`
- **Old Findings**: `FINDINGS.md` (historical)

### Reference Code
- **QMapShack**: `/tmp/qmapshack/src/qmapshack/map/garmin/CGarminTyp.cpp`
- **Repository**: https://github.com/Maproom/qmapshack

### Test Files
- `testdata/binary/M00000.typ` - CodePage 1252
- `testdata/binary/M03690.typ` - CodePage 1252
- `testdata/binary/oh_3690.typ` - CodePage 1250 (Hungarian)

## Next Steps

### Completed ✓

1. ✅ **Update header parsing** to read array metadata
2. ✅ **Implement array entry reader** (3/4/5 byte formats)
3. ✅ **Implement type/subtype decoder** (bit unpacking)
4. ✅ **Rewrite point data parser** using index/data structure
5. ✅ **Test with all 3 test files** - all parsing successfully!

### Immediate (Next priorities)

1. **Implement color table reader** - properly parse RGB/BGR palettes
2. **Implement bitmap bit unpacker** - decode 1/2/4/8 bpp pixel data correctly
3. **Fix label position calculation** - currently estimating bitmap size, should parse it
4. **Implement day/night bitmap handling** - read separate night bitmaps when present
5. **Implement text color parsing** - day/night label colors

### Follow-up

6. Complete polyline data parser (currently stubbed)
7. Complete polygon data parser (currently stubbed)
8. Implement draw order array parsing
9. Add validation and error handling
10. Round-trip testing (text → binary conversion)

## Code Status

### Implemented ✅
- ✅ Project structure
- ✅ CLI framework
- ✅ Model types
- ✅ Text writer (XPM encoding)
- ✅ Codepage handling (CP1250, CP1252, UTF-8)
- ✅ Header parsing with array metadata
- ✅ Section reading using index arrays
- ✅ Array entry reader (3/4/5 byte formats)
- ✅ Type/subtype bit decoder
- ✅ Variable-length label parser
- ✅ Point/line/polygon array iteration

### Needs Implementation
- ⚠️ Color table reader (currently skipping)
- ⚠️ Bitmap bit unpacker (currently estimating size)
- ⚠️ Day/night bitmap handling (partially implemented)
- ⚠️ Text color parser (not implemented)
- ⚠️ Polyline data parsing (structure done, data stubbed)
- ⚠️ Polygon data parsing (structure done, data stubbed)
- ⚠️ Draw order array parsing (not implemented)

---

**Status**: Array-based parser implemented and working! All 3 test files parse successfully.

**Next Action**: Implement color table and bitmap parsing for complete data extraction.

**Breakthrough #1**: QMapShack source code provided complete format understanding!

**Breakthrough #2**: Array-based parsing working - type codes, labels, and codepage decoding all functional!

**Test Results**:
- ✅ M00000.typ - parsing with CP1252, extracting types and labels
- ✅ M03690.typ - parsing with CP1252, Hungarian labels working
- ✅ oh_3690.typ - parsing with CP1250, proper Hungarian encoding (Főváros, Étterem, etc.)

**Last Updated**: 2025-12-26 15:30 UTC
