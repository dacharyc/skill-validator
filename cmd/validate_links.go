package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/links"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var validateLinksCmd = &cobra.Command{
	Use:   "links <path>",
	Short: "Check external link validity (HTTP/HTTPS)",
	Long:  "Validates external (HTTP/HTTPS) links in SKILL.md. Internal (relative) links are checked by validate structure.",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidateLinks,
}

func init() {
	validateCmd.AddCommand(validateLinksCmd)
}

func runValidateLinks(cmd *cobra.Command, args []string) error {
	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	switch mode {
	case validator.SingleSkill:
		r := runLinkChecks(dirs[0])
		return outputReport(r)
	case validator.MultiSkill:
		mr := &validator.MultiReport{}
		for _, dir := range dirs {
			r := runLinkChecks(dir)
			mr.Skills = append(mr.Skills, r)
			mr.Errors += r.Errors
			mr.Warnings += r.Warnings
		}
		return outputMultiReport(mr)
	}
	return nil
}

func runLinkChecks(dir string) *validator.Report {
	rpt := &validator.Report{SkillDir: dir}

	s, err := validator.LoadSkill(dir)
	if err != nil {
		rpt.Results = append(rpt.Results, validator.Result{
			Level: validator.Error, Category: "Links", Message: err.Error(),
		})
		rpt.Errors = 1
		return rpt
	}

	rpt.Results = append(rpt.Results, links.CheckLinks(dir, s.Body)...)

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
			Level: validator.Pass, Category: "Links", Message: "all link checks passed",
		})
	}

	return rpt
}
