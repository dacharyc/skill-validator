package cmd

import (
	"github.com/spf13/cobra"
)

var scoreCmd = &cobra.Command{
	Use:   "score",
	Short: "LLM-as-judge quality scoring for skills",
	Long:  "Parent command for LLM-based quality scoring. Use 'score evaluate' to score skills and 'score report' to view cached results.",
}

func init() {
	rootCmd.AddCommand(scoreCmd)
}
