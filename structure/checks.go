package structure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agent-ecosystem/skill-validator/types"
	"github.com/agent-ecosystem/skill-validator/util"
)

// recognizedDirs lists the directory names defined by the skill spec.
var recognizedDirs = map[string]bool{
	"scripts":    true,
	"references": true,
	"assets":     true,
}

// rootFileCategory classifies root-level files in a skill directory.
type rootFileCategory int

const (
	categoryExtraneous rootFileCategory = iota
	categoryReference
	categoryScript
	categoryAsset
)

// scriptExtensions lists file extensions that are script-equivalent.
var scriptExtensions = map[string]bool{
	".py": true, ".sh": true, ".bash": true, ".js": true, ".ts": true,
	".jsx": true, ".tsx": true, ".rb": true, ".go": true, ".rs": true,
	".pl": true, ".lua": true, ".php": true, ".r": true,
}

// classifyRootFile determines the category of a non-SKILL.md root file.
func classifyRootFile(name string) rootFileCategory {
	if util.IsExtraneousFile(name) {
		return categoryExtraneous
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".md" {
		return categoryReference
	}
	if scriptExtensions[ext] {
		return categoryScript
	}
	return categoryAsset
}

// CheckStructure validates the directory layout of a skill package. It checks
// for the required SKILL.md file, flags unrecognized directories and extraneous
// root files, and warns about deep nesting in recognized directories.
func CheckStructure(dir string) []types.Result {
	ctx := types.ResultContext{Category: "Structure"}
	var results []types.Result

	// Check SKILL.md exists
	skillPath := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		results = append(results, ctx.ErrorFile("SKILL.md", "SKILL.md not found"))
		return results
	}
	results = append(results, ctx.PassFile("SKILL.md", "SKILL.md found"))

	// Check directories
	entries, err := os.ReadDir(dir)
	if err != nil {
		results = append(results, ctx.Errorf("reading directory: %v", err))
		return results
	}

	supportFileCount := 0
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip hidden files/dirs
		}
		if !entry.IsDir() {
			if name != "SKILL.md" {
				cat := classifyRootFile(name)
				if cat == categoryExtraneous {
					results = append(results, extraneousFileResult(ctx, name))
				} else {
					supportFileCount++
				}
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
						name, fileCount, util.PluralS(fileCount), hint,
					)
				}
			}
			results = append(results, ctx.Warn(msg))
		}
	}

	if supportFileCount >= 5 {
		results = append(results, ctx.Infof(
			"%d support files at root — consider organizing them into references/, scripts/, and assets/ subdirectories",
			supportFileCount,
		))
	}

	// Check for deep nesting in recognized directories
	for dirName := range recognizedDirs {
		subdir := filepath.Join(dir, dirName)
		if _, err := os.Stat(subdir); os.IsNotExist(err) {
			continue
		}
		err := checkNesting(ctx, subdir, dirName)
		if err != nil {
			results = append(results, err...)
		}
	}

	return results
}

func extraneousFileResult(ctx types.ResultContext, name string) types.Result {
	lower := strings.ToLower(name)
	if lower == "agents.md" {
		return ctx.WarnFile(name, fmt.Sprintf(
			"%s is for repo-level agent configuration, not skill content — "+
				"move it outside the skill directory (e.g. to the repository root) "+
				"where agents discover it automatically",
			name,
		))
	}
	return ctx.WarnFile(name, fmt.Sprintf(
		"%s is not needed in a skill — agents may load it into their context window, "+
			"taking space from your actual task (Anthropic best practices: skills should only "+
			"contain files that directly support agent functionality)",
		name,
	))
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

func checkNesting(ctx types.ResultContext, dir, prefix string) []types.Result {
	var results []types.Result
	entries, err := os.ReadDir(dir)
	if err != nil {
		return results
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			results = append(results, ctx.Warnf("deep nesting detected: %s/%s/", prefix, entry.Name()))
		}
	}
	return results
}
