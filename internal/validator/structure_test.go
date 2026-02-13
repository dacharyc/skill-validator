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

	t.Run("unknown directory", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		os.MkdirAll(filepath.Join(dir, "extras"), 0755)
		results := checkStructure(dir)
		requireResult(t, results, Warning, "unknown directory: extras/")
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
		requireResultContaining(t, results, Warning, "avoid unnecessary context window usage")
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
