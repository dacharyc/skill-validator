package evaluate

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dacharyc/skill-validator/judge"
	"github.com/dacharyc/skill-validator/util"
)

// ReportList formats cached results in list mode.
func ReportList(w io.Writer, results []*judge.CachedResult, skillDir, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	case "markdown":
		_, _ = fmt.Fprintf(w, "## Cached scores for: %s\n\n", skillDir)
		_, _ = fmt.Fprintf(w, "| File | Model | Scored At | Provider |\n")
		_, _ = fmt.Fprintf(w, "| --- | --- | --- | --- |\n")
		for _, r := range results {
			scored := r.ScoredAt.Local().Format("2006-01-02 15:04:05")
			_, _ = fmt.Fprintf(w, "| %s | %s | %s | %s |\n", r.File, r.Model, scored, r.Provider)
		}
		return nil
	default:
		_, _ = fmt.Fprintf(w, "\n%sCached scores for: %s%s\n\n", ColorBold, skillDir, ColorReset)
		_, _ = fmt.Fprintf(w, "  %-28s %-30s %-20s %s\n", "File", "Model", "Scored At", "Provider")
		_, _ = fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 90))
		for _, r := range results {
			scored := r.ScoredAt.Local().Format("2006-01-02 15:04:05")
			_, _ = fmt.Fprintf(w, "  %-28s %-30s %-20s %s\n", r.File, r.Model, scored, r.Provider)
		}
		_, _ = fmt.Fprintln(w)
		return nil
	}
}

// ReportCompare formats cached results in comparison mode.
func ReportCompare(w io.Writer, results []*judge.CachedResult, skillDir, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	case "markdown":
		reportCompareMarkdown(w, results, skillDir)
		return nil
	default:
		reportCompareText(w, results, skillDir)
		return nil
	}
}

func reportCompareText(w io.Writer, results []*judge.CachedResult, skillDir string) {
	byFile := groupByFile(results)
	files := util.SortedKeys(byFile)

	_, _ = fmt.Fprintf(w, "\n%sScore comparison for: %s%s\n", ColorBold, skillDir, ColorReset)

	for _, file := range files {
		entries := byFile[file]
		_, _ = fmt.Fprintf(w, "\n%s%s%s\n", ColorBold, file, ColorReset)

		models := uniqueModels(entries)
		isSkill := file == "SKILL.md"

		_, _ = fmt.Fprintf(w, "  %-22s", "Dimension")
		for _, m := range models {
			_, _ = fmt.Fprintf(w, " %-15s", truncateModel(m))
		}
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 22+16*len(models)))

		if isSkill {
			printCompareRow(w, "Clarity", entries, models, "clarity")
			printCompareRow(w, "Actionability", entries, models, "actionability")
			printCompareRow(w, "Token Efficiency", entries, models, "token_efficiency")
			printCompareRow(w, "Scope Discipline", entries, models, "scope_discipline")
			printCompareRow(w, "Directive Precision", entries, models, "directive_precision")
			printCompareRow(w, "Novelty", entries, models, "novelty")
			printCompareRow(w, "Overall", entries, models, "overall")
		} else {
			printCompareRow(w, "Clarity", entries, models, "clarity")
			printCompareRow(w, "Instructional Value", entries, models, "instructional_value")
			printCompareRow(w, "Token Efficiency", entries, models, "token_efficiency")
			printCompareRow(w, "Novelty", entries, models, "novelty")
			printCompareRow(w, "Skill Relevance", entries, models, "skill_relevance")
			printCompareRow(w, "Overall", entries, models, "overall")
		}
	}
	_, _ = fmt.Fprintln(w)
}

func printCompareRow(w io.Writer, label string, entries []*judge.CachedResult, models []string, key string) {
	_, _ = fmt.Fprintf(w, "  %-22s", label)

	modelScores := buildModelScores(entries)

	for _, m := range models {
		scores := modelScores[m]
		if scores == nil {
			_, _ = fmt.Fprintf(w, " %-15s", "-")
			continue
		}
		val, ok := scores[key]
		if !ok {
			_, _ = fmt.Fprintf(w, " %-15s", "-")
			continue
		}
		switch v := val.(type) {
		case float64:
			if key == "overall" {
				_, _ = fmt.Fprintf(w, " %-15s", fmt.Sprintf("%.2f/5", v))
			} else {
				_, _ = fmt.Fprintf(w, " %-15s", fmt.Sprintf("%d/5", int(v)))
			}
		default:
			_, _ = fmt.Fprintf(w, " %-15v", v)
		}
	}
	_, _ = fmt.Fprintln(w)
}

func reportCompareMarkdown(w io.Writer, results []*judge.CachedResult, skillDir string) {
	byFile := groupByFile(results)
	files := util.SortedKeys(byFile)

	_, _ = fmt.Fprintf(w, "## Score comparison for: %s\n", skillDir)

	for _, file := range files {
		entries := byFile[file]
		models := uniqueModels(entries)
		isSkill := file == "SKILL.md"

		_, _ = fmt.Fprintf(w, "\n### %s\n\n", file)

		_, _ = fmt.Fprintf(w, "| Dimension |")
		for _, m := range models {
			_, _ = fmt.Fprintf(w, " %s |", m)
		}
		_, _ = fmt.Fprintf(w, "\n| --- |")
		for range models {
			_, _ = fmt.Fprintf(w, " ---: |")
		}
		_, _ = fmt.Fprintf(w, "\n")

		modelScores := buildModelScores(entries)

		if isSkill {
			printCompareRowMD(w, "Clarity", models, modelScores, "clarity")
			printCompareRowMD(w, "Actionability", models, modelScores, "actionability")
			printCompareRowMD(w, "Token Efficiency", models, modelScores, "token_efficiency")
			printCompareRowMD(w, "Scope Discipline", models, modelScores, "scope_discipline")
			printCompareRowMD(w, "Directive Precision", models, modelScores, "directive_precision")
			printCompareRowMD(w, "Novelty", models, modelScores, "novelty")
			printCompareRowMD(w, "**Overall**", models, modelScores, "overall")
		} else {
			printCompareRowMD(w, "Clarity", models, modelScores, "clarity")
			printCompareRowMD(w, "Instructional Value", models, modelScores, "instructional_value")
			printCompareRowMD(w, "Token Efficiency", models, modelScores, "token_efficiency")
			printCompareRowMD(w, "Novelty", models, modelScores, "novelty")
			printCompareRowMD(w, "Skill Relevance", models, modelScores, "skill_relevance")
			printCompareRowMD(w, "**Overall**", models, modelScores, "overall")
		}
	}
}

func printCompareRowMD(w io.Writer, label string, models []string, modelScores map[string]map[string]any, key string) {
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

// --- Helpers ---

func groupByFile(results []*judge.CachedResult) map[string][]*judge.CachedResult {
	byFile := make(map[string][]*judge.CachedResult)
	for _, r := range results {
		byFile[r.File] = append(byFile[r.File], r)
	}
	return byFile
}

func uniqueModels(entries []*judge.CachedResult) []string {
	var models []string
	seen := make(map[string]bool)
	for _, e := range entries {
		if !seen[e.Model] {
			models = append(models, e.Model)
			seen[e.Model] = true
		}
	}
	return models
}

func buildModelScores(entries []*judge.CachedResult) map[string]map[string]any {
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
	return modelScores
}

// ReportDefault formats the most recent cached results per file.
func ReportDefault(w io.Writer, results []*judge.CachedResult, skillDir, format string) error {
	latest := judge.LatestByFile(results)

	if format == "json" {
		vals := make([]*judge.CachedResult, 0, len(latest))
		for _, v := range latest {
			vals = append(vals, v)
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(vals)
	}

	if format == "markdown" {
		reportDefaultMarkdown(w, latest, skillDir)
		return nil
	}

	reportDefaultText(w, latest, skillDir)
	return nil
}

func reportDefaultText(w io.Writer, latest map[string]*judge.CachedResult, skillDir string) {
	_, _ = fmt.Fprintf(w, "\n%sCached scores for: %s%s\n", ColorBold, skillDir, ColorReset)

	if r, ok := latest["SKILL.md"]; ok {
		printCachedSkillScores(w, r)
		delete(latest, "SKILL.md")
	}

	refs := make([]string, 0, len(latest))
	for f := range latest {
		refs = append(refs, f)
	}
	sort.Strings(refs)

	for _, f := range refs {
		printCachedRefScores(w, latest[f])
	}

	_, _ = fmt.Fprintln(w)
}

func printCachedSkillScores(w io.Writer, r *judge.CachedResult) {
	var scores judge.SkillScores
	if err := json.Unmarshal(r.Scores, &scores); err != nil {
		_, _ = fmt.Fprintf(w, "\n  Could not parse cached SKILL.md scores\n")
		return
	}

	_, _ = fmt.Fprintf(w, "\n%sSKILL.md Scores%s  %s(model: %s, scored: %s)%s\n",
		ColorBold, ColorReset,
		ColorCyan, r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"), ColorReset)

	printDimScore(w, "Clarity", scores.Clarity)
	printDimScore(w, "Actionability", scores.Actionability)
	printDimScore(w, "Token Efficiency", scores.TokenEfficiency)
	printDimScore(w, "Scope Discipline", scores.ScopeDiscipline)
	printDimScore(w, "Directive Precision", scores.DirectivePrecision)
	printDimScore(w, "Novelty", scores.Novelty)
	_, _ = fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 30))
	_, _ = fmt.Fprintf(w, "  %sOverall:              %.2f/5%s\n", ColorBold, scores.Overall, ColorReset)

	if scores.BriefAssessment != "" {
		_, _ = fmt.Fprintf(w, "\n  %s\"%s\"%s\n", ColorCyan, scores.BriefAssessment, ColorReset)
	}
	if scores.NovelInfo != "" {
		_, _ = fmt.Fprintf(w, "  %sNovel details: %s%s\n", ColorCyan, scores.NovelInfo, ColorReset)
	}
}

func printCachedRefScores(w io.Writer, r *judge.CachedResult) {
	var scores judge.RefScores
	if err := json.Unmarshal(r.Scores, &scores); err != nil {
		_, _ = fmt.Fprintf(w, "\n  Could not parse cached scores for %s\n", r.File)
		return
	}

	_, _ = fmt.Fprintf(w, "\n%sReference: %s%s  %s(model: %s, scored: %s)%s\n",
		ColorBold, r.File, ColorReset,
		ColorCyan, r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"), ColorReset)

	printDimScore(w, "Clarity", scores.Clarity)
	printDimScore(w, "Instructional Value", scores.InstructionalValue)
	printDimScore(w, "Token Efficiency", scores.TokenEfficiency)
	printDimScore(w, "Novelty", scores.Novelty)
	printDimScore(w, "Skill Relevance", scores.SkillRelevance)
	_, _ = fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 30))
	_, _ = fmt.Fprintf(w, "  %sOverall:              %.2f/5%s\n", ColorBold, scores.Overall, ColorReset)

	if scores.BriefAssessment != "" {
		_, _ = fmt.Fprintf(w, "\n  %s\"%s\"%s\n", ColorCyan, scores.BriefAssessment, ColorReset)
	}
	if scores.NovelInfo != "" {
		_, _ = fmt.Fprintf(w, "  %sNovel details: %s%s\n", ColorCyan, scores.NovelInfo, ColorReset)
	}
}

func reportDefaultMarkdown(w io.Writer, latest map[string]*judge.CachedResult, skillDir string) {
	_, _ = fmt.Fprintf(w, "## Cached scores for: %s\n", skillDir)

	if r, ok := latest["SKILL.md"]; ok {
		printCachedSkillScoresMD(w, r)
		delete(latest, "SKILL.md")
	}

	refs := make([]string, 0, len(latest))
	for f := range latest {
		refs = append(refs, f)
	}
	sort.Strings(refs)

	for _, f := range refs {
		printCachedRefScoresMD(w, latest[f])
	}
}

func printCachedSkillScoresMD(w io.Writer, r *judge.CachedResult) {
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

func printCachedRefScoresMD(w io.Writer, r *judge.CachedResult) {
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

func truncateModel(model string) string {
	if len(model) > 14 {
		return model[:11] + "..."
	}
	return model
}
