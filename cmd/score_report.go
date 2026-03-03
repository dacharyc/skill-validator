package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/judge"
	"github.com/dacharyc/skill-validator/report"
)

var (
	reportList    bool
	reportCompare bool
	reportModel   string
)

var scoreReportCmd = &cobra.Command{
	Use:   "report <path>",
	Short: "View cached LLM scores",
	Long: `View and compare cached LLM quality scores without making API calls.

By default, shows the most recent scores for each file. Use flags to
list all cached entries, compare across models, or filter by model.`,
	Args: cobra.ExactArgs(1),
	RunE: runScoreReport,
}

func init() {
	scoreReportCmd.Flags().BoolVar(&reportList, "list", false, "list all cached score entries with metadata")
	scoreReportCmd.Flags().BoolVar(&reportCompare, "compare", false, "compare scores across models side-by-side")
	scoreReportCmd.Flags().StringVar(&reportModel, "model", "", "filter to scores from a specific model")
	scoreCmd.AddCommand(scoreReportCmd)
}

func runScoreReport(cmd *cobra.Command, args []string) error {
	absDir, err := resolvePath(args)
	if err != nil {
		return err
	}

	cacheDir := judge.CacheDir(absDir)
	results, err := judge.ListCached(cacheDir)
	if err != nil {
		return fmt.Errorf("reading cache: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No cached scores found. Run 'score evaluate' first.")
		return nil
	}

	if reportModel != "" {
		results = judge.FilterByModel(results, reportModel)
		if len(results) == 0 {
			fmt.Printf("No cached scores found for model %q.\n", reportModel)
			return nil
		}
	}

	switch {
	case reportList:
		return report.List(os.Stdout, results, absDir, outputFormat)
	case reportCompare:
		return report.Compare(os.Stdout, results, absDir, outputFormat)
	default:
		return report.Default(os.Stdout, results, absDir, outputFormat)
	}
}
