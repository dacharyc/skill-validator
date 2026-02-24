package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var perFileContamination bool

var analyzeContaminationCmd = &cobra.Command{
	Use:   "contamination <path>",
	Short: "Assess cross-language contamination",
	Long:  "Detects cross-language contamination: multi-interface tools, language mismatches, scope breadth.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyzeContamination,
}

func init() {
	analyzeContaminationCmd.Flags().BoolVar(&perFileContamination, "per-file", false, "show per-file reference analysis")
	analyzeCmd.AddCommand(analyzeContaminationCmd)
}

func runAnalyzeContamination(cmd *cobra.Command, args []string) error {
	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	switch mode {
	case validator.SingleSkill:
		r := runContaminationAnalysis(dirs[0])
		return outputReportWithPerFile(r, perFileContamination)
	case validator.MultiSkill:
		mr := &validator.MultiReport{}
		for _, dir := range dirs {
			r := runContaminationAnalysis(dir)
			mr.Skills = append(mr.Skills, r)
			mr.Errors += r.Errors
			mr.Warnings += r.Warnings
		}
		return outputMultiReportWithPerFile(mr, perFileContamination)
	}
	return nil
}

func runContaminationAnalysis(dir string) *validator.Report {
	rpt := &validator.Report{SkillDir: dir}

	s, err := validator.LoadSkill(dir)
	if err != nil {
		rpt.Results = append(rpt.Results, validator.Result{
			Level: validator.Error, Category: "Contamination", Message: err.Error(),
		})
		rpt.Errors = 1
		return rpt
	}

	// Get code languages from content analysis
	cr := content.Analyze(s.RawContent)
	skillName := filepath.Base(dir)
	rpt.ContaminationReport = contamination.Analyze(skillName, s.RawContent, cr.CodeLanguages)

	rpt.Results = append(rpt.Results, validator.Result{
		Level: validator.Pass, Category: "Contamination", Message: "contamination analysis complete",
	})

	validator.AnalyzeReferences(dir, rpt)

	return rpt
}
