package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dyuri/typconv/internal/img"
	"github.com/dyuri/typconv/internal/model"
	"github.com/dyuri/typconv/pkg/typconv"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "typconv",
	Short: "Convert Garmin TYP files between binary and text formats",
	Long: `typconv is a tool for working with Garmin TYP files.

It can convert between binary and text formats, extract TYP files from
.img containers, inspect file metadata, validate structure, and export
to JSON format.

This is the first native Linux implementation of the binary TYP format.`,
}

func init() {
	rootCmd.AddCommand(bin2txtCmd)
	rootCmd.AddCommand(txt2binCmd)
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
}

// bin2txt command
var bin2txtCmd = &cobra.Command{
	Use:   "bin2txt <input.typ>",
	Short: "Convert binary TYP to text format",
	Long: `Convert a binary TYP file to mkgmap-compatible text format.

The output can be edited and converted back to binary with txt2bin.`,
	Args: cobra.ExactArgs(1),
	RunE: runBin2Txt,
}

func init() {
	bin2txtCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	bin2txtCmd.Flags().String("format", "mkgmap", "Output format: mkgmap, json")
	bin2txtCmd.Flags().Bool("no-xpm", false, "Skip XPM bitmap data")
	bin2txtCmd.Flags().Bool("no-labels", false, "Skip label strings")
}

func runBin2Txt(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	outputPath, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	noXPM, _ := cmd.Flags().GetBool("no-xpm")
	noLabels, _ := cmd.Flags().GetBool("no-labels")

	// Open input file
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer f.Close()

	// Get file size
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat input file: %w", err)
	}

	// Parse binary TYP
	typ, err := typconv.ParseBinaryTYP(f, stat.Size())
	if err != nil {
		return fmt.Errorf("parse TYP file: %w", err)
	}

	// Apply filters
	if noXPM {
		stripXPMData(typ)
	}
	if noLabels {
		stripLabels(typ)
	}

	// Determine output writer
	var output *os.File
	if outputPath == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer output.Close()
	}

	// Write output
	switch format {
	case "mkgmap":
		return typconv.WriteTextTYP(output, typ)
	case "json":
		return writeJSONTYP(output, typ)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func stripXPMData(typ *model.TYPFile) {
	for i := range typ.Points {
		typ.Points[i].DayIcon = nil
		typ.Points[i].NightIcon = nil
	}
	for i := range typ.Lines {
		typ.Lines[i].DayPattern = nil
		typ.Lines[i].NightPattern = nil
	}
	for i := range typ.Polygons {
		typ.Polygons[i].DayPattern = nil
		typ.Polygons[i].NightPattern = nil
	}
}

func stripLabels(typ *model.TYPFile) {
	for i := range typ.Points {
		typ.Points[i].Labels = make(map[string]string)
	}
	for i := range typ.Lines {
		typ.Lines[i].Labels = make(map[string]string)
	}
	for i := range typ.Polygons {
		typ.Polygons[i].Labels = make(map[string]string)
	}
}

func writeJSONTYP(w *os.File, typ *model.TYPFile) error {
	// Create JSON-friendly structure
	output := map[string]interface{}{
		"header": map[string]interface{}{
			"fid":      typ.Header.FID,
			"pid":      typ.Header.PID,
			"codepage": typ.Header.CodePage,
		},
		"points":   convertPointsToJSON(typ.Points),
		"lines":    convertLinesToJSON(typ.Lines),
		"polygons": convertPolygonsToJSON(typ.Polygons),
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func convertPointsToJSON(points []model.PointType) []map[string]interface{} {
	result := make([]map[string]interface{}, len(points))
	for i, pt := range points {
		entry := map[string]interface{}{
			"type":    pt.Type,
			"subtype": pt.SubType,
		}

		// Add colors
		if pt.DayColor != (model.Color{}) {
			entry["dayColor"] = colorToHex(pt.DayColor)
		}
		if pt.NightColor != (model.Color{}) {
			entry["nightColor"] = colorToHex(pt.NightColor)
		}

		// Add labels
		if len(pt.Labels) > 0 {
			entry["labels"] = pt.Labels
		}

		// Add bitmaps
		if pt.DayIcon != nil {
			entry["dayIcon"] = bitmapToJSON(pt.DayIcon)
		}
		if pt.NightIcon != nil {
			entry["nightIcon"] = bitmapToJSON(pt.NightIcon)
		}

		result[i] = entry
	}
	return result
}

func convertLinesToJSON(lines []model.LineType) []map[string]interface{} {
	result := make([]map[string]interface{}, len(lines))
	for i, lt := range lines {
		entry := map[string]interface{}{
			"type":    lt.Type,
			"subtype": lt.SubType,
		}

		// Add colors
		if lt.DayColor != (model.Color{}) {
			entry["dayColor"] = colorToHex(lt.DayColor)
		}
		if lt.NightColor != (model.Color{}) {
			entry["nightColor"] = colorToHex(lt.NightColor)
		}
		if lt.DayBorderColor != (model.Color{}) {
			entry["dayBorderColor"] = colorToHex(lt.DayBorderColor)
		}
		if lt.NightBorderColor != (model.Color{}) {
			entry["nightBorderColor"] = colorToHex(lt.NightBorderColor)
		}

		// Add width
		if lt.LineWidth > 0 {
			entry["lineWidth"] = lt.LineWidth
		}
		if lt.BorderWidth > 0 {
			entry["borderWidth"] = lt.BorderWidth
		}

		// Add labels
		if len(lt.Labels) > 0 {
			entry["labels"] = lt.Labels
		}

		// Add patterns
		if lt.DayPattern != nil {
			entry["dayPattern"] = bitmapToJSON(lt.DayPattern)
		}
		if lt.NightPattern != nil {
			entry["nightPattern"] = bitmapToJSON(lt.NightPattern)
		}

		result[i] = entry
	}
	return result
}

func convertPolygonsToJSON(polygons []model.PolygonType) []map[string]interface{} {
	result := make([]map[string]interface{}, len(polygons))
	for i, poly := range polygons {
		entry := map[string]interface{}{
			"type":    poly.Type,
			"subtype": poly.SubType,
		}

		// Add colors
		if poly.DayColor != (model.Color{}) {
			entry["dayColor"] = colorToHex(poly.DayColor)
		}
		if poly.NightColor != (model.Color{}) {
			entry["nightColor"] = colorToHex(poly.NightColor)
		}

		// Add labels
		if len(poly.Labels) > 0 {
			entry["labels"] = poly.Labels
		}

		// Add patterns
		if poly.DayPattern != nil {
			entry["dayPattern"] = bitmapToJSON(poly.DayPattern)
		}
		if poly.NightPattern != nil {
			entry["nightPattern"] = bitmapToJSON(poly.NightPattern)
		}

		result[i] = entry
	}
	return result
}

func bitmapToJSON(bm *model.Bitmap) map[string]interface{} {
	result := map[string]interface{}{
		"width":  bm.Width,
		"height": bm.Height,
	}

	// Add palette
	if len(bm.Palette) > 0 {
		palette := make([]string, len(bm.Palette))
		for i, c := range bm.Palette {
			palette[i] = colorToHex(c)
		}
		result["palette"] = palette
		result["colors"] = len(bm.Palette)
	}

	// Add pixel data as array of color indices
	result["pixels"] = bm.Data

	return result
}

func colorToHex(c model.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// txt2bin command
var txt2binCmd = &cobra.Command{
	Use:   "txt2bin <input.txt>",
	Short: "Convert text to binary TYP format",
	Long: `Convert mkgmap text format to binary TYP file.

The binary file can be used with Garmin devices and map software.`,
	Args: cobra.ExactArgs(1),
	RunE: runTxt2Bin,
}

func init() {
	txt2binCmd.Flags().StringP("output", "o", "", "Output file (required)")
	txt2binCmd.MarkFlagRequired("output")
	txt2binCmd.Flags().Int("fid", 0, "Override Family ID")
	txt2binCmd.Flags().Int("pid", 0, "Override Product ID")
	txt2binCmd.Flags().Int("codepage", 1252, "Character encoding")
}

func runTxt2Bin(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	outputPath, _ := cmd.Flags().GetString("output")
	fid, _ := cmd.Flags().GetInt("fid")
	pid, _ := cmd.Flags().GetInt("pid")
	codepage, _ := cmd.Flags().GetInt("codepage")

	// Open input file
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer f.Close()

	// Parse text TYP
	typ, err := typconv.ParseTextTYP(f)
	if err != nil {
		return fmt.Errorf("parse text TYP: %w", err)
	}

	// Override header fields if specified
	if fid != 0 {
		typ.Header.FID = fid
	}
	if pid != 0 {
		typ.Header.PID = pid
	}
	// Only override CodePage if explicitly specified
	// Otherwise, use the CodePage from the text file
	if codepage != 0 && codepage != 1252 {
		// User explicitly specified a non-default codepage
		typ.Header.CodePage = codepage
	} else if typ.Header.CodePage == 0 {
		// No CodePage in file and no explicit override, use default
		typ.Header.CodePage = 1252
	}
	// Otherwise, use the CodePage from the parsed file

	// Create output file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	// Write binary TYP
	if err := typconv.WriteBinaryTYP(out, typ); err != nil {
		return fmt.Errorf("write binary TYP: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully converted %s to %s\n", inputPath, outputPath)
	fmt.Fprintf(os.Stderr, "  CodePage: %d, FID: %d, PID: %d\n", typ.Header.CodePage, typ.Header.FID, typ.Header.PID)
	fmt.Fprintf(os.Stderr, "  Points: %d, Lines: %d, Polygons: %d\n",
		len(typ.Points), len(typ.Lines), len(typ.Polygons))

	return nil
}

// extract command
var extractCmd = &cobra.Command{
	Use:   "extract <input.img>",
	Short: "Extract TYP from .img file",
	Long: `Extract TYP files from Garmin .img container files.

.img files can contain map data and TYP files. This command extracts
the TYP files for separate processing.`,
	Args: cobra.ExactArgs(1),
	RunE: runExtract,
}

func init() {
	extractCmd.Flags().StringP("output", "o", "", "Output directory (required for extraction)")
	extractCmd.Flags().BoolP("list", "l", false, "List TYP files without extracting")
	extractCmd.Flags().Bool("all", false, "Extract all TYP files (default: first only)")
}

func runExtract(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	outputPath, _ := cmd.Flags().GetString("output")
	list, _ := cmd.Flags().GetBool("list")
	all, _ := cmd.Flags().GetBool("all")

	// For listing, we still need to extract to a temp directory
	extractDir := outputPath
	if list || extractDir == "" {
		// Use temp directory for listing or if no output specified
		tempDir, err := os.MkdirTemp("", "typconv-extract-*")
		if err != nil {
			return fmt.Errorf("create temp directory: %w", err)
		}
		if list {
			// Clean up temp directory after listing
			defer os.RemoveAll(tempDir)
		}
		extractDir = tempDir
	}

	// Extract TYP files from .img
	extractedFiles, err := img.ExtractTYP(inputPath, extractDir)
	if err != nil {
		return err
	}

	// If listing, just show the files and return
	if list {
		fmt.Printf("Found %d TYP file(s) in %s:\n", len(extractedFiles), filepath.Base(inputPath))
		for _, file := range extractedFiles {
			// Get file info
			stat, err := os.Stat(file)
			if err != nil {
				fmt.Printf("  - %s (error reading: %v)\n", filepath.Base(file), err)
				continue
			}
			fmt.Printf("  - %s (%d bytes)\n", filepath.Base(file), stat.Size())
		}
		return nil
	}

	// If not extracting all, keep only the first file
	if !all && len(extractedFiles) > 1 {
		// Remove extra files
		for i := 1; i < len(extractedFiles); i++ {
			os.Remove(extractedFiles[i])
		}
		extractedFiles = extractedFiles[:1]
		fmt.Printf("Extracted first TYP file (use --all to extract all files)\n")
	}

	// Show what was extracted
	fmt.Printf("Extracted %d TYP file(s) to %s:\n", len(extractedFiles), extractDir)
	for _, file := range extractedFiles {
		stat, _ := os.Stat(file)
		fmt.Printf("  - %s (%d bytes)\n", filepath.Base(file), stat.Size())
	}

	return nil
}

// info command
var infoCmd = &cobra.Command{
	Use:   "info <input.typ>",
	Short: "Display TYP file information",
	Long: `Display metadata and statistics about a TYP file.

Shows FID, PID, CodePage, and counts of point/line/polygon types.`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func init() {
	infoCmd.Flags().Bool("json", false, "Output as JSON")
	infoCmd.Flags().Bool("brief", false, "Show only summary")
}

func runInfo(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")
	brief, _ := cmd.Flags().GetBool("brief")

	// Open input file
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer f.Close()

	// Get file size
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat input file: %w", err)
	}

	// Parse binary TYP
	typ, err := typconv.ParseBinaryTYP(f, stat.Size())
	if err != nil {
		return fmt.Errorf("parse TYP file: %w", err)
	}

	// Output based on format
	if jsonOutput {
		return outputInfoJSON(inputPath, typ, stat.Size())
	}
	return outputInfoText(inputPath, typ, stat.Size(), brief)
}

func outputInfoText(path string, typ *model.TYPFile, fileSize int64, brief bool) error {
	if brief {
		// Brief mode: just the counts
		fmt.Printf("%s: FID=%d PID=%d CP=%d Points=%d Lines=%d Polygons=%d\n",
			path,
			typ.Header.FID,
			typ.Header.PID,
			typ.Header.CodePage,
			len(typ.Points),
			len(typ.Lines),
			len(typ.Polygons))
		return nil
	}

	// Full human-readable output
	fmt.Printf("TYP File: %s\n", path)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	// Header information
	fmt.Println("Header:")
	fmt.Printf("  Family ID (FID):  %d\n", typ.Header.FID)
	fmt.Printf("  Product ID (PID): %d\n", typ.Header.PID)
	fmt.Printf("  CodePage:         %d (%s)\n", typ.Header.CodePage, getCodePageName(typ.Header.CodePage))
	fmt.Println()

	// Type counts
	fmt.Println("Feature Types:")
	fmt.Printf("  Points:           %d types\n", len(typ.Points))
	fmt.Printf("  Lines:            %d types\n", len(typ.Lines))
	fmt.Printf("  Polygons:         %d types\n", len(typ.Polygons))
	fmt.Printf("  Total:            %d types\n", len(typ.Points)+len(typ.Lines)+len(typ.Polygons))
	fmt.Println()

	// File size
	fmt.Printf("File Size:          %s (%d bytes)\n", formatBytes(fileSize), fileSize)
	fmt.Println()

	// Type details (if not too many)
	if len(typ.Points) > 0 && len(typ.Points) <= 20 {
		fmt.Println("Point Types:")
		for _, pt := range typ.Points {
			fmt.Printf("  0x%04x", pt.Type)
			if pt.SubType > 0 {
				fmt.Printf(" (subtype 0x%x)", pt.SubType)
			}
			if len(pt.Labels) > 0 {
				// Get first label
				for _, label := range pt.Labels {
					fmt.Printf(" - %s", label)
					break
				}
			}
			fmt.Println()
		}
		fmt.Println()
	}

	if len(typ.Lines) > 0 && len(typ.Lines) <= 20 {
		fmt.Println("Line Types:")
		for _, lt := range typ.Lines {
			fmt.Printf("  0x%04x", lt.Type)
			if lt.SubType > 0 {
				fmt.Printf(" (subtype 0x%x)", lt.SubType)
			}
			if len(lt.Labels) > 0 {
				for _, label := range lt.Labels {
					fmt.Printf(" - %s", label)
					break
				}
			}
			fmt.Println()
		}
		fmt.Println()
	}

	if len(typ.Polygons) > 0 && len(typ.Polygons) <= 20 {
		fmt.Println("Polygon Types:")
		for _, poly := range typ.Polygons {
			fmt.Printf("  0x%04x", poly.Type)
			if poly.SubType > 0 {
				fmt.Printf(" (subtype 0x%x)", poly.SubType)
			}
			if len(poly.Labels) > 0 {
				for _, label := range poly.Labels {
					fmt.Printf(" - %s", label)
					break
				}
			}
			fmt.Println()
		}
	}

	return nil
}

func outputInfoJSON(path string, typ *model.TYPFile, fileSize int64) error {
	info := map[string]interface{}{
		"file": path,
		"header": map[string]interface{}{
			"fid":      typ.Header.FID,
			"pid":      typ.Header.PID,
			"codepage": typ.Header.CodePage,
		},
		"counts": map[string]int{
			"points":   len(typ.Points),
			"lines":    len(typ.Lines),
			"polygons": len(typ.Polygons),
			"total":    len(typ.Points) + len(typ.Lines) + len(typ.Polygons),
		},
		"fileSize": fileSize,
	}

	// Add type lists
	points := make([]map[string]interface{}, len(typ.Points))
	for i, pt := range typ.Points {
		ptInfo := map[string]interface{}{
			"type":    pt.Type,
			"subtype": pt.SubType,
		}
		if len(pt.Labels) > 0 {
			labels := make(map[string]string)
			for k, v := range pt.Labels {
				labels[k] = v
			}
			ptInfo["labels"] = labels
		}
		points[i] = ptInfo
	}
	info["points"] = points

	lines := make([]map[string]interface{}, len(typ.Lines))
	for i, lt := range typ.Lines {
		ltInfo := map[string]interface{}{
			"type":    lt.Type,
			"subtype": lt.SubType,
		}
		if len(lt.Labels) > 0 {
			labels := make(map[string]string)
			for k, v := range lt.Labels {
				labels[k] = v
			}
			ltInfo["labels"] = labels
		}
		lines[i] = ltInfo
	}
	info["lines"] = lines

	polygons := make([]map[string]interface{}, len(typ.Polygons))
	for i, poly := range typ.Polygons {
		polyInfo := map[string]interface{}{
			"type":    poly.Type,
			"subtype": poly.SubType,
		}
		if len(poly.Labels) > 0 {
			labels := make(map[string]string)
			for k, v := range poly.Labels {
				labels[k] = v
			}
			polyInfo["labels"] = labels
		}
		polygons[i] = polyInfo
	}
	info["polygons"] = polygons

	// Pretty print JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

func getCodePageName(cp int) string {
	switch cp {
	case 1252:
		return "Windows-1252 (Western European)"
	case 1250:
		return "Windows-1250 (Central European)"
	case 1251:
		return "Windows-1251 (Cyrillic)"
	case 1254:
		return "Windows-1254 (Turkish)"
	case 437:
		return "CP437 (IBM PC)"
	case 65001:
		return "UTF-8"
	default:
		return "Unknown"
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// validate command
var validateCmd = &cobra.Command{
	Use:   "validate <input.typ>",
	Short: "Validate TYP file structure",
	Long: `Validate TYP file structure and contents.

Checks for format errors, invalid type codes, and structural issues.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().Bool("strict", false, "Fail on warnings")
}

func runValidate(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	strict, _ := cmd.Flags().GetBool("strict")

	// Open input file
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer f.Close()

	// Get file size
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat input file: %w", err)
	}

	// Parse binary TYP
	typ, err := typconv.ParseBinaryTYP(f, stat.Size())
	if err != nil {
		return fmt.Errorf("parse TYP file: %w", err)
	}

	// Validate the file
	validator := newValidator(strict)
	validator.validate(typ, inputPath)

	// Print results
	validator.printResults()

	// Return error if validation failed
	if validator.hasErrors() || (strict && validator.hasWarnings()) {
		return fmt.Errorf("validation failed")
	}

	return nil
}

// Validator holds validation state
type validator struct {
	strict   bool
	errors   []string
	warnings []string
	file     string
}

func newValidator(strict bool) *validator {
	return &validator{
		strict:   strict,
		errors:   make([]string, 0),
		warnings: make([]string, 0),
	}
}

func (v *validator) error(msg string, args ...interface{}) {
	v.errors = append(v.errors, fmt.Sprintf(msg, args...))
}

func (v *validator) warning(msg string, args ...interface{}) {
	v.warnings = append(v.warnings, fmt.Sprintf(msg, args...))
}

func (v *validator) hasErrors() bool {
	return len(v.errors) > 0
}

func (v *validator) hasWarnings() bool {
	return len(v.warnings) > 0
}

func (v *validator) validate(typ *model.TYPFile, file string) {
	v.file = file

	// Validate header
	v.validateHeader(&typ.Header)

	// Validate points
	v.validatePoints(typ.Points)

	// Validate lines
	v.validateLines(typ.Lines)

	// Validate polygons
	v.validatePolygons(typ.Polygons)
}

func (v *validator) validateHeader(h *model.Header) {
	// Check CodePage
	validCodePages := map[int]bool{
		437: true, 1250: true, 1251: true, 1252: true, 1254: true, 65001: true,
	}
	if !validCodePages[h.CodePage] {
		v.warning("Unusual CodePage: %d (common values: 1252, 1250, 1251, 437)", h.CodePage)
	}

	// Check FID/PID ranges
	if h.FID < 0 || h.FID > 65535 {
		v.error("Invalid FID: %d (must be 0-65535)", h.FID)
	}
	if h.PID < 0 || h.PID > 65535 {
		v.error("Invalid PID: %d (must be 0-65535)", h.PID)
	}
}

func (v *validator) validatePoints(points []model.PointType) {
	if len(points) == 0 {
		v.warning("No point types defined")
		return
	}

	seenTypes := make(map[int]bool)
	for i, pt := range points {
		// Check for duplicate types
		typeKey := pt.Type<<8 | pt.SubType
		if seenTypes[typeKey] {
			v.warning("Duplicate point type: 0x%04x (subtype 0x%x)", pt.Type, pt.SubType)
		}
		seenTypes[typeKey] = true

		// Validate type code (extended types can go beyond 0xFFFF)
		if pt.Type < 0 || pt.Type > 0x1FFFF {
			v.error("Point %d: invalid type code 0x%x (must be 0x00-0x1FFFF)", i, pt.Type)
		}
		if pt.Type > 0xFFFF {
			v.warning("Point %d: extended type code 0x%x", i, pt.Type)
		}

		// Validate subtype
		if pt.SubType < 0 || pt.SubType > 0x1F {
			v.warning("Point %d: unusual subtype 0x%x (expected 0x00-0x1F)", i, pt.SubType)
		}

		// Validate bitmaps
		if pt.DayIcon != nil {
			v.validateBitmap(pt.DayIcon, fmt.Sprintf("Point %d day icon", i))
		}
		if pt.NightIcon != nil {
			v.validateBitmap(pt.NightIcon, fmt.Sprintf("Point %d night icon", i))
		}

		// Check for labels
		if len(pt.Labels) == 0 {
			v.warning("Point 0x%04x has no labels", pt.Type)
		}
	}
}

func (v *validator) validateLines(lines []model.LineType) {
	if len(lines) == 0 {
		v.warning("No line types defined")
		return
	}

	seenTypes := make(map[int]bool)
	for i, lt := range lines {
		// Check for duplicate types
		typeKey := lt.Type<<8 | lt.SubType
		if seenTypes[typeKey] {
			v.warning("Duplicate line type: 0x%04x (subtype 0x%x)", lt.Type, lt.SubType)
		}
		seenTypes[typeKey] = true

		// Validate type code (extended types can go beyond 0xFFFF)
		if lt.Type < 0 || lt.Type > 0x1FFFF {
			v.error("Line %d: invalid type code 0x%x (must be 0x00-0x1FFFF)", i, lt.Type)
		}
		if lt.Type > 0xFFFF {
			v.warning("Line %d: extended type code 0x%x", i, lt.Type)
		}

		// Validate widths
		if lt.LineWidth < 0 || lt.LineWidth > 255 {
			v.warning("Line %d: unusual line width %d", i, lt.LineWidth)
		}
		if lt.BorderWidth < 0 || lt.BorderWidth > 255 {
			v.warning("Line %d: unusual border width %d", i, lt.BorderWidth)
		}
		if lt.BorderWidth > 0 && lt.LineWidth == 0 {
			v.warning("Line %d: has border but no line width", i)
		}

		// Validate patterns
		if lt.DayPattern != nil {
			v.validateBitmap(lt.DayPattern, fmt.Sprintf("Line %d day pattern", i))
		}
		if lt.NightPattern != nil {
			v.validateBitmap(lt.NightPattern, fmt.Sprintf("Line %d night pattern", i))
		}
	}
}

func (v *validator) validatePolygons(polygons []model.PolygonType) {
	if len(polygons) == 0 {
		v.warning("No polygon types defined")
		return
	}

	seenTypes := make(map[int]bool)
	for i, poly := range polygons {
		// Check for duplicate types
		typeKey := poly.Type<<8 | poly.SubType
		if seenTypes[typeKey] {
			v.warning("Duplicate polygon type: 0x%04x (subtype 0x%x)", poly.Type, poly.SubType)
		}
		seenTypes[typeKey] = true

		// Validate type code (extended types can go beyond 0xFFFF)
		if poly.Type < 0 || poly.Type > 0x1FFFF {
			v.error("Polygon %d: invalid type code 0x%x (must be 0x00-0x1FFFF)", i, poly.Type)
		}
		if poly.Type > 0xFFFF {
			v.warning("Polygon %d: extended type code 0x%x", i, poly.Type)
		}

		// Validate patterns
		if poly.DayPattern != nil {
			v.validateBitmap(poly.DayPattern, fmt.Sprintf("Polygon %d day pattern", i))
		}
		if poly.NightPattern != nil {
			v.validateBitmap(poly.NightPattern, fmt.Sprintf("Polygon %d night pattern", i))
		}
	}
}

func (v *validator) validateBitmap(bm *model.Bitmap, context string) {
	// Check dimensions
	if bm.Width <= 0 || bm.Width > 256 {
		v.error("%s: invalid width %d", context, bm.Width)
	}
	if bm.Height <= 0 || bm.Height > 256 {
		v.error("%s: invalid height %d", context, bm.Height)
	}

	// Warn about unusually large bitmaps
	if bm.Width > 64 || bm.Height > 64 {
		v.warning("%s: unusually large bitmap %dx%d", context, bm.Width, bm.Height)
	}

	// Check palette
	if len(bm.Palette) == 0 {
		v.warning("%s: empty palette", context)
	}
	if len(bm.Palette) > 256 {
		v.error("%s: palette too large (%d colors)", context, len(bm.Palette))
	}

	// Check pixel data
	if len(bm.Data) == 0 {
		v.error("%s: no pixel data", context)
	}
}

func (v *validator) printResults() {
	fmt.Printf("Validating: %s\n", v.file)
	fmt.Println(strings.Repeat("=", 50))

	if len(v.errors) == 0 && len(v.warnings) == 0 {
		fmt.Println("✓ Valid TYP file - no issues found")
		return
	}

	// Print errors
	if len(v.errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(v.errors))
		for _, err := range v.errors {
			fmt.Printf("  ✗ %s\n", err)
		}
	}

	// Print warnings
	if len(v.warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(v.warnings))
		for _, warn := range v.warnings {
			fmt.Printf("  ⚠ %s\n", warn)
		}
	}

	// Summary
	fmt.Println()
	if len(v.errors) > 0 {
		fmt.Printf("Validation failed: %d error(s)", len(v.errors))
		if len(v.warnings) > 0 {
			fmt.Printf(", %d warning(s)", len(v.warnings))
		}
		fmt.Println()
	} else if len(v.warnings) > 0 {
		fmt.Printf("Validation passed with %d warning(s)\n", len(v.warnings))
		if v.strict {
			fmt.Println("(use without --strict to ignore warnings)")
		}
	}
}

// version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("typconv version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
	},
}
