package structure

import (
	"fmt"

	"github.com/dacharyc/skill-validator/internal/skill"
	"github.com/dacharyc/skill-validator/internal/validator"
)

// ValidateMulti validates each directory and returns an aggregated report.
func ValidateMulti(dirs []string) *validator.MultiReport {
	mr := &validator.MultiReport{}
	for _, dir := range dirs {
		r := Validate(dir)
		mr.Skills = append(mr.Skills, r)
		mr.Errors += r.Errors
		mr.Warnings += r.Warnings
	}
	return mr
}

// Validate runs all checks against the skill in the given directory.
func Validate(dir string) *validator.Report {
	report := &validator.Report{SkillDir: dir}

	// Structure checks
	structResults := CheckStructure(dir)
	report.Results = append(report.Results, structResults...)

	// Check if SKILL.md was found; if not, skip further checks
	hasSkillMD := false
	for _, r := range structResults {
		if r.Level == validator.Pass && r.Message == "SKILL.md found" {
			hasSkillMD = true
			break
		}
	}
	if !hasSkillMD {
		report.Tally()
		return report
	}

	// Parse skill
	s, err := skill.Load(dir)
	if err != nil {
		report.Results = append(report.Results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: err.Error()})
		report.Tally()
		return report
	}

	// Frontmatter checks
	report.Results = append(report.Results, CheckFrontmatter(s)...)

	// Token checks
	tokenResults, tokenCounts, otherCounts := CheckTokens(dir, s.Body)
	report.Results = append(report.Results, tokenResults...)
	report.TokenCounts = tokenCounts
	report.OtherTokenCounts = otherCounts

	// Holistic structure check: is this actually a skill?
	report.Results = append(report.Results, checkSkillRatio(report.TokenCounts, report.OtherTokenCounts)...)

	// Markdown structure checks (unclosed code fences)
	report.Results = append(report.Results, CheckMarkdown(dir, s.Body)...)

	// Internal link checks (broken relative links are a structural issue)
	report.Results = append(report.Results, CheckInternalLinks(dir, s.Body)...)

	report.Tally()
	return report
}

func checkSkillRatio(standard []validator.TokenCount, other []validator.TokenCount) []validator.Result {
	standardTotal := 0
	for _, tc := range standard {
		standardTotal += tc.Tokens
	}
	otherTotal := 0
	for _, tc := range other {
		otherTotal += tc.Tokens
	}

	if otherTotal > 25_000 && standardTotal > 0 && otherTotal > standardTotal*10 {
		return []validator.Result{{
			Level:    validator.Error,
			Category: "Overall",
			Message: fmt.Sprintf(
				"this content doesn't appear to be structured as a skill — "+
					"there are %s tokens of non-standard content but only %s tokens in the "+
					"standard skill structure (SKILL.md + references). This ratio suggests a "+
					"build pipeline issue or content that belongs in a different format, not a skill. "+
					"Per the spec, a skill should contain a focused SKILL.md with optional references, "+
					"scripts, and assets.",
				formatTokenCount(otherTotal), formatTokenCount(standardTotal),
			),
		}}
	}

	return nil
}

func formatTokenCount(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
