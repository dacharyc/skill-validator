package evaluate

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dacharyc/skill-validator/judge"
)

// ANSI color constants for terminal output.
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorRed    = "\033[31m"
)

// FormatResults formats a single EvalResult in the given format.
func FormatResults(w io.Writer, results []*EvalResult, format, display string) error {
	if len(results) == 0 {
		return nil
	}
	if len(results) == 1 {
		switch format {
		case "json":
			return PrintJSON(w, results)
		case "markdown":
			PrintMarkdown(w, results[0], display)
			return nil
		default:
			PrintText(w, results[0], display)
			return nil
		}
	}
	return FormatMultiResults(w, results, format, display)
}

// FormatMultiResults formats multiple EvalResults in the given format.
func FormatMultiResults(w io.Writer, results []*EvalResult, format, display string) error {
	switch format {
	case "json":
		return PrintJSON(w, results)
	case "markdown":
		PrintMultiMarkdown(w, results, display)
		return nil
	default:
		for i, r := range results {
			if i > 0 {
				fmt.Fprintf(w, "\n%s\n", strings.Repeat("━", 60))
			}
			PrintText(w, r, display)
		}
		return nil
	}
}

// PrintText writes a human-readable text representation of an EvalResult.
func PrintText(w io.Writer, result *EvalResult, display string) {
	fmt.Fprintf(w, "\n%sScoring skill: %s%s\n", ColorBold, result.SkillDir, ColorReset)

	if result.SkillScores != nil {
		fmt.Fprintf(w, "\n%sSKILL.md Scores%s\n", ColorBold, ColorReset)
		printDimScore(w, "Clarity", result.SkillScores.Clarity)
		printDimScore(w, "Actionability", result.SkillScores.Actionability)
		printDimScore(w, "Token Efficiency", result.SkillScores.TokenEfficiency)
		printDimScore(w, "Scope Discipline", result.SkillScores.ScopeDiscipline)
		printDimScore(w, "Directive Precision", result.SkillScores.DirectivePrecision)
		printDimScore(w, "Novelty", result.SkillScores.Novelty)
		fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 30))
		fmt.Fprintf(w, "  %sOverall:              %.2f/5%s\n", ColorBold, result.SkillScores.Overall, ColorReset)

		if result.SkillScores.BriefAssessment != "" {
			fmt.Fprintf(w, "\n  %s\"%s\"%s\n", ColorCyan, result.SkillScores.BriefAssessment, ColorReset)
		}

		if result.SkillScores.NovelInfo != "" {
			fmt.Fprintf(w, "  %sNovel details: %s%s\n", ColorCyan, result.SkillScores.NovelInfo, ColorReset)
		}
	}

	if display == "files" && len(result.RefResults) > 0 {
		for _, ref := range result.RefResults {
			fmt.Fprintf(w, "\n%sReference: %s%s\n", ColorBold, ref.File, ColorReset)
			printDimScore(w, "Clarity", ref.Scores.Clarity)
			printDimScore(w, "Instructional Value", ref.Scores.InstructionalValue)
			printDimScore(w, "Token Efficiency", ref.Scores.TokenEfficiency)
			printDimScore(w, "Novelty", ref.Scores.Novelty)
			printDimScore(w, "Skill Relevance", ref.Scores.SkillRelevance)
			fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 30))
			fmt.Fprintf(w, "  %sOverall:              %.2f/5%s\n", ColorBold, ref.Scores.Overall, ColorReset)

			if ref.Scores.BriefAssessment != "" {
				fmt.Fprintf(w, "\n  %s\"%s\"%s\n", ColorCyan, ref.Scores.BriefAssessment, ColorReset)
			}

			if ref.Scores.NovelInfo != "" {
				fmt.Fprintf(w, "  %sNovel details: %s%s\n", ColorCyan, ref.Scores.NovelInfo, ColorReset)
			}
		}
	}

	if result.RefAggregate != nil {
		fmt.Fprintf(w, "\n%sReference Scores (%d file%s)%s\n", ColorBold, len(result.RefResults), pluralS(len(result.RefResults)), ColorReset)
		printDimScore(w, "Clarity", result.RefAggregate.Clarity)
		printDimScore(w, "Instructional Value", result.RefAggregate.InstructionalValue)
		printDimScore(w, "Token Efficiency", result.RefAggregate.TokenEfficiency)
		printDimScore(w, "Novelty", result.RefAggregate.Novelty)
		printDimScore(w, "Skill Relevance", result.RefAggregate.SkillRelevance)
		fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 30))
		fmt.Fprintf(w, "  %sOverall:              %.2f/5%s\n", ColorBold, result.RefAggregate.Overall, ColorReset)
	}

	fmt.Fprintln(w)
}

func printDimScore(w io.Writer, name string, score int) {
	color := ColorGreen
	if score <= 2 {
		color = ColorRed
	} else if score <= 3 {
		color = ColorYellow
	}
	padding := max(22-len(name), 1)
	fmt.Fprintf(w, "  %s:%s%s%d/5%s\n", name, strings.Repeat(" ", padding), color, score, ColorReset)
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// --- JSON output ---

// EvalJSONOutput is the top-level JSON envelope.
type EvalJSONOutput struct {
	Skills []EvalJSONSkill `json:"skills"`
}

// EvalJSONSkill is one skill entry in JSON output.
type EvalJSONSkill struct {
	SkillDir     string             `json:"skill_dir"`
	SkillScores  *judge.SkillScores `json:"skill_scores,omitempty"`
	RefScores    []EvalJSONRef      `json:"reference_scores,omitempty"`
	RefAggregate *judge.RefScores   `json:"reference_aggregate,omitempty"`
}

// EvalJSONRef is one reference file entry in JSON output.
type EvalJSONRef struct {
	File   string           `json:"file"`
	Scores *judge.RefScores `json:"scores"`
}

// PrintJSON writes results as indented JSON.
func PrintJSON(w io.Writer, results []*EvalResult) error {
	out := EvalJSONOutput{
		Skills: make([]EvalJSONSkill, len(results)),
	}
	for i, r := range results {
		skill := EvalJSONSkill{
			SkillDir:     r.SkillDir,
			SkillScores:  r.SkillScores,
			RefAggregate: r.RefAggregate,
		}
		for _, ref := range r.RefResults {
			skill.RefScores = append(skill.RefScores, EvalJSONRef(ref))
		}
		out.Skills[i] = skill
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// --- Markdown output ---

// PrintMarkdown writes a single EvalResult as Markdown.
func PrintMarkdown(w io.Writer, result *EvalResult, display string) {
	_, _ = fmt.Fprintf(w, "## Scoring skill: %s\n", result.SkillDir)

	if result.SkillScores != nil {
		_, _ = fmt.Fprintf(w, "\n### SKILL.md Scores\n\n")
		_, _ = fmt.Fprintf(w, "| Dimension | Score |\n")
		_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
		_, _ = fmt.Fprintf(w, "| Clarity | %d/5 |\n", result.SkillScores.Clarity)
		_, _ = fmt.Fprintf(w, "| Actionability | %d/5 |\n", result.SkillScores.Actionability)
		_, _ = fmt.Fprintf(w, "| Token Efficiency | %d/5 |\n", result.SkillScores.TokenEfficiency)
		_, _ = fmt.Fprintf(w, "| Scope Discipline | %d/5 |\n", result.SkillScores.ScopeDiscipline)
		_, _ = fmt.Fprintf(w, "| Directive Precision | %d/5 |\n", result.SkillScores.DirectivePrecision)
		_, _ = fmt.Fprintf(w, "| Novelty | %d/5 |\n", result.SkillScores.Novelty)
		_, _ = fmt.Fprintf(w, "| **Overall** | **%.2f/5** |\n", result.SkillScores.Overall)

		if result.SkillScores.BriefAssessment != "" {
			_, _ = fmt.Fprintf(w, "\n> %s\n", result.SkillScores.BriefAssessment)
		}

		if result.SkillScores.NovelInfo != "" {
			_, _ = fmt.Fprintf(w, "\n*Novel details: %s*\n", result.SkillScores.NovelInfo)
		}
	}

	if display == "files" && len(result.RefResults) > 0 {
		for _, ref := range result.RefResults {
			printRefScoresMarkdown(w, ref.File, ref.Scores)
		}
	}

	if result.RefAggregate != nil {
		_, _ = fmt.Fprintf(w, "\n### Reference Scores (%d file%s)\n\n", len(result.RefResults), pluralS(len(result.RefResults)))
		_, _ = fmt.Fprintf(w, "| Dimension | Score |\n")
		_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
		_, _ = fmt.Fprintf(w, "| Clarity | %d/5 |\n", result.RefAggregate.Clarity)
		_, _ = fmt.Fprintf(w, "| Instructional Value | %d/5 |\n", result.RefAggregate.InstructionalValue)
		_, _ = fmt.Fprintf(w, "| Token Efficiency | %d/5 |\n", result.RefAggregate.TokenEfficiency)
		_, _ = fmt.Fprintf(w, "| Novelty | %d/5 |\n", result.RefAggregate.Novelty)
		_, _ = fmt.Fprintf(w, "| Skill Relevance | %d/5 |\n", result.RefAggregate.SkillRelevance)
		_, _ = fmt.Fprintf(w, "| **Overall** | **%.2f/5** |\n", result.RefAggregate.Overall)
	}
}

// PrintMultiMarkdown writes multiple EvalResults as Markdown, separated by rules.
func PrintMultiMarkdown(w io.Writer, results []*EvalResult, display string) {
	for i, r := range results {
		if i > 0 {
			_, _ = fmt.Fprintf(w, "\n---\n\n")
		}
		PrintMarkdown(w, r, display)
	}
}

func printRefScoresMarkdown(w io.Writer, file string, scores *judge.RefScores) {
	_, _ = fmt.Fprintf(w, "\n### Reference: %s\n\n", file)
	_, _ = fmt.Fprintf(w, "| Dimension | Score |\n")
	_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
	_, _ = fmt.Fprintf(w, "| Clarity | %d/5 |\n", scores.Clarity)
	_, _ = fmt.Fprintf(w, "| Instructional Value | %d/5 |\n", scores.InstructionalValue)
	_, _ = fmt.Fprintf(w, "| Token Efficiency | %d/5 |\n", scores.TokenEfficiency)
	_, _ = fmt.Fprintf(w, "| Novelty | %d/5 |\n", scores.Novelty)
	_, _ = fmt.Fprintf(w, "| Skill Relevance | %d/5 |\n", scores.SkillRelevance)
	_, _ = fmt.Fprintf(w, "| **Overall** | **%.2f/5** |\n", scores.Overall)

	if scores.BriefAssessment != "" {
		_, _ = fmt.Fprintf(w, "\n> %s\n", scores.BriefAssessment)
	}

	if scores.NovelInfo != "" {
		_, _ = fmt.Fprintf(w, "\n*Novel details: %s*\n", scores.NovelInfo)
	}
}
