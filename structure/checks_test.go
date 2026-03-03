package structure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dacharyc/skill-validator/skillcheck"
)

func TestCheckStructure(t *testing.T) {
	t.Run("missing SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		results := CheckStructure(dir)
		requireResult(t, results, skillcheck.Error, "SKILL.md not found")
	})

	t.Run("only SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "---\nname: test\n---\n")
		results := CheckStructure(dir)
		requireResult(t, results, skillcheck.Pass, "SKILL.md found")
		requireNoLevel(t, results, skillcheck.Error)
		requireNoLevel(t, results, skillcheck.Warning)
	})

	t.Run("recognized directories", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
			t.Fatal(err)
		}
		results := CheckStructure(dir)
		requireResult(t, results, skillcheck.Pass, "SKILL.md found")
		requireNoLevel(t, results, skillcheck.Warning)
	})

	t.Run("unknown directory empty", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		if err := os.MkdirAll(filepath.Join(dir, "extras"), 0o755); err != nil {
			t.Fatal(err)
		}
		results := CheckStructure(dir)
		requireResult(t, results, skillcheck.Warning, "unknown directory: extras/")
	})

	t.Run("unknown directory with files suggests both dirs", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "rules/rule1.md", "rule one")
		writeFile(t, dir, "rules/rule2.md", "rule two")
		writeFile(t, dir, "rules/rule3.md", "rule three")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "unknown directory: rules/ (contains 3 files)")
		requireResultContaining(t, results, skillcheck.Warning, "won't discover these files")
		requireResultContaining(t, results, skillcheck.Warning, "should this be references/ or assets/?")
	})

	t.Run("unknown directory hint omits references when it exists", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "extras/file.md", "content")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "should this be assets/?")
		requireNoResultContaining(t, results, skillcheck.Warning, "references/")
	})

	t.Run("unknown directory hint omits assets when it exists", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "extras/file.md", "content")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "should this be references/?")
		requireNoResultContaining(t, results, skillcheck.Warning, "assets/")
	})

	t.Run("unknown directory hint omitted when both exist", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, "extras/file.md", "content")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "won't discover these files")
		requireNoResultContaining(t, results, skillcheck.Warning, "should this be")
	})

	t.Run("unknown directory with hidden files excluded from count", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "extras/visible.md", "content")
		writeFile(t, dir, "extras/.hidden", "secret")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "unknown directory: extras/ (contains 1 file)")
	})

	t.Run("AGENTS.md has specific warning", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "AGENTS.md", "agent config")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "repo-level agent configuration")
		requireResultContaining(t, results, skillcheck.Warning, "move it outside the skill directory")
	})

	t.Run("known extraneous file README.md", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "README.md", "readme")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "README.md is not needed in a skill")
		requireResultContaining(t, results, skillcheck.Warning, "Anthropic best practices")
	})

	t.Run("known extraneous file CHANGELOG.md", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "CHANGELOG.md", "changes")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "CHANGELOG.md is not needed in a skill")
	})

	t.Run("known extraneous file LICENSE", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "LICENSE", "mit")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "LICENSE is not needed in a skill")
	})

	t.Run("unknown file at root", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "notes.txt", "some notes")
		results := CheckStructure(dir)
		requireResultContaining(t, results, skillcheck.Warning, "unexpected file at root: notes.txt")
		requireResultContaining(t, results, skillcheck.Warning, "move it into references/ or assets/")
		requireResultContaining(t, results, skillcheck.Warning, "otherwise remove it")
	})

	t.Run("deep nesting", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		if err := os.MkdirAll(filepath.Join(dir, "references", "subdir"), 0o755); err != nil {
			t.Fatal(err)
		}
		results := CheckStructure(dir)
		requireResult(t, results, skillcheck.Warning, "deep nesting detected: references/subdir/")
	})

	t.Run("hidden files and dirs are skipped", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, ".hidden", "secret")
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		results := CheckStructure(dir)
		requireResult(t, results, skillcheck.Pass, "SKILL.md found")
		requireNoLevel(t, results, skillcheck.Warning)
	})

	t.Run("hidden dirs inside recognized dirs are skipped", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		if err := os.MkdirAll(filepath.Join(dir, "references", ".hidden"), 0o755); err != nil {
			t.Fatal(err)
		}
		results := CheckStructure(dir)
		requireNoLevel(t, results, skillcheck.Warning)
	})
}
