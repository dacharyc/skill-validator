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
	return outputReportWithPerFile(r, false)
}

func outputReportWithPerFile(r *validator.Report, perFile bool) error {
	switch outputFormat {
	case "json":
		if err := report.PrintJSON(os.Stdout, r, perFile); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	default:
		report.Print(os.Stdout, r, perFile)
	}
	if r.Errors > 0 {
		os.Exit(1)
	}
	return nil
}

func outputMultiReport(mr *validator.MultiReport) error {
	return outputMultiReportWithPerFile(mr, false)
}

func outputMultiReportWithPerFile(mr *validator.MultiReport, perFile bool) error {
	switch outputFormat {
	case "json":
		if err := report.PrintMultiJSON(os.Stdout, mr, perFile); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	default:
		report.PrintMulti(os.Stdout, mr, perFile)
	}
	if mr.Errors > 0 {
		os.Exit(1)
	}
	return nil
}
