package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dacharyc/skill-validator/internal/report"
	"github.com/dacharyc/skill-validator/internal/validator"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: skill-validator <path-to-skill-directory>\n")
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
	report.Print(os.Stdout, r)

	if r.Errors > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}
