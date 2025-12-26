# TYP Binary Format Investigation - oh_3690.typ

## Summary

The file `testdata/binary/oh_3690.typ` uses a binary format that differs from our initial specification. This document captures our reverse engineering progress.

## File Analysis

### Basic Info
- **Size**: 73,171 bytes (71KB)
- **Signature**: "GARMIN TYP" at offset 0x02-0x0B ✓
- **Version**: 1 (at offset 0x0C-0x0D)

### Header Structure (0x00-0x7F)

```
0x00-0x01: 0x009C (156) - Unknown field
0x02-0x0B: "GARMIN TYP" - Signature
0x0C-0x0D: 0x0001 (1) - Version
0x0E-0x0F: 0x007C (124) - Unknown
0x10-0x11: 0x0605 (1541) - Possible FID
0x12-0x13: 0x1E15 (7701) - Possible PID
... more fields ...
```

### Data Locations

**Type Codes Found**:
- 0x2F06 (POI - Waypoint) at offset 0x1F9A (8,090)
- 0x2F04 (POI - Gas Station) at offset 0x2F80 (12,160)
- 0x2F02 (POI - Parking) at offset 0x3402 (13,314)
- 0x6400 (City - Large) at offset 0x38D9 (14,553)

**Label Strings**:
- "Residential" at 0x00A2
- "Playground" at 0x00B5
- "Military" at 0x0154
- "Parking" at 0x0164
- And 1,600+ more strings

## Section Directory Search

### Attempted Formats

**1. Standard format (our initial spec)**:
- Count (uint16) + Type(1) + Offset(4) + Length(4) + Reserved(3) per entry
- **Result**: No matches found in first 512 bytes

**2. Simplified format (8-byte entries)**:
- Count (uint16) + Offset(4) + Length(4) per entry (no type field)
- **Result**: No valid matches

**3. Header-embedded offsets**:
- Checked if header contains direct pointers to sections
- **Result**: No clear pattern found

### Observations

1. **No traditional section directory found** using standard patterns
2. **Type data exists** - we can see valid Garmin type codes in the file
3. **Data is structured** - labels and type codes are present in recognizable patterns
4. **Format may be variant** - possibly uses a different directory structure or no directory at all

## Possible Explanations

### Theory 1: Non-Standard Format
This file might be created by a tool that uses a proprietary/non-standard TYP format.

### Theory 2: Compressed or Encrypted Sections
Some sections might use compression or encoding we're not recognizing.

### Theory 3: Different Version
The version field shows "1" but the format might have changed between versions.

### Theory 4: No Section Directory
The file might use a simpler format:
- Fixed offsets for each section
- Sequential layout (points, then lines, then polygons)
- Markers/delimiters between sections instead of a directory

## Next Steps

### Immediate Actions

1. **Get Reference Output**:
   ```bash
   wine img2typ.exe oh_3690.typ
   # Compare with expected text format
   ```

2. **Study mkgmap Source**:
   - Look at TYP binary writer in mkgmap Java code
   - Find exact format specification

3. **Try Alternative Tools**:
   - GPSMapEdit
   - cGPSmapper
   - TYPViewer

### Code Improvements

1. **Add Debug Mode**:
   - Verbose logging of parser decisions
   - Dump detected structures
   - Show where parser fails

2. **Flexible Parser**:
   - Try multiple format variants
   - Fall back to heuristic parsing
   - Extract what we can even if full parse fails

3. **Manual Override**:
   - Allow user to specify section offsets
   - Support partial parsing

## Resources Needed

- [ ] Access to img2typ.exe or similar tool
- [ ] mkgmap source code analysis
- [ ] TYP binary format specification (if exists)
- [ ] More sample TYP files for comparison
- [ ] Community forums (GPSPower, OSM, etc.)

## Workarounds

Until we crack the format:

1. **Use Windows Tools**:
   - Convert with img2typ under Wine
   - Use that as reference

2. **Partial Implementation**:
   - Parse what we can
   - Skip unknown sections
   - Extract labels and type codes heuristically

3. **Focus on Text→Binary**:
   - Implement text parser first
   - Use known-good text files
   - Binary writer can create files we understand

## Contact Points

- mkgmap mailing list
- OpenStreetMap Garmin forum
- GPSPower forum
- GarminDev mailing list (if still active)

---

**Status**: Investigation ongoing - need reference implementation or detailed spec
**Last Updated**: 2025-12-26
