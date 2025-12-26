package binary

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestReadHeader tests basic header parsing
func TestReadHeader(t *testing.T) {
	// Create a minimal TYP header
	buf := make([]byte, 64)

	// Offset 0x0A: Version = 1
	binary.LittleEndian.PutUint16(buf[0x0A:], 1)

	// Offset 0x0C: CodePage = 1252
	binary.LittleEndian.PutUint16(buf[0x0C:], 1252)

	// Offset 0x0E: FID = 3511
	binary.LittleEndian.PutUint16(buf[0x0E:], 3511)

	// Offset 0x10: PID = 1
	binary.LittleEndian.PutUint16(buf[0x10:], 1)

	reader := NewReader(bytes.NewReader(buf), int64(len(buf)))
	header, err := reader.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader failed: %v", err)
	}

	if header.Version != 1 {
		t.Errorf("Version = %d, want 1", header.Version)
	}
	if header.CodePage != 1252 {
		t.Errorf("CodePage = %d, want 1252", header.CodePage)
	}
	if header.FID != 3511 {
		t.Errorf("FID = %d, want 3511", header.FID)
	}
	if header.PID != 1 {
		t.Errorf("PID = %d, want 1", header.PID)
	}
}

// TestReadSectionDirectory tests section directory parsing
func TestReadSectionDirectory(t *testing.T) {
	buf := make([]byte, 100)

	// Section count = 2
	binary.LittleEndian.PutUint16(buf[0:], 2)

	// Section 1: Type=0x01, Offset=0x100, Length=0x50
	buf[2] = 0x01
	binary.LittleEndian.PutUint32(buf[3:], 0x100)
	binary.LittleEndian.PutUint32(buf[7:], 0x50)

	// Section 2: Type=0x02, Offset=0x150, Length=0x30
	buf[14] = 0x02
	binary.LittleEndian.PutUint32(buf[15:], 0x150)
	binary.LittleEndian.PutUint32(buf[19:], 0x30)

	reader := NewReader(bytes.NewReader(buf), int64(len(buf)))
	sections, err := reader.ReadSectionDirectory(0)
	if err != nil {
		t.Fatalf("ReadSectionDirectory failed: %v", err)
	}

	if len(sections) != 2 {
		t.Fatalf("Got %d sections, want 2", len(sections))
	}

	if sections[0].Type != 0x01 {
		t.Errorf("Section 0 Type = 0x%x, want 0x01", sections[0].Type)
	}
	if sections[0].Offset != 0x100 {
		t.Errorf("Section 0 Offset = 0x%x, want 0x100", sections[0].Offset)
	}
	if sections[0].Length != 0x50 {
		t.Errorf("Section 0 Length = 0x%x, want 0x50", sections[0].Length)
	}

	if sections[1].Type != 0x02 {
		t.Errorf("Section 1 Type = 0x%x, want 0x02", sections[1].Type)
	}
}

// TestReadPointTypeMinimal tests parsing a minimal point type
func TestReadPointTypeMinimal(t *testing.T) {
	buf := make([]byte, 1024) // Make buffer large enough for readPointType
	pos := 0

	// Type code: 0x2f06
	binary.LittleEndian.PutUint16(buf[pos:], 0x2f06)
	pos += 2

	// SubType: 0x00
	buf[pos] = 0x00
	pos++

	// Flags: 0x00 (no icon, no colors)
	buf[pos] = 0x00
	pos++

	// Label count: 1
	buf[pos] = 1
	pos++

	// Language code: 0x04 (English)
	buf[pos] = 0x04
	pos++

	// Label text: "Test" + null terminator
	copy(buf[pos:], "Test\x00")
	expectedBytes := pos + 5

	reader := NewReader(bytes.NewReader(buf), int64(len(buf)))
	pt, bytesRead, err := reader.readPointType(0)
	if err != nil {
		t.Fatalf("readPointType failed: %v", err)
	}

	if pt.Type != 0x2f06 {
		t.Errorf("Type = 0x%x, want 0x2f06", pt.Type)
	}
	if pt.SubType != 0x00 {
		t.Errorf("SubType = 0x%x, want 0x00", pt.SubType)
	}
	if len(pt.Labels) != 1 {
		t.Fatalf("Got %d labels, want 1", len(pt.Labels))
	}
	if pt.Labels["04"] != "Test" {
		t.Errorf("Label[04] = %q, want %q", pt.Labels["04"], "Test")
	}
	if bytesRead != expectedBytes {
		t.Errorf("bytesRead = %d, want %d", bytesRead, expectedBytes)
	}
}
