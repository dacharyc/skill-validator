package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/report"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate skill structure or links",
	Long:  "Parent command for structure and link validation subcommands.",
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func outputReport(r *validator.Report) error {
	switch outputFormat {
	case "json":
		if err := report.PrintJSON(os.Stdout, r); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	default:
		report.Print(os.Stdout, r)
	}
	if r.Errors > 0 {
		os.Exit(1)
	}
	return nil
}

func outputMultiReport(mr *validator.MultiReport) error {
	switch outputFormat {
	case "json":
		if err := report.PrintMultiJSON(os.Stdout, mr); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	default:
		report.PrintMulti(os.Stdout, mr)
	}
	if mr.Errors > 0 {
		os.Exit(1)
	}
	return nil
}
