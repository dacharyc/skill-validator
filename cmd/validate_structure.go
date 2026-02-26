package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/structure"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var (
	skipOrphans     bool
	strictStructure bool
)

var validateStructureCmd = &cobra.Command{
	Use:   "structure <path>",
	Short: "Validate skill structure (spec compliance, tokens, code fences, internal links)",
	Long:  "Checks that a skill directory conforms to the spec: structure, frontmatter fields, token limits, skill ratio, code fence integrity, and internal link validity.",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidateStructure,
}

func init() {
	validateStructureCmd.Flags().BoolVar(&skipOrphans, "skip-orphans", false,
		"skip orphan file detection (unreferenced files in scripts/, references/, assets/)")
	validateStructureCmd.Flags().BoolVar(&strictStructure, "strict", false, "treat warnings as errors (exit 1 instead of 2)")
	validateCmd.AddCommand(validateStructureCmd)
}

func runValidateStructure(cmd *cobra.Command, args []string) error {
	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	opts := structure.Options{SkipOrphans: skipOrphans}
	eopts := exitOpts{strict: strictStructure}

	switch mode {
	case validator.SingleSkill:
		r := structure.Validate(dirs[0], opts)
		return outputReportWithExitOpts(r, false, eopts)
	case validator.MultiSkill:
		mr := structure.ValidateMulti(dirs, opts)
		return outputMultiReportWithExitOpts(mr, false, eopts)
	}
	return nil
}
