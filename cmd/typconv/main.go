package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

It can convert between binary and text formats, extract TYP files
from .img containers, and validate TYP file structure.

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
		// TODO: Implement JSON output
		return fmt.Errorf("JSON format not yet implemented")
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

	// TODO: Implement extract
	_ = inputPath
	_ = outputPath
	_ = list
	_ = all

	return fmt.Errorf("extract not yet implemented")
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

	// TODO: Implement validate
	_ = inputPath
	_ = strict

	return fmt.Errorf("validate not yet implemented")
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
