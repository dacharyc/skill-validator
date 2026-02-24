package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var perFileContent bool

var analyzeContentCmd = &cobra.Command{
	Use:   "content <path>",
	Short: "Analyze content quality metrics",
	Long:  "Computes content metrics: word count, code block ratio, imperative ratio, information density, instruction specificity, and more.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyzeContent,
}

func init() {
	analyzeContentCmd.Flags().BoolVar(&perFileContent, "per-file", false, "show per-file reference analysis")
	analyzeCmd.AddCommand(analyzeContentCmd)
}

func runAnalyzeContent(cmd *cobra.Command, args []string) error {
	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	switch mode {
	case validator.SingleSkill:
		r := runContentAnalysis(dirs[0])
		return outputReportWithPerFile(r, perFileContent)
	case validator.MultiSkill:
		mr := &validator.MultiReport{}
		for _, dir := range dirs {
			r := runContentAnalysis(dir)
			mr.Skills = append(mr.Skills, r)
			mr.Errors += r.Errors
			mr.Warnings += r.Warnings
		}
		return outputMultiReportWithPerFile(mr, perFileContent)
	}
	return nil
}

func runContentAnalysis(dir string) *validator.Report {
	rpt := &validator.Report{SkillDir: dir}

	s, err := validator.LoadSkill(dir)
	if err != nil {
		rpt.Results = append(rpt.Results, validator.Result{
			Level: validator.Error, Category: "Content", Message: err.Error(),
		})
		rpt.Errors = 1
		return rpt
	}

	rpt.ContentReport = content.Analyze(s.RawContent)
	rpt.Results = append(rpt.Results, validator.Result{
		Level: validator.Pass, Category: "Content", Message: "content analysis complete",
	})

	validator.AnalyzeReferences(dir, rpt)

	return rpt
}
