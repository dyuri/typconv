package binary

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dyuri/typconv/internal/model"
)

// Reader handles parsing of binary TYP files
type Reader struct {
	r      io.ReaderAt
	size   int64
	endian binary.ByteOrder // Garmin uses little-endian
}

// NewReader creates a new binary TYP reader
func NewReader(r io.ReaderAt, size int64) *Reader {
	return &Reader{
		r:      r,
		size:   size,
		endian: binary.LittleEndian,
	}
}

// Parse reads the entire TYP file and returns the internal model
func (r *Reader) Parse() (*model.TYPFile, error) {
	typ := model.NewTYPFile()

	// Read header
	header, err := r.ReadHeader()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	typ.Header = *header

	// TODO: Read section directory
	// TODO: Parse point types
	// TODO: Parse line types
	// TODO: Parse polygon types
	// TODO: Parse draw order
	// TODO: Parse bitmaps

	return typ, fmt.Errorf("binary parser not yet implemented")
}

// ReadHeader reads and parses the TYP file header
func (r *Reader) ReadHeader() (*model.Header, error) {
	// Allocate buffer for header (estimated 64 bytes)
	buf := make([]byte, 64)
	if _, err := r.r.ReadAt(buf, 0); err != nil {
		return nil, fmt.Errorf("read header bytes: %w", err)
	}

	// TODO: Verify magic/signature bytes
	// TODO: Parse version, codepage, FID, PID from correct offsets
	// TODO: Validate header structure

	// Placeholder implementation
	header := &model.Header{
		Version:  0, // TODO: Parse from binary
		CodePage: 0, // TODO: Parse from binary
		FID:      0, // TODO: Parse from binary
		PID:      0, // TODO: Parse from binary
	}

	return header, fmt.Errorf("header parsing not yet implemented")
}

// Section represents a section in the TYP file
type Section struct {
	Type   byte   // Section type (1=points, 2=lines, 3=polygons, etc.)
	Offset uint32 // Offset from file start
	Length uint32 // Section length in bytes
}

// ReadSectionDirectory reads the section directory table of contents
func (r *Reader) ReadSectionDirectory(offset int64) ([]Section, error) {
	// TODO: Read section count
	// TODO: Parse section entries
	// TODO: Validate section structure

	return nil, fmt.Errorf("section directory parsing not yet implemented")
}

// ReadPointTypes reads all point type definitions from a section
func (r *Reader) ReadPointTypes(section Section) ([]model.PointType, error) {
	// TODO: Iterate through section
	// TODO: Parse each point type entry
	// TODO: Handle variable-length records

	return nil, fmt.Errorf("point type parsing not yet implemented")
}

// readPointType reads a single point type entry
func (r *Reader) readPointType(offset int64) (model.PointType, int, error) {
	// TODO: Read type code
	// TODO: Read subtype
	// TODO: Read flags
	// TODO: Parse icon if present
	// TODO: Read labels
	// TODO: Read colors

	return model.PointType{}, 0, fmt.Errorf("point type parsing not yet implemented")
}

// ReadLineTypes reads all line type definitions from a section
func (r *Reader) ReadLineTypes(section Section) ([]model.LineType, error) {
	// TODO: Parse line types similar to point types

	return nil, fmt.Errorf("line type parsing not yet implemented")
}

// ReadPolygonTypes reads all polygon type definitions from a section
func (r *Reader) ReadPolygonTypes(section Section) ([]model.PolygonType, error) {
	// TODO: Parse polygon types similar to point types

	return nil, fmt.Errorf("polygon type parsing not yet implemented")
}

// readBitmap reads bitmap data at the specified offset
func (r *Reader) readBitmap(offset int64) (*model.Bitmap, int, error) {
	// TODO: Read width, height
	// TODO: Read color mode
	// TODO: Read palette
	// TODO: Read pixel data

	return nil, 0, fmt.Errorf("bitmap parsing not yet implemented")
}

// readString reads a null-terminated string at the specified offset
func (r *Reader) readString(offset int64, maxLen int) (string, int, error) {
	buf := make([]byte, maxLen)
	if _, err := r.r.ReadAt(buf, offset); err != nil {
		return "", 0, err
	}

	// Find null terminator
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), i + 1, nil
		}
	}

	// No null terminator found within maxLen
	return string(buf), maxLen, nil
}
