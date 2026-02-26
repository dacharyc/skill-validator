package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/validator"
)

const version = "v0.6.0"

var outputFormat string

var rootCmd = &cobra.Command{
	Use:   "skill-validator",
	Short: "Validate and analyze agent skills",
	Long:  "A CLI for validating skill directory structure, analyzing content quality, and detecting cross-language contamination.",
}

func init() {
	rootCmd.Version = version
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format: text or json")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(ExitCobra)
	}
}

// resolvePath resolves a path argument to an absolute directory path.
func resolvePath(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("path argument required")
	}

	dir := args[0]
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s is not a valid directory", dir)
	}

	return absDir, nil
}

// detectAndResolve resolves the path and detects skills.
func detectAndResolve(args []string) (string, validator.SkillMode, []string, error) {
	absDir, err := resolvePath(args)
	if err != nil {
		return "", 0, nil, err
	}

	mode, dirs := validator.DetectSkills(absDir)
	if mode == validator.NoSkill {
		return "", 0, nil, fmt.Errorf("no skills found in %s (expected SKILL.md or subdirectories containing SKILL.md)", args[0])
	}

	return absDir, mode, dirs, nil
}
