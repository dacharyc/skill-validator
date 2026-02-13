package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func checkStructure(dir string) []Result {
	var results []Result

	// Check SKILL.md exists
	skillPath := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		results = append(results, Result{Level: Error, Category: "Structure", Message: "SKILL.md not found"})
		return results
	}
	results = append(results, Result{Level: Pass, Category: "Structure", Message: "SKILL.md found"})

	// Check directories
	entries, err := os.ReadDir(dir)
	if err != nil {
		results = append(results, Result{Level: Error, Category: "Structure", Message: fmt.Sprintf("reading directory: %v", err)})
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
			results = append(results, Result{Level: Warning, Category: "Structure", Message: fmt.Sprintf("unknown directory: %s/", name)})
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

func extraneousFileResult(name string) Result {
	lower := strings.ToLower(name)
	if _, known := knownExtraneousFiles[lower]; known {
		return Result{
			Level:    Warning,
			Category: "Structure",
			Message: fmt.Sprintf(
				"%s is not needed in a skill — agents may load it into their context window, "+
					"taking space from your actual task (Anthropic best practices: skills should only "+
					"contain files that directly support agent functionality)",
				name,
			),
		}
	}
	return Result{
		Level:    Warning,
		Category: "Structure",
		Message: fmt.Sprintf(
			"unexpected file at root: %s — if agents don't need this file, "+
				"remove it to avoid unnecessary context window usage",
			name,
		),
	}
}

func checkNesting(dir string, prefix string) []Result {
	var results []Result
	entries, err := os.ReadDir(dir)
	if err != nil {
		return results
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			results = append(results, Result{
				Level:    Warning,
				Category: "Structure",
				Message:  fmt.Sprintf("deep nesting detected: %s/%s/", prefix, entry.Name()),
			})
		}
	}
	return results
}
