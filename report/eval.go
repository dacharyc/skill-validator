package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dacharyc/skill-validator/evaluate"
	"github.com/dacharyc/skill-validator/types"
	"github.com/dacharyc/skill-validator/util"
)

// FormatEvalResults formats a single EvalResult in the given format.
func FormatEvalResults(w io.Writer, results []*evaluate.Result, format, display string) error {
	if len(results) == 0 {
		return nil
	}
	if len(results) == 1 {
		switch format {
		case "json":
			return PrintEvalJSON(w, results)
		case "markdown":
			PrintEvalMarkdown(w, results[0], display)
			return nil
		default:
			PrintEvalText(w, results[0], display)
			return nil
		}
	}
	return FormatMultiEvalResults(w, results, format, display)
}

// FormatMultiEvalResults formats multiple EvalResults in the given format.
func FormatMultiEvalResults(w io.Writer, results []*evaluate.Result, format, display string) error {
	switch format {
	case "json":
		return PrintEvalJSON(w, results)
	case "markdown":
		PrintMultiEvalMarkdown(w, results, display)
		return nil
	default:
		for i, r := range results {
			if i > 0 {
				_, _ = fmt.Fprintf(w, "\n%s\n", strings.Repeat("━", 60))
			}
			PrintEvalText(w, r, display)
		}
		return nil
	}
}

// PrintEvalText writes a human-readable text representation of an EvalResult.
func PrintEvalText(w io.Writer, result *evaluate.Result, display string) {
	_, _ = fmt.Fprintf(w, "\n%sScoring skill: %s%s\n", colorBold, result.SkillDir, colorReset)

	if result.SkillScores != nil {
		_, _ = fmt.Fprintf(w, "\n%sSKILL.md Scores%s\n", colorBold, colorReset)
		printScoredText(w, result.SkillScores)
	}

	if display == "files" && len(result.RefResults) > 0 {
		for _, ref := range result.RefResults {
			_, _ = fmt.Fprintf(w, "\n%sReference: %s%s\n", colorBold, ref.File, colorReset)
			printScoredText(w, ref.Scores)
		}
	}

	if result.RefAggregate != nil {
		_, _ = fmt.Fprintf(w, "\n%sReference Scores (%d file%s)%s\n", colorBold, len(result.RefResults), util.PluralS(len(result.RefResults)), colorReset)
		printScoredText(w, result.RefAggregate)
	}

	_, _ = fmt.Fprintln(w)
}

// printScoredText writes all dimensions, overall, assessment, and novel details for a Scored value.
func printScoredText(w io.Writer, s types.Scored) {
	for _, d := range s.DimensionScores() {
		printDimScore(w, d.Label, d.Value)
	}
	_, _ = fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 30))
	_, _ = fmt.Fprintf(w, "  %sOverall:              %.2f/5%s\n", colorBold, s.OverallScore(), colorReset)

	if s.Assessment() != "" {
		_, _ = fmt.Fprintf(w, "\n  %s\"%s\"%s\n", colorCyan, s.Assessment(), colorReset)
	}
	if s.NovelDetails() != "" {
		_, _ = fmt.Fprintf(w, "  %sNovel details: %s%s\n", colorCyan, s.NovelDetails(), colorReset)
	}
}

func printDimScore(w io.Writer, name string, score int) {
	color := colorGreen
	if score <= 2 {
		color = colorRed
	} else if score <= 3 {
		color = colorYellow
	}
	padding := max(22-len(name), 1)
	_, _ = fmt.Fprintf(w, "  %s:%s%s%d/5%s\n", name, strings.Repeat(" ", padding), color, score, colorReset)
}

// --- JSON output ---

// EvalJSONOutput is the top-level JSON envelope.
type EvalJSONOutput struct {
	Skills []EvalJSONSkill `json:"skills"`
}

// EvalJSONSkill is one skill entry in JSON output.
type EvalJSONSkill struct {
	SkillDir     string        `json:"skill_dir"`
	SkillScores  any           `json:"skill_scores,omitempty"`
	RefScores    []EvalJSONRef `json:"reference_scores,omitempty"`
	RefAggregate any           `json:"reference_aggregate,omitempty"`
}

// EvalJSONRef is one reference file entry in JSON output.
type EvalJSONRef struct {
	File   string `json:"file"`
	Scores any    `json:"scores"`
}

// PrintEvalJSON writes results as indented JSON.
func PrintEvalJSON(w io.Writer, results []*evaluate.Result) error {
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
			skill.RefScores = append(skill.RefScores, EvalJSONRef{File: ref.File, Scores: ref.Scores})
		}
		out.Skills[i] = skill
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// --- Markdown output ---

// PrintEvalMarkdown writes a single EvalResult as Markdown.
func PrintEvalMarkdown(w io.Writer, result *evaluate.Result, display string) {
	_, _ = fmt.Fprintf(w, "## Scoring skill: %s\n", result.SkillDir)

	if result.SkillScores != nil {
		_, _ = fmt.Fprintf(w, "\n### SKILL.md Scores\n\n")
		printScoredMarkdown(w, result.SkillScores)
	}

	if display == "files" && len(result.RefResults) > 0 {
		for _, ref := range result.RefResults {
			_, _ = fmt.Fprintf(w, "\n### Reference: %s\n\n", ref.File)
			printScoredMarkdown(w, ref.Scores)
		}
	}

	if result.RefAggregate != nil {
		_, _ = fmt.Fprintf(w, "\n### Reference Scores (%d file%s)\n\n", len(result.RefResults), util.PluralS(len(result.RefResults)))
		printScoredMarkdown(w, result.RefAggregate)
	}
}

// PrintMultiEvalMarkdown writes multiple EvalResults as Markdown, separated by rules.
func PrintMultiEvalMarkdown(w io.Writer, results []*evaluate.Result, display string) {
	for i, r := range results {
		if i > 0 {
			_, _ = fmt.Fprintf(w, "\n---\n\n")
		}
		PrintEvalMarkdown(w, r, display)
	}
}

// printScoredMarkdown writes a markdown table for all dimensions plus overall, assessment, and novel details.
func printScoredMarkdown(w io.Writer, s types.Scored) {
	_, _ = fmt.Fprintf(w, "| Dimension | Score |\n")
	_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
	for _, d := range s.DimensionScores() {
		_, _ = fmt.Fprintf(w, "| %s | %d/5 |\n", d.Label, d.Value)
	}
	_, _ = fmt.Fprintf(w, "| **Overall** | **%.2f/5** |\n", s.OverallScore())

	if s.Assessment() != "" {
		_, _ = fmt.Fprintf(w, "\n> %s\n", s.Assessment())
	}
	if s.NovelDetails() != "" {
		_, _ = fmt.Fprintf(w, "\n*Novel details: %s*\n", s.NovelDetails())
	}
}
