package main

import (
	"fmt"
	"os"

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

	// TODO: Implement info
	_ = inputPath
	_ = jsonOutput
	_ = brief

	return fmt.Errorf("info not yet implemented")
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
