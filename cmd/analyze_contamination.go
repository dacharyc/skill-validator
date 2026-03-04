package cmd

import (
	"github.com/spf13/cobra"

	"github.com/agent-ecosystem/skill-validator/orchestrate"
	"github.com/agent-ecosystem/skill-validator/types"
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
	case types.SingleSkill:
		r := orchestrate.RunContaminationAnalysis(dirs[0])
		return outputReportWithPerFile(r, perFileContamination)
	case types.MultiSkill:
		mr := &types.MultiReport{}
		for _, dir := range dirs {
			r := orchestrate.RunContaminationAnalysis(dir)
			mr.Skills = append(mr.Skills, r)
			mr.Errors += r.Errors
			mr.Warnings += r.Warnings
		}
		return outputMultiReportWithPerFile(mr, perFileContamination)
	}
	return nil
}
