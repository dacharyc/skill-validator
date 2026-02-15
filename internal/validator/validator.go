package validator

import (
	"os"
	"path/filepath"
	"sort"
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

// ReadReferencesMarkdownFiles reads all .md files from <dir>/references/ and returns
// a map from filename to content. Returns nil if no references dir or no .md files
// are found.
func ReadReferencesMarkdownFiles(dir string) map[string]string {
	refsDir := filepath.Join(dir, "references")
	entries, err := os.ReadDir(refsDir)
	if err != nil {
		return nil
	}

	files := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(refsDir, entry.Name()))
		if err != nil {
			continue
		}
		files[entry.Name()] = string(data)
	}

	if len(files) == 0 {
		return nil
	}
	return files
}

// AnalyzeReferences runs content and contamination analysis on reference markdown
// files. It populates the aggregate ReferencesContentReport, ReferencesContaminationReport,
// and per-file ReferenceReports on the given report.
func AnalyzeReferences(dir string, rpt *Report) {
	files := ReadReferencesMarkdownFiles(dir)
	if files == nil {
		return
	}

	// Sort filenames for deterministic ordering
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	// Per-file analysis
	var parts []string
	for _, name := range names {
		fileContent := files[name]
		parts = append(parts, fileContent)

		fr := ReferenceFileReport{File: name}
		fr.ContentReport = content.Analyze(fileContent)
		skillName := filepath.Base(dir)
		fr.ContaminationReport = contamination.Analyze(skillName, fileContent, fr.ContentReport.CodeLanguages)
		rpt.ReferenceReports = append(rpt.ReferenceReports, fr)
	}

	// Aggregate analysis on concatenated content
	concatenated := strings.Join(parts, "\n")
	rpt.ReferencesContentReport = content.Analyze(concatenated)
	skillName := filepath.Base(dir)
	rpt.ReferencesContaminationReport = contamination.Analyze(skillName, concatenated, rpt.ReferencesContentReport.CodeLanguages)
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
