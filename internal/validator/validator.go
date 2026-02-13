package validator

import (
	"github.com/dacharyc/skill-validator/internal/skill"
)

// Level represents the severity of a validation result.
type Level int

const (
	Pass    Level = iota
	Warning
	Error
)

// Result represents a single validation finding.
type Result struct {
	Level    Level
	Category string
	Message  string
}

// Report holds all validation results and token counts.
type Report struct {
	SkillDir    string
	Results     []Result
	TokenCounts []TokenCount
	Errors      int
	Warnings    int
}

// Validate runs all checks against the skill in the given directory.
func Validate(dir string) *Report {
	report := &Report{SkillDir: dir}

	// Structure checks
	structResults := checkStructure(dir)
	report.Results = append(report.Results, structResults...)

	// Check if SKILL.md was found; if not, skip further checks
	hasSkillMD := false
	for _, r := range structResults {
		if r.Level == Pass && r.Message == "SKILL.md found" {
			hasSkillMD = true
			break
		}
	}
	if !hasSkillMD {
		report.tally()
		return report
	}

	// Parse skill
	s, err := skill.Load(dir)
	if err != nil {
		report.Results = append(report.Results, Result{Level: Error, Category: "Frontmatter", Message: err.Error()})
		report.tally()
		return report
	}

	// Frontmatter checks
	report.Results = append(report.Results, checkFrontmatter(s)...)

	// Link checks
	report.Results = append(report.Results, checkLinks(dir, s.Body)...)

	// Token checks
	tokenResults, tokenCounts := checkTokens(dir, s.Body)
	report.Results = append(report.Results, tokenResults...)
	report.TokenCounts = tokenCounts

	report.tally()
	return report
}

func (r *Report) tally() {
	r.Errors = 0
	r.Warnings = 0
	for _, result := range r.Results {
		switch result.Level {
		case Error:
			r.Errors++
		case Warning:
			r.Warnings++
		}
	}
}
