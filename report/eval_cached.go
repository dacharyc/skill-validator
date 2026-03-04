package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/agent-ecosystem/skill-validator/judge"
	"github.com/agent-ecosystem/skill-validator/types"
	"github.com/agent-ecosystem/skill-validator/util"
)

// List formats cached results in list mode.
func List(w io.Writer, results []*judge.CachedResult, skillDir, format string) error {
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
		_, _ = fmt.Fprintf(w, "\n%sCached scores for: %s%s\n\n", colorBold, skillDir, colorReset)
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

// Compare formats cached results in comparison mode.
func Compare(w io.Writer, results []*judge.CachedResult, skillDir, format string) error {
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
	byFile := groupCachedByFile(results)
	files := util.SortedKeys(byFile)

	_, _ = fmt.Fprintf(w, "\n%sScore comparison for: %s%s\n", colorBold, skillDir, colorReset)

	for _, file := range files {
		entries := byFile[file]
		_, _ = fmt.Fprintf(w, "\n%s%s%s\n", colorBold, file, colorReset)

		models := uniqueModels(entries)
		modelScored := buildModelScored(entries)

		_, _ = fmt.Fprintf(w, "  %-22s", "Dimension")
		for _, m := range models {
			_, _ = fmt.Fprintf(w, " %-15s", truncateModel(m))
		}
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 22+16*len(models)))

		dims := dimensionLabels(entries)
		for _, label := range dims {
			printCompareRowScored(w, label, models, modelScored, false)
		}
		printCompareRowScored(w, "Overall", models, modelScored, true)
	}
	_, _ = fmt.Fprintln(w)
}

func printCompareRowScored(w io.Writer, label string, models []string, modelScored map[string]types.Scored, isOverall bool) {
	_, _ = fmt.Fprintf(w, "  %-22s", label)
	for _, m := range models {
		s := modelScored[m]
		if s == nil {
			_, _ = fmt.Fprintf(w, " %-15s", "-")
			continue
		}
		if isOverall {
			_, _ = fmt.Fprintf(w, " %-15s", fmt.Sprintf("%.2f/5", s.OverallScore()))
		} else {
			val := dimValueByLabel(s, label)
			_, _ = fmt.Fprintf(w, " %-15s", fmt.Sprintf("%d/5", val))
		}
	}
	_, _ = fmt.Fprintln(w)
}

func reportCompareMarkdown(w io.Writer, results []*judge.CachedResult, skillDir string) {
	byFile := groupCachedByFile(results)
	files := util.SortedKeys(byFile)

	_, _ = fmt.Fprintf(w, "## Score comparison for: %s\n", skillDir)

	for _, file := range files {
		entries := byFile[file]
		models := uniqueModels(entries)
		modelScored := buildModelScored(entries)

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

		dims := dimensionLabels(entries)
		for _, label := range dims {
			printCompareRowScoredMD(w, label, models, modelScored, false)
		}
		printCompareRowScoredMD(w, "**Overall**", models, modelScored, true)
	}
}

func printCompareRowScoredMD(w io.Writer, label string, models []string, modelScored map[string]types.Scored, isOverall bool) {
	_, _ = fmt.Fprintf(w, "| %s |", label)
	for _, m := range models {
		s := modelScored[m]
		if s == nil {
			_, _ = fmt.Fprintf(w, " - |")
			continue
		}
		if isOverall {
			_, _ = fmt.Fprintf(w, " **%.2f/5** |", s.OverallScore())
		} else {
			lookupLabel := strings.TrimPrefix(strings.TrimSuffix(label, "**"), "**")
			val := dimValueByLabel(s, lookupLabel)
			_, _ = fmt.Fprintf(w, " %d/5 |", val)
		}
	}
	_, _ = fmt.Fprintf(w, "\n")
}

// --- Helpers ---

func groupCachedByFile(results []*judge.CachedResult) map[string][]*judge.CachedResult {
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

func buildModelScored(entries []*judge.CachedResult) map[string]types.Scored {
	m := make(map[string]types.Scored)
	for _, e := range entries {
		if _, ok := m[e.Model]; ok {
			continue
		}
		if s, err := judge.DeserializeScored(e); err == nil {
			m[e.Model] = s
		}
	}
	return m
}

func dimensionLabels(entries []*judge.CachedResult) []string {
	for _, e := range entries {
		if s, err := judge.DeserializeScored(e); err == nil {
			dims := s.DimensionScores()
			labels := make([]string, len(dims))
			for i, d := range dims {
				labels[i] = d.Label
			}
			return labels
		}
	}
	return nil
}

func dimValueByLabel(s types.Scored, label string) int {
	for _, d := range s.DimensionScores() {
		if d.Label == label {
			return d.Value
		}
	}
	return 0
}

// Default formats the most recent cached results per file.
func Default(w io.Writer, results []*judge.CachedResult, skillDir, format string) error {
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
	_, _ = fmt.Fprintf(w, "\n%sCached scores for: %s%s\n", colorBold, skillDir, colorReset)

	if r, ok := latest["SKILL.md"]; ok {
		printCachedScoresText(w, r)
		delete(latest, "SKILL.md")
	}

	refs := make([]string, 0, len(latest))
	for f := range latest {
		refs = append(refs, f)
	}
	sort.Strings(refs)

	for _, f := range refs {
		printCachedScoresText(w, latest[f])
	}

	_, _ = fmt.Fprintln(w)
}

func printCachedScoresText(w io.Writer, r *judge.CachedResult) {
	scored, err := judge.DeserializeScored(r)
	if err != nil {
		_, _ = fmt.Fprintf(w, "\n  Could not parse cached scores for %s\n", r.File)
		return
	}

	if r.Type == "skill" || r.File == "SKILL.md" {
		_, _ = fmt.Fprintf(w, "\n%sSKILL.md Scores%s  %s(model: %s, scored: %s)%s\n",
			colorBold, colorReset,
			colorCyan, r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"), colorReset)
	} else {
		_, _ = fmt.Fprintf(w, "\n%sReference: %s%s  %s(model: %s, scored: %s)%s\n",
			colorBold, r.File, colorReset,
			colorCyan, r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"), colorReset)
	}

	printScoredText(w, scored)
}

func reportDefaultMarkdown(w io.Writer, latest map[string]*judge.CachedResult, skillDir string) {
	_, _ = fmt.Fprintf(w, "## Cached scores for: %s\n", skillDir)

	if r, ok := latest["SKILL.md"]; ok {
		printCachedScoresMarkdown(w, r)
		delete(latest, "SKILL.md")
	}

	refs := make([]string, 0, len(latest))
	for f := range latest {
		refs = append(refs, f)
	}
	sort.Strings(refs)

	for _, f := range refs {
		printCachedScoresMarkdown(w, latest[f])
	}
}

func printCachedScoresMarkdown(w io.Writer, r *judge.CachedResult) {
	scored, err := judge.DeserializeScored(r)
	if err != nil {
		_, _ = fmt.Fprintf(w, "\nCould not parse cached scores for %s\n", r.File)
		return
	}

	if r.Type == "skill" || r.File == "SKILL.md" {
		_, _ = fmt.Fprintf(w, "\n### SKILL.md Scores\n\n")
	} else {
		_, _ = fmt.Fprintf(w, "\n### Reference: %s\n\n", r.File)
	}
	_, _ = fmt.Fprintf(w, "*Model: %s, scored: %s*\n\n", r.Model, r.ScoredAt.Local().Format("2006-01-02 15:04"))
	printScoredMarkdown(w, scored)
}

func truncateModel(model string) string {
	if len(model) > 14 {
		return model[:11] + "..."
	}
	return model
}
