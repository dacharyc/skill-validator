package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/content"
	"github.com/dacharyc/skill-validator/skill"
	"github.com/dacharyc/skill-validator/skillcheck"
	"github.com/dacharyc/skill-validator/types"
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
	case types.SingleSkill:
		r := runContentAnalysis(dirs[0])
		return outputReportWithPerFile(r, perFileContent)
	case types.MultiSkill:
		mr := &types.MultiReport{}
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

func runContentAnalysis(dir string) *types.Report {
	rpt := &types.Report{SkillDir: dir}

	s, err := skill.Load(dir)
	if err != nil {
		rpt.Results = append(rpt.Results,
			types.ResultContext{Category: "Content"}.Error(err.Error()))
		rpt.Errors = 1
		return rpt
	}

	rpt.ContentReport = content.Analyze(s.RawContent)
	rpt.Results = append(rpt.Results,
		types.ResultContext{Category: "Content"}.Pass("content analysis complete"))

	skillcheck.AnalyzeReferences(dir, rpt)

	return rpt
}
