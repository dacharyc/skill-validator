package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{Pass, "pass"},
		{Info, "info"},
		{Warning, "warning"},
		{Error, "error"},
		{Level(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestTally(t *testing.T) {
	r := &Report{
		Results: []Result{
			{Level: Pass, Category: "A", Message: "ok"},
			{Level: Error, Category: "B", Message: "bad"},
			{Level: Warning, Category: "C", Message: "meh"},
			{Level: Error, Category: "D", Message: "also bad"},
			{Level: Info, Category: "E", Message: "fyi"},
		},
	}
	r.Tally()
	if r.Errors != 2 {
		t.Errorf("Errors = %d, want 2", r.Errors)
	}
	if r.Warnings != 1 {
		t.Errorf("Warnings = %d, want 1", r.Warnings)
	}
}

func TestTally_Empty(t *testing.T) {
	r := &Report{Errors: 5, Warnings: 3}
	r.Tally()
	if r.Errors != 0 {
		t.Errorf("Errors = %d, want 0", r.Errors)
	}
	if r.Warnings != 0 {
		t.Errorf("Warnings = %d, want 0", r.Warnings)
	}
}

func TestLoadSkill(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "---\nname: test-skill\ndescription: A test\n---\n# Hello\n")

	s, err := LoadSkill(dir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}
	if s.Frontmatter.Name != "test-skill" {
		t.Errorf("Name = %q, want test-skill", s.Frontmatter.Name)
	}
}

func TestLoadSkill_Missing(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadSkill(dir)
	if err == nil {
		t.Error("expected error for missing SKILL.md")
	}
}

func TestReadSkillRaw(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: raw\n---\n# Raw Skill\nSome content.\n"
	writeSkill(t, dir, content)

	got := ReadSkillRaw(dir)
	if got != content {
		t.Errorf("ReadSkillRaw = %q, want %q", got, content)
	}
}

func TestReadSkillRaw_Missing(t *testing.T) {
	dir := t.TempDir()
	got := ReadSkillRaw(dir)
	if got != "" {
		t.Errorf("ReadSkillRaw = %q, want empty", got)
	}
}

func TestReadReferencesMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	refsDir := filepath.Join(dir, "references")
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "guide.md"), []byte("# Guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "notes.md"), []byte("# Notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "data.json"), []byte(`{"skip": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(refsDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	files := ReadReferencesMarkdownFiles(dir)
	if files == nil {
		t.Fatal("expected non-nil map")
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
	if files["guide.md"] != "# Guide" {
		t.Errorf("guide.md content = %q", files["guide.md"])
	}
	if files["notes.md"] != "# Notes" {
		t.Errorf("notes.md content = %q", files["notes.md"])
	}
}

func TestReadReferencesMarkdownFiles_NoDir(t *testing.T) {
	dir := t.TempDir()
	files := ReadReferencesMarkdownFiles(dir)
	if files != nil {
		t.Errorf("expected nil, got %v", files)
	}
}

func TestReadReferencesMarkdownFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := ReadReferencesMarkdownFiles(dir)
	if files != nil {
		t.Errorf("expected nil for empty references dir, got %v", files)
	}
}

func TestReadReferencesMarkdownFiles_OnlyNonMd(t *testing.T) {
	dir := t.TempDir()
	refsDir := filepath.Join(dir, "references")
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "data.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := ReadReferencesMarkdownFiles(dir)
	if files != nil {
		t.Errorf("expected nil when only non-.md files, got %v", files)
	}
}

func TestAnalyzeReferences_WithFiles(t *testing.T) {
	dir := t.TempDir()
	refsDir := filepath.Join(dir, "references")
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "alpha.md"), []byte("# Alpha\nUse this tool."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "beta.md"), []byte("# Beta\nAnother reference."), 0o644); err != nil {
		t.Fatal(err)
	}

	rpt := &Report{SkillDir: dir}
	AnalyzeReferences(dir, rpt)

	if rpt.ReferencesContentReport == nil {
		t.Fatal("expected aggregate ReferencesContentReport")
	}
	if rpt.ReferencesContaminationReport == nil {
		t.Fatal("expected aggregate ReferencesContaminationReport")
	}
	if len(rpt.ReferenceReports) != 2 {
		t.Fatalf("expected 2 per-file reports, got %d", len(rpt.ReferenceReports))
	}
	// Sorted alphabetically
	if rpt.ReferenceReports[0].File != "alpha.md" {
		t.Errorf("first file = %q, want alpha.md", rpt.ReferenceReports[0].File)
	}
	if rpt.ReferenceReports[1].File != "beta.md" {
		t.Errorf("second file = %q, want beta.md", rpt.ReferenceReports[1].File)
	}
	for _, fr := range rpt.ReferenceReports {
		if fr.ContentReport == nil {
			t.Errorf("%s: expected ContentReport", fr.File)
		}
		if fr.ContaminationReport == nil {
			t.Errorf("%s: expected ContaminationReport", fr.File)
		}
	}
}

func TestAnalyzeReferences_NoFiles(t *testing.T) {
	dir := t.TempDir()
	rpt := &Report{SkillDir: dir}
	AnalyzeReferences(dir, rpt)

	if rpt.ReferencesContentReport != nil {
		t.Error("expected nil ReferencesContentReport")
	}
	if rpt.ReferencesContaminationReport != nil {
		t.Error("expected nil ReferencesContaminationReport")
	}
	if len(rpt.ReferenceReports) != 0 {
		t.Error("expected no ReferenceReports")
	}
}

// writeSkill creates a SKILL.md file in the given directory.
func writeSkill(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
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
		if err := os.MkdirAll(filepath.Join(dir, "no-skill"), 0o755); err != nil {
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
		if err := os.MkdirAll(parent, 0o755); err != nil {
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
