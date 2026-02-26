package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/validator"
)

// PrintMarkdown writes the report as GitHub-flavored markdown to the given writer.
func PrintMarkdown(w io.Writer, r *validator.Report, perFile bool) error {
	_, _ = fmt.Fprintf(w, "## Validating skill: %s\n", r.SkillDir)

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
		_, _ = fmt.Fprintf(w, "\n### %s\n\n", cat)
		for _, res := range grouped[cat] {
			prefix := markdownLevelPrefix(res.Level)
			_, _ = fmt.Fprintf(w, "- %s %s\n", prefix, res.Message)
		}
	}

	// Token counts
	if len(r.TokenCounts) > 0 {
		_, _ = fmt.Fprintf(w, "\n### Tokens\n\n")
		_, _ = fmt.Fprintf(w, "| File | Tokens |\n")
		_, _ = fmt.Fprintf(w, "| --- | ---: |\n")

		total := 0
		for _, tc := range r.TokenCounts {
			total += tc.Tokens
			_, _ = fmt.Fprintf(w, "| %s | %s |\n", tc.File, formatNumber(tc.Tokens))
		}
		_, _ = fmt.Fprintf(w, "| **Total** | **%s** |\n", formatNumber(total))
	}

	// Other files token counts
	if len(r.OtherTokenCounts) > 0 {
		_, _ = fmt.Fprintf(w, "\n### Other files\n\n")
		_, _ = fmt.Fprintf(w, "| File | Tokens |\n")
		_, _ = fmt.Fprintf(w, "| --- | ---: |\n")

		total := 0
		for _, tc := range r.OtherTokenCounts {
			total += tc.Tokens
			_, _ = fmt.Fprintf(w, "| %s | %s |\n", tc.File, formatNumber(tc.Tokens))
		}
		_, _ = fmt.Fprintf(w, "| **Total (other)** | **%s** |\n", formatNumber(total))
	}

	// Content analysis
	if r.ContentReport != nil {
		printMarkdownContentReport(w, "Content Analysis", r.ContentReport)
	}

	// References content analysis
	if r.ReferencesContentReport != nil {
		printMarkdownContentReport(w, "References Content Analysis", r.ReferencesContentReport)
	}

	// Per-file content analysis
	if perFile && len(r.ReferenceReports) > 0 {
		for _, fr := range r.ReferenceReports {
			if fr.ContentReport != nil {
				printMarkdownContentReport(w, fmt.Sprintf("[%s] Content Analysis", fr.File), fr.ContentReport)
			}
		}
	}

	// Contamination analysis
	if r.ContaminationReport != nil {
		printMarkdownContaminationReport(w, "Contamination Analysis", r.ContaminationReport)
	}

	// References contamination analysis
	if r.ReferencesContaminationReport != nil {
		printMarkdownContaminationReport(w, "References Contamination Analysis", r.ReferencesContaminationReport)
	}

	// Per-file contamination analysis
	if perFile && len(r.ReferenceReports) > 0 {
		for _, fr := range r.ReferenceReports {
			if fr.ContaminationReport != nil {
				printMarkdownContaminationReport(w, fmt.Sprintf("[%s] Contamination Analysis", fr.File), fr.ContaminationReport)
			}
		}
	}

	// Summary
	_, _ = fmt.Fprintln(w)
	if r.Errors == 0 && r.Warnings == 0 {
		_, _ = fmt.Fprintf(w, "**Result: passed**\n")
	} else {
		parts := []string{}
		if r.Errors > 0 {
			parts = append(parts, fmt.Sprintf("%d error%s", r.Errors, pluralize(r.Errors)))
		}
		if r.Warnings > 0 {
			parts = append(parts, fmt.Sprintf("%d warning%s", r.Warnings, pluralize(r.Warnings)))
		}
		_, _ = fmt.Fprintf(w, "**Result: %s**\n", strings.Join(parts, ", "))
	}

	return nil
}

// PrintMultiMarkdown writes the multi-skill report as GitHub-flavored markdown.
func PrintMultiMarkdown(w io.Writer, mr *validator.MultiReport, perFile bool) error {
	for i, r := range mr.Skills {
		if i > 0 {
			_, _ = fmt.Fprintf(w, "\n---\n\n")
		}
		if err := PrintMarkdown(w, r, perFile); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(w, "\n---\n\n")

	passed := 0
	failed := 0
	for _, r := range mr.Skills {
		if r.Errors == 0 {
			passed++
		} else {
			failed++
		}
	}

	_, _ = fmt.Fprintf(w, "**%d skill%s validated: ", len(mr.Skills), pluralize(len(mr.Skills)))
	if failed == 0 {
		_, _ = fmt.Fprintf(w, "all passed**\n")
	} else {
		skillParts := []string{}
		if passed > 0 {
			skillParts = append(skillParts, fmt.Sprintf("%d passed", passed))
		}
		skillParts = append(skillParts, fmt.Sprintf("%d failed", failed))
		_, _ = fmt.Fprintf(w, "%s**\n", strings.Join(skillParts, ", "))
	}

	countParts := []string{}
	if mr.Errors > 0 {
		countParts = append(countParts, fmt.Sprintf("%d error%s", mr.Errors, pluralize(mr.Errors)))
	}
	if mr.Warnings > 0 {
		countParts = append(countParts, fmt.Sprintf("%d warning%s", mr.Warnings, pluralize(mr.Warnings)))
	}
	if len(countParts) > 0 {
		_, _ = fmt.Fprintf(w, "**Total: %s**\n", strings.Join(countParts, ", "))
	}

	return nil
}

func markdownLevelPrefix(level validator.Level) string {
	switch level {
	case validator.Pass:
		return "**Pass:**"
	case validator.Info:
		return "**Info:**"
	case validator.Warning:
		return "**Warning:**"
	case validator.Error:
		return "**Error:**"
	default:
		return ""
	}
}

func printMarkdownContentReport(w io.Writer, title string, cr *content.Report) {
	_, _ = fmt.Fprintf(w, "\n### %s\n\n", title)
	_, _ = fmt.Fprintf(w, "| Metric | Value |\n")
	_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
	_, _ = fmt.Fprintf(w, "| Word count | %s |\n", formatNumber(cr.WordCount))
	_, _ = fmt.Fprintf(w, "| Code block ratio | %.2f |\n", cr.CodeBlockRatio)
	_, _ = fmt.Fprintf(w, "| Imperative ratio | %.2f |\n", cr.ImperativeRatio)
	_, _ = fmt.Fprintf(w, "| Information density | %.2f |\n", cr.InformationDensity)
	_, _ = fmt.Fprintf(w, "| Instruction specificity | %.2f |\n", cr.InstructionSpecificity)
	_, _ = fmt.Fprintf(w, "| Sections | %d |\n", cr.SectionCount)
	_, _ = fmt.Fprintf(w, "| List items | %d |\n", cr.ListItemCount)
	_, _ = fmt.Fprintf(w, "| Code blocks | %d |\n", cr.CodeBlockCount)
}

func printMarkdownContaminationReport(w io.Writer, title string, rr *contamination.Report) {
	_, _ = fmt.Fprintf(w, "\n### %s\n\n", title)
	_, _ = fmt.Fprintf(w, "| Metric | Value |\n")
	_, _ = fmt.Fprintf(w, "| --- | --- |\n")
	_, _ = fmt.Fprintf(w, "| Contamination level | %s |\n", rr.ContaminationLevel)
	_, _ = fmt.Fprintf(w, "| Contamination score | %.2f |\n", rr.ContaminationScore)
	if rr.PrimaryCategory != "" {
		_, _ = fmt.Fprintf(w, "| Primary language category | %s |\n", rr.PrimaryCategory)
	}
	_, _ = fmt.Fprintf(w, "| Scope breadth | %d |\n", rr.ScopeBreadth)

	if rr.LanguageMismatch && len(rr.MismatchedCategories) > 0 {
		_, _ = fmt.Fprintf(w, "\n- **Warning: Language mismatch:** %s (%d categor%s differ from primary)\n",
			strings.Join(rr.MismatchedCategories, ", "),
			len(rr.MismatchedCategories), ySuffix(len(rr.MismatchedCategories)))
	}
	if len(rr.MultiInterfaceTools) > 0 {
		_, _ = fmt.Fprintf(w, "- **Multi-interface tool detected:** %s\n",
			strings.Join(rr.MultiInterfaceTools, ", "))
	}
}
