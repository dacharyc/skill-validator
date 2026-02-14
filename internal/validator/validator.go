package validator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/skill"
)

// Level represents the severity of a validation result.
type Level int

const (
	Pass    Level = iota
	Info
	Warning
	Error
)

// String returns the lowercase name of the level.
func (l Level) String() string {
	switch l {
	case Pass:
		return "pass"
	case Info:
		return "info"
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

// TokenCount holds the token count for a single file.
type TokenCount struct {
	File   string
	Tokens int
}

// Report holds all validation results and token counts.
type Report struct {
	SkillDir             string
	Results              []Result
	TokenCounts          []TokenCount
	OtherTokenCounts     []TokenCount
	ContentReport        *content.Report
	ContaminationReport  *contamination.Report
	Errors               int
	Warnings             int
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

// LoadSkill loads and returns the skill from the given directory.
// This is used by commands that need the parsed skill (e.g., links, content, contamination).
func LoadSkill(dir string) (*skill.Skill, error) {
	return skill.Load(dir)
}

// ReadSkillRaw reads the raw SKILL.md content from a directory without parsing
// frontmatter. This is used as a fallback for content/contamination analysis when
// frontmatter parsing fails.
func ReadSkillRaw(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

// Tally counts errors and warnings in the report.
func (r *Report) Tally() {
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
