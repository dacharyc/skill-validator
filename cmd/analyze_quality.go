package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/quality"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var analyzeQualityCmd = &cobra.Command{
	Use:   "quality <path>",
	Short: "Check link validity and code fence structure",
	Long:  "Validates links (relative and HTTP) and checks for unclosed code fences in SKILL.md and references.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyzeQuality,
}

func init() {
	analyzeCmd.AddCommand(analyzeQualityCmd)
}

func runAnalyzeQuality(cmd *cobra.Command, args []string) error {
	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	switch mode {
	case validator.SingleSkill:
		r := runQualityChecks(dirs[0])
		return outputReport(r)
	case validator.MultiSkill:
		mr := &validator.MultiReport{}
		for _, dir := range dirs {
			r := runQualityChecks(dir)
			mr.Skills = append(mr.Skills, r)
			mr.Errors += r.Errors
			mr.Warnings += r.Warnings
		}
		return outputMultiReport(mr)
	}
	return nil
}

func runQualityChecks(dir string) *validator.Report {
	rpt := &validator.Report{SkillDir: dir}

	s, err := validator.LoadSkill(dir)
	if err != nil {
		rpt.Results = append(rpt.Results, validator.Result{
			Level: validator.Error, Category: "Quality", Message: err.Error(),
		})
		rpt.Errors = 1
		return rpt
	}

	rpt.Results = append(rpt.Results, quality.CheckLinks(dir, s.Body)...)
	rpt.Results = append(rpt.Results, quality.CheckMarkdown(dir, s.Body)...)

	// Tally
	for _, r := range rpt.Results {
		switch r.Level {
		case validator.Error:
			rpt.Errors++
		case validator.Warning:
			rpt.Warnings++
		}
	}

	// If no results at all, add a pass result
	if len(rpt.Results) == 0 {
		rpt.Results = append(rpt.Results, validator.Result{
			Level: validator.Pass, Category: "Quality", Message: "all quality checks passed",
		})
	}

	return rpt
}

// AppendQualityResults runs quality checks and appends results to an existing report.
func AppendQualityResults(rpt *validator.Report, dir string, body string) {
	rpt.Results = append(rpt.Results, quality.CheckLinks(dir, body)...)
	rpt.Results = append(rpt.Results, quality.CheckMarkdown(dir, body)...)
}
