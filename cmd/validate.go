package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/report"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var validateCmd = &cobra.Command{
	Use:   "validate <path>",
	Short: "Validate skill spec compliance (structure, frontmatter, tokens)",
	Long:  "Checks that a skill directory conforms to the spec: structure, frontmatter fields, token limits, and skill ratio.",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	switch mode {
	case validator.SingleSkill:
		r := validator.Validate(dirs[0])
		return outputReport(r)
	case validator.MultiSkill:
		mr := validator.ValidateMulti(dirs)
		return outputMultiReport(mr)
	}
	return nil
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
