package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/agent-ecosystem/skill-validator/orchestrate"
	"github.com/agent-ecosystem/skill-validator/types"
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

	ctx := context.Background()

	switch mode {
	case types.SingleSkill:
		r := orchestrate.RunLinkChecks(ctx, dirs[0])
		return outputReport(r)
	case types.MultiSkill:
		mr := &types.MultiReport{}
		for _, dir := range dirs {
			r := orchestrate.RunLinkChecks(ctx, dir)
			mr.Skills = append(mr.Skills, r)
			mr.Errors += r.Errors
			mr.Warnings += r.Warnings
		}
		return outputMultiReport(mr)
	}
	return nil
}
