package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/structure"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var validateStructureCmd = &cobra.Command{
	Use:   "structure <path>",
	Short: "Validate skill structure (spec compliance, tokens, code fences)",
	Long:  "Checks that a skill directory conforms to the spec: structure, frontmatter fields, token limits, skill ratio, and code fence integrity.",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidateStructure,
}

func init() {
	validateCmd.AddCommand(validateStructureCmd)
}

func runValidateStructure(cmd *cobra.Command, args []string) error {
	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	switch mode {
	case validator.SingleSkill:
		r := structure.Validate(dirs[0])
		return outputReport(r)
	case validator.MultiSkill:
		mr := structure.ValidateMulti(dirs)
		return outputMultiReport(mr)
	}
	return nil
}
