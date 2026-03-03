package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/judge"
)

var (
	reportList    bool
	reportCompare bool
	reportModel   string
)

// Color constants for terminal output (local to score_report).
const (
	reportColorReset  = "\033[0m"
	reportColorBold   = "\033[1m"
	reportColorGreen  = "\033[32m"
	reportColorYellow = "\033[33m"
	reportColorCyan   = "\033[36m"
	reportColorRed    = "\033[31m"
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
		return outputReportList(results, absDir)
	case reportCompare:
		return outputReportCompare(results, absDir)
	default:
		return outputReportDefault(results, absDir)
	}
}

// --- List mode ---

func outputReportList(results []*judge.CachedResult, skillDir string) error {
	if outputFormat == "json" {
		return outputReportListJSON(results)
	}
	if outputFormat == "markdown" {
		outputReportListMarkdown(os.Stdout, results, skillDir)
		return nil
	}

	fmt.Printf("\n%sCached scores for: %s%s\n\n", reportColorBold, skillDir, reportColorReset)
	fmt.Printf("  %-28s %-30s %-20s %s\n", "File", "Model", "Scored At", "Provider")
	fmt.Printf("  %s\n", strings.Repeat("─", 90))

	for _, r := range results {
		scored := r.ScoredAt.Local().Format("2006-01-02 15:04:05")
		fmt.Printf("  %-28s %-30s %-20s %s\n", r.File, r.Model, scored, r.Provider)
	}
	fmt.Println()

	return nil
}

func outputReportListJSON(results []*judge.CachedResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// --- Compare mode ---

func outputReportCompare(results []*judge.CachedResult, skillDir string) error {
	if outputFormat == "json" {
		return outputReportCompareJSON(results)
	}
	if outputFormat == "markdown" {
		outputReportCompareMarkdown(os.Stdout, results, skillDir)
		return nil
	}

	byFile := make(map[string][]*judge.CachedResult)
	for _, r := range results {
		byFile[r.File] = append(byFile[r.File], r)
	}

	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	fmt.Printf("\n%sScore comparison for: %s%s\n", reportColorBold, skillDir, reportColorReset)

	for _, file := range files {
		entries := byFile[file]
		fmt.Printf("\n%s%s%s\n", reportColorBold, file, reportColorReset)

		// Get unique models
		models := make([]string, 0)
		seen := make(map[string]bool)
		for _, e := range entries {
			if !seen[e.Model] {
				models = append(models, e.Model)
				seen[e.Model] = true
			}
		}

		// Determine dimensions based on file type
		isSkill := file == "SKILL.md"

		// Print header
		fmt.Printf("  %-22s", "Dimension")
		for _, m := range models {
			fmt.Printf(" %-15s", truncateModel(m))
		}
		fmt.Println()
		fmt.Printf("  %s\n", strings.Repeat("─", 22+16*len(models)))

		if isSkill {
			printCompareRow("Clarity", entries, models, "clarity")
			printCompareRow("Actionability", entries, models, "actionability")
			printCompareRow("Token Efficiency", entries, models, "token_efficiency")
			printCompareRow("Scope Discipline", entries, models, "scope_discipline")
			printCompareRow("Directive Precision", entries, models, "directive_precision")
			printCompareRow("Novelty", entries, models, "novelty")
			printCompareRow("Overall", entries, models, "overall")
		} else {
			printCompareRow("Clarity", entries, models, "clarity")
			printCompareRow("Instructional Value", entries, models, "instructional_value")
			printCompareRow("Token Efficiency", entries, models, "token_efficiency")
			printCompareRow("Novelty", entries, models, "novelty")
			printCompareRow("Skill Relevance", entries, models, "skill_relevance")
			printCompareRow("Overall", entries, models, "overall")
		}
	}
	fmt.Println()

	return nil
}

func printCompareRow(label string, entries []*judge.CachedResult, models []string, key string) {
	fmt.Printf("  %-22s", label)

	// Build model→scores map using the most recent entry per model
	modelScores := make(map[string]map[string]any)
	for _, e := range entries {
		if _, ok := modelScores[e.Model]; ok {
			continue // already have a newer one (results are sorted newest-first)
		}
		var scores map[string]any
		if err := json.Unmarshal(e.Scores, &scores); err == nil {
			modelScores[e.Model] = scores
		}
	}

	for _, m := range models {
		scores := modelScores[m]
		if scores == nil {
			fmt.Printf(" %-15s", "-")
			continue
		}
		val, ok := scores[key]
		if !ok {
			fmt.Printf(" %-15s", "-")
			continue
		}
		switch v := val.(type) {
		case float64:
			if key == "overall" {
				fmt.Printf(" %-15s", fmt.Sprintf("%.2f/5", v))
			} else {
				fmt.Printf(" %-15s", fmt.Sprintf("%d/5", int(v)))
			}
		default:
			fmt.Printf(" %-15v", v)
		}
	}
	fmt.Println()
}

func truncateModel(model string) string {
	if len(model) > 14 {
		return model[:11] + "..."
	}
	return model
}

func outputReportCompareJSON(results []*judge.CachedResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// --- Default mode (most recent per file) ---

func outputReportDefault(results []*judge.CachedResult, skillDir string) error {
	latest := judge.LatestByFile(results)

	if outputFormat == "json" {
		vals := make([]*judge.CachedResult, 0, len(latest))
		for _, v := range latest {
			vals = append(vals, v)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(vals)
	}

	if outputFormat == "markdown" {
		outputReportDefaultMarkdown(os.Stdout, results, skillDir)
		return nil
	}

	fmt.Printf("\n%sCached scores for: %s%s\n", reportColorBold, skillDir, reportColorReset)

	// Show SKILL.md first, then references sorted alphabetically
	if r, ok := latest["SKILL.md"]; ok {
		printCachedSkillScores(r)
		delete(latest, "SKILL.md")
	}

	refs := make([]string, 0, len(latest))
	for f := range latest {
		refs = append(refs, f)
	}
	sort.Strings(refs)

	for _, f := range refs {
		printCachedRefScores(latest[f])
	}

	fmt.Println()
	return nil
}

func printCachedSkillScores(r *judge.CachedResult) {
	var scores judge.SkillScores
	if err := json.Unmarshal(r.Scores, &scores); err != nil {
		fmt.Printf("\n  Could not parse cached SKILL.md scores\n")
		return
	}

	fmt.Printf("\n%sSKILL.md Scores%s  %s(model: %s, scored: %s)%s\n",
		reportColorBold, reportColorReset,
		reportColorCyan, r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"), reportColorReset)

	reportPrintDimScore("Clarity", scores.Clarity)
	reportPrintDimScore("Actionability", scores.Actionability)
	reportPrintDimScore("Token Efficiency", scores.TokenEfficiency)
	reportPrintDimScore("Scope Discipline", scores.ScopeDiscipline)
	reportPrintDimScore("Directive Precision", scores.DirectivePrecision)
	reportPrintDimScore("Novelty", scores.Novelty)
	fmt.Printf("  %s\n", strings.Repeat("─", 30))
	fmt.Printf("  %sOverall:              %.2f/5%s\n", reportColorBold, scores.Overall, reportColorReset)

	if scores.BriefAssessment != "" {
		fmt.Printf("\n  %s\"%s\"%s\n", reportColorCyan, scores.BriefAssessment, reportColorReset)
	}

	if scores.NovelInfo != "" {
		fmt.Printf("  %sNovel details: %s%s\n", reportColorCyan, scores.NovelInfo, reportColorReset)
	}
}

func reportPrintDimScore(name string, score int) {
	color := reportColorGreen
	if score <= 2 {
		color = reportColorRed
	} else if score <= 3 {
		color = reportColorYellow
	}
	padding := max(22-len(name), 1)
	fmt.Printf("  %s:%s%s%d/5%s\n", name, strings.Repeat(" ", padding), color, score, reportColorReset)
}

func printCachedRefScores(r *judge.CachedResult) {
	var scores judge.RefScores
	if err := json.Unmarshal(r.Scores, &scores); err != nil {
		fmt.Printf("\n  Could not parse cached scores for %s\n", r.File)
		return
	}

	fmt.Printf("\n%sReference: %s%s  %s(model: %s, scored: %s)%s\n",
		reportColorBold, r.File, reportColorReset,
		reportColorCyan, r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"), reportColorReset)

	reportPrintDimScore("Clarity", scores.Clarity)
	reportPrintDimScore("Instructional Value", scores.InstructionalValue)
	reportPrintDimScore("Token Efficiency", scores.TokenEfficiency)
	reportPrintDimScore("Novelty", scores.Novelty)
	reportPrintDimScore("Skill Relevance", scores.SkillRelevance)
	fmt.Printf("  %s\n", strings.Repeat("─", 30))
	fmt.Printf("  %sOverall:              %.2f/5%s\n", reportColorBold, scores.Overall, reportColorReset)

	if scores.BriefAssessment != "" {
		fmt.Printf("\n  %s\"%s\"%s\n", reportColorCyan, scores.BriefAssessment, reportColorReset)
	}

	if scores.NovelInfo != "" {
		fmt.Printf("  %sNovel details: %s%s\n", reportColorCyan, scores.NovelInfo, reportColorReset)
	}
}
