package evaluate

import (
	"bytes"
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
