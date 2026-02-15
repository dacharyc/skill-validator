package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/links"
	"github.com/dacharyc/skill-validator/internal/report"
	"github.com/dacharyc/skill-validator/internal/structure"
	"github.com/dacharyc/skill-validator/internal/validator"
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

	r := structure.Validate(dir)
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
		for _, res := range r.Results {
			if res.Level == validator.Error {
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

	r := structure.Validate(dir)
	if r.Errors == 0 {
		t.Error("expected errors for invalid skill")
	}
}

func TestValidateCommand_MultiSkill(t *testing.T) {
	dir := fixtureDir(t, "multi-skill")

	mode, dirs := validator.DetectSkills(dir)
	if mode != validator.MultiSkill {
		t.Fatalf("expected MultiSkill, got %d", mode)
	}

	mr := structure.ValidateMulti(dirs)
	if len(mr.Skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(mr.Skills))
	}
}

func TestValidateLinks_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	s, err := validator.LoadSkill(dir)
	if err != nil {
		t.Fatal(err)
	}

	// External link checks: valid-skill has no HTTP links, so no results
	linkResults := links.CheckLinks(dir, s.Body)
	if linkResults != nil {
		t.Errorf("expected nil for skill with no HTTP links, got %d results", len(linkResults))
	}

	// Internal links are now checked by structure validation
	r := structure.Validate(dir)
	foundLink := false
	for _, res := range r.Results {
		if res.Level == validator.Pass && strings.Contains(res.Message, "references/guide.md") {
			foundLink = true
		}
	}
	if !foundLink {
		t.Error("expected passing internal link check for references/guide.md in structure results")
	}
}

func TestValidateLinks_InvalidSkill(t *testing.T) {
	dir := fixtureDir(t, "invalid-skill")

	s, err := validator.LoadSkill(dir)
	if err != nil {
		t.Fatal(err)
	}

	// External link checks: invalid-skill has an HTTP link
	linkResults := links.CheckLinks(dir, s.Body)
	if len(linkResults) == 0 {
		t.Error("expected at least one external link check result")
	}

	// Internal links are now checked by structure validation
	r := structure.Validate(dir)
	foundBroken := false
	for _, res := range r.Results {
		if res.Level == validator.Error && strings.Contains(res.Message, "missing.md") {
			foundBroken = true
		}
	}
	if !foundBroken {
		t.Error("expected broken internal link error for references/missing.md in structure results")
	}
}

func TestAnalyzeContent_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	s, err := validator.LoadSkill(dir)
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

	s, err := validator.LoadSkill(dir)
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

	s, err := validator.LoadSkill(dir)
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

	s, err := validator.LoadSkill(dir)
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

func TestCheckCommand_AllChecks(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": true,
	}

	r := runAllChecks(dir, enabled)

	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
		for _, res := range r.Results {
			if res.Level == validator.Error {
				t.Logf("  error: %s: %s", res.Category, res.Message)
			}
		}
	}

	// Should have results from all check groups
	categories := map[string]bool{}
	for _, res := range r.Results {
		categories[res.Category] = true
	}
	if !categories["Structure"] {
		t.Error("expected Structure results")
	}
	if !categories["Frontmatter"] {
		t.Error("expected Frontmatter results")
	}
	// valid-skill has no HTTP links, so no "Links" category results are expected.
	// Internal links are checked by structure validation under the "Structure" category.

	// Should have content and contamination reports
	if r.ContentReport == nil {
		t.Error("expected ContentReport to be set")
	}
	if r.ContaminationReport == nil {
		t.Error("expected ContaminationReport to be set")
	}

	// valid-skill has references/guide.md
	if r.ReferencesContentReport == nil {
		t.Error("expected ReferencesContentReport to be set for valid-skill")
	}
	if r.ReferencesContaminationReport == nil {
		t.Error("expected ReferencesContaminationReport to be set for valid-skill")
	}
	if len(r.ReferenceReports) == 0 {
		t.Error("expected per-file ReferenceReports to be set for valid-skill")
	}
}

func TestCheckCommand_OnlyStructure(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	enabled := map[string]bool{
		"structure":     true,
		"links":         false,
		"content":       false,
		"contamination": false,
	}

	r := runAllChecks(dir, enabled)

	// Should have Markdown results (code fence checks are part of structure now)
	hasMarkdown := false
	for _, res := range r.Results {
		if res.Category == "Markdown" {
			hasMarkdown = true
		}
	}
	if !hasMarkdown {
		t.Error("expected Markdown results from structure validation")
	}

	// Should NOT have links/content/contamination results
	for _, res := range r.Results {
		if res.Category == "Links" {
			t.Errorf("unexpected Links result: %s: %s", res.Category, res.Message)
		}
	}
	if r.ContentReport != nil {
		t.Error("expected ContentReport to be nil when content is disabled")
	}
	if r.ReferencesContentReport != nil {
		t.Error("expected ReferencesContentReport to be nil when content is disabled")
	}
	if r.ContaminationReport != nil {
		t.Error("expected ContaminationReport to be nil when contamination is disabled")
	}
	if r.ReferencesContaminationReport != nil {
		t.Error("expected ReferencesContaminationReport to be nil when contamination is disabled")
	}
	if len(r.ReferenceReports) != 0 {
		t.Error("expected no ReferenceReports when both content and contamination are disabled")
	}
}

func TestCheckCommand_OnlyLinks(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	enabled := map[string]bool{
		"structure":     false,
		"links":         true,
		"content":       false,
		"contamination": false,
	}

	r := runAllChecks(dir, enabled)

	// Should NOT have structure results
	for _, res := range r.Results {
		if res.Category == "Structure" || res.Category == "Frontmatter" || res.Category == "Tokens" {
			t.Errorf("unexpected structure result: %s: %s", res.Category, res.Message)
		}
	}

	// Should NOT have Markdown results (those are part of structure now)
	for _, res := range r.Results {
		if res.Category == "Markdown" {
			t.Errorf("unexpected Markdown result in links-only check: %s: %s", res.Category, res.Message)
		}
	}
}

func TestCheckCommand_SkipContamination(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": false,
	}

	r := runAllChecks(dir, enabled)

	if r.ContentReport == nil {
		t.Error("expected ContentReport when content is enabled")
	}
	if r.ContaminationReport != nil {
		t.Error("expected ContaminationReport to be nil when contamination is skipped")
	}
	// Content reference fields should be populated, but contamination ones nil
	if r.ReferencesContentReport == nil {
		t.Error("expected ReferencesContentReport when content is enabled")
	}
	if r.ReferencesContaminationReport != nil {
		t.Error("expected ReferencesContaminationReport to be nil when contamination is skipped")
	}
	// Per-file reports should have content but not contamination
	if len(r.ReferenceReports) == 0 {
		t.Fatal("expected ReferenceReports when content is enabled")
	}
	for _, fr := range r.ReferenceReports {
		if fr.ContentReport == nil {
			t.Error("expected per-file ContentReport when content is enabled")
		}
		if fr.ContaminationReport != nil {
			t.Error("expected nil per-file ContaminationReport when contamination is skipped")
		}
	}
}

func TestCheckCommand_OnlyContentContamination(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")

	enabled := map[string]bool{
		"structure":     false,
		"links":         false,
		"content":       true,
		"contamination": true,
	}

	r := runAllChecks(dir, enabled)

	if r.ContentReport == nil {
		t.Error("expected ContentReport")
	}
	if r.ContaminationReport == nil {
		t.Error("expected ContaminationReport")
	}

	// Content should have code blocks
	if r.ContentReport.CodeBlockCount != 4 {
		t.Errorf("expected 4 code blocks, got %d", r.ContentReport.CodeBlockCount)
	}

	// Contamination should detect mongodb
	foundMongo := false
	for _, tool := range r.ContaminationReport.MultiInterfaceTools {
		if tool == "mongodb" {
			foundMongo = true
		}
	}
	if !foundMongo {
		t.Error("expected mongodb multi-interface tool detection")
	}
}

func TestCheckCommand_BrokenFrontmatter_AllChecks(t *testing.T) {
	dir := fixtureDir(t, "broken-frontmatter")

	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": true,
	}

	r := runAllChecks(dir, enabled)

	// Should have a frontmatter parse error from structure
	if r.Errors == 0 {
		t.Error("expected errors for broken frontmatter")
	}
	foundFMError := false
	for _, res := range r.Results {
		if res.Level == validator.Error && res.Category == "Frontmatter" {
			foundFMError = true
		}
	}
	if !foundFMError {
		t.Error("expected Frontmatter error result")
	}

	// Content analysis should still be populated (fallback to raw file read)
	if r.ContentReport == nil {
		t.Fatal("expected ContentReport despite broken frontmatter")
	}
	if r.ContentReport.WordCount == 0 {
		t.Error("expected non-zero word count from content analysis")
	}
	if r.ContentReport.CodeBlockCount != 2 {
		t.Errorf("expected 2 code blocks (bash, python), got %d", r.ContentReport.CodeBlockCount)
	}
	if len(r.ContentReport.CodeLanguages) != 2 {
		t.Errorf("expected 2 code languages, got %d: %v",
			len(r.ContentReport.CodeLanguages), r.ContentReport.CodeLanguages)
	}

	// Contamination analysis should still be populated
	if r.ContaminationReport == nil {
		t.Fatal("expected ContaminationReport despite broken frontmatter")
	}
	if r.ContaminationReport.ContaminationLevel == "" {
		t.Error("expected non-empty contamination level")
	}

	// Link checks should be skipped (need parsed skill for link checks)
	for _, res := range r.Results {
		if res.Category == "Links" {
			t.Errorf("unexpected Links result for broken-frontmatter skill: %s: %s",
				res.Category, res.Message)
		}
	}
}

func TestCheckCommand_BrokenFrontmatter_OnlyContent(t *testing.T) {
	dir := fixtureDir(t, "broken-frontmatter")

	enabled := map[string]bool{
		"structure":     false,
		"links":         false,
		"content":       true,
		"contamination": false,
	}

	r := runAllChecks(dir, enabled)

	// Content analysis should work even without structure
	if r.ContentReport == nil {
		t.Fatal("expected ContentReport for content-only check")
	}
	if r.ContentReport.WordCount == 0 {
		t.Error("expected non-zero word count")
	}
	if r.ContentReport.StrongMarkers == 0 {
		t.Error("expected strong markers (must, always, never)")
	}
}

func TestCheckCommand_BrokenFrontmatter_OnlyContamination(t *testing.T) {
	dir := fixtureDir(t, "broken-frontmatter")

	enabled := map[string]bool{
		"structure":     false,
		"links":         false,
		"content":       false,
		"contamination": true,
	}

	r := runAllChecks(dir, enabled)

	// Contamination analysis should work even without content analysis enabled
	if r.ContaminationReport == nil {
		t.Fatal("expected ContaminationReport for contamination-only check")
	}
	// Should have detected code languages from the raw content
	if len(r.ContaminationReport.CodeLanguages) != 2 {
		t.Errorf("expected 2 code languages, got %d: %v",
			len(r.ContaminationReport.CodeLanguages), r.ContaminationReport.CodeLanguages)
	}
}

func TestReadSkillRaw(t *testing.T) {
	dir := fixtureDir(t, "broken-frontmatter")

	raw := validator.ReadSkillRaw(dir)
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
	files := validator.ReadReferencesMarkdownFiles(dir)
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
	files := validator.ReadReferencesMarkdownFiles(dir)
	if files != nil {
		t.Errorf("expected nil for dir without references, got %d files", len(files))
	}
}

func TestReadSkillRaw_MissingFile(t *testing.T) {
	dir := t.TempDir()

	raw := validator.ReadSkillRaw(dir)
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
		for _, g := range []string{"structure", "links", "content", "contamination"} {
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
		if !enabled["structure"] || !enabled["links"] {
			t.Error("expected structure and links enabled")
		}
		if enabled["content"] || enabled["contamination"] {
			t.Error("expected content and contamination disabled")
		}
	})

	t.Run("skip contamination", func(t *testing.T) {
		enabled, err := resolveCheckGroups("", "contamination")
		if err != nil {
			t.Fatal(err)
		}
		if !enabled["structure"] || !enabled["links"] || !enabled["content"] {
			t.Error("expected structure, links, content enabled")
		}
		if enabled["contamination"] {
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
		// This is checked in runCheck, but test the logic anyway
		if checkOnly != "" && checkSkip != "" {
			// Reset for test isolation
		}
	})
}

func TestCheckCommand_JSONOutput(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")

	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": true,
	}

	r := runAllChecks(dir, enabled)

	// Render as JSON and verify structure
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	// Build a simplified JSON to verify fields exist
	type jsonCheck struct {
		ContentAnalysis       *content.Report       `json:"content_analysis,omitempty"`
		ContaminationAnalysis *contamination.Report  `json:"contamination_analysis,omitempty"`
	}

	out := jsonCheck{
		ContentAnalysis:       r.ContentReport,
		ContaminationAnalysis: r.ContaminationReport,
	}

	if err := enc.Encode(out); err != nil {
		t.Fatal(err)
	}

	// Parse back and verify
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}

	ca, ok := parsed["content_analysis"].(map[string]any)
	if !ok {
		t.Fatal("expected content_analysis object in JSON")
	}
	if ca["word_count"].(float64) == 0 {
		t.Error("expected non-zero word_count in JSON")
	}
	if ca["code_block_count"].(float64) != 4 {
		t.Errorf("expected 4 code_block_count, got %v", ca["code_block_count"])
	}

	ra, ok := parsed["contamination_analysis"].(map[string]any)
	if !ok {
		t.Fatal("expected contamination_analysis object in JSON")
	}
	if ra["contamination_level"].(string) == "" {
		t.Error("expected non-empty contamination_level in JSON")
	}
	if ra["contamination_score"].(float64) <= 0 {
		t.Error("expected positive contamination_score in JSON")
	}
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
	if mode != validator.SingleSkill {
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
	if mode != validator.MultiSkill {
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

// --- Individual analysis function tests ---

func TestRunContaminationAnalysis_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	r := runContaminationAnalysis(dir)
	if r.ContaminationReport == nil {
		t.Fatal("expected ContaminationReport to be set")
	}
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
	}
	hasPass := false
	for _, res := range r.Results {
		if res.Level == validator.Pass && res.Category == "Contamination" {
			hasPass = true
		}
	}
	if !hasPass {
		t.Error("expected pass result with Contamination category")
	}

	// valid-skill has references/guide.md — analyze contamination should cover it
	if r.ReferencesContentReport == nil {
		t.Error("expected ReferencesContentReport to be set for valid-skill")
	}
	if r.ReferencesContaminationReport == nil {
		t.Error("expected ReferencesContaminationReport to be set for valid-skill")
	}
	if len(r.ReferenceReports) == 0 {
		t.Fatal("expected per-file ReferenceReports for valid-skill")
	}
	if r.ReferenceReports[0].File != "guide.md" {
		t.Errorf("expected first reference file to be guide.md, got %s", r.ReferenceReports[0].File)
	}
}

func TestRunContaminationAnalysis_RichSkill(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")
	r := runContaminationAnalysis(dir)
	if r.ContaminationReport == nil {
		t.Fatal("expected ContaminationReport to be set")
	}
	if r.ContaminationReport.ContaminationScore <= 0 {
		t.Error("expected positive contamination score for rich-skill")
	}

	// rich-skill has no references directory
	if r.ReferencesContentReport != nil {
		t.Error("expected nil ReferencesContentReport for skill without references")
	}
	if r.ReferencesContaminationReport != nil {
		t.Error("expected nil ReferencesContaminationReport for skill without references")
	}
	if len(r.ReferenceReports) != 0 {
		t.Error("expected no ReferenceReports for skill without references")
	}
}

func TestRunContaminationAnalysis_BrokenDir(t *testing.T) {
	dir := t.TempDir() // no SKILL.md
	r := runContaminationAnalysis(dir)
	if r.Errors != 1 {
		t.Errorf("expected 1 error, got %d", r.Errors)
	}
	if r.ContaminationReport != nil {
		t.Error("expected nil ContaminationReport for broken dir")
	}
	if r.ReferencesContaminationReport != nil {
		t.Error("expected nil ReferencesContaminationReport for broken dir")
	}
	if len(r.ReferenceReports) != 0 {
		t.Error("expected no ReferenceReports for broken dir")
	}
}

func TestRunContentAnalysis_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	r := runContentAnalysis(dir)
	if r.ContentReport == nil {
		t.Fatal("expected ContentReport to be set")
	}
	if r.ContentReport.WordCount == 0 {
		t.Error("expected non-zero word count")
	}
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
	}

	// valid-skill has references/guide.md, so aggregate reports should be set
	if r.ReferencesContentReport == nil {
		t.Fatal("expected ReferencesContentReport to be set for valid-skill")
	}
	if r.ReferencesContentReport.WordCount == 0 {
		t.Error("expected non-zero word count in references content report")
	}
	if r.ReferencesContaminationReport == nil {
		t.Fatal("expected ReferencesContaminationReport to be set for valid-skill")
	}

	// Per-file reports should also be populated
	if len(r.ReferenceReports) == 0 {
		t.Fatal("expected per-file ReferenceReports for valid-skill")
	}
	if r.ReferenceReports[0].File != "guide.md" {
		t.Errorf("expected first reference file to be guide.md, got %s", r.ReferenceReports[0].File)
	}
	if r.ReferenceReports[0].ContentReport == nil {
		t.Error("expected per-file ContentReport to be set")
	}
	if r.ReferenceReports[0].ContaminationReport == nil {
		t.Error("expected per-file ContaminationReport to be set")
	}
}

func TestRunContentAnalysis_BrokenDir(t *testing.T) {
	dir := t.TempDir()
	r := runContentAnalysis(dir)
	if r.Errors != 1 {
		t.Errorf("expected 1 error, got %d", r.Errors)
	}
	if r.ContentReport != nil {
		t.Error("expected nil ContentReport for broken dir")
	}
	if r.ReferencesContentReport != nil {
		t.Error("expected nil ReferencesContentReport for broken dir")
	}
	if r.ReferencesContaminationReport != nil {
		t.Error("expected nil ReferencesContaminationReport for broken dir")
	}
	if len(r.ReferenceReports) != 0 {
		t.Error("expected no ReferenceReports for broken dir")
	}
}

func TestRunContentAnalysis_NoReferences(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")
	r := runContentAnalysis(dir)
	if r.ContentReport == nil {
		t.Fatal("expected ContentReport to be set")
	}
	// rich-skill has no references directory, so ReferencesContentReport should be nil
	if r.ReferencesContentReport != nil {
		t.Error("expected nil ReferencesContentReport for skill without references")
	}
}

func TestRunLinkChecks_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	r := runLinkChecks(dir)
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
		for _, res := range r.Results {
			if res.Level == validator.Error {
				t.Logf("  error: %s: %s", res.Category, res.Message)
			}
		}
	}
	hasLinks := false
	for _, res := range r.Results {
		if res.Category == "Links" {
			hasLinks = true
		}
	}
	if !hasLinks {
		t.Error("expected Links results from link checks")
	}
}

func TestRunLinkChecks_InvalidSkill(t *testing.T) {
	dir := fixtureDir(t, "invalid-skill")
	r := runLinkChecks(dir)
	if r.Errors == 0 {
		t.Error("expected errors for invalid skill with broken links")
	}
}

func TestRunLinkChecks_BrokenDir(t *testing.T) {
	dir := t.TempDir()
	r := runLinkChecks(dir)
	if r.Errors != 1 {
		t.Errorf("expected 1 error, got %d", r.Errors)
	}
}

// --- Multi-skill path tests through command handlers ---

func TestRunAllChecks_MultiSkill(t *testing.T) {
	dir := fixtureDir(t, "multi-skill")
	_, dirs := validator.DetectSkills(dir)

	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": true,
	}

	mr := &validator.MultiReport{}
	for _, d := range dirs {
		r := runAllChecks(d, enabled)
		mr.Skills = append(mr.Skills, r)
		mr.Errors += r.Errors
		mr.Warnings += r.Warnings
	}

	if len(mr.Skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(mr.Skills))
	}

	// Each skill should have content and contamination reports
	for i, r := range mr.Skills {
		if r.ContentReport == nil {
			t.Errorf("skill %d: expected ContentReport", i)
		}
		if r.ContaminationReport == nil {
			t.Errorf("skill %d: expected ContaminationReport", i)
		}
	}
}

// --- JSON output end-to-end through report package ---

func TestOutputJSON_FullCheck_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": true,
	}
	r := runAllChecks(dir, enabled)

	var buf bytes.Buffer
	if err := report.PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed["passed"] != true {
		t.Error("expected passed=true")
	}
	if _, ok := parsed["content_analysis"]; !ok {
		t.Error("expected content_analysis in JSON")
	}
	if _, ok := parsed["references_content_analysis"]; !ok {
		t.Error("expected references_content_analysis in JSON for valid-skill")
	}
	if _, ok := parsed["contamination_analysis"]; !ok {
		t.Error("expected contamination_analysis in JSON")
	}
	if _, ok := parsed["references_contamination_analysis"]; !ok {
		t.Error("expected references_contamination_analysis in JSON for valid-skill")
	}
	if _, ok := parsed["token_counts"]; !ok {
		t.Error("expected token_counts in JSON")
	}
	// Without --per-file, reference_reports should be absent
	if _, ok := parsed["reference_reports"]; ok {
		t.Error("expected no reference_reports in JSON without --per-file")
	}
}

func TestOutputJSON_FullCheck_RichSkill(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")
	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": true,
	}
	r := runAllChecks(dir, enabled)

	var buf bytes.Buffer
	if err := report.PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify contamination fields in JSON
	ca := parsed["contamination_analysis"].(map[string]any)
	if ca["contamination_level"].(string) == "" {
		t.Error("expected non-empty contamination_level")
	}
	if ca["contamination_score"].(float64) <= 0 {
		t.Error("expected positive contamination_score")
	}
	if ca["language_mismatch"] != true {
		t.Error("expected language_mismatch=true")
	}

	tools := ca["multi_interface_tools"].([]any)
	foundMongo := false
	for _, tool := range tools {
		if tool.(string) == "mongodb" {
			foundMongo = true
		}
	}
	if !foundMongo {
		t.Error("expected mongodb in multi_interface_tools")
	}

	// Verify content fields in JSON
	co := parsed["content_analysis"].(map[string]any)
	if co["word_count"].(float64) == 0 {
		t.Error("expected non-zero word_count")
	}
	if co["code_block_count"].(float64) != 4 {
		t.Errorf("expected 4 code_block_count, got %v", co["code_block_count"])
	}
}

func TestOutputJSON_MultiSkill(t *testing.T) {
	dir := fixtureDir(t, "multi-skill")
	_, dirs := validator.DetectSkills(dir)

	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": true,
	}

	mr := &validator.MultiReport{}
	for _, d := range dirs {
		r := runAllChecks(d, enabled)
		mr.Skills = append(mr.Skills, r)
		mr.Errors += r.Errors
		mr.Warnings += r.Warnings
	}

	var buf bytes.Buffer
	if err := report.PrintMultiJSON(&buf, mr, false); err != nil {
		t.Fatalf("PrintMultiJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	skills := parsed["skills"].([]any)
	if len(skills) != 3 {
		t.Fatalf("expected 3 skills in JSON, got %d", len(skills))
	}

	// Each skill should have contamination_analysis
	for i, s := range skills {
		skill := s.(map[string]any)
		if _, ok := skill["contamination_analysis"]; !ok {
			t.Errorf("skill %d: expected contamination_analysis in JSON", i)
		}
		if _, ok := skill["content_analysis"]; !ok {
			t.Errorf("skill %d: expected content_analysis in JSON", i)
		}
	}
}

func TestOutputJSON_PerFile_ValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	enabled := map[string]bool{
		"structure":     true,
		"links":         true,
		"content":       true,
		"contamination": true,
	}
	r := runAllChecks(dir, enabled)

	var buf bytes.Buffer
	if err := report.PrintJSON(&buf, r, true); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// With --per-file, reference_reports should be present
	rr, ok := parsed["reference_reports"].([]any)
	if !ok {
		t.Fatal("expected reference_reports array in JSON with --per-file")
	}
	if len(rr) == 0 {
		t.Fatal("expected at least one reference report")
	}

	first := rr[0].(map[string]any)
	if first["file"].(string) != "guide.md" {
		t.Errorf("expected file=guide.md, got %s", first["file"])
	}
	if _, ok := first["content_analysis"]; !ok {
		t.Error("expected content_analysis in per-file report")
	}
	if _, ok := first["contamination_analysis"]; !ok {
		t.Error("expected contamination_analysis in per-file report")
	}
}

func TestRunContaminationAnalysis_ReferencesValidSkill(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")
	r := runContaminationAnalysis(dir)

	if r.ReferencesContaminationReport == nil {
		t.Fatal("expected ReferencesContaminationReport for valid-skill")
	}
	if len(r.ReferenceReports) == 0 {
		t.Fatal("expected per-file ReferenceReports for valid-skill")
	}
}

func TestRunContaminationAnalysis_NoReferences(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")
	r := runContaminationAnalysis(dir)

	// rich-skill has no references directory
	if r.ReferencesContaminationReport != nil {
		t.Error("expected nil ReferencesContaminationReport for skill without references")
	}
	if len(r.ReferenceReports) != 0 {
		t.Error("expected no ReferenceReports for skill without references")
	}
}

func TestRunContentAnalysis_NoReferencesContamination(t *testing.T) {
	dir := fixtureDir(t, "rich-skill")
	r := runContentAnalysis(dir)

	// rich-skill has no references directory
	if r.ReferencesContaminationReport != nil {
		t.Error("expected nil ReferencesContaminationReport for skill without references")
	}
	if len(r.ReferenceReports) != 0 {
		t.Error("expected no ReferenceReports for skill without references")
	}
}

func TestCheckCommand_OnlyContent_ReferencesHaveContentOnly(t *testing.T) {
	dir := fixtureDir(t, "valid-skill")

	enabled := map[string]bool{
		"structure":     false,
		"links":         false,
		"content":       true,
		"contamination": false,
	}

	r := runAllChecks(dir, enabled)

	if r.ReferencesContentReport == nil {
		t.Error("expected ReferencesContentReport when content is enabled")
	}
	if r.ReferencesContaminationReport != nil {
		t.Error("expected nil ReferencesContaminationReport when contamination is disabled")
	}
	for _, fr := range r.ReferenceReports {
		if fr.ContentReport == nil {
			t.Error("expected per-file ContentReport when content is enabled")
		}
		if fr.ContaminationReport != nil {
			t.Error("expected nil per-file ContaminationReport when contamination is disabled")
		}
	}
}
