# Binary TYP Format Specification

This document describes the binary format of Garmin TYP files based on reverse engineering and community documentation.

**Status**: Work in Progress - findings will be documented as we implement the parser.

## Overview

TYP files use a binary format with the following characteristics:
- **Byte Order**: Little-endian
- **Structure**: Section-based with a directory
- **Strings**: Null-terminated, encoding specified by CodePage
- **Alignment**: Variable, no padding between records

## File Structure

```
┌─────────────────────────────────┐
│ File Header (64 bytes)          │ Identification and metadata
├─────────────────────────────────┤
│ Section Directory               │ Table of contents
├─────────────────────────────────┤
│ Point Types Section             │ POI definitions
├─────────────────────────────────┤
│ Line Types Section              │ Road/path definitions
├─────────────────────────────────┤
│ Polygon Types Section           │ Area definitions
├─────────────────────────────────┤
│ Draw Order Section              │ Rendering priority
├─────────────────────────────────┤
│ Icon/Bitmap Data                │ Embedded images
└─────────────────────────────────┘
```

## File Header

**Offset**: 0x00
**Size**: 64 bytes (estimated)

| Offset | Size | Type   | Description                          |
|--------|------|--------|--------------------------------------|
| 0x00   | 10   | byte[] | Magic/Signature (varies)             |
| 0x0A   | 2    | uint16 | Version                              |
| 0x0C   | 2    | uint16 | CodePage (1252, 1250, etc.)          |
| 0x0E   | 2    | uint16 | Family ID (FID)                      |
| 0x10   | 2    | uint16 | Product ID (PID)                     |
| 0x12   | ?    | byte[] | Reserved/Unknown                     |
| ?      | 4    | uint32 | Section Directory Offset             |

**Note**: Exact header structure needs verification with real files.

### Common CodePage Values

| Value | Encoding              | Region             |
|-------|-----------------------|--------------------|
| 1252  | Windows-1252          | Western Europe     |
| 1250  | Windows-1250          | Central Europe     |
| 1251  | Windows-1251          | Cyrillic           |
| 65001 | UTF-8                 | Unicode            |

## Section Directory

**Location**: Referenced by header offset

### Directory Header

| Offset | Size | Type   | Description                          |
|--------|------|--------|--------------------------------------|
| 0x00   | 2    | uint16 | Number of sections                   |

### Section Entry

Each entry is 12 bytes (estimated):

| Offset | Size | Type   | Description                          |
|--------|------|--------|--------------------------------------|
| 0x00   | 1    | uint8  | Section type                         |
| 0x01   | 4    | uint32 | Offset from file start               |
| 0x05   | 4    | uint32 | Section length in bytes              |
| 0x09   | 3    | byte[] | Reserved                             |

### Section Types

| Value | Section                |
|-------|------------------------|
| 0x01  | Point Types            |
| 0x02  | Line Types             |
| 0x03  | Polygon Types          |
| 0x04  | Draw Order             |
| 0x??  | (Other types TBD)      |

## Point Type Entry

**Location**: Point Types Section
**Size**: Variable

### Structure

| Offset | Size     | Type       | Description                       |
|--------|----------|------------|-----------------------------------|
| 0x00   | 2        | uint16     | Type code (e.g., 0x2f06)          |
| 0x02   | 1        | uint8      | SubType (0x00-0x1F)               |
| 0x03   | 1        | uint8      | Flags byte                        |
| 0x04   | variable | Bitmap?    | Icon data (if flags & 0x01)       |
| ?      | 1        | uint8      | Number of labels                  |
| ?      | variable | Label[]    | Language-specific labels          |
| ?      | 3        | RGB        | Day color (if flags & 0x02)       |
| ?      | 3        | RGB        | Night color (if flags & 0x04)     |
| ?      | 1        | uint8      | Font style (if flags & ?)         |

### Flags Byte (Offset 0x03)

| Bit | Meaning                |
|-----|------------------------|
| 0   | Has icon bitmap        |
| 1   | Has day color          |
| 2   | Has night color        |
| 3   | Extended labels?       |
| 4-7 | Unknown                |

### Label Entry

| Offset | Size     | Type    | Description                         |
|--------|----------|---------|-------------------------------------|
| 0x00   | 1        | uint8   | Language code                       |
| 0x01   | variable | string  | Null-terminated text                |

### Language Codes

| Value | Language  |
|-------|-----------|
| 0x01  | French    |
| 0x02  | German    |
| 0x03  | Dutch     |
| 0x04  | English   |
| 0x05  | Italian   |
| ...   | (TBD)     |

### RGB Color

| Offset | Size | Type  | Description |
|--------|------|-------|-------------|
| 0x00   | 1    | uint8 | Red         |
| 0x01   | 1    | uint8 | Green       |
| 0x02   | 1    | uint8 | Blue        |

## Line Type Entry

**Location**: Line Types Section
**Size**: Variable

**Structure**: TBD (similar to Point Type but with line-specific fields)

Key fields:
- Type code
- SubType
- Line width
- Border width
- Colors (line and border, day/night)
- Line style (solid, dashed, dotted)
- Pattern bitmap (optional)

## Polygon Type Entry

**Location**: Polygon Types Section
**Size**: Variable

**Structure**: TBD (similar to Point Type but with polygon-specific fields)

Key fields:
- Type code
- SubType
- Fill pattern bitmap (optional)
- Colors (fill, day/night)
- Font style for labels

## Bitmap Format

TYP files use a custom bitmap format similar to XPM but in binary.

### Structure (Preliminary)

| Offset | Size     | Type    | Description                         |
|--------|----------|---------|-------------------------------------|
| 0x00   | 1        | uint8   | Width in pixels                     |
| 0x01   | 1        | uint8   | Height in pixels                    |
| 0x02   | 1        | uint8   | Color mode (1/4/8/32)               |
| 0x03   | 1        | uint8   | Number of colors in palette         |
| 0x04   | variable | RGB[]   | Color palette                       |
| ?      | variable | byte[]  | Pixel data (indexed or direct)      |

### Color Modes

| Value | Mode       | Description                    |
|-------|------------|--------------------------------|
| 1     | Monochrome | 1 bit per pixel                |
| 4     | 16-color   | 4 bits per pixel               |
| 8     | 256-color  | 8 bits per pixel (indexed)     |
| 32    | True color | 24-bit RGB + 8-bit alpha       |

### Pixel Data

- **Indexed modes**: Each pixel is an index into the palette
- **True color**: Direct RGB values
- **Order**: Row-major (top-to-bottom, left-to-right)

### Transparency

One palette entry (typically index 0) can represent transparency:
- R=0, G=0, B=0, Alpha=0 indicates transparent pixel

## Draw Order Section

**Location**: Draw Order Section
**Size**: Variable

**Purpose**: Defines rendering priority for types

**Structure**: TBD

Likely format:
- Number of entries
- List of type codes in rendering order
- Separate lists for points, lines, polygons

## Unknown/Reserved Fields

Areas requiring further investigation:

### Header
- Exact signature/magic bytes
- Reserved fields at offsets 0x12-0x3F
- Section directory pointer exact location

### Type Entries
- Extended type encoding (SubType > 0x1F)
- All flag bit meanings
- Font style encoding
- Line style encoding

### Sections
- Draw order exact format
- Other section types beyond 0x01-0x04
- Version-specific differences

## Validation Rules

Based on observations:

1. **Type Codes**: Must be in Garmin's valid ranges
   - Points: 0x0100-0x7FFF
   - Lines: 0x0001-0x003F (basic), extended ranges TBD
   - Polygons: 0x0001-0x009F (basic), extended ranges TBD

2. **SubTypes**: 0x00-0x1F (5 bits), extended encoding TBD

3. **FID/PID**: Should match map family

4. **Bitmap dimensions**: Typically small (8x8 to 32x32)

## References

- mkgmap source code (Java implementation)
- Community forum discussions
- img2typ behavior analysis
- Real TYP file hex dumps

## Updates

This document will be updated as we:
1. Parse real TYP files
2. Compare with img2typ output
3. Test edge cases
4. Discover version differences

**Last Updated**: 2025-12-26
