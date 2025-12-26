#!/usr/bin/env python3
"""Analyze TYP file structure to help reverse engineer the format."""

import struct
import sys

def analyze_typ(filename):
    with open(filename, 'rb') as f:
        data = f.read()

    print(f"=== TYP File Analysis: {filename} ===")
    print(f"File size: {len(data)} bytes ({len(data)//1024}KB)\n")

    # Header analysis
    print("HEADER ANALYSIS:")
    print(f"  0x00-0x01: {struct.unpack('<H', data[0:2])[0]:5d} (0x{struct.unpack('<H', data[0:2])[0]:04x})")

    if data[2:12] == b'GARMIN TYP':
        print(f"  0x02-0x0B: 'GARMIN TYP' ✓")
        print(f"  0x0C-0x0D: {struct.unpack('<H', data[0x0C:0x0E])[0]:5d} (0x{struct.unpack('<H', data[0x0C:0x0E])[0]:04x}) - likely version")
        print(f"  0x0E-0x0F: {struct.unpack('<H', data[0x0E:0x10])[0]:5d} (0x{struct.unpack('<H', data[0x0E:0x10])[0]:04x})")
        print(f"  0x10-0x11: {struct.unpack('<H', data[0x10:0x12])[0]:5d} (0x{struct.unpack('<H', data[0x10:0x12])[0]:04x})")
        print(f"  0x12-0x13: {struct.unpack('<H', data[0x12:0x14])[0]:5d} (0x{struct.unpack('<H', data[0x12:0x14])[0]:04x})")

    print("\nHex dump of first 128 bytes:")
    for i in range(0, 128, 16):
        line = data[i:i+16]
        hex_str = ' '.join(f'{b:02x}' for b in line)
        ascii_str = ''.join(chr(b) if 32 <= b < 127 else '.' for b in line)
        print(f"  {i:04x}: {hex_str:48} | {ascii_str}")

    # Search for common Garmin type codes
    print("\n\nSEARCHING FOR GARMIN TYPE CODES:")
    common_types = {
        0x2f01: "POI - Misc",
        0x2f02: "POI - Parking",
        0x2f03: "POI - Restaurant",
        0x2f04: "POI - Gas Station",
        0x2f05: "POI - Hotel",
        0x2f06: "POI - Waypoint",
        0x6400: "City - Large",
        0x6401: "City - Medium",
    }

    found_types = []
    for type_code, name in common_types.items():
        for i in range(len(data) - 2):
            val = struct.unpack('<H', data[i:i+2])[0]
            if val == type_code:
                found_types.append((i, type_code, name))
                break

    if found_types:
        print(f"Found {len(found_types)} type codes:")
        for offset, code, name in sorted(found_types)[:10]:
            print(f"  0x{offset:04x} ({offset:6d}): 0x{code:04x} - {name}")
            # Show context
            ctx_start = max(0, offset - 8)
            ctx = data[ctx_start:offset+20]
            print(f"    Context: {ctx.hex()}")
    else:
        print("  No common type codes found in expected format")

    # Look for ASCII strings (potential labels)
    print("\n\nASCII STRINGS (potential labels):")
    in_string = False
    string_start = 0
    strings = []

    for i, byte in enumerate(data):
        if 32 <= byte < 127:  # Printable ASCII
            if not in_string:
                in_string = True
                string_start = i
        else:
            if in_string and i - string_start >= 4:  # String of 4+ chars
                s = data[string_start:i].decode('ascii')
                if any(c.isalpha() for c in s):  # Has letters
                    strings.append((string_start, s))
            in_string = False

    # Show first 20 strings
    for offset, s in strings[:20]:
        if len(s) > 50:
            s = s[:47] + "..."
        print(f"  0x{offset:04x}: {s!r}")

    if len(strings) > 20:
        print(f"  ... and {len(strings)-20} more strings")

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print("Usage: analyze_typ.py <file.typ>")
        sys.exit(1)

    analyze_typ(sys.argv[1])
