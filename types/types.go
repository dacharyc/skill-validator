// Package types defines the core data types used throughout the
// skill-validator: validation results, severity levels, token counts,
// skill modes, and aggregated reports.
package types

import (
	"github.com/dacharyc/skill-validator/contamination"
	"github.com/dacharyc/skill-validator/content"
)

// Level represents the severity of a validation result.
type Level int

const (
	Pass Level = iota
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
	File     string // path relative to skill dir, e.g. "SKILL.md", "references/guide.md"
	Line     int    // 0 = no line info
}

// TokenCount holds the token count for a single file.
type TokenCount struct {
	File   string
	Tokens int
}

// ReferenceFileReport holds per-file content and contamination analysis for a single reference file.
type ReferenceFileReport struct {
	File                string
	ContentReport       *content.Report
	ContaminationReport *contamination.Report
}

// Report holds all validation results and token counts.
type Report struct {
	SkillDir                      string
	Results                       []Result
	TokenCounts                   []TokenCount
	OtherTokenCounts              []TokenCount
	ContentReport                 *content.Report
	ReferencesContentReport       *content.Report
	ContaminationReport           *contamination.Report
	ReferencesContaminationReport *contamination.Report
	ReferenceReports              []ReferenceFileReport
	Errors                        int
	Warnings                      int
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

// SkillMode indicates what kind of skill directory was detected.
type SkillMode int

const (
	NoSkill SkillMode = iota
	SingleSkill
	MultiSkill
)

// MultiReport holds aggregated results from validating multiple skills.
type MultiReport struct {
	Skills   []*Report
	Errors   int
	Warnings int
}
