package report

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/dacharyc/skill-validator/internal/validator"
)

// PrintAnnotations writes GitHub Actions workflow command annotations for
// errors and warnings in the report. Pass and Info results are skipped.
// workDir is the working directory used to compute relative file paths;
// in CI this is typically the repository root.
func PrintAnnotations(w io.Writer, r *validator.Report, workDir string) {
	for _, res := range r.Results {
		line := formatAnnotation(r.SkillDir, res, workDir)
		if line != "" {
			_, _ = fmt.Fprintln(w, line)
		}
	}
}

// PrintMultiAnnotations writes annotations for all skills in a multi-report.
func PrintMultiAnnotations(w io.Writer, mr *validator.MultiReport, workDir string) {
	for _, r := range mr.Skills {
		PrintAnnotations(w, r, workDir)
	}
}

func formatAnnotation(skillDir string, res validator.Result, workDir string) string {
	var cmd string
	switch res.Level {
	case validator.Error:
		cmd = "error"
	case validator.Warning:
		cmd = "warning"
	default:
		return ""
	}

	// Build the parameters string
	var params string
	if res.File != "" {
		// Compose path relative to the working directory so GitHub Actions
		// can map annotations to files in the PR diff view.
		absPath := filepath.Join(skillDir, res.File)
		relPath, err := filepath.Rel(workDir, absPath)
		if err != nil {
			relPath = absPath // fall back to absolute if Rel fails
		}
		params = fmt.Sprintf(" file=%s", relPath)
		if res.Line > 0 {
			params += fmt.Sprintf(",line=%d", res.Line)
		}
		params += fmt.Sprintf(",title=%s", res.Category)
	} else {
		params = fmt.Sprintf(" title=%s", res.Category)
	}

	return fmt.Sprintf("::%s%s::%s", cmd, params, res.Message)
}
