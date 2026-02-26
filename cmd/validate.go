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
	return outputReportWithExitOpts(r, false, exitOpts{})
}

func outputReportWithPerFile(r *validator.Report, perFile bool) error {
	return outputReportWithExitOpts(r, perFile, exitOpts{})
}

func outputReportWithExitOpts(r *validator.Report, perFile bool, opts exitOpts) error {
	switch outputFormat {
	case "json":
		if err := report.PrintJSON(os.Stdout, r, perFile); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	case "markdown":
		if err := report.PrintMarkdown(os.Stdout, r, perFile); err != nil {
			return fmt.Errorf("writing markdown: %w", err)
		}
	default:
		report.Print(os.Stdout, r, perFile)
	}
	if emitAnnotations {
		wd, _ := os.Getwd()
		report.PrintAnnotations(os.Stdout, r, wd)
	}
	if code := opts.resolve(r.Errors, r.Warnings); code != 0 {
		os.Exit(code)
	}
	return nil
}

func outputMultiReport(mr *validator.MultiReport) error {
	return outputMultiReportWithExitOpts(mr, false, exitOpts{})
}

func outputMultiReportWithPerFile(mr *validator.MultiReport, perFile bool) error {
	return outputMultiReportWithExitOpts(mr, perFile, exitOpts{})
}

func outputMultiReportWithExitOpts(mr *validator.MultiReport, perFile bool, opts exitOpts) error {
	switch outputFormat {
	case "json":
		if err := report.PrintMultiJSON(os.Stdout, mr, perFile); err != nil {
			return fmt.Errorf("writing JSON: %w", err)
		}
	case "markdown":
		if err := report.PrintMultiMarkdown(os.Stdout, mr, perFile); err != nil {
			return fmt.Errorf("writing markdown: %w", err)
		}
	default:
		report.PrintMulti(os.Stdout, mr, perFile)
	}
	if emitAnnotations {
		wd, _ := os.Getwd()
		report.PrintMultiAnnotations(os.Stdout, mr, wd)
	}
	if code := opts.resolve(mr.Errors, mr.Warnings); code != 0 {
		os.Exit(code)
	}
	return nil
}
