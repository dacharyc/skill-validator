package evaluate

import (
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

func TestResolveCacheDir_Default(t *testing.T) {
	opts := Options{}
	got := resolveCacheDir(opts, "/tmp/skill")
	want := judge.CacheDir("/tmp/skill")
	if got != want {
		t.Errorf("resolveCacheDir default = %q, want %q", got, want)
	}
}

func TestResolveCacheDir_Override(t *testing.T) {
	opts := Options{CacheDir: "/custom/cache"}
	got := resolveCacheDir(opts, "/tmp/skill")
	if got != "/custom/cache" {
		t.Errorf("resolveCacheDir override = %q, want /custom/cache", got)
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

	result, err := EvaluateSkill(context.Background(), dir, client, Options{SkillOnly: true, MaxLen: 8000})
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

	result, err := EvaluateSkill(context.Background(), dir, client, Options{RefsOnly: true, MaxLen: 8000})
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

	result, err := EvaluateSkill(context.Background(), dir, client, Options{MaxLen: 8000})
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

	result, err := EvaluateSkill(context.Background(), dir, client, Options{MaxLen: 8000})
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
	_, err := EvaluateSkill(context.Background(), "/nonexistent", client, Options{})
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
}

func TestEvaluateSkill_LLMError(t *testing.T) {
	dir := makeSkillDir(t, nil)
	client := &mockLLMClient{errors: []error{fmt.Errorf("API down")}}

	_, err := EvaluateSkill(context.Background(), dir, client, Options{MaxLen: 8000})
	if err == nil {
		t.Fatal("expected error when LLM fails")
	}
}

func TestEvaluateSkill_CacheRoundTrip(t *testing.T) {
	dir := makeSkillDir(t, nil)
	client := &mockLLMClient{responses: []string{skillJSON}}

	// First call — scores and caches
	result1, err := EvaluateSkill(context.Background(), dir, client, Options{MaxLen: 8000})
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}

	// Second call — should use cache (no more mock responses needed)
	client2 := &mockLLMClient{} // empty: would fail if called
	result2, err := EvaluateSkill(context.Background(), dir, client2, Options{MaxLen: 8000})
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
	_, err := EvaluateSkill(context.Background(), dir, client, Options{MaxLen: 8000})
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}

	// Rescore should call LLM again
	client2 := &mockLLMClient{responses: []string{skillJSON}}
	_, err = EvaluateSkill(context.Background(), dir, client2, Options{Rescore: true, MaxLen: 8000})
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

	result, err := EvaluateSingleFile(context.Background(), refPath, client, Options{MaxLen: 8000})
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
	_, err := EvaluateSingleFile(context.Background(), "/tmp/foo.txt", &mockLLMClient{}, Options{})
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

	_, err := EvaluateSingleFile(context.Background(), mdPath, &mockLLMClient{}, Options{})
	if err == nil {
		t.Fatal("expected error for missing parent skill")
	}
}

func TestEvaluateSingleFile_CacheRoundTrip(t *testing.T) {
	dir := makeSkillDir(t, map[string]string{"cached.md": "# Cached"})
	refPath := filepath.Join(dir, "references", "cached.md")
	client := &mockLLMClient{responses: []string{refJSON}}

	// First call — caches
	_, err := EvaluateSingleFile(context.Background(), refPath, client, Options{MaxLen: 8000})
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}

	// Second call — from cache
	client2 := &mockLLMClient{}
	result, err := EvaluateSingleFile(context.Background(), refPath, client2, Options{MaxLen: 8000})
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

	var progressEvents []string
	opts := Options{
		MaxLen: 8000,
		Progress: func(event, detail string) {
			progressEvents = append(progressEvents, event+": "+detail)
		},
	}
	result, err := EvaluateSkill(context.Background(), dir, client, opts)
	if err != nil {
		t.Fatalf("EvaluateSkill should not fail entirely: %v", err)
	}
	if result.SkillScores == nil {
		t.Error("expected SkillScores even when ref fails")
	}
	if len(result.RefResults) != 0 {
		t.Errorf("expected 0 refs (scoring failed), got %d", len(result.RefResults))
	}
	found := false
	for _, e := range progressEvents {
		if strings.Contains(e, "error") && strings.Contains(e, "scoring") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error progress event, got: %v", progressEvents)
	}
}
