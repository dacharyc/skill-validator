package cmd

import (
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze skill content or contamination",
	Long:  "Parent command for content and contamination analysis subcommands.",
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
