package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/dacharyc/skill-validator/internal/judge"
)

func outputReportListMarkdown(w io.Writer, results []*judge.CachedResult, skillDir string) {
	_, _ = fmt.Fprintf(w, "## Cached scores for: %s\n\n", skillDir)
	_, _ = fmt.Fprintf(w, "| File | Model | Scored At | Provider |\n")
	_, _ = fmt.Fprintf(w, "| --- | --- | --- | --- |\n")

	for _, r := range results {
		scored := r.ScoredAt.Local().Format("2006-01-02 15:04:05")
		_, _ = fmt.Fprintf(w, "| %s | %s | %s | %s |\n", r.File, r.Model, scored, r.Provider)
	}
}

func outputReportCompareMarkdown(w io.Writer, results []*judge.CachedResult, skillDir string) {
	byFile := make(map[string][]*judge.CachedResult)
	for _, r := range results {
		byFile[r.File] = append(byFile[r.File], r)
	}

	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	_, _ = fmt.Fprintf(w, "## Score comparison for: %s\n", skillDir)

	for _, file := range files {
		entries := byFile[file]

		// Get unique models
		models := make([]string, 0)
		seen := make(map[string]bool)
		for _, e := range entries {
			if !seen[e.Model] {
				models = append(models, e.Model)
				seen[e.Model] = true
			}
		}

		isSkill := file == "SKILL.md"

		_, _ = fmt.Fprintf(w, "\n### %s\n\n", file)

		// Build header
		_, _ = fmt.Fprintf(w, "| Dimension |")
		for _, m := range models {
			_, _ = fmt.Fprintf(w, " %s |", m)
		}
		_, _ = fmt.Fprintf(w, "\n| --- |")
		for range models {
			_, _ = fmt.Fprintf(w, " ---: |")
		}
		_, _ = fmt.Fprintf(w, "\n")

		// Build model→scores map
		modelScores := make(map[string]map[string]any)
		for _, e := range entries {
			if _, ok := modelScores[e.Model]; ok {
				continue
			}
			var scores map[string]any
			if err := json.Unmarshal(e.Scores, &scores); err == nil {
				modelScores[e.Model] = scores
			}
		}

		if isSkill {
			printCompareRowMarkdown(w, "Clarity", models, modelScores, "clarity")
			printCompareRowMarkdown(w, "Actionability", models, modelScores, "actionability")
			printCompareRowMarkdown(w, "Token Efficiency", models, modelScores, "token_efficiency")
			printCompareRowMarkdown(w, "Scope Discipline", models, modelScores, "scope_discipline")
			printCompareRowMarkdown(w, "Directive Precision", models, modelScores, "directive_precision")
			printCompareRowMarkdown(w, "Novelty", models, modelScores, "novelty")
			printCompareRowMarkdown(w, "**Overall**", models, modelScores, "overall")
		} else {
			printCompareRowMarkdown(w, "Clarity", models, modelScores, "clarity")
			printCompareRowMarkdown(w, "Instructional Value", models, modelScores, "instructional_value")
			printCompareRowMarkdown(w, "Token Efficiency", models, modelScores, "token_efficiency")
			printCompareRowMarkdown(w, "Novelty", models, modelScores, "novelty")
			printCompareRowMarkdown(w, "Skill Relevance", models, modelScores, "skill_relevance")
			printCompareRowMarkdown(w, "**Overall**", models, modelScores, "overall")
		}
	}
}

func printCompareRowMarkdown(w io.Writer, label string, models []string, modelScores map[string]map[string]any, key string) {
	_, _ = fmt.Fprintf(w, "| %s |", label)
	for _, m := range models {
		scores := modelScores[m]
		if scores == nil {
			_, _ = fmt.Fprintf(w, " - |")
			continue
		}
		val, ok := scores[key]
		if !ok {
			_, _ = fmt.Fprintf(w, " - |")
			continue
		}
		switch v := val.(type) {
		case float64:
			if key == "overall" {
				_, _ = fmt.Fprintf(w, " **%.2f/5** |", v)
			} else {
				_, _ = fmt.Fprintf(w, " %d/5 |", int(v))
			}
		default:
			_, _ = fmt.Fprintf(w, " %v |", v)
		}
	}
	_, _ = fmt.Fprintf(w, "\n")
}

func outputReportDefaultMarkdown(w io.Writer, results []*judge.CachedResult, skillDir string) {
	latest := judge.LatestByFile(results)

	_, _ = fmt.Fprintf(w, "## Cached scores for: %s\n", skillDir)

	// Show SKILL.md first
	if r, ok := latest["SKILL.md"]; ok {
		printCachedSkillScoresMarkdown(w, r)
		delete(latest, "SKILL.md")
	}

	refs := make([]string, 0, len(latest))
	for f := range latest {
		refs = append(refs, f)
	}
	sort.Strings(refs)

	for _, f := range refs {
		printCachedRefScoresMarkdown(w, latest[f])
	}
}

func printCachedSkillScoresMarkdown(w io.Writer, r *judge.CachedResult) {
	var scores judge.SkillScores
	if err := json.Unmarshal(r.Scores, &scores); err != nil {
		_, _ = fmt.Fprintf(w, "\nCould not parse cached SKILL.md scores\n")
		return
	}

	_, _ = fmt.Fprintf(w, "\n### SKILL.md Scores\n\n")
	_, _ = fmt.Fprintf(w, "*Model: %s, scored: %s*\n\n", r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"))
	_, _ = fmt.Fprintf(w, "| Dimension | Score |\n")
	_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
	_, _ = fmt.Fprintf(w, "| Clarity | %d/5 |\n", scores.Clarity)
	_, _ = fmt.Fprintf(w, "| Actionability | %d/5 |\n", scores.Actionability)
	_, _ = fmt.Fprintf(w, "| Token Efficiency | %d/5 |\n", scores.TokenEfficiency)
	_, _ = fmt.Fprintf(w, "| Scope Discipline | %d/5 |\n", scores.ScopeDiscipline)
	_, _ = fmt.Fprintf(w, "| Directive Precision | %d/5 |\n", scores.DirectivePrecision)
	_, _ = fmt.Fprintf(w, "| Novelty | %d/5 |\n", scores.Novelty)
	_, _ = fmt.Fprintf(w, "| **Overall** | **%.2f/5** |\n", scores.Overall)

	if scores.BriefAssessment != "" {
		_, _ = fmt.Fprintf(w, "\n> %s\n", scores.BriefAssessment)
	}

	if scores.NovelInfo != "" {
		_, _ = fmt.Fprintf(w, "\n*Novel details: %s*\n", scores.NovelInfo)
	}
}

func printCachedRefScoresMarkdown(w io.Writer, r *judge.CachedResult) {
	var scores judge.RefScores
	if err := json.Unmarshal(r.Scores, &scores); err != nil {
		_, _ = fmt.Fprintf(w, "\nCould not parse cached scores for %s\n", r.File)
		return
	}

	_, _ = fmt.Fprintf(w, "\n### Reference: %s\n\n", r.File)
	_, _ = fmt.Fprintf(w, "*Model: %s, scored: %s*\n\n", r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"))
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
