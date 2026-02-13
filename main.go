package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dacharyc/skill-validator/internal/report"
	"github.com/dacharyc/skill-validator/internal/validator"
)

func main() {
	var outputFormat string
	flag.StringVar(&outputFormat, "output", "text", "output format: text or json")
	flag.StringVar(&outputFormat, "o", "text", "output format: text or json (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: skill-validator [-o format] <path-to-skill-directory>\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if outputFormat != "text" && outputFormat != "json" {
		fmt.Fprintf(os.Stderr, "Error: unknown output format %q (expected text or json)\n", outputFormat)
		os.Exit(2)
	}

	args := flag.Args()

	if len(args) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	dir := args[0]

	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(2)
	}

	// Verify directory exists
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a valid directory\n", dir)
		os.Exit(2)
	}

	r := validator.Validate(absDir)

	switch outputFormat {
	case "json":
		if err := report.PrintJSON(os.Stdout, r); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
			os.Exit(2)
		}
	default:
		report.Print(os.Stdout, r)
	}

	if r.Errors > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}
