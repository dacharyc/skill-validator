package evaluate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/judge"
)

func TestFindParentSkillDir(t *testing.T) {
	// Create a temp directory with a SKILL.md
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "my-skill")
	refsDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	refFile := filepath.Join(refsDir, "example.md")
	if err := os.WriteFile(refFile, []byte("# ref"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := FindParentSkillDir(refFile)
	if err != nil {
		t.Fatalf("FindParentSkillDir() error = %v", err)
	}
	if got != skillDir {
		t.Errorf("FindParentSkillDir() = %q, want %q", got, skillDir)
	}
}

func TestFindParentSkillDir_NotFound(t *testing.T) {
	tmp := t.TempDir()
	noSkill := filepath.Join(tmp, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(noSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(noSkill, "test.md")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := FindParentSkillDir(filePath)
	if err == nil {
		t.Fatal("expected error for missing SKILL.md")
	}
	if !strings.Contains(err.Error(), "could not find parent SKILL.md") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintText(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/my-skill",
		SkillScores: &judge.SkillScores{
			Clarity:            4,
			Actionability:      3,
			TokenEfficiency:    5,
			ScopeDiscipline:    4,
			DirectivePrecision: 4,
			Novelty:            3,
			Overall:            3.83,
			BriefAssessment:    "Good skill",
		},
	}

	var buf bytes.Buffer
	PrintText(&buf, result, "aggregate")
	out := buf.String()

	if !strings.Contains(out, "Scoring skill: /tmp/my-skill") {
		t.Errorf("expected skill dir header, got: %s", out)
	}
	if !strings.Contains(out, "SKILL.md Scores") {
		t.Errorf("expected SKILL.md Scores header, got: %s", out)
	}
	if !strings.Contains(out, "3.83/5") {
		t.Errorf("expected overall score, got: %s", out)
	}
	if !strings.Contains(out, "Good skill") {
		t.Errorf("expected assessment, got: %s", out)
	}
}

func TestPrintJSON(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/my-skill",
		SkillScores: &judge.SkillScores{
			Clarity: 4,
			Overall: 4.0,
		},
	}

	var buf bytes.Buffer
	err := PrintJSON(&buf, []*EvalResult{result})
	if err != nil {
		t.Fatalf("PrintJSON() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"skill_dir"`) {
		t.Errorf("expected JSON skill_dir field, got: %s", out)
	}
	if !strings.Contains(out, `"clarity"`) {
		t.Errorf("expected JSON clarity field, got: %s", out)
	}
}

func TestPrintMarkdown(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/my-skill",
		SkillScores: &judge.SkillScores{
			Clarity:            4,
			Actionability:      3,
			TokenEfficiency:    5,
			ScopeDiscipline:    4,
			DirectivePrecision: 4,
			Novelty:            3,
			Overall:            3.83,
			BriefAssessment:    "Good skill",
		},
	}

	var buf bytes.Buffer
	PrintMarkdown(&buf, result, "aggregate")
	out := buf.String()

	if !strings.Contains(out, "## Scoring skill:") {
		t.Errorf("expected markdown header, got: %s", out)
	}
	if !strings.Contains(out, "| Clarity | 4/5 |") {
		t.Errorf("expected clarity row, got: %s", out)
	}
	if !strings.Contains(out, "**3.83/5**") {
		t.Errorf("expected overall score, got: %s", out)
	}
}

func TestFormatResults_SingleText(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/test",
		SkillScores: &judge.SkillScores{
			Overall: 4.0,
		},
	}

	var buf bytes.Buffer
	err := FormatResults(&buf, []*EvalResult{result}, "text", "aggregate")
	if err != nil {
		t.Fatalf("FormatResults() error = %v", err)
	}

	if !strings.Contains(buf.String(), "Scoring skill:") {
		t.Errorf("expected text output, got: %s", buf.String())
	}
}

func TestFormatResults_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := FormatResults(&buf, nil, "text", "aggregate")
	if err != nil {
		t.Fatalf("FormatResults() error = %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got: %s", buf.String())
	}
}

func TestPrintMultiMarkdown(t *testing.T) {
	results := []*EvalResult{
		{SkillDir: "/tmp/skill-a", SkillScores: &judge.SkillScores{Overall: 4.0}},
		{SkillDir: "/tmp/skill-b", SkillScores: &judge.SkillScores{Overall: 3.0}},
	}

	var buf bytes.Buffer
	PrintMultiMarkdown(&buf, results, "aggregate")
	out := buf.String()

	if !strings.Contains(out, "skill-a") {
		t.Errorf("expected skill-a, got: %s", out)
	}
	if !strings.Contains(out, "skill-b") {
		t.Errorf("expected skill-b, got: %s", out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("expected separator, got: %s", out)
	}
}

func TestPrintText_WithRefs(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/my-skill",
		RefResults: []RefEvalResult{
			{
				File: "example.md",
				Scores: &judge.RefScores{
					Clarity:            4,
					InstructionalValue: 3,
					TokenEfficiency:    5,
					Novelty:            4,
					SkillRelevance:     4,
					Overall:            4.0,
					BriefAssessment:    "Good ref",
				},
			},
		},
		RefAggregate: &judge.RefScores{
			Clarity:            4,
			InstructionalValue: 3,
			TokenEfficiency:    5,
			Novelty:            4,
			SkillRelevance:     4,
			Overall:            4.0,
		},
	}

	// Test "files" display mode shows individual refs
	var buf bytes.Buffer
	PrintText(&buf, result, "files")
	out := buf.String()

	if !strings.Contains(out, "Reference: example.md") {
		t.Errorf("expected ref header in files mode, got: %s", out)
	}

	// Test "aggregate" display mode hides individual refs
	buf.Reset()
	PrintText(&buf, result, "aggregate")
	out = buf.String()

	if strings.Contains(out, "Reference: example.md") {
		t.Errorf("should not show individual refs in aggregate mode, got: %s", out)
	}
	if !strings.Contains(out, "Reference Scores (1 file)") {
		t.Errorf("expected aggregate ref header, got: %s", out)
	}
}

// --- Formatting coverage tests ---

func TestFormatResults_SingleJSON(t *testing.T) {
	result := &EvalResult{
		SkillDir:    "/tmp/test",
		SkillScores: &judge.SkillScores{Clarity: 4, Overall: 4.0},
	}

	var buf bytes.Buffer
	err := FormatResults(&buf, []*EvalResult{result}, "json", "aggregate")
	if err != nil {
		t.Fatalf("FormatResults(json) error = %v", err)
	}
	if !strings.Contains(buf.String(), `"skill_dir"`) {
		t.Errorf("expected JSON output, got: %s", buf.String())
	}
}

func TestFormatResults_SingleMarkdown(t *testing.T) {
	result := &EvalResult{
		SkillDir:    "/tmp/test",
		SkillScores: &judge.SkillScores{Clarity: 4, Overall: 4.0},
	}

	var buf bytes.Buffer
	err := FormatResults(&buf, []*EvalResult{result}, "markdown", "aggregate")
	if err != nil {
		t.Fatalf("FormatResults(markdown) error = %v", err)
	}
	if !strings.Contains(buf.String(), "## Scoring skill:") {
		t.Errorf("expected markdown output, got: %s", buf.String())
	}
}

func TestFormatMultiResults_Text(t *testing.T) {
	results := []*EvalResult{
		{SkillDir: "/tmp/a", SkillScores: &judge.SkillScores{Overall: 4.0}},
		{SkillDir: "/tmp/b", SkillScores: &judge.SkillScores{Overall: 3.0}},
	}

	var buf bytes.Buffer
	err := FormatMultiResults(&buf, results, "text", "aggregate")
	if err != nil {
		t.Fatalf("FormatMultiResults(text) error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "/tmp/a") || !strings.Contains(out, "/tmp/b") {
		t.Errorf("expected both skills, got: %s", out)
	}
	if !strings.Contains(out, "━") {
		t.Errorf("expected separator, got: %s", out)
	}
}

func TestFormatMultiResults_JSON(t *testing.T) {
	results := []*EvalResult{
		{SkillDir: "/tmp/a", SkillScores: &judge.SkillScores{Overall: 4.0}},
		{SkillDir: "/tmp/b", SkillScores: &judge.SkillScores{Overall: 3.0}},
	}

	var buf bytes.Buffer
	err := FormatMultiResults(&buf, results, "json", "aggregate")
	if err != nil {
		t.Fatalf("FormatMultiResults(json) error = %v", err)
	}
	if !strings.Contains(buf.String(), "/tmp/a") {
		t.Errorf("expected skill dir in JSON, got: %s", buf.String())
	}
}

func TestFormatMultiResults_Markdown(t *testing.T) {
	results := []*EvalResult{
		{SkillDir: "/tmp/a", SkillScores: &judge.SkillScores{Overall: 4.0}},
		{SkillDir: "/tmp/b", SkillScores: &judge.SkillScores{Overall: 3.0}},
	}

	var buf bytes.Buffer
	err := FormatMultiResults(&buf, results, "markdown", "aggregate")
	if err != nil {
		t.Fatalf("FormatMultiResults(markdown) error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "---") {
		t.Errorf("expected markdown separator, got: %s", out)
	}
}

func TestFormatResults_MultiDelegatesToFormatMulti(t *testing.T) {
	results := []*EvalResult{
		{SkillDir: "/tmp/a"},
		{SkillDir: "/tmp/b"},
	}

	var buf bytes.Buffer
	err := FormatResults(&buf, results, "text", "aggregate")
	if err != nil {
		t.Fatalf("FormatResults with 2 results error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "/tmp/a") || !strings.Contains(out, "/tmp/b") {
		t.Errorf("expected both skills, got: %s", out)
	}
}

func TestPrintMarkdown_WithRefsFiles(t *testing.T) {
	result := &EvalResult{
		SkillDir:    "/tmp/my-skill",
		SkillScores: &judge.SkillScores{Clarity: 4, Overall: 4.0},
		RefResults: []RefEvalResult{
			{
				File: "ref.md",
				Scores: &judge.RefScores{
					Clarity: 4, InstructionalValue: 3,
					TokenEfficiency: 5, Novelty: 4, SkillRelevance: 4,
					Overall: 4.0, BriefAssessment: "Good", NovelInfo: "Proprietary API",
				},
			},
		},
		RefAggregate: &judge.RefScores{
			Clarity: 4, InstructionalValue: 3, TokenEfficiency: 5,
			Novelty: 4, SkillRelevance: 4, Overall: 4.0,
		},
	}

	var buf bytes.Buffer
	PrintMarkdown(&buf, result, "files")
	out := buf.String()

	if !strings.Contains(out, "### Reference: ref.md") {
		t.Errorf("expected ref header in files mode, got: %s", out)
	}
	if !strings.Contains(out, "Proprietary API") {
		t.Errorf("expected novel info, got: %s", out)
	}
	if !strings.Contains(out, "### Reference Scores") {
		t.Errorf("expected aggregate ref header, got: %s", out)
	}
}

func TestPrintMarkdown_WithNovelInfo(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/test",
		SkillScores: &judge.SkillScores{
			Clarity: 4, Overall: 4.0,
			BriefAssessment: "Assessment", NovelInfo: "Internal API",
		},
	}

	var buf bytes.Buffer
	PrintMarkdown(&buf, result, "aggregate")
	out := buf.String()

	if !strings.Contains(out, "> Assessment") {
		t.Errorf("expected assessment blockquote, got: %s", out)
	}
	if !strings.Contains(out, "*Novel details: Internal API*") {
		t.Errorf("expected novel info, got: %s", out)
	}
}

func TestPrintText_NovelInfo(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/test",
		SkillScores: &judge.SkillScores{
			Clarity: 4, Overall: 4.0,
			NovelInfo: "Proprietary details",
		},
	}

	var buf bytes.Buffer
	PrintText(&buf, result, "aggregate")
	out := buf.String()
	if !strings.Contains(out, "Novel details: Proprietary details") {
		t.Errorf("expected novel info in text, got: %s", out)
	}
}

func TestPrintText_RefFilesWithNovelInfo(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/test",
		RefResults: []RefEvalResult{
			{
				File: "ref.md",
				Scores: &judge.RefScores{
					Clarity: 4, InstructionalValue: 3, TokenEfficiency: 5,
					Novelty: 4, SkillRelevance: 4, Overall: 4.0,
					NovelInfo: "Internal endpoint",
				},
			},
		},
	}

	var buf bytes.Buffer
	PrintText(&buf, result, "files")
	out := buf.String()
	if !strings.Contains(out, "Novel details: Internal endpoint") {
		t.Errorf("expected ref novel info, got: %s", out)
	}
}

func TestPrintJSON_WithRefs(t *testing.T) {
	result := &EvalResult{
		SkillDir: "/tmp/test",
		RefResults: []RefEvalResult{
			{File: "ref.md", Scores: &judge.RefScores{Clarity: 4, Overall: 4.0}},
		},
		RefAggregate: &judge.RefScores{Clarity: 4, Overall: 4.0},
	}

	var buf bytes.Buffer
	err := PrintJSON(&buf, []*EvalResult{result})
	if err != nil {
		t.Fatalf("PrintJSON error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"reference_scores"`) {
		t.Errorf("expected reference_scores in JSON, got: %s", out)
	}
	if !strings.Contains(out, `"reference_aggregate"`) {
		t.Errorf("expected reference_aggregate in JSON, got: %s", out)
	}
}

func TestPrintDimScore_Colors(t *testing.T) {
	var buf bytes.Buffer

	// High score (green)
	printDimScore(&buf, "Test", 5)
	if !strings.Contains(buf.String(), ColorGreen) {
		t.Errorf("score 5 should use green, got: %s", buf.String())
	}

	// Medium score (yellow)
	buf.Reset()
	printDimScore(&buf, "Test", 3)
	if !strings.Contains(buf.String(), ColorYellow) {
		t.Errorf("score 3 should use yellow, got: %s", buf.String())
	}

	// Low score (red)
	buf.Reset()
	printDimScore(&buf, "Test", 2)
	if !strings.Contains(buf.String(), ColorRed) {
		t.Errorf("score 2 should use red, got: %s", buf.String())
	}
}

// --- Mock LLM client ---

type mockLLMClient struct {
	responses []string
	errors    []error
	callIdx   int
}

func (m *mockLLMClient) Complete(_ context.Context, _, _ string) (string, error) {
	idx := m.callIdx
	m.callIdx++
	if idx < len(m.errors) && m.errors[idx] != nil {
		return "", m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "", fmt.Errorf("no more mock responses (call %d)", idx)
}

func (m *mockLLMClient) Provider() string  { return "mock" }
func (m *mockLLMClient) ModelName() string { return "mock-model" }

// skillJSON is a valid JSON response for skill scoring (all dims, low novelty).
const skillJSON = `{"clarity":4,"actionability":5,"token_efficiency":3,"scope_discipline":4,"directive_precision":4,"novelty":2,"brief_assessment":"Solid."}`

// refJSON is a valid JSON response for reference scoring (all dims, low novelty).
const refJSON = `{"clarity":4,"instructional_value":3,"token_efficiency":4,"novelty":2,"skill_relevance":4,"brief_assessment":"Good ref."}`

// makeSkillDir creates a temp skill directory with SKILL.md and optional refs.
func makeSkillDir(t *testing.T, refs map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "test-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillContent := "---\nname: test-skill\ndescription: A test skill\n---\n# Test Skill\nInstructions here.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(refs) > 0 {
		refsDir := filepath.Join(dir, "references")
		if err := os.MkdirAll(refsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, content := range refs {
			if err := os.WriteFile(filepath.Join(refsDir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

// --- EvaluateSkill tests ---

func TestEvaluateSkill_SkillOnly(t *testing.T) {
	dir := makeSkillDir(t, map[string]string{"ref.md": "# Ref"})
	client := &mockLLMClient{responses: []string{skillJSON}}

	result, err := EvaluateSkill(context.Background(), dir, client, EvalOptions{SkillOnly: true, MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("EvaluateSkill error = %v", err)
	}
	if result.SkillScores == nil {
		t.Fatal("expected SkillScores")
	}
	if len(result.RefResults) != 0 {
		t.Errorf("expected no refs with SkillOnly, got %d", len(result.RefResults))
	}
}

func TestEvaluateSkill_RefsOnly(t *testing.T) {
	dir := makeSkillDir(t, map[string]string{"ref.md": "# Ref"})
	client := &mockLLMClient{responses: []string{refJSON}}

	result, err := EvaluateSkill(context.Background(), dir, client, EvalOptions{RefsOnly: true, MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("EvaluateSkill error = %v", err)
	}
	if result.SkillScores != nil {
		t.Error("expected nil SkillScores with RefsOnly")
	}
	if len(result.RefResults) != 1 {
		t.Fatalf("expected 1 ref result, got %d", len(result.RefResults))
	}
	if result.RefResults[0].File != "ref.md" {
		t.Errorf("ref file = %q, want ref.md", result.RefResults[0].File)
	}
}

func TestEvaluateSkill_Both(t *testing.T) {
	dir := makeSkillDir(t, map[string]string{"a.md": "# A", "b.md": "# B"})
	client := &mockLLMClient{responses: []string{skillJSON, refJSON, refJSON}}

	result, err := EvaluateSkill(context.Background(), dir, client, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("EvaluateSkill error = %v", err)
	}
	if result.SkillScores == nil {
		t.Fatal("expected SkillScores")
	}
	if len(result.RefResults) != 2 {
		t.Fatalf("expected 2 ref results, got %d", len(result.RefResults))
	}
	if result.RefAggregate == nil {
		t.Error("expected RefAggregate")
	}
	// Refs should be sorted alphabetically
	if result.RefResults[0].File != "a.md" {
		t.Errorf("first ref = %q, want a.md", result.RefResults[0].File)
	}
}

func TestEvaluateSkill_NoRefs(t *testing.T) {
	dir := makeSkillDir(t, nil)
	client := &mockLLMClient{responses: []string{skillJSON}}

	result, err := EvaluateSkill(context.Background(), dir, client, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("EvaluateSkill error = %v", err)
	}
	if result.SkillScores == nil {
		t.Fatal("expected SkillScores")
	}
	if len(result.RefResults) != 0 {
		t.Errorf("expected 0 ref results, got %d", len(result.RefResults))
	}
	if result.RefAggregate != nil {
		t.Error("expected nil RefAggregate with no refs")
	}
}

func TestEvaluateSkill_BadDir(t *testing.T) {
	client := &mockLLMClient{}
	_, err := EvaluateSkill(context.Background(), "/nonexistent", client, EvalOptions{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
}

func TestEvaluateSkill_LLMError(t *testing.T) {
	dir := makeSkillDir(t, nil)
	client := &mockLLMClient{errors: []error{fmt.Errorf("API down")}}

	_, err := EvaluateSkill(context.Background(), dir, client, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error when LLM fails")
	}
}

func TestEvaluateSkill_CacheRoundTrip(t *testing.T) {
	dir := makeSkillDir(t, nil)
	client := &mockLLMClient{responses: []string{skillJSON}}

	// First call — scores and caches
	result1, err := EvaluateSkill(context.Background(), dir, client, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}

	// Second call — should use cache (no more mock responses needed)
	client2 := &mockLLMClient{} // empty: would fail if called
	result2, err := EvaluateSkill(context.Background(), dir, client2, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("cached call error = %v", err)
	}
	if result2.SkillScores.Clarity != result1.SkillScores.Clarity {
		t.Errorf("cached clarity = %d, want %d", result2.SkillScores.Clarity, result1.SkillScores.Clarity)
	}
}

func TestEvaluateSkill_Rescore(t *testing.T) {
	dir := makeSkillDir(t, nil)
	client := &mockLLMClient{responses: []string{skillJSON}}

	// First call populates cache
	_, err := EvaluateSkill(context.Background(), dir, client, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}

	// Rescore should call LLM again
	client2 := &mockLLMClient{responses: []string{skillJSON}}
	_, err = EvaluateSkill(context.Background(), dir, client2, EvalOptions{Rescore: true, MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("rescore call error = %v", err)
	}
	if client2.callIdx == 0 {
		t.Error("rescore should have called LLM, but callIdx is 0")
	}
}

// --- EvaluateSingleFile tests ---

func TestEvaluateSingleFile_Success(t *testing.T) {
	dir := makeSkillDir(t, map[string]string{"example.md": "# Example ref"})
	refPath := filepath.Join(dir, "references", "example.md")
	client := &mockLLMClient{responses: []string{refJSON}}

	result, err := EvaluateSingleFile(context.Background(), refPath, client, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("EvaluateSingleFile error = %v", err)
	}
	if result.SkillDir != dir {
		t.Errorf("SkillDir = %q, want %q", result.SkillDir, dir)
	}
	if len(result.RefResults) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(result.RefResults))
	}
	if result.RefResults[0].File != "example.md" {
		t.Errorf("ref file = %q, want example.md", result.RefResults[0].File)
	}
}

func TestEvaluateSingleFile_NonMD(t *testing.T) {
	_, err := EvaluateSingleFile(context.Background(), "/tmp/foo.txt", &mockLLMClient{}, EvalOptions{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for non-.md file")
	}
	if !strings.Contains(err.Error(), ".md files") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEvaluateSingleFile_NoParentSkill(t *testing.T) {
	tmp := t.TempDir()
	mdPath := filepath.Join(tmp, "orphan.md")
	if err := os.WriteFile(mdPath, []byte("# Orphan"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := EvaluateSingleFile(context.Background(), mdPath, &mockLLMClient{}, EvalOptions{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for missing parent skill")
	}
}

func TestEvaluateSingleFile_CacheRoundTrip(t *testing.T) {
	dir := makeSkillDir(t, map[string]string{"cached.md": "# Cached"})
	refPath := filepath.Join(dir, "references", "cached.md")
	client := &mockLLMClient{responses: []string{refJSON}}

	// First call — caches
	_, err := EvaluateSingleFile(context.Background(), refPath, client, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}

	// Second call — from cache
	client2 := &mockLLMClient{}
	result, err := EvaluateSingleFile(context.Background(), refPath, client2, EvalOptions{MaxLen: 8000}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("cached call error = %v", err)
	}
	if result.RefResults[0].Scores.Clarity != 4 {
		t.Errorf("cached clarity = %d, want 4", result.RefResults[0].Scores.Clarity)
	}
}

func TestEvaluateSkill_RefScoringError(t *testing.T) {
	dir := makeSkillDir(t, map[string]string{"bad.md": "# Bad"})
	client := &mockLLMClient{
		responses: []string{skillJSON},
		errors:    []error{nil, fmt.Errorf("ref scoring failed")},
	}

	var stderr bytes.Buffer
	result, err := EvaluateSkill(context.Background(), dir, client, EvalOptions{MaxLen: 8000}, &stderr)
	if err != nil {
		t.Fatalf("EvaluateSkill should not fail entirely: %v", err)
	}
	if result.SkillScores == nil {
		t.Error("expected SkillScores even when ref fails")
	}
	if len(result.RefResults) != 0 {
		t.Errorf("expected 0 refs (scoring failed), got %d", len(result.RefResults))
	}
	if !strings.Contains(stderr.String(), "Error scoring") {
		t.Errorf("expected error in stderr, got: %s", stderr.String())
	}
}
