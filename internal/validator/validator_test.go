package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Run("valid skill", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\nname: "+dirName(dir)+"\ndescription: A valid skill\n---\n# Body\n")
		report := Validate(dir)
		if report.Errors != 0 {
			t.Errorf("expected 0 errors, got %d", report.Errors)
			for _, r := range report.Results {
				if r.Level == Error {
					t.Logf("  error: %s: %s", r.Category, r.Message)
				}
			}
		}
	})

	t.Run("missing SKILL.md stops early", func(t *testing.T) {
		dir := t.TempDir()
		report := Validate(dir)
		if report.Errors != 1 {
			t.Errorf("expected 1 error, got %d", report.Errors)
		}
		requireResult(t, report.Results, Error, "SKILL.md not found")
		// Should not have any frontmatter/link/token results
		for _, r := range report.Results {
			if r.Category != "Structure" {
				t.Errorf("unexpected category %q when SKILL.md missing", r.Category)
			}
		}
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		dir := t.TempDir()
		// Invalid name, missing description, broken link
		writeSkill(t, dir, "---\nname: BAD\ndescription: \"\"\n---\n[broken](references/nope.md)\n")
		report := Validate(dir)
		if report.Errors < 3 {
			t.Errorf("expected at least 3 errors, got %d", report.Errors)
			for _, r := range report.Results {
				if r.Level == Error {
					t.Logf("  error: %s: %s", r.Category, r.Message)
				}
			}
		}
	})

	t.Run("tally counts errors and warnings", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\nname: "+dirName(dir)+"\ndescription: desc\ncustom: field\n---\n# Body\n")
		writeFile(t, dir, "extras/file.txt", "content")
		report := Validate(dir)
		if report.Warnings < 1 {
			t.Errorf("expected at least 1 warning, got %d", report.Warnings)
		}
	})

	t.Run("token counts populated", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\nname: "+dirName(dir)+"\ndescription: desc\n---\n# Body content\n")
		writeFile(t, dir, "references/ref.md", "Reference text.")
		report := Validate(dir)
		if len(report.TokenCounts) != 2 {
			t.Errorf("expected 2 token counts, got %d", len(report.TokenCounts))
		}
	})

	t.Run("other token counts populated", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\nname: "+dirName(dir)+"\ndescription: desc\n---\n# Body content\n")
		writeFile(t, dir, "AGENTS.md", "Some agent content here.")
		writeFile(t, dir, "rules/rule1.md", "Rule one.")
		report := Validate(dir)
		if len(report.OtherTokenCounts) != 2 {
			t.Errorf("expected 2 other token counts, got %d", len(report.OtherTokenCounts))
			for _, c := range report.OtherTokenCounts {
				t.Logf("  %s: %d tokens", c.File, c.Tokens)
			}
		}
	})

	t.Run("no other token counts for standard structure", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\nname: "+dirName(dir)+"\ndescription: desc\n---\n# Body content\n")
		writeFile(t, dir, "references/ref.md", "Reference text.")
		report := Validate(dir)
		if len(report.OtherTokenCounts) != 0 {
			t.Errorf("expected 0 other token counts, got %d", len(report.OtherTokenCounts))
		}
	})

	t.Run("not structured as a skill error", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\nname: "+dirName(dir)+"\ndescription: desc\n---\n# Body\n")
		// Add a massive amount of non-standard content
		writeFile(t, dir, "AGENTS.md", generateContent(30_000))
		report := Validate(dir)
		requireResultContaining(t, report.Results, Error, "doesn't appear to be structured as a skill")
		requireResultContaining(t, report.Results, Error, "build pipeline issue")
	})

	t.Run("no skill ratio error when other content is small", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\nname: "+dirName(dir)+"\ndescription: desc\n---\n# Body\n")
		writeFile(t, dir, "extra.md", "A small extra file.")
		report := Validate(dir)
		requireNoResultContaining(t, report.Results, Error, "doesn't appear to be structured as a skill")
	})

	t.Run("unparseable frontmatter", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "---\n: invalid: yaml: [broken\n---\nBody\n")
		report := Validate(dir)
		if report.Errors != 1 {
			t.Errorf("expected 1 error, got %d", report.Errors)
		}
		requireResultContaining(t, report.Results, Error, "parsing frontmatter YAML")
	})
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

func TestValidateMulti(t *testing.T) {
	dir := t.TempDir()
	// Create two skills: one valid, one invalid
	goodDir := filepath.Join(dir, "good")
	writeSkill(t, goodDir, "---\nname: good\ndescription: A good skill\n---\n# Body\n")
	badDir := filepath.Join(dir, "bad")
	writeSkill(t, badDir, "---\nname: BAD\ndescription: \"\"\n---\n# Body\n")

	mr := ValidateMulti([]string{goodDir, badDir})

	if len(mr.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(mr.Skills))
	}
	if mr.Skills[0].Errors != 0 {
		t.Errorf("expected good skill to have 0 errors, got %d", mr.Skills[0].Errors)
	}
	if mr.Skills[1].Errors == 0 {
		t.Errorf("expected bad skill to have errors")
	}
	if mr.Errors != mr.Skills[0].Errors+mr.Skills[1].Errors {
		t.Errorf("expected aggregated errors %d, got %d", mr.Skills[0].Errors+mr.Skills[1].Errors, mr.Errors)
	}
	if mr.Warnings != mr.Skills[0].Warnings+mr.Skills[1].Warnings {
		t.Errorf("expected aggregated warnings %d, got %d", mr.Skills[0].Warnings+mr.Skills[1].Warnings, mr.Warnings)
	}
}

func TestValidate_MultiSkillFixture(t *testing.T) {
	// Integration test using testdata/multi-skill
	fixtureDir := "../../testdata/multi-skill"
	mode, dirs := DetectSkills(fixtureDir)
	if mode != MultiSkill {
		t.Fatalf("expected MultiSkill, got %d", mode)
	}
	if len(dirs) != 3 {
		t.Fatalf("expected 3 skill dirs, got %d: %v", len(dirs), dirs)
	}

	mr := ValidateMulti(dirs)
	if len(mr.Skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(mr.Skills))
	}

	// skill-alpha and skill-beta should pass
	for _, r := range mr.Skills {
		base := filepath.Base(r.SkillDir)
		if base == "skill-alpha" || base == "skill-beta" {
			if r.Errors != 0 {
				t.Errorf("%s: expected 0 errors, got %d", base, r.Errors)
				for _, res := range r.Results {
					if res.Level == Error {
						t.Logf("  %s: %s", res.Category, res.Message)
					}
				}
			}
		}
		// skill-gamma should have errors
		if base == "skill-gamma" {
			if r.Errors == 0 {
				t.Errorf("skill-gamma: expected errors, got 0")
			}
		}
	}

	if mr.Errors == 0 {
		t.Error("expected aggregated errors > 0")
	}
}
