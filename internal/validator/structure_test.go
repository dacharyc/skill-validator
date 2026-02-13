package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckStructure(t *testing.T) {
	t.Run("missing SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		results := checkStructure(dir)
		requireResult(t, results, Error, "SKILL.md not found")
	})

	t.Run("only SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "---\nname: test\n---\n")
		results := checkStructure(dir)
		requireResult(t, results, Pass, "SKILL.md found")
		requireNoLevel(t, results, Error)
		requireNoLevel(t, results, Warning)
	})

	t.Run("recognized directories", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		os.MkdirAll(filepath.Join(dir, "scripts"), 0755)
		os.MkdirAll(filepath.Join(dir, "references"), 0755)
		os.MkdirAll(filepath.Join(dir, "assets"), 0755)
		results := checkStructure(dir)
		requireResult(t, results, Pass, "SKILL.md found")
		requireNoLevel(t, results, Warning)
	})

	t.Run("unknown directory empty", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		os.MkdirAll(filepath.Join(dir, "extras"), 0755)
		results := checkStructure(dir)
		requireResult(t, results, Warning, "unknown directory: extras/")
	})

	t.Run("unknown directory with files suggests both dirs", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "rules/rule1.md", "rule one")
		writeFile(t, dir, "rules/rule2.md", "rule two")
		writeFile(t, dir, "rules/rule3.md", "rule three")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "unknown directory: rules/ (contains 3 files)")
		requireResultContaining(t, results, Warning, "won't discover these files")
		requireResultContaining(t, results, Warning, "should this be references/ or assets/?")
	})

	t.Run("unknown directory hint omits references when it exists", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		os.MkdirAll(filepath.Join(dir, "references"), 0755)
		writeFile(t, dir, "extras/file.md", "content")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "should this be assets/?")
		requireNoResultContaining(t, results, Warning, "references/")
	})

	t.Run("unknown directory hint omits assets when it exists", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		os.MkdirAll(filepath.Join(dir, "assets"), 0755)
		writeFile(t, dir, "extras/file.md", "content")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "should this be references/?")
		requireNoResultContaining(t, results, Warning, "assets/")
	})

	t.Run("unknown directory hint omitted when both exist", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		os.MkdirAll(filepath.Join(dir, "references"), 0755)
		os.MkdirAll(filepath.Join(dir, "assets"), 0755)
		writeFile(t, dir, "extras/file.md", "content")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "won't discover these files")
		requireNoResultContaining(t, results, Warning, "should this be")
	})

	t.Run("unknown directory with hidden files excluded from count", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "extras/visible.md", "content")
		writeFile(t, dir, "extras/.hidden", "secret")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "unknown directory: extras/ (contains 1 file)")
	})

	t.Run("AGENTS.md has specific warning", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "AGENTS.md", "agent config")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "repo-level agent configuration")
		requireResultContaining(t, results, Warning, "move it outside the skill directory")
	})

	t.Run("known extraneous file README.md", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "README.md", "readme")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "README.md is not needed in a skill")
		requireResultContaining(t, results, Warning, "Anthropic best practices")
	})

	t.Run("known extraneous file CHANGELOG.md", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "CHANGELOG.md", "changes")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "CHANGELOG.md is not needed in a skill")
	})

	t.Run("known extraneous file LICENSE", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "LICENSE", "mit")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "LICENSE is not needed in a skill")
	})

	t.Run("unknown file at root", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "notes.txt", "some notes")
		results := checkStructure(dir)
		requireResultContaining(t, results, Warning, "unexpected file at root: notes.txt")
		requireResultContaining(t, results, Warning, "move it into references/ or assets/")
		requireResultContaining(t, results, Warning, "otherwise remove it")
	})

	t.Run("deep nesting", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		os.MkdirAll(filepath.Join(dir, "references", "subdir"), 0755)
		results := checkStructure(dir)
		requireResult(t, results, Warning, "deep nesting detected: references/subdir/")
	})

	t.Run("hidden files and dirs are skipped", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, ".hidden", "secret")
		os.MkdirAll(filepath.Join(dir, ".git"), 0755)
		results := checkStructure(dir)
		requireResult(t, results, Pass, "SKILL.md found")
		requireNoLevel(t, results, Warning)
	})

	t.Run("hidden dirs inside recognized dirs are skipped", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		os.MkdirAll(filepath.Join(dir, "references", ".hidden"), 0755)
		results := checkStructure(dir)
		requireNoLevel(t, results, Warning)
	})
}
