package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agent-ecosystem/skill-validator/contamination"
	"github.com/agent-ecosystem/skill-validator/content"
	"github.com/agent-ecosystem/skill-validator/links"
	"github.com/agent-ecosystem/skill-validator/orchestrate"
	"github.com/agent-ecosystem/skill-validator/skill"
	"github.com/agent-ecosystem/skill-validator/skillcheck"
	"github.com/agent-ecosystem/skill-validator/structure"
	"github.com/agent-ecosystem/skill-validator/types"
)

// fixtureDir returns the absolute path to a testdata fixture.
func fixtureDir(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("fixture %q not found: %v", name, err)
	}
	return dir
}

func TestValidateCommand_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	r := structure.Validate(dir, structure.Options{})
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
		for _, res := range r.Results {
			if res.Level == types.Error {
				t.Logf("  error: %s: %s", res.Category, res.Message)
			}
		}
	}

	// Validate should check structure and frontmatter
	hasStructure := false
	hasFrontmatter := false
	hasMarkdown := false
	for _, res := range r.Results {
		if res.Category == "Structure" {
			hasStructure = true
		}
		if res.Category == "Frontmatter" {
			hasFrontmatter = true
		}
		if res.Category == "Markdown" {
			hasMarkdown = true
		}
	}
	if !hasStructure {
		t.Error("expected Structure results from validate")
	}
	if !hasFrontmatter {
		t.Error("expected Frontmatter results from validate")
	}
	if !hasMarkdown {
		t.Error("expected Markdown results from validate (code fence checks)")
	}

	// Validate should NOT include Links results (those are in validate links)
	for _, res := range r.Results {
		if res.Category == "Links" {
			t.Error("validate should not include Links results (moved to validate links)")
		}
	}

	// Should have token counts
	if len(r.TokenCounts) == 0 {
		t.Error("expected token counts from validate")
	}
}

func TestValidateCommand_InvalidSkill(t *testing.T) {
	dir := fixtureDir(t, "invalid-skill")

	r := structure.Validate(dir, structure.Options{})
	if r.Errors == 0 {
		t.Error("expected errors for invalid skill")
	}
}

func TestValidateCommand_MultiSkill(t *testing.T) {
	dir := fixtureDir(t, "multi-skill")

	mode, dirs := skillcheck.DetectSkills(dir)
	if mode != types.MultiSkill {
		t.Fatalf("expected MultiSkill, got %d", mode)
	}

	mr := structure.ValidateMulti(dirs, structure.Options{})
	if len(mr.Skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(mr.Skills))
	}
}

func TestValidateLinks_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	// External link checks: valid-skill has no HTTP links, so no results
	linkResults := links.CheckLinks(t.Context(), dir, s.Body)
	if linkResults != nil {
		t.Errorf("expected nil for skill with no HTTP links, got %d results", len(linkResults))
	}

	// Internal links are now checked by structure validation
	r := structure.Validate(dir, structure.Options{})
	foundLink := false
	for _, res := range r.Results {
		if res.Level == types.Pass && strings.Contains(res.Message, "references/guide.md") {
			foundLink = true
		}
	}
	if !foundLink {
		t.Error("expected passing internal link check for references/guide.md in structure results")
	}
}

func TestValidateLinks_InvalidSkill(t *testing.T) {
	dir := fixtureDir(t, "invalid-skill")

	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	// External link checks: invalid-skill has an HTTP link
	linkResults := links.CheckLinks(t.Context(), dir, s.Body)
	if len(linkResults) == 0 {
		t.Error("expected at least one external link check result")
	}

	// Internal links are now checked by structure validation
	r := structure.Validate(dir, structure.Options{})
	foundBroken := false
	for _, res := range r.Results {
		if res.Level == types.Error && strings.Contains(res.Message, "missing.md") {
			foundBroken = true
		}
	}
	if !foundBroken {
		t.Error("expected broken internal link error for references/missing.md in structure results")
	}
}

func TestAnalyzeContent_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	cr := content.Analyze(s.RawContent)

	if cr.WordCount == 0 {
		t.Error("expected non-zero word count")
	}
	if cr.SectionCount != 2 {
		t.Errorf("expected 2 sections (## Usage, ## Notes), got %d", cr.SectionCount)
	}
}

func TestAnalyzeContent_RichSkill(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")

	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	cr := content.Analyze(s.RawContent)

	if cr.WordCount == 0 {
		t.Error("expected non-zero word count")
	}
	if cr.CodeBlockCount != 4 {
		t.Errorf("expected 4 code blocks, got %d", cr.CodeBlockCount)
	}
	if cr.CodeBlockRatio <= 0 {
		t.Error("expected positive code block ratio")
	}
	if len(cr.CodeLanguages) != 4 {
		t.Errorf("expected 4 code languages (bash, javascript, python, yaml), got %d: %v",
			len(cr.CodeLanguages), cr.CodeLanguages)
	}
	if cr.ImperativeCount == 0 {
		t.Error("expected imperative sentences")
	}
	if cr.StrongMarkers < 3 {
		t.Errorf("expected at least 3 strong markers (must, always, never), got %d", cr.StrongMarkers)
	}
	if cr.WeakMarkers < 2 {
		t.Errorf("expected at least 2 weak markers (may, consider, could, optional, suggested), got %d", cr.WeakMarkers)
	}
	if cr.InstructionSpecificity <= 0 {
		t.Error("expected positive instruction specificity")
	}
	if cr.ListItemCount != 4 {
		t.Errorf("expected 4 list items, got %d", cr.ListItemCount)
	}
	if cr.SectionCount < 3 {
		t.Errorf("expected at least 3 sections, got %d", cr.SectionCount)
	}
}

func TestAnalyzeContamination_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	cr := content.Analyze(s.RawContent)
	rr := contamination.Analyze(filepath.Base(dir), s.RawContent, cr.CodeLanguages)

	if rr.ContaminationLevel != "low" {
		t.Errorf("expected low contamination for valid-skill, got %s", rr.ContaminationLevel)
	}
}

func TestAnalyzeContamination_RichSkill(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")

	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	cr := content.Analyze(s.RawContent)
	rr := contamination.Analyze(filepath.Base(dir), s.RawContent, cr.CodeLanguages)

	// rich-skill mentions mongodb, has bash+javascript+python+yaml code blocks
	if len(rr.MultiInterfaceTools) == 0 {
		t.Error("expected multi-interface tool detection (mongodb)")
	}
	if !rr.LanguageMismatch {
		t.Error("expected language mismatch (multiple language categories)")
	}
	if rr.ContaminationScore <= 0 {
		t.Error("expected positive contamination score")
	}
	if rr.ContaminationLevel == "low" {
		t.Errorf("expected medium or high contamination for rich-skill, got low (score=%f)", rr.ContaminationScore)
	}
	if rr.ScopeBreadth < 3 {
		t.Errorf("expected scope breadth >= 3, got %d", rr.ScopeBreadth)
	}
}

func TestReadSkillRaw(t *testing.T) {
	dir := fixtureDir(t, "broken-frontmatter")

	raw := skillcheck.ReadSkillRaw(dir)
	if raw == "" {
		t.Fatal("expected non-empty raw content")
	}
	if !strings.Contains(raw, "# Broken Frontmatter Skill") {
		t.Error("expected raw content to contain skill heading")
	}
	if !strings.Contains(raw, "npm install express") {
		t.Error("expected raw content to contain code block content")
	}
}

func TestReadReferencesMarkdownFiles_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	files := skillcheck.ReadReferencesMarkdownFiles(dir)
	if files == nil {
		t.Fatal("expected non-nil map for valid-skill with references")
	}
	if len(files) == 0 {
		t.Fatal("expected at least one reference file")
	}
	guideContent, ok := files["guide.md"]
	if !ok {
		t.Fatal("expected guide.md in reference files map")
	}
	if !strings.Contains(guideContent, "Reference Guide") {
		t.Error("expected guide.md content to contain 'Reference Guide'")
	}
}

func TestReadReferencesMarkdownFiles_NoReferences(t *testing.T) {
	dir := t.TempDir()
	files := skillcheck.ReadReferencesMarkdownFiles(dir)
	if files != nil {
		t.Errorf("expected nil for dir without references, got %d files", len(files))
	}
}

func TestReadSkillRaw_MissingFile(t *testing.T) {
	dir := t.TempDir()

	raw := skillcheck.ReadSkillRaw(dir)
	if raw != "" {
		t.Errorf("expected empty string for missing SKILL.md, got %d bytes", len(raw))
	}
}

func TestResolveCheckGroups(t *testing.T) {
	t.Run("default all enabled", func(t *testing.T) {
		enabled, err := resolveCheckGroups("", "")
		if err != nil {
			t.Fatal(err)
		}
		for _, g := range []orchestrate.CheckGroup{
			orchestrate.GroupStructure, orchestrate.GroupLinks,
			orchestrate.GroupContent, orchestrate.GroupContamination,
		} {
			if !enabled[g] {
				t.Errorf("expected %s enabled by default", g)
			}
		}
	})

	t.Run("only structure,links", func(t *testing.T) {
		enabled, err := resolveCheckGroups("structure,links", "")
		if err != nil {
			t.Fatal(err)
		}
		if !enabled[orchestrate.GroupStructure] || !enabled[orchestrate.GroupLinks] {
			t.Error("expected structure and links enabled")
		}
		if enabled[orchestrate.GroupContent] || enabled[orchestrate.GroupContamination] {
			t.Error("expected content and contamination disabled")
		}
	})

	t.Run("skip contamination", func(t *testing.T) {
		enabled, err := resolveCheckGroups("", "contamination")
		if err != nil {
			t.Fatal(err)
		}
		if !enabled[orchestrate.GroupStructure] || !enabled[orchestrate.GroupLinks] || !enabled[orchestrate.GroupContent] {
			t.Error("expected structure, links, content enabled")
		}
		if enabled[orchestrate.GroupContamination] {
			t.Error("expected contamination disabled")
		}
	})

	t.Run("invalid group", func(t *testing.T) {
		_, err := resolveCheckGroups("structure,bogus", "")
		if err == nil {
			t.Error("expected error for invalid group")
		}
	})

	t.Run("mutual exclusion", func(t *testing.T) {
		// This is checked in runCheck; covered by integration tests
	})
}

// --- End-to-end command handler tests ---

func TestResolvePath_ValidDir(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	resolved, err := resolvePath([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != dir {
		t.Errorf("expected %s, got %s", dir, resolved)
	}
}

func TestResolvePath_NoArgs(t *testing.T) {
	_, err := resolvePath([]string{})
	if err == nil {
		t.Error("expected error for empty args")
	}
}

func TestResolvePath_NotADirectory(t *testing.T) {
	// Point at a file, not a directory
	path := filepath.Join(fixtureDir(t, "valid-skill"), "SKILL.md")
	_, err := resolvePath([]string{path})
	if err == nil {
		t.Error("expected error for file path")
	}
	if !strings.Contains(err.Error(), "not a valid directory") {
		t.Errorf("expected 'not a valid directory' error, got: %v", err)
	}
}

func TestResolvePath_NonexistentPath(t *testing.T) {
	_, err := resolvePath([]string{"/nonexistent/path/that/does/not/exist"})
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestDetectAndResolve_SingleSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	_, mode, dirs, err := detectAndResolve([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != types.SingleSkill {
		t.Errorf("expected SingleSkill, got %d", mode)
	}
	if len(dirs) != 1 {
		t.Errorf("expected 1 dir, got %d", len(dirs))
	}
}

func TestDetectAndResolve_MultiSkill(t *testing.T) {
	dir := fixtureDir(t, "multi-skill")
	_, mode, dirs, err := detectAndResolve([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != types.MultiSkill {
		t.Errorf("expected MultiSkill, got %d", mode)
	}
	if len(dirs) < 2 {
		t.Errorf("expected multiple dirs, got %d", len(dirs))
	}
}

func TestDetectAndResolve_NoSkill(t *testing.T) {
	dir := t.TempDir()
	_, _, _, err := detectAndResolve([]string{dir})
	if err == nil {
		t.Error("expected error for directory with no skills")
	}
	if !strings.Contains(err.Error(), "no skills found") {
		t.Errorf("expected 'no skills found' error, got: %v", err)
	}
}
