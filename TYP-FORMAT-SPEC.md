# Garmin TYP File Format Specification

**Based on QMapShack implementation analysis**
**Source**: https://github.com/Maproom/qmapshack/blob/master/src/qmapshack/map/garmin/CGarminTyp.cpp
**Date**: 2025-12-26

## Overview

The Garmin TYP file format uses an **index/data array structure** rather than sequential records. Each geometry type section (POI/Points, Polylines, Polygons) has:
1. An **index array** containing type codes and offsets
2. A **data section** containing the actual type definitions

This two-level structure allows random access to type definitions.

## File Structure

```
┌─────────────────────────────────┐
│ Header                          │ <- Contains section metadata
├─────────────────────────────────┤
│ Variable-length data            │ <- Strings, extra data, etc.
├─────────────────────────────────┤
│ Points Index Array              │ <- arrayModulo × N entries
├─────────────────────────────────┤
│ Polylines Index Array           │
├─────────────────────────────────┤
│ Polygons Index Array            │
├─────────────────────────────────┤
│ Draw Order Array                │
├─────────────────────────────────┤
│ Points Data Section             │ <- Variable-length records
├─────────────────────────────────┤
│ Polylines Data Section          │
├─────────────────────────────────┤
│ Polygons Data Section           │
└─────────────────────────────────┘
```

## Header Format

**All multi-byte values are little-endian (Intel byte order)**

### Header Fields

| Offset | Size | Type   | Field              | Description                           |
|--------|------|--------|--------------------|---------------------------------------|
| 0x00   | 2    | uint16 | Descriptor         | Often equals header length            |
| 0x02   | 10   | char[] | Signature          | "GARMIN TYP" (not null-terminated)    |
| 0x0C   | 2    | uint16 | Version            | Format version (usually 1)            |
| 0x0E   | 2    | uint16 | Year               | Creation year - 1900                  |
| 0x10   | 1    | uint8  | Month              | Creation month (0-based!)             |
| 0x11   | 1    | uint8  | Day                | Creation day                          |
| 0x12   | 1    | uint8  | Hour               | Creation hour                         |
| 0x13   | 1    | uint8  | Minutes            | Creation minutes                      |
| 0x14   | 1    | uint8  | Seconds            | Creation seconds                      |
| 0x15   | 2    | uint16 | CodePage           | Text encoding (1252, 1250, 65001, etc)|
| 0x17   | 4    | uint32 | Points Data Offset | Byte offset to points data section    |
| 0x1B   | 4    | uint32 | Points Data Length | Length of points data section         |
| 0x1F   | 4    | uint32 | Lines Data Offset  | Byte offset to polylines data section |
| 0x23   | 4    | uint32 | Lines Data Length  | Length of polylines data section      |
| 0x27   | 4    | uint32 | Polygons Data Offset| Byte offset to polygons data section |
| 0x2B   | 4    | uint32 | Polygons Data Length| Length of polygons data section      |
| 0x2F   | 2    | uint16 | PID                | Product ID                            |
| 0x31   | 2    | uint16 | FID                | Family ID                             |
| 0x33   | 4    | uint32 | Points Array Offset| Byte offset to points index array     |
| 0x37   | 2    | uint16 | Points Array Modulo| Size of each array entry (3, 4, or 5) |
| 0x39   | 4    | uint32 | Points Array Size  | Total size of points array in bytes   |
| 0x3D   | 4    | uint32 | Lines Array Offset | Byte offset to polylines index array  |
| 0x41   | 2    | uint16 | Lines Array Modulo | Size of each array entry              |
| 0x43   | 4    | uint32 | Lines Array Size   | Total size of polylines array         |
| 0x47   | 4    | uint32 | Polygons Array Offset| Byte offset to polygons index array |
| 0x4B   | 2    | uint16 | Polygons Array Modulo| Size of each array entry            |
| 0x4D   | 4    | uint32 | Polygons Array Size| Total size of polygons array          |
| 0x51   | 4    | uint32 | Order Array Offset | Byte offset to draw order array       |
| 0x55   | 2    | uint16 | Order Array Modulo | Size of each array entry (usually 5)  |
| 0x57   | 4    | uint32 | Order Array Size   | Total size of draw order array        |

**Total minimum header size**: 0x5B (91 bytes)

### CodePage Values

| Value | Encoding                  |
|-------|---------------------------|
| 1250  | Windows-1250 (Central European, includes Hungarian) |
| 1252  | Windows-1252 (Western European) |
| 65001 | UTF-8                     |

## Index Array Structure

### Array Basics

Each section has an associated index array:
- **Location**: `arrayOffset`
- **Entry count**: `arraySize / arrayModulo`
- **Entry size**: `arrayModulo` bytes (3, 4, or 5)

### Array Entry Format

The array entry format varies by `arrayModulo`:

#### 5-byte entries (most common):
```
Bytes 0-1: Type/Subtype (uint16, bit-packed)
Bytes 2-4: Data offset (uint32, 24-bit value)
```

#### 4-byte entries:
```
Bytes 0-1: Type/Subtype (uint16, bit-packed)
Bytes 2-3: Data offset (uint16)
```

#### 3-byte entries:
```
Bytes 0-1: Type/Subtype (uint16, bit-packed)
Byte 2:    Data offset (uint8)
```

### Type/Subtype Encoding

The 16-bit type/subtype field is bit-packed:

```c++
// Decode (from QMapShack):
t16_2 = (t16_1 >> 5) | ((t16_1 & 0x1f) << 11);
typ = t16_2 & 0x7FF;         // 11 bits for type
subtyp = t16_1 & 0x01F;      // 5 bits for subtype

// Extended type (if bit 0x2000 is set):
if (t16_1 & 0x2000) {
    typ = 0x10000 | (typ << 8) | subtyp;
} else {
    typ = (typ << 8) + subtyp;
}
```

The `offset` value is relative to the section's `dataOffset`.

## Point (POI) Record Format

**Location**: `sectPoints.dataOffset + offset` (from array entry)

### Record Structure

| Offset | Size | Type  | Field          | Description                          |
|--------|------|-------|----------------|--------------------------------------|
| 0      | 1    | uint8 | Flags          | See flag bits below                  |
| 1      | 1    | uint8 | Width          | Icon width in pixels                 |
| 2      | 1    | uint8 | Height         | Icon height in pixels                |
| 3      | 1    | uint8 | Number of colors| Palette size                       |
| 4      | 1    | uint8 | Color type     | Color/transparency mode              |
| 5+     | var  | -     | Color table    | Palette (3 bytes RGB per color)      |
| ?      | var  | -     | Bitmap data    | Pixel data (bit-packed)              |
| ?      | var  | -     | Labels         | Optional (if bit 2 set in flags)     |
| ?      | var  | -     | Text colors    | Optional (if bit 3 set in flags)     |

### Flag Bits (Byte 0)

| Bit | Mask | Meaning                              |
|-----|------|--------------------------------------|
| 0-1 | 0x03 | Day/Night mode (0=day only, 2=day+night shared, 3=separate) |
| 2   | 0x04 | Has localization (labels)            |
| 3   | 0x08 | Has text color settings              |

### Color Type Values (Byte 4)

| Value | Meaning                              |
|-------|--------------------------------------|
| 0x00  | Standard color                       |
| 0x10  | With transparency (alpha channel)    |
| 0x20  | With alpha in color table            |

### Bitmap Data

The bitmap is bit-packed based on the color depth:
- **1 bpp**: 1 bit per pixel (monochrome)
- **2 bpp**: 2 bits per pixel (4 colors)
- **4 bpp**: 4 bits per pixel (16 colors)
- **8 bpp**: 8 bits per pixel (256 colors)

Bits per pixel is calculated from number of colors: `ceil(log2(ncolors))`

### Label Data Format (if flag bit 2 set)

```
Byte 0:    Length byte (if bit 0 clear, read second byte for uint16 length)
If length has bit 0 clear:
  Byte 1:  High byte of length
  n = 2    (length is 2 bytes)
Else:
  n = 1    (length is 1 byte)

len -= n;  // Subtract length field size

Then repeating for each language:
  Byte:      Language code (uint8)
  len -= 2 * n  // Each byte costs 2*n in the length counter

  Then for each byte of the null-terminated string:
    Byte:    Character (0x00 = terminator)
    len -= 2 * n  // Each byte (including null) costs 2*n

Continue until len == 0

NOTE: This is the QMapShack algorithm - every single byte (language code +
all string bytes including null terminator) decrements the counter by 2*n.
For a label with n=1, lang=0x03, string "Test\0" (5 bytes):
  Initial len includes: n + (2*n × 6) = 1 + 12 = 13
  After header: len = 12
  After processing: 12 - 2 - 2 - 2 - 2 - 2 - 2 = 0
```

**Language code examples**:
- 0x00: Unspecified
- 0x03: English
- 0x04: French
- 0x05: German
- etc.

The string is encoded using the codepage from the header.

### Text Color Format (if flag bit 3 set)

```
Byte 0:    Label type and color flags
           Bits 0-2: Label type (0=standard, 1=none, 2=small, 3=normal, 4=large)
           Bit 3:    Has day color (read 3 bytes RGB)
           Bit 4:    Has night color (read 3 bytes RGB)

If bit 3 set:
  Bytes 1-3: Day color (B, G, R) - note BGR order!

If bit 4 set:
  Bytes:     Night color (B, G, R)
```

## Polyline Record Format

**Location**: `sectPolylines.dataOffset + offset`

Structure is similar to points but includes line-specific properties:
- Line width
- Line style (solid, dashed, etc.)
- Border width and color
- Optional pattern bitmap (for textured lines)
- Labels (same format as points)
- Text colors (same format as points)

## Polygon Record Format

**Location**: `sectPolygons.dataOffset + offset`

Structure includes polygon-specific properties:
- Fill pattern
- Border pen
- Optional fill bitmap (32×32 pattern)
- Labels (same format as points)
- Text colors (same format as points)

## Draw Order Array

**Location**: `sectOrder.arrayOffset`
**Entry size**: Usually 5 bytes (arrayModulo = 5)

Controls the rendering order of polygons (drawn first = background).

### Entry Format

```
Byte 0:    Type (uint8)
Bytes 1-4: Subtype bitmap (uint32)
```

If `subtype == 0`, the entire type is prioritized.
Otherwise, bits in the subtype bitmap indicate which subtypes to prioritize.

## Example: Parsing Point Types

### Step 1: Read Header
```
descriptor = read_uint16(0x00)          // e.g., 0x005B
signature = read_string(0x02, 10)       // "GARMIN TYP"
codepage = read_uint16(0x15)            // e.g., 1252
points_data_offset = read_uint32(0x17)  // e.g., 0x00000C5E
points_data_length = read_uint32(0x1B)
points_array_offset = read_uint32(0x33) // e.g., 0x0000005B
points_array_modulo = read_uint16(0x37) // e.g., 5
points_array_size = read_uint32(0x39)   // e.g., 0x00000050
```

### Step 2: Iterate Array
```
num_points = points_array_size / points_array_modulo

for i in 0..num_points:
    array_pos = points_array_offset + (i * points_array_modulo)

    // Read array entry
    type_subtype = read_uint16(array_pos)
    offset = read_uint24(array_pos + 2)  // or uint16/uint8 depending on modulo

    // Decode type/subtype
    typ, subtyp = decode_type_subtype(type_subtype)

    // Read point data
    data_pos = points_data_offset + offset
    flags = read_uint8(data_pos + 0)
    width = read_uint8(data_pos + 1)
    height = read_uint8(data_pos + 2)
    ncolors = read_uint8(data_pos + 3)
    ctype = read_uint8(data_pos + 4)

    // Read color table, bitmap, labels, etc.
    ...
```

## Test File Analysis

### M00000.typ

```
Header length: 0x5B (91)
CodePage: 1252 (Windows Western European)
Points array: offset=0x005B, modulo=5, size=?
Points data: offset=?, length=?
```

### oh_3690.typ

```
Header length: 0x9C (156)
CodePage: 1250 (Windows Central European - Hungarian)
Points array: offset=?, modulo=?, size=?
Points data: offset=?, length=?
```

## Implementation Notes

1. **Byte Order**: All multi-byte integers are little-endian
2. **Text Encoding**: Use codepage from header (1250, 1252, 65001=UTF-8)
3. **Color Order**: Colors in file are BGR, not RGB
4. **Bounds Checking**: Always verify offsets are within file bounds
5. **Variable Length**: Record sizes vary - use flags to determine what fields are present
6. **Index Required**: Must use array to find records, can't read sequentially

## References

- **QMapShack Source**: https://github.com/Maproom/qmapshack/blob/master/src/qmapshack/map/garmin/CGarminTyp.cpp
- **Original Research**: http://ati.land.cz/gps/typdecomp/editor.cgi (noted in QMapShack)
- **pinns.co.uk**: https://www.pinns.co.uk/osm/typformat.html (partially accurate)

## Revision History

- **2025-12-28**: Corrected label length calculation algorithm (lines 185-211) - every byte costs 2*n, not a single subtraction per entry
- **2025-12-26**: Initial specification based on QMapShack code analysis

---

**Status**: Complete header and point record format documented. Polyline and polygon formats need detailed documentation from QMapShack code.
