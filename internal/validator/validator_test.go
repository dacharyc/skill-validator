package validator

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSkill creates a SKILL.md file in the given directory.
func writeSkill(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectSkills(t *testing.T) {
	t.Run("single skill", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\nname: test\n---\n")
		mode, dirs := DetectSkills(dir)
		if mode != SingleSkill {
			t.Errorf("expected SingleSkill, got %d", mode)
		}
		if len(dirs) != 1 || dirs[0] != dir {
			t.Errorf("expected [%s], got %v", dir, dirs)
		}
	})

	t.Run("multi skill", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, filepath.Join(dir, "alpha"), "---\nname: alpha\n---\n")
		writeSkill(t, filepath.Join(dir, "beta"), "---\nname: beta\n---\n")
		mode, dirs := DetectSkills(dir)
		if mode != MultiSkill {
			t.Errorf("expected MultiSkill, got %d", mode)
		}
		if len(dirs) != 2 {
			t.Fatalf("expected 2 dirs, got %d", len(dirs))
		}
		// os.ReadDir returns sorted entries
		if filepath.Base(dirs[0]) != "alpha" || filepath.Base(dirs[1]) != "beta" {
			t.Errorf("expected [alpha, beta], got [%s, %s]", filepath.Base(dirs[0]), filepath.Base(dirs[1]))
		}
	})

	t.Run("no skills", func(t *testing.T) {
		dir := t.TempDir()
		mode, dirs := DetectSkills(dir)
		if mode != NoSkill {
			t.Errorf("expected NoSkill, got %d", mode)
		}
		if dirs != nil {
			t.Errorf("expected nil dirs, got %v", dirs)
		}
	})

	t.Run("SKILL.md at root takes precedence", func(t *testing.T) {
		dir := t.TempDir()
		// Root has SKILL.md AND subdirs with SKILL.md
		writeSkill(t, dir, "---\nname: root\n---\n")
		writeSkill(t, filepath.Join(dir, "sub"), "---\nname: sub\n---\n")
		mode, dirs := DetectSkills(dir)
		if mode != SingleSkill {
			t.Errorf("expected SingleSkill (root precedence), got %d", mode)
		}
		if len(dirs) != 1 || dirs[0] != dir {
			t.Errorf("expected [%s], got %v", dir, dirs)
		}
	})

	t.Run("skips hidden dirs", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, filepath.Join(dir, ".hidden"), "---\nname: hidden\n---\n")
		writeSkill(t, filepath.Join(dir, "visible"), "---\nname: visible\n---\n")
		mode, dirs := DetectSkills(dir)
		if mode != MultiSkill {
			t.Errorf("expected MultiSkill, got %d", mode)
		}
		if len(dirs) != 1 {
			t.Fatalf("expected 1 dir (hidden skipped), got %d", len(dirs))
		}
		if filepath.Base(dirs[0]) != "visible" {
			t.Errorf("expected visible, got %s", filepath.Base(dirs[0]))
		}
	})

	t.Run("ignores subdirs without SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		// Create a subdir without SKILL.md
		if err := os.MkdirAll(filepath.Join(dir, "no-skill"), 0755); err != nil {
			t.Fatal(err)
		}
		writeSkill(t, filepath.Join(dir, "has-skill"), "---\nname: has-skill\n---\n")
		mode, dirs := DetectSkills(dir)
		if mode != MultiSkill {
			t.Errorf("expected MultiSkill, got %d", mode)
		}
		if len(dirs) != 1 {
			t.Fatalf("expected 1 dir, got %d", len(dirs))
		}
		if filepath.Base(dirs[0]) != "has-skill" {
			t.Errorf("expected has-skill, got %s", filepath.Base(dirs[0]))
		}
	})

	t.Run("follows symlinks", func(t *testing.T) {
		dir := t.TempDir()
		// Create a real skill dir outside
		realDir := filepath.Join(dir, "real")
		writeSkill(t, realDir, "---\nname: real\n---\n")
		// Create a parent with a symlink
		parent := filepath.Join(dir, "parent")
		if err := os.MkdirAll(parent, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realDir, filepath.Join(parent, "linked")); err != nil {
			t.Fatal(err)
		}
		mode, dirs := DetectSkills(parent)
		if mode != MultiSkill {
			t.Errorf("expected MultiSkill, got %d", mode)
		}
		if len(dirs) != 1 {
			t.Fatalf("expected 1 dir, got %d", len(dirs))
		}
	})
}
