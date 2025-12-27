package img

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IMG file header (partial - only fields we need)
type IMGHeader struct {
	XORByte   uint8
	Reserved1 [15]byte
	Signature [7]byte // "DSKIMG" or "DSDIMG"
	Reserved2 [42]byte
	// Additional fields for block size calculation
	Identifier [7]byte  // 0x41-0x47
	Byte48     uint8
	Desc1      [20]byte // 0x49-0x5C
	Reserved3  [4]byte  // 0x5D-0x60
	E1         uint8    // 0x61 - for block size calculation
	E2         uint8    // 0x62 - for block size calculation
}

// FAT block structure
type FATBlock struct {
	Flag     uint8
	Name     [8]byte
	Type     [3]byte
	Size     uint32
	Part     uint16
	Reserved [14]byte
	Blocks   [240]uint16
}

// Subfile part location
type SubfilePart struct {
	Offset uint32
	Size   uint32
}

// ExtractTYP extracts TYP file(s) from a Garmin .img container file
// Returns a list of extracted TYP file paths
func ExtractTYP(imgPath string, outputDir string) ([]string, error) {
	// Open the IMG file
	file, err := os.Open(imgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open img file: %w", err)
	}
	defer file.Close()

	// Read and verify header
	var header IMGHeader
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Verify signature
	sig := strings.TrimRight(string(header.Signature[:]), "\x00")
	if sig != "DSKIMG" && sig != "DSDIMG" {
		return nil, fmt.Errorf("invalid IMG file signature: %s (expected DSKIMG or DSDIMG)", sig)
	}

	// Calculate block size from header
	blockSize := uint32(1 << (header.E1 + header.E2))

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse FAT blocks to find TYP subfiles
	typParts := make(map[string]SubfilePart)

	// Start reading FAT blocks from offset 0x600 (1536 bytes - after IMG header)
	offset := int64(0x600)

	for {
		// Seek to FAT block offset
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek to FAT block: %w", err)
		}

		// Read FAT block
		var fatBlock FATBlock
		if err := binary.Read(file, binary.LittleEndian, &fatBlock); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read FAT block: %w", err)
		}

		// Check if we've reached the end of FAT (flag == 0x00)
		if fatBlock.Flag == 0x00 {
			break
		}

		// Valid FAT blocks have flag == 0x01
		if fatBlock.Flag != 0x01 {
			// Skip invalid blocks
			offset += 512
			continue
		}

		// Get subfile name and type
		name := strings.TrimRight(string(fatBlock.Name[:]), "\x00 ")
		typ := strings.TrimRight(string(fatBlock.Type[:]), "\x00 ")

		// Check if this is a TYP subfile
		if typ == "TYP" {
			// Calculate actual file offset from FAT blocks
			fileOffset := calculateFileOffset(fatBlock.Blocks[:], blockSize)

			typParts[name] = SubfilePart{
				Offset: fileOffset,
				Size:   fatBlock.Size,
			}
		}

		// Move to next FAT block (512 bytes per block)
		offset += 512
	}

	// Extract all TYP files
	var extractedFiles []string
	for name, part := range typParts {
		// Seek to TYP file location
		if _, err := file.Seek(int64(part.Offset), io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek to TYP file %s: %w", name, err)
		}

		// Read TYP file data
		typData := make([]byte, part.Size)
		if _, err := io.ReadFull(file, typData); err != nil {
			return nil, fmt.Errorf("failed to read TYP file %s: %w", name, err)
		}

		// Create output file
		outputPath := filepath.Join(outputDir, name+".typ")
		outFile, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file %s: %w", outputPath, err)
		}

		// Write TYP data
		if _, err := outFile.Write(typData); err != nil {
			outFile.Close()
			return nil, fmt.Errorf("failed to write TYP file %s: %w", outputPath, err)
		}
		outFile.Close()

		extractedFiles = append(extractedFiles, outputPath)
	}

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no TYP files found in %s", imgPath)
	}

	return extractedFiles, nil
}

// calculateFileOffset calculates the actual file offset from FAT block numbers
func calculateFileOffset(blocks []uint16, blockSize uint32) uint32 {
	// Find the first non-zero block
	for _, block := range blocks {
		if block != 0 && block != 0xFFFF {
			// Calculate offset using the IMG's block size
			return uint32(block) * blockSize
		}
	}
	return 0
}
