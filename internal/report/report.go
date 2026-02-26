package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/validator"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func Print(w io.Writer, r *validator.Report, perFile bool) {
	_, _ = fmt.Fprintf(w, "\n%sValidating skill: %s%s\n", colorBold, r.SkillDir, colorReset)

	// Group results by category, preserving order of first appearance
	var categories []string
	grouped := make(map[string][]validator.Result)
	for _, res := range r.Results {
		if _, exists := grouped[res.Category]; !exists {
			categories = append(categories, res.Category)
		}
		grouped[res.Category] = append(grouped[res.Category], res)
	}

	for _, cat := range categories {
		_, _ = fmt.Fprintf(w, "\n%s%s%s\n", colorBold, cat, colorReset)
		for _, res := range grouped[cat] {
			icon, color := formatLevel(res.Level)
			_, _ = fmt.Fprintf(w, "  %s%s %s%s\n", color, icon, res.Message, colorReset)
		}
	}

	// Token counts
	if len(r.TokenCounts) > 0 {
		_, _ = fmt.Fprintf(w, "\n%sTokens%s\n", colorBold, colorReset)

		maxFileLen := len("Total")
		for _, tc := range r.TokenCounts {
			if len(tc.File) > maxFileLen {
				maxFileLen = len(tc.File)
			}
		}

		total := 0
		for _, tc := range r.TokenCounts {
			total += tc.Tokens
			padding := maxFileLen - len(tc.File) + 2
			_, _ = fmt.Fprintf(w, "  %s%s:%s%s%s tokens\n", colorCyan, tc.File, colorReset, strings.Repeat(" ", padding), formatNumber(tc.Tokens))
		}

		separator := strings.Repeat("─", maxFileLen+20)
		_, _ = fmt.Fprintf(w, "  %s\n", separator)
		padding := maxFileLen - len("Total") + 2
		_, _ = fmt.Fprintf(w, "  %sTotal:%s%s%s tokens\n", colorBold, colorReset, strings.Repeat(" ", padding), formatNumber(total))
	}

	// Other files token counts
	if len(r.OtherTokenCounts) > 0 {
		_, _ = fmt.Fprintf(w, "\n%sOther files (outside standard structure)%s\n", colorBold, colorReset)

		maxFileLen := len("Total (other)")
		for _, tc := range r.OtherTokenCounts {
			if len(tc.File) > maxFileLen {
				maxFileLen = len(tc.File)
			}
		}

		total := 0
		for _, tc := range r.OtherTokenCounts {
			total += tc.Tokens
			padding := maxFileLen - len(tc.File) + 2
			countColor := ""
			countColorEnd := ""
			if tc.Tokens > 25_000 {
				countColor = colorRed
				countColorEnd = colorReset
			} else if tc.Tokens > 10_000 {
				countColor = colorYellow
				countColorEnd = colorReset
			}
			_, _ = fmt.Fprintf(w, "  %s%s:%s%s%s%s tokens%s\n", colorCyan, tc.File, colorReset, strings.Repeat(" ", padding), countColor, formatNumber(tc.Tokens), countColorEnd)
		}

		separator := strings.Repeat("─", maxFileLen+20)
		_, _ = fmt.Fprintf(w, "  %s\n", separator)
		label := "Total (other)"
		padding := maxFileLen - len(label) + 2
		totalColor := ""
		totalColorEnd := ""
		if total > 100_000 {
			totalColor = colorRed
			totalColorEnd = colorReset
		} else if total > 25_000 {
			totalColor = colorYellow
			totalColorEnd = colorReset
		}
		_, _ = fmt.Fprintf(w, "  %s%s:%s%s%s%s tokens%s\n", colorBold, label, colorReset, strings.Repeat(" ", padding), totalColor, formatNumber(total), totalColorEnd)
	}

	// Content analysis
	if r.ContentReport != nil {
		printContentReport(w, "Content Analysis", r.ContentReport)
	}

	// References content analysis
	if r.ReferencesContentReport != nil {
		printContentReport(w, "References Content Analysis", r.ReferencesContentReport)
	}

	// Per-file content analysis
	if perFile && len(r.ReferenceReports) > 0 {
		for _, fr := range r.ReferenceReports {
			if fr.ContentReport != nil {
				printContentReport(w, fmt.Sprintf("  [%s] Content Analysis", fr.File), fr.ContentReport)
			}
		}
	}

	// Contamination analysis
	if r.ContaminationReport != nil {
		printContaminationReport(w, "Contamination Analysis", r.ContaminationReport)
	}

	// References contamination analysis
	if r.ReferencesContaminationReport != nil {
		printContaminationReport(w, "References Contamination Analysis", r.ReferencesContaminationReport)
	}

	// Per-file contamination analysis
	if perFile && len(r.ReferenceReports) > 0 {
		for _, fr := range r.ReferenceReports {
			if fr.ContaminationReport != nil {
				printContaminationReport(w, fmt.Sprintf("  [%s] Contamination Analysis", fr.File), fr.ContaminationReport)
			}
		}
	}

	// Summary
	_, _ = fmt.Fprintln(w)
	if r.Errors == 0 && r.Warnings == 0 {
		_, _ = fmt.Fprintf(w, "%s%sResult: passed%s\n", colorBold, colorGreen, colorReset)
	} else {
		parts := []string{}
		if r.Errors > 0 {
			parts = append(parts, fmt.Sprintf("%s%d error%s%s", colorRed, r.Errors, pluralize(r.Errors), colorReset))
		}
		if r.Warnings > 0 {
			parts = append(parts, fmt.Sprintf("%s%d warning%s%s", colorYellow, r.Warnings, pluralize(r.Warnings), colorReset))
		}
		_, _ = fmt.Fprintf(w, "%sResult: %s%s\n", colorBold, strings.Join(parts, ", "), colorReset)
	}
	_, _ = fmt.Fprintln(w)
}

// PrintMulti prints each skill report separated by a line, with an overall summary.
func PrintMulti(w io.Writer, mr *validator.MultiReport, perFile bool) {
	for i, r := range mr.Skills {
		if i > 0 {
			_, _ = fmt.Fprintf(w, "\n%s\n", strings.Repeat("━", 60))
		}
		Print(w, r, perFile)
	}

	passed := 0
	failed := 0
	for _, r := range mr.Skills {
		if r.Errors == 0 {
			passed++
		} else {
			failed++
		}
	}

	_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("━", 60))
	_, _ = fmt.Fprintf(w, "\n%s%d skill%s validated: ", colorBold, len(mr.Skills), pluralize(len(mr.Skills)))
	if failed == 0 {
		_, _ = fmt.Fprintf(w, "%sall passed%s\n", colorGreen, colorReset)
	} else {
		skillParts := []string{}
		if passed > 0 {
			skillParts = append(skillParts, fmt.Sprintf("%s%d passed%s", colorGreen, passed, colorReset))
		}
		skillParts = append(skillParts, fmt.Sprintf("%s%d failed%s", colorRed, failed, colorReset))
		_, _ = fmt.Fprintf(w, "%s%s\n", strings.Join(skillParts, ", "), colorReset)
	}

	countParts := []string{}
	if mr.Errors > 0 {
		countParts = append(countParts, fmt.Sprintf("%s%d error%s%s", colorRed, mr.Errors, pluralize(mr.Errors), colorReset))
	}
	if mr.Warnings > 0 {
		countParts = append(countParts, fmt.Sprintf("%s%d warning%s%s", colorYellow, mr.Warnings, pluralize(mr.Warnings), colorReset))
	}
	if len(countParts) > 0 {
		_, _ = fmt.Fprintf(w, "%sTotal: %s%s\n", colorBold, strings.Join(countParts, ", "), colorReset)
	}
	_, _ = fmt.Fprintln(w)
}

func printContentReport(w io.Writer, title string, cr *content.Report) {
	_, _ = fmt.Fprintf(w, "\n%s%s%s\n", colorBold, title, colorReset)
	_, _ = fmt.Fprintf(w, "  Word count:               %s\n", formatNumber(cr.WordCount))
	_, _ = fmt.Fprintf(w, "  Code block ratio:         %.2f\n", cr.CodeBlockRatio)
	_, _ = fmt.Fprintf(w, "  Imperative ratio:         %.2f\n", cr.ImperativeRatio)
	_, _ = fmt.Fprintf(w, "  Information density:      %.2f\n", cr.InformationDensity)
	_, _ = fmt.Fprintf(w, "  Instruction specificity:  %.2f\n", cr.InstructionSpecificity)
	_, _ = fmt.Fprintf(w, "  Sections: %d  |  List items: %d  |  Code blocks: %d\n",
		cr.SectionCount, cr.ListItemCount, cr.CodeBlockCount)
}

func printContaminationReport(w io.Writer, title string, rr *contamination.Report) {
	_, _ = fmt.Fprintf(w, "\n%s%s%s\n", colorBold, title, colorReset)
	levelColor := colorGreen
	switch rr.ContaminationLevel {
	case "high":
		levelColor = colorRed
	case "medium":
		levelColor = colorYellow
	}
	_, _ = fmt.Fprintf(w, "  Contamination level: %s%s%s (score: %.2f)\n", levelColor, rr.ContaminationLevel, colorReset, rr.ContaminationScore)
	if rr.PrimaryCategory != "" {
		_, _ = fmt.Fprintf(w, "  Primary language category: %s\n", rr.PrimaryCategory)
	}
	if rr.LanguageMismatch && len(rr.MismatchedCategories) > 0 {
		_, _ = fmt.Fprintf(w, "  %s⚠ Language mismatch: %s (%d categor%s differ from primary)%s\n",
			colorYellow, strings.Join(rr.MismatchedCategories, ", "),
			len(rr.MismatchedCategories), ySuffix(len(rr.MismatchedCategories)), colorReset)
	}
	if len(rr.MultiInterfaceTools) > 0 {
		_, _ = fmt.Fprintf(w, "  %sℹ Multi-interface tool detected: %s%s\n",
			colorCyan, strings.Join(rr.MultiInterfaceTools, ", "), colorReset)
	}
	_, _ = fmt.Fprintf(w, "  Scope breadth: %d\n", rr.ScopeBreadth)
}

func formatLevel(level validator.Level) (string, string) {
	switch level {
	case validator.Pass:
		return "✓", colorGreen
	case validator.Info:
		return "ℹ", colorCyan
	case validator.Warning:
		return "⚠", colorYellow
	case validator.Error:
		return "✗", colorRed
	default:
		return "?", colorReset
	}
}

func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	// Insert commas
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func ySuffix(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
