package structure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dacharyc/skill-validator/internal/validator"
)

var recognizedDirs = map[string]bool{
	"scripts":    true,
	"references": true,
	"assets":     true,
}

// Files commonly found in repos but not intended for agent consumption.
// Per Anthropic best practices: "A skill should only contain essential files
// that directly support its functionality."
// See: github.com/anthropics/skills → skill-creator
var knownExtraneousFiles = map[string]string{
	"readme.md":             "README.md",
	"readme":                "README",
	"changelog.md":          "CHANGELOG.md",
	"changelog":             "CHANGELOG",
	"license":               "LICENSE",
	"license.md":            "LICENSE.md",
	"license.txt":           "LICENSE.txt",
	"contributing.md":       "CONTRIBUTING.md",
	"code_of_conduct.md":    "CODE_OF_CONDUCT.md",
	"installation_guide.md": "INSTALLATION_GUIDE.md",
	"quick_reference.md":    "QUICK_REFERENCE.md",
	"makefile":              "Makefile",
	".gitignore":            ".gitignore",
}

func CheckStructure(dir string) []validator.Result {
	var results []validator.Result

	// Check SKILL.md exists
	skillPath := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		results = append(results, validator.Result{Level: validator.Error, Category: "Structure", Message: "SKILL.md not found"})
		return results
	}
	results = append(results, validator.Result{Level: validator.Pass, Category: "Structure", Message: "SKILL.md found"})

	// Check directories
	entries, err := os.ReadDir(dir)
	if err != nil {
		results = append(results, validator.Result{Level: validator.Error, Category: "Structure", Message: fmt.Sprintf("reading directory: %v", err)})
		return results
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip hidden files/dirs
		}
		if !entry.IsDir() {
			if name != "SKILL.md" {
				results = append(results, extraneousFileResult(name))
			}
			continue
		}
		if !recognizedDirs[name] {
			msg := fmt.Sprintf("unknown directory: %s/", name)
			if subEntries, err := os.ReadDir(filepath.Join(dir, name)); err == nil {
				fileCount := 0
				for _, se := range subEntries {
					if !strings.HasPrefix(se.Name(), ".") {
						fileCount++
					}
				}
				if fileCount > 0 {
					hint := unknownDirHint(dir)
					msg = fmt.Sprintf(
						"unknown directory: %s/ (contains %d file%s) — agents using the standard skill structure won't discover these files%s",
						name, fileCount, pluralS(fileCount), hint,
					)
				}
			}
			results = append(results, validator.Result{Level: validator.Warning, Category: "Structure", Message: msg})
		}
	}

	// Check for deep nesting in recognized directories
	for dirName := range recognizedDirs {
		subdir := filepath.Join(dir, dirName)
		if _, err := os.Stat(subdir); os.IsNotExist(err) {
			continue
		}
		err := checkNesting(subdir, dirName)
		if err != nil {
			results = append(results, err...)
		}
	}

	return results
}

func extraneousFileResult(name string) validator.Result {
	lower := strings.ToLower(name)
	if lower == "agents.md" {
		return validator.Result{
			Level:    validator.Warning,
			Category: "Structure",
			Message: fmt.Sprintf(
				"%s is for repo-level agent configuration, not skill content — "+
					"move it outside the skill directory (e.g. to the repository root) "+
					"where agents discover it automatically",
				name,
			),
		}
	}
	if _, known := knownExtraneousFiles[lower]; known {
		return validator.Result{
			Level:    validator.Warning,
			Category: "Structure",
			Message: fmt.Sprintf(
				"%s is not needed in a skill — agents may load it into their context window, "+
					"taking space from your actual task (Anthropic best practices: skills should only "+
					"contain files that directly support agent functionality)",
				name,
			),
		}
	}
	return validator.Result{
		Level:    validator.Warning,
		Category: "Structure",
		Message: fmt.Sprintf(
			"unexpected file at root: %s — if agents need this file, move it into "+
				"references/ or assets/ as appropriate; otherwise remove it to avoid "+
				"unnecessary context window usage",
			name,
		),
	}
}

func unknownDirHint(dir string) string {
	var candidates []string
	if _, err := os.Stat(filepath.Join(dir, "references")); os.IsNotExist(err) {
		candidates = append(candidates, "references/")
	}
	if _, err := os.Stat(filepath.Join(dir, "assets")); os.IsNotExist(err) {
		candidates = append(candidates, "assets/")
	}
	if len(candidates) == 0 {
		return ""
	}
	return fmt.Sprintf("; should this be %s?", strings.Join(candidates, " or "))
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func checkNesting(dir, prefix string) []validator.Result {
	var results []validator.Result
	entries, err := os.ReadDir(dir)
	if err != nil {
		return results
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			results = append(results, validator.Result{
				Level:    validator.Warning,
				Category: "Structure",
				Message:  fmt.Sprintf("deep nesting detected: %s/%s/", prefix, entry.Name()),
			})
		}
	}
	return results
}
