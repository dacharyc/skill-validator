package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/judge"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var (
	evalProvider       string
	evalModel          string
	evalBaseURL        string
	evalRescore        bool
	evalSkillOnly      bool
	evalRefsOnly       bool
	evalDisplay        string
	evalFullContent    bool
	evalMaxTokensStyle string
)

var scoreEvaluateCmd = &cobra.Command{
	Use:   "evaluate <path>",
	Short: "Score a skill using an LLM judge",
	Long: `Score a skill's quality using an LLM-as-judge approach.

The path can be:
  - A skill directory (containing SKILL.md) — scores SKILL.md and references
  - A multi-skill parent directory — scores each skill
  - A specific .md file — scores just that reference file

Requires an API key via environment variable:
  ANTHROPIC_API_KEY (for --provider anthropic, the default)
  OPENAI_API_KEY    (for --provider openai)`,
	Args: cobra.ExactArgs(1),
	RunE: runScoreEvaluate,
}

func init() {
	scoreEvaluateCmd.Flags().StringVar(&evalProvider, "provider", "anthropic", "LLM provider: anthropic or openai")
	scoreEvaluateCmd.Flags().StringVar(&evalModel, "model", "", "model name (default: claude-sonnet-4-5-20250929 for anthropic, gpt-4o for openai)")
	scoreEvaluateCmd.Flags().StringVar(&evalBaseURL, "base-url", "", "API base URL (for openai-compatible endpoints)")
	scoreEvaluateCmd.Flags().BoolVar(&evalRescore, "rescore", false, "re-score and overwrite cached results")
	scoreEvaluateCmd.Flags().BoolVar(&evalSkillOnly, "skill-only", false, "score only SKILL.md, skip reference files")
	scoreEvaluateCmd.Flags().BoolVar(&evalRefsOnly, "refs-only", false, "score only reference files, skip SKILL.md")
	scoreEvaluateCmd.Flags().StringVar(&evalDisplay, "display", "aggregate", "reference score display: aggregate or files")
	scoreEvaluateCmd.Flags().BoolVar(&evalFullContent, "full-content", false, "send full file content to LLM (default: truncate to 8,000 chars)")
	scoreEvaluateCmd.Flags().StringVar(&evalMaxTokensStyle, "max-tokens-style", "auto", "token parameter style: auto, max_tokens, or max_completion_tokens")
	scoreCmd.AddCommand(scoreEvaluateCmd)
}

// skillEvalResult holds the complete scoring output for one skill.
type skillEvalResult struct {
	SkillDir     string
	SkillScores  *judge.SkillScores
	RefResults   []refEvalResult
	RefAggregate *judge.RefScores
}

type refEvalResult struct {
	File   string
	Scores *judge.RefScores
}

func runScoreEvaluate(cmd *cobra.Command, args []string) error {
	if evalSkillOnly && evalRefsOnly {
		return fmt.Errorf("--skill-only and --refs-only are mutually exclusive")
	}

	if evalDisplay != "aggregate" && evalDisplay != "files" {
		return fmt.Errorf("--display must be \"aggregate\" or \"files\"")
	}

	// Validate --max-tokens-style
	switch evalMaxTokensStyle {
	case "auto", "max_tokens", "max_completion_tokens":
		// valid
	default:
		return fmt.Errorf("--max-tokens-style must be \"auto\", \"max_tokens\", or \"max_completion_tokens\"")
	}

	// Resolve API key
	apiKey, err := resolveAPIKey(evalProvider)
	if err != nil {
		return err
	}

	client, err := judge.NewClient(evalProvider, apiKey, evalBaseURL, evalModel, evalMaxTokensStyle)
	if err != nil {
		return err
	}

	ctx := context.Background()
	path := args[0]

	// Check if path is a file (single reference scoring)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path not found: %s", path)
	}

	if !info.IsDir() {
		return runScoreSingleFile(ctx, absPath, client, evalMaxLen())
	}

	// Directory mode — detect skills
	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	switch mode {
	case validator.SingleSkill:
		result, err := evaluateSkill(ctx, dirs[0], client, evalMaxLen())
		if err != nil {
			return err
		}
		return outputEvalResult(result)

	case validator.MultiSkill:
		var results []skillEvalResult
		for _, dir := range dirs {
			result, err := evaluateSkill(ctx, dir, client, evalMaxLen())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error scoring %s: %v\n", filepath.Base(dir), err)
				continue
			}
			results = append(results, *result)
		}
		return outputMultiEvalResults(results)
	}

	return nil
}

func evalMaxLen() int {
	if evalFullContent {
		return 0
	}
	return judge.DefaultMaxContentLen
}

func evaluateSkill(ctx context.Context, dir string, client judge.LLMClient, maxLen int) (*skillEvalResult, error) {
	result := &skillEvalResult{SkillDir: dir}
	cacheDir := judge.CacheDir(dir)
	skillName := filepath.Base(dir)

	// Load skill
	s, err := validator.LoadSkill(dir)
	if err != nil {
		return nil, fmt.Errorf("loading skill: %w", err)
	}

	// Score SKILL.md
	if !evalRefsOnly {
		fmt.Fprintf(os.Stderr, "  Scoring %s/SKILL.md...\n", skillName)

		cacheKey := judge.CacheKey(client.Provider(), client.ModelName(), "skill", skillName, "SKILL.md")

		if !evalRescore {
			if cached, ok := judge.GetCached(cacheDir, cacheKey); ok {
				var scores judge.SkillScores
				if err := json.Unmarshal(cached.Scores, &scores); err == nil {
					result.SkillScores = &scores
					fmt.Fprintf(os.Stderr, "  Scoring %s/SKILL.md... (cached)\n", skillName)
				}
			}
		}

		if result.SkillScores == nil {
			scores, err := judge.ScoreSkill(ctx, s.RawContent, client, maxLen)
			if err != nil {
				return nil, fmt.Errorf("scoring SKILL.md: %w", err)
			}
			result.SkillScores = scores

			// Save to cache
			scoresJSON, _ := json.Marshal(scores)
			cacheResult := &judge.CachedResult{
				Provider:    client.Provider(),
				Model:       client.ModelName(),
				File:        "SKILL.md",
				Type:        "skill",
				ContentHash: judge.ContentHash(s.RawContent),
				ScoredAt:    time.Now().UTC(),
				Scores:      scoresJSON,
			}
			if err := judge.SaveCache(cacheDir, cacheKey, cacheResult); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: could not save cache: %v\n", err)
			}
		}
	}

	// Score reference files
	if !evalSkillOnly {
		refFiles := validator.ReadReferencesMarkdownFiles(dir)
		if refFiles != nil {
			skillDesc := s.Frontmatter.Description

			// Sort for deterministic ordering
			names := make([]string, 0, len(refFiles))
			for name := range refFiles {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				content := refFiles[name]
				fmt.Fprintf(os.Stderr, "  Scoring %s/references/%s...\n", skillName, name)

				cacheKey := judge.CacheKey(client.Provider(), client.ModelName(), "ref:"+name, skillName, name)
				var refScores *judge.RefScores

				if !evalRescore {
					if cached, ok := judge.GetCached(cacheDir, cacheKey); ok {
						var scores judge.RefScores
						if err := json.Unmarshal(cached.Scores, &scores); err == nil {
							refScores = &scores
							fmt.Fprintf(os.Stderr, "  Scoring %s/references/%s... (cached)\n", skillName, name)
						}
					}
				}

				if refScores == nil {
					scores, err := judge.ScoreReference(ctx, content, s.Frontmatter.Name, skillDesc, client, maxLen)
					if err != nil {
						fmt.Fprintf(os.Stderr, "  Error scoring %s: %v\n", name, err)
						continue
					}
					refScores = scores

					scoresJSON, _ := json.Marshal(scores)
					cacheResult := &judge.CachedResult{
						Provider:    client.Provider(),
						Model:       client.ModelName(),
						File:        name,
						Type:        "ref:" + name,
						ContentHash: judge.ContentHash(content),
						ScoredAt:    time.Now().UTC(),
						Scores:      scoresJSON,
					}
					if err := judge.SaveCache(cacheDir, cacheKey, cacheResult); err != nil {
						fmt.Fprintf(os.Stderr, "  Warning: could not save cache: %v\n", err)
					}
				}

				result.RefResults = append(result.RefResults, refEvalResult{File: name, Scores: refScores})
			}

			// Aggregate
			if len(result.RefResults) > 0 {
				var allScores []*judge.RefScores
				for _, r := range result.RefResults {
					allScores = append(allScores, r.Scores)
				}
				result.RefAggregate = judge.AggregateRefScores(allScores)
			}
		}
	}

	return result, nil
}

func runScoreSingleFile(ctx context.Context, absPath string, client judge.LLMClient, maxLen int) error {
	if !strings.HasSuffix(strings.ToLower(absPath), ".md") {
		return fmt.Errorf("single-file scoring only supports .md files: %s", absPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Walk up to find parent skill directory
	skillDir, err := findParentSkillDir(absPath)
	if err != nil {
		return err
	}

	// Load parent skill for context
	s, err := validator.LoadSkill(skillDir)
	if err != nil {
		return fmt.Errorf("loading parent skill: %w", err)
	}

	fileName := filepath.Base(absPath)
	skillName := s.Frontmatter.Name
	if skillName == "" {
		skillName = filepath.Base(skillDir)
	}

	fmt.Fprintf(os.Stderr, "  Scoring %s (parent: %s)...\n", fileName, skillName)

	cacheDir := judge.CacheDir(skillDir)
	cacheKey := judge.CacheKey(client.Provider(), client.ModelName(), "ref:"+fileName, skillName, fileName)

	if !evalRescore {
		if cached, ok := judge.GetCached(cacheDir, cacheKey); ok {
			var scores judge.RefScores
			if err := json.Unmarshal(cached.Scores, &scores); err == nil {
				fmt.Fprintf(os.Stderr, "  Scoring %s... (cached)\n", fileName)
				result := &skillEvalResult{
					SkillDir:   skillDir,
					RefResults: []refEvalResult{{File: fileName, Scores: &scores}},
				}
				return outputEvalResult(result)
			}
		}
	}

	scores, err := judge.ScoreReference(ctx, string(content), skillName, s.Frontmatter.Description, client, maxLen)
	if err != nil {
		return fmt.Errorf("scoring %s: %w", fileName, err)
	}

	// Save to cache
	scoresJSON, _ := json.Marshal(scores)
	cacheResult := &judge.CachedResult{
		Provider:    client.Provider(),
		Model:       client.ModelName(),
		File:        fileName,
		Type:        "ref:" + fileName,
		ContentHash: judge.ContentHash(string(content)),
		ScoredAt:    time.Now().UTC(),
		Scores:      scoresJSON,
	}
	if err := judge.SaveCache(cacheDir, cacheKey, cacheResult); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: could not save cache: %v\n", err)
	}

	result := &skillEvalResult{
		SkillDir:   skillDir,
		RefResults: []refEvalResult{{File: fileName, Scores: scores}},
	}
	return outputEvalResult(result)
}

func findParentSkillDir(filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	// Check up to 3 levels
	for range 3 {
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("could not find parent SKILL.md for %s (checked up to 3 directories)", filePath)
}

func resolveAPIKey(provider string) (string, error) {
	switch strings.ToLower(provider) {
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return "", fmt.Errorf("ANTHROPIC_API_KEY environment variable not set\n  Set it with: export ANTHROPIC_API_KEY=your-key-here")
		}
		return key, nil
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return "", fmt.Errorf("OPENAI_API_KEY environment variable not set\n  Set it with: export OPENAI_API_KEY=your-key-here")
		}
		return key, nil
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// --- Output formatting ---

const (
	evalColorReset  = "\033[0m"
	evalColorBold   = "\033[1m"
	evalColorGreen  = "\033[32m"
	evalColorYellow = "\033[33m"
	evalColorCyan   = "\033[36m"
	evalColorRed    = "\033[31m"
)

func outputEvalResult(result *skillEvalResult) error {
	switch outputFormat {
	case "json":
		return outputEvalJSON([]*skillEvalResult{result})
	case "markdown":
		printEvalResultMarkdown(os.Stdout, result)
		return nil
	default:
		printEvalResult(result)
		return nil
	}
}

func outputMultiEvalResults(results []skillEvalResult) error {
	switch outputFormat {
	case "json":
		ptrs := make([]*skillEvalResult, len(results))
		for i := range results {
			ptrs[i] = &results[i]
		}
		return outputEvalJSON(ptrs)
	case "markdown":
		printMultiEvalResultsMarkdown(os.Stdout, results)
		return nil
	default:
		for i, r := range results {
			if i > 0 {
				fmt.Printf("\n%s\n", strings.Repeat("━", 60))
			}
			printEvalResult(&r)
		}
		return nil
	}
}

func printEvalResult(result *skillEvalResult) {
	fmt.Printf("\n%sScoring skill: %s%s\n", evalColorBold, result.SkillDir, evalColorReset)

	if result.SkillScores != nil {
		fmt.Printf("\n%sSKILL.md Scores%s\n", evalColorBold, evalColorReset)
		printDimScore("Clarity", result.SkillScores.Clarity)
		printDimScore("Actionability", result.SkillScores.Actionability)
		printDimScore("Token Efficiency", result.SkillScores.TokenEfficiency)
		printDimScore("Scope Discipline", result.SkillScores.ScopeDiscipline)
		printDimScore("Directive Precision", result.SkillScores.DirectivePrecision)
		printDimScore("Novelty", result.SkillScores.Novelty)
		fmt.Printf("  %s\n", strings.Repeat("─", 30))
		fmt.Printf("  %sOverall:              %.2f/5%s\n", evalColorBold, result.SkillScores.Overall, evalColorReset)

		if result.SkillScores.BriefAssessment != "" {
			fmt.Printf("\n  %s\"%s\"%s\n", evalColorCyan, result.SkillScores.BriefAssessment, evalColorReset)
		}

		if result.SkillScores.NovelInfo != "" {
			fmt.Printf("  %sNovel details: %s%s\n", evalColorCyan, result.SkillScores.NovelInfo, evalColorReset)
		}
	}

	if evalDisplay == "files" && len(result.RefResults) > 0 {
		for _, ref := range result.RefResults {
			fmt.Printf("\n%sReference: %s%s\n", evalColorBold, ref.File, evalColorReset)
			printDimScore("Clarity", ref.Scores.Clarity)
			printDimScore("Instructional Value", ref.Scores.InstructionalValue)
			printDimScore("Token Efficiency", ref.Scores.TokenEfficiency)
			printDimScore("Novelty", ref.Scores.Novelty)
			printDimScore("Skill Relevance", ref.Scores.SkillRelevance)
			fmt.Printf("  %s\n", strings.Repeat("─", 30))
			fmt.Printf("  %sOverall:              %.2f/5%s\n", evalColorBold, ref.Scores.Overall, evalColorReset)

			if ref.Scores.BriefAssessment != "" {
				fmt.Printf("\n  %s\"%s\"%s\n", evalColorCyan, ref.Scores.BriefAssessment, evalColorReset)
			}

			if ref.Scores.NovelInfo != "" {
				fmt.Printf("  %sNovel details: %s%s\n", evalColorCyan, ref.Scores.NovelInfo, evalColorReset)
			}
		}
	}

	if result.RefAggregate != nil {
		fmt.Printf("\n%sReference Scores (%d file%s)%s\n", evalColorBold, len(result.RefResults), pluralS(len(result.RefResults)), evalColorReset)
		printDimScore("Clarity", result.RefAggregate.Clarity)
		printDimScore("Instructional Value", result.RefAggregate.InstructionalValue)
		printDimScore("Token Efficiency", result.RefAggregate.TokenEfficiency)
		printDimScore("Novelty", result.RefAggregate.Novelty)
		printDimScore("Skill Relevance", result.RefAggregate.SkillRelevance)
		fmt.Printf("  %s\n", strings.Repeat("─", 30))
		fmt.Printf("  %sOverall:              %.2f/5%s\n", evalColorBold, result.RefAggregate.Overall, evalColorReset)
	}

	fmt.Println()
}

func printDimScore(name string, score int) {
	color := evalColorGreen
	if score <= 2 {
		color = evalColorRed
	} else if score <= 3 {
		color = evalColorYellow
	}
	padding := max(22-len(name), 1)
	fmt.Printf("  %s:%s%s%d/5%s\n", name, strings.Repeat(" ", padding), color, score, evalColorReset)
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// --- JSON output ---

type evalJSONOutput struct {
	Skills []evalJSONSkill `json:"skills"`
}

type evalJSONSkill struct {
	SkillDir     string             `json:"skill_dir"`
	SkillScores  *judge.SkillScores `json:"skill_scores,omitempty"`
	RefScores    []evalJSONRef      `json:"reference_scores,omitempty"`
	RefAggregate *judge.RefScores   `json:"reference_aggregate,omitempty"`
}

type evalJSONRef struct {
	File   string           `json:"file"`
	Scores *judge.RefScores `json:"scores"`
}

func outputEvalJSON(results []*skillEvalResult) error {
	out := evalJSONOutput{
		Skills: make([]evalJSONSkill, len(results)),
	}
	for i, r := range results {
		skill := evalJSONSkill{
			SkillDir:     r.SkillDir,
			SkillScores:  r.SkillScores,
			RefAggregate: r.RefAggregate,
		}
		for _, ref := range r.RefResults {
			skill.RefScores = append(skill.RefScores, evalJSONRef(ref))
		}
		out.Skills[i] = skill
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
