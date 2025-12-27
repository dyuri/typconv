// Package typconv provides functions for working with Garmin TYP files.
//
// This package can be used as a library to parse, convert, and generate
// TYP files programmatically.
//
// Example usage:
//
//	f, _ := os.Open("map.typ")
//	defer f.Close()
//	stat, _ := f.Stat()
//
//	typ, err := typconv.ParseBinaryTYP(f, stat.Size())
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	out, _ := os.Create("map.txt")
//	defer out.Close()
//	typconv.WriteTextTYP(out, typ)
package typconv

import (
	"io"

	"github.com/dyuri/typconv/internal/binary"
	"github.com/dyuri/typconv/internal/model"
	"github.com/dyuri/typconv/internal/text"
)

// ParseBinaryTYP reads a binary TYP file and returns the internal model.
//
// The reader must support ReadAt for random access. The size parameter
// should be the total file size in bytes.
//
// Example:
//
//	f, _ := os.Open("map.typ")
//	defer f.Close()
//	stat, _ := f.Stat()
//	typ, err := ParseBinaryTYP(f, stat.Size())
func ParseBinaryTYP(r io.ReaderAt, size int64) (*model.TYPFile, error) {
	reader := binary.NewReader(r, size)
	return reader.Parse()
}

// WriteTextTYP writes a TYP file in mkgmap text format.
//
// The output is compatible with the mkgmap TYP compiler and can be
// edited with a text editor.
//
// Example:
//
//	out, _ := os.Create("map.txt")
//	defer out.Close()
//	err := WriteTextTYP(out, typ)
func WriteTextTYP(w io.Writer, typ *model.TYPFile) error {
	writer := text.NewWriter(w)
	return writer.Write(typ)
}

// ParseTextTYP reads a mkgmap text format TYP file.
//
// The input should be in mkgmap-compatible text format with
// [_id], [_point], [_line], and [_polygon] sections.
//
// Example:
//
//	f, _ := os.Open("map.txt")
//	defer f.Close()
//	typ, err := ParseTextTYP(f)
func ParseTextTYP(r io.Reader) (*model.TYPFile, error) {
	reader := text.NewReader(r)
	return reader.Read()
}

// WriteBinaryTYP writes a binary TYP file.
//
// Currently not implemented.
func WriteBinaryTYP(w io.Writer, typ *model.TYPFile) error {
	// TODO: Implement binary writer
	return ErrNotImplemented
}

// ValidationError represents a validation issue found in a TYP file
type ValidationError struct {
	Field   string // Field name or location
	Message string // Error description
	Level   string // "error" or "warning"
}

// Validate checks a TYP file for structural and semantic errors.
//
// Returns a list of validation errors/warnings. An empty list means
// the file is valid.
//
// Currently not implemented.
func Validate(typ *model.TYPFile) []ValidationError {
	// TODO: Implement validation
	// - Check type code ranges
	// - Verify FID/PID
	// - Validate bitmap dimensions
	// - Check for duplicate type codes
	// - Verify label encoding
	return nil
}

// Common errors
var (
	ErrNotImplemented = &Error{Code: "not_implemented", Message: "feature not yet implemented"}
	ErrInvalidFormat  = &Error{Code: "invalid_format", Message: "invalid file format"}
	ErrInvalidHeader  = &Error{Code: "invalid_header", Message: "invalid TYP header"}
)

// Error represents a typconv error
type Error struct {
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}
