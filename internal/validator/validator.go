package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dacharyc/skill-validator/internal/skill"
)

// Level represents the severity of a validation result.
type Level int

const (
	Pass    Level = iota
	Warning
	Error
)

// String returns the lowercase name of the level.
func (l Level) String() string {
	switch l {
	case Pass:
		return "pass"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// Result represents a single validation finding.
type Result struct {
	Level    Level
	Category string
	Message  string
}

// Report holds all validation results and token counts.
type Report struct {
	SkillDir         string
	Results          []Result
	TokenCounts      []TokenCount
	OtherTokenCounts []TokenCount
	Errors           int
	Warnings         int
}

// SkillMode indicates what kind of skill directory was detected.
type SkillMode int

const (
	NoSkill    SkillMode = iota
	SingleSkill
	MultiSkill
)

// DetectSkills determines whether dir is a single skill, a multi-skill
// parent, or contains no skills. It follows symlinks when checking
// subdirectories.
func DetectSkills(dir string) (SkillMode, []string) {
	// If the directory itself contains SKILL.md, it's a single skill.
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
		return SingleSkill, []string{dir}
	}

	// Scan immediate subdirectories for SKILL.md.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return NoSkill, nil
	}

	var skillDirs []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		subdir := filepath.Join(dir, name)
		// Use os.Stat (not entry.IsDir()) to follow symlinks.
		info, err := os.Stat(subdir)
		if err != nil || !info.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(subdir, "SKILL.md")); err == nil {
			skillDirs = append(skillDirs, subdir)
		}
	}

	if len(skillDirs) > 0 {
		return MultiSkill, skillDirs
	}
	return NoSkill, nil
}

// MultiReport holds aggregated results from validating multiple skills.
type MultiReport struct {
	Skills   []*Report
	Errors   int
	Warnings int
}

// ValidateMulti validates each directory and returns an aggregated report.
func ValidateMulti(dirs []string) *MultiReport {
	mr := &MultiReport{}
	for _, dir := range dirs {
		r := Validate(dir)
		mr.Skills = append(mr.Skills, r)
		mr.Errors += r.Errors
		mr.Warnings += r.Warnings
	}
	return mr
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

	// Markdown checks
	report.Results = append(report.Results, checkMarkdown(dir, s.Body)...)

	// Token checks
	tokenResults, tokenCounts, otherCounts := checkTokens(dir, s.Body)
	report.Results = append(report.Results, tokenResults...)
	report.TokenCounts = tokenCounts
	report.OtherTokenCounts = otherCounts

	// Holistic structure check: is this actually a skill?
	report.Results = append(report.Results, checkSkillRatio(report.TokenCounts, report.OtherTokenCounts)...)

	report.tally()
	return report
}

func checkSkillRatio(standard []TokenCount, other []TokenCount) []Result {
	standardTotal := 0
	for _, tc := range standard {
		standardTotal += tc.Tokens
	}
	otherTotal := 0
	for _, tc := range other {
		otherTotal += tc.Tokens
	}

	if otherTotal > 25_000 && standardTotal > 0 && otherTotal > standardTotal*10 {
		return []Result{{
			Level:    Error,
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
