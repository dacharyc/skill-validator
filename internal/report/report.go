package report

import (
	"fmt"
	"io"
	"strings"

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

func Print(w io.Writer, r *validator.Report) {
	fmt.Fprintf(w, "\n%sValidating skill: %s%s\n", colorBold, r.SkillDir, colorReset)

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
		fmt.Fprintf(w, "\n%s%s%s\n", colorBold, cat, colorReset)
		for _, res := range grouped[cat] {
			icon, color := formatLevel(res.Level)
			fmt.Fprintf(w, "  %s%s %s%s\n", color, icon, res.Message, colorReset)
		}
	}

	// Token counts
	if len(r.TokenCounts) > 0 {
		fmt.Fprintf(w, "\n%sTokens%s\n", colorBold, colorReset)

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
			fmt.Fprintf(w, "  %s%s:%s%s%s tokens\n", colorCyan, tc.File, colorReset, strings.Repeat(" ", padding), formatNumber(tc.Tokens))
		}

		separator := strings.Repeat("─", maxFileLen+20)
		fmt.Fprintf(w, "  %s\n", separator)
		padding := maxFileLen - len("Total") + 2
		fmt.Fprintf(w, "  %sTotal:%s%s%s tokens\n", colorBold, colorReset, strings.Repeat(" ", padding), formatNumber(total))
	}

	// Other files token counts
	if len(r.OtherTokenCounts) > 0 {
		fmt.Fprintf(w, "\n%sOther files (outside standard structure)%s\n", colorBold, colorReset)

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
			fmt.Fprintf(w, "  %s%s:%s%s%s%s tokens%s\n", colorCyan, tc.File, colorReset, strings.Repeat(" ", padding), countColor, formatNumber(tc.Tokens), countColorEnd)
		}

		separator := strings.Repeat("─", maxFileLen+20)
		fmt.Fprintf(w, "  %s\n", separator)
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
		fmt.Fprintf(w, "  %s%s:%s%s%s%s tokens%s\n", colorBold, label, colorReset, strings.Repeat(" ", padding), totalColor, formatNumber(total), totalColorEnd)
	}

	// Summary
	fmt.Fprintln(w)
	if r.Errors == 0 && r.Warnings == 0 {
		fmt.Fprintf(w, "%s%sResult: passed%s\n", colorBold, colorGreen, colorReset)
	} else {
		parts := []string{}
		if r.Errors > 0 {
			parts = append(parts, fmt.Sprintf("%s%d error%s%s", colorRed, r.Errors, pluralize(r.Errors), colorReset))
		}
		if r.Warnings > 0 {
			parts = append(parts, fmt.Sprintf("%s%d warning%s%s", colorYellow, r.Warnings, pluralize(r.Warnings), colorReset))
		}
		fmt.Fprintf(w, "%sResult: %s%s\n", colorBold, strings.Join(parts, ", "), colorReset)
	}
	fmt.Fprintln(w)
}

func formatLevel(level validator.Level) (string, string) {
	switch level {
	case validator.Pass:
		return "✓", colorGreen
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
