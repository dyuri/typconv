# Reverse Engineering Notes

This document tracks the process and findings from reverse engineering the Garmin binary TYP format.

## Methodology

### Tools Used

1. **Hex Editors**
   - `xxd` - Command-line hex dump
   - `hexdump` - Alternative hex viewer
   - Hex Fiend (GUI) - For visual analysis

2. **Reference Implementation**
   - `img2typ.exe` - Windows tool (via Wine)
   - Used to generate reference text output

3. **Sample Files**
   - OpenHiking maps
   - OpenMTBMap
   - Custom test files

4. **Analysis Tools**
   - `diff` - Compare hex dumps
   - Custom Go scripts for pattern detection

### Process

```bash
# 1. Collect sample binary TYP files
wget https://openhiking.eu/downloads/openhiking-CE.typ
wget https://openmtbmap.org/downloads/mtbmap.typ

# 2. Convert with img2typ (reference)
wine img2typ.exe openhiking-CE.typ
# Creates openhiking-CE.txt

# 3. Create hex dumps
xxd openhiking-CE.typ > openhiking-CE.hex
cat openhiking-CE.txt > openhiking-CE.txt

# 4. Correlate text with binary
# Look for patterns:
# - Type codes (e.g., 0x2f06) in hex at specific offsets
# - Label strings in ASCII in hex dump
# - Color values (RGB bytes)
# - Bitmap dimensions and data
```

## Findings

### Header Structure

#### Initial Discovery (2025-12-26)

Looking at multiple TYP files, the header appears to be approximately 64 bytes:

```
Offset 0x00-0x09: Variable signature/magic bytes
Offset 0x0A-0x0B: Version (uint16, LE)
Offset 0x0C-0x0D: CodePage (uint16, LE)
Offset 0x0E-0x0F: FID (uint16, LE)
Offset 0x10-0x11: PID (uint16, LE)
```

**Examples from real files:**

OpenHiking (FID=3511, PID=1, CodePage=1250):
```
00000000: ???? ???? ???? ???? ???? 0100 ba04 b704  ................
          [sig?]              ver  CP   FID  PID
```

**Note**: Need to examine more files to confirm exact signature pattern.

### Section Directory

Located at a header-specified offset (needs confirmation of exact location).

**Structure observed:**
```
+0x00: Count (uint16) - number of sections
+0x02: Section entries (12 bytes each?)
```

**Section entry format (tentative):**
```
+0x00: Type byte (0x01=points, 0x02=lines, 0x03=polygons)
+0x01: Offset (uint32, LE)
+0x05: Length (uint32, LE)
+0x09: Reserved (3 bytes?)
```

### Point Type Structure

**Observations from correlating binary with text:**

When `img2typ` outputs:
```
[_point]
Type=0x2f06
String1=0x04,Trail Junction
DayColor=#ff0000
```

The binary shows (approximate offsets):
```
06 2f 00 [flags] ... 01 04 54 72 61 69 6c 20 4a ...
^^type^^          ^count ^lang ^^"Trail J"...
```

**Flags byte** appears to indicate:
- Bit 0: Has icon
- Bit 1: Has day color
- Bit 2: Has night color

### Bitmap Format

**Pattern observed:**

```
Width Height ColorMode NumColors [Palette] [Data]
  08     08      08        10      [RGB...]  [pixels...]
```

For an 8x8, 16-color icon:
- Width: 0x08
- Height: 0x08
- Color mode: 0x08 (8-bit indexed)
- Palette entries: 10 (0x10) colors
- Palette: 16 * 3 bytes RGB = 48 bytes
- Pixel data: 64 bytes (8*8)

**Transparency**: Palette entry with R=G=B=0 often represents transparent.

### Language Codes

From comparing text output with binary:

| Code | Language   | Evidence                           |
|------|------------|------------------------------------|
| 0x01 | French     | Seen in multi-language TYP files   |
| 0x02 | German     | Common in European maps            |
| 0x03 | Dutch      | OpenBenelux maps                   |
| 0x04 | English    | Most common in our samples         |
| 0x05 | Italian    | Italian maps                       |

### Draw Order Section

**Status**: Not yet analyzed in detail

Hypothesis: Simple list of type codes in rendering order.

## Open Questions

### Critical Unknowns

1. **Header signature**: What are the exact magic bytes?
   - Do they vary by version?
   - Is there a fixed signature at all?

2. **Section directory offset**: Where exactly is it stored in the header?
   - Fixed offset?
   - Variable location?

3. **Extended types**: How are subtypes >0x1F encoded?
   - Is there an extended format?
   - Different section type?

4. **Draw order format**: Exact structure?
   - Separate lists for each type?
   - Single combined list?
   - Priority values or just order?

### Minor Unknowns

5. **Font style encoding**: What do the font style bytes mean?
6. **Line style values**: Confirmed values for solid/dashed/dotted?
7. **Version differences**: Do different versions use different formats?
8. **Reserved fields**: Purpose of unknown/reserved header bytes?

## Test Files Analyzed

### OpenHiking CE
- **FID**: 3511
- **PID**: 1
- **CodePage**: 1250
- **Size**: ~45KB
- **Notes**: Complex multi-language, many point types

### OpenMTBMap
- **FID**: 6277
- **PID**: 1
- **CodePage**: 1252
- **Size**: ~38KB
- **Notes**: Focus on trails, bike-specific types

### Minimal Test File
- **FID**: 9999
- **PID**: 1
- **CodePage**: 1252
- **Size**: ~2KB
- **Notes**: Created manually with 3 simple point types for testing

## Correlation Examples

### Example 1: Finding Type Code

**Text output (img2typ):**
```
[_point]
Type=0x2f06
```

**Binary (xxd):**
```
000001a0: 06 2f 00 03 01 04 54 72 61 69 6c 20 4a 75 6e 63  ./....Trail Junc
         ^^type^^
```

Type `0x2f06` appears as `06 2f` (little-endian).

### Example 2: Finding Label String

**Text output:**
```
String1=0x04,Trail Junction
```

**Binary:**
```
000001a0: 06 2f 00 03 01 04 54 72 61 69 6c 20 4a 75 6e 63  ./....Trail Junc
                     ^count ^lang ^^ASCII "Trail Junc"...
000001b0: 74 69 6f 6e 00                                   tion.
         ^^"tion" ^^null terminator
```

### Example 3: Finding Color

**Text output:**
```
DayColor=#ff0000
```

**Binary:**
```
000001c0: ff 00 00
         ^^R ^^G ^^B
```

RGB(255, 0, 0) = red.

## Validation Strategy

### Approach 1: Round-Trip Testing

```bash
# Start with known-good binary
original.typ

# Convert to text with img2typ (reference)
wine img2typ.exe original.typ
# Creates original.txt

# Convert with our tool
./typconv bin2txt original.typ -o our.txt

# Compare outputs
diff original.txt our.txt
```

### Approach 2: Binary Comparison

```bash
# Convert original to text
./typconv bin2txt original.typ -o temp.txt

# Convert back to binary
./typconv txt2bin temp.txt -o recreated.typ

# Compare key fields (not byte-for-byte, but semantically)
./typconv diff original.typ recreated.typ
```

### Approach 3: Incremental Testing

1. Start with minimal TYP (single point type, no icon)
2. Add complexity incrementally:
   - Add icon
   - Add colors
   - Add multiple labels
   - Add line types
   - Add polygon types
3. Validate each step

## Implementation Strategy

### Phase 1: Read-Only Parser

**Goal**: Parse binary → internal model

1. Implement header parsing
2. Implement section directory reading
3. Implement point type parsing (no bitmaps)
4. Test with real files
5. Add bitmap parsing
6. Add line and polygon types

**Success criteria**: Can parse 90%+ of real TYP files without errors.

### Phase 2: Text Writer

**Goal**: Internal model → text format

1. Implement header writing
2. Implement point type writing
3. Test output against img2typ reference
4. Add bitmap (XPM) writing
5. Add line and polygon writing

**Success criteria**: Output matches img2typ reference (semantically).

### Phase 3: Text Parser + Binary Writer

**Goal**: Full round-trip

1. Parse mkgmap text format
2. Write binary format
3. Test round-trip accuracy

**Success criteria**: binary → text → binary preserves data.

## Debugging Tips

### Finding Offsets

```bash
# Search for type code 0x2f06 in hex dump
xxd file.typ | grep "06 2f"

# Search for ASCII string
xxd file.typ | grep -i "trail"

# Show bytes around an offset
xxd -s 0x1a0 -l 64 file.typ
```

### Comparing Files

```bash
# Side-by-side hex comparison
diff -y <(xxd file1.typ) <(xxd file2.typ) | less

# Binary diff
cmp -l file1.typ file2.typ
```

### Pattern Detection

```go
// Find recurring patterns in binary
func findPatterns(data []byte) {
    // Look for repeating structures
    // E.g., section entries every 12 bytes
}
```

## Resources

### Documentation
- [mkgmap TYP compiler docs](https://www.mkgmap.org.uk/doc/typ-compiler)
- [cferrero.net TYP guide](https://www.cferrero.net/maps/guide_to_TYPs.html)
- [OSM Wiki Garmin page](https://wiki.openstreetmap.org/wiki/OSM_Map_On_Garmin)

### Community
- OSM forums - Garmin section
- mkgmap mailing list
- GarminDev forums (archived)

### Sample Files
- [OpenHiking downloads](https://openhiking.eu/)
- [OpenMTBMap downloads](https://openmtbmap.org/)
- [Freizeitkarte](https://www.freizeitkarte-osm.de/)

## Update Log

**2025-12-26**: Initial document created
- Documented methodology
- Recorded initial header findings
- Listed open questions
- Planned validation strategy

**Next steps**:
- Collect more diverse TYP files
- Analyze header signatures across versions
- Map out complete point type structure
- Begin parser implementation

---

**Note**: This is a living document. Update it as we discover new information during implementation.
