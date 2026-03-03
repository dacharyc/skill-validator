package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/evaluate"
	"github.com/dacharyc/skill-validator/judge"
	"github.com/dacharyc/skill-validator/util"
)

func TestPrintEvalText(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir: "/tmp/my-skill",
		SkillScores: &judge.SkillScores{
			Clarity: 4, Actionability: 3, TokenEfficiency: 5,
			ScopeDiscipline: 4, DirectivePrecision: 4, Novelty: 3,
			Overall: 3.83, BriefAssessment: "Good skill",
		},
	}

	var buf bytes.Buffer
	PrintEvalText(&buf, result, "aggregate")
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

func TestPrintEvalJSON(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir:    "/tmp/my-skill",
		SkillScores: &judge.SkillScores{Clarity: 4, Overall: 4.0},
	}

	var buf bytes.Buffer
	err := PrintEvalJSON(&buf, []*evaluate.EvalResult{result})
	if err != nil {
		t.Fatalf("PrintEvalJSON() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"skill_dir"`) {
		t.Errorf("expected JSON skill_dir field, got: %s", out)
	}
	if !strings.Contains(out, `"clarity"`) {
		t.Errorf("expected JSON clarity field, got: %s", out)
	}
}

func TestPrintEvalMarkdown(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir: "/tmp/my-skill",
		SkillScores: &judge.SkillScores{
			Clarity: 4, Actionability: 3, TokenEfficiency: 5,
			ScopeDiscipline: 4, DirectivePrecision: 4, Novelty: 3,
			Overall: 3.83, BriefAssessment: "Good skill",
		},
	}

	var buf bytes.Buffer
	PrintEvalMarkdown(&buf, result, "aggregate")
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

func TestFormatEvalResults_SingleText(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir:    "/tmp/test",
		SkillScores: &judge.SkillScores{Overall: 4.0},
	}

	var buf bytes.Buffer
	err := FormatEvalResults(&buf, []*evaluate.EvalResult{result}, "text", "aggregate")
	if err != nil {
		t.Fatalf("FormatEvalResults() error = %v", err)
	}
	if !strings.Contains(buf.String(), "Scoring skill:") {
		t.Errorf("expected text output, got: %s", buf.String())
	}
}

func TestFormatEvalResults_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := FormatEvalResults(&buf, nil, "text", "aggregate")
	if err != nil {
		t.Fatalf("FormatEvalResults() error = %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got: %s", buf.String())
	}
}

func TestPrintMultiEvalMarkdown(t *testing.T) {
	results := []*evaluate.EvalResult{
		{SkillDir: "/tmp/skill-a", SkillScores: &judge.SkillScores{Overall: 4.0}},
		{SkillDir: "/tmp/skill-b", SkillScores: &judge.SkillScores{Overall: 3.0}},
	}

	var buf bytes.Buffer
	PrintMultiEvalMarkdown(&buf, results, "aggregate")
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

func TestPrintEvalText_WithRefs(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir: "/tmp/my-skill",
		RefResults: []evaluate.RefEvalResult{
			{
				File: "example.md",
				Scores: &judge.RefScores{
					Clarity: 4, InstructionalValue: 3, TokenEfficiency: 5,
					Novelty: 4, SkillRelevance: 4, Overall: 4.0,
					BriefAssessment: "Good ref",
				},
			},
		},
		RefAggregate: &judge.RefScores{
			Clarity: 4, InstructionalValue: 3, TokenEfficiency: 5,
			Novelty: 4, SkillRelevance: 4, Overall: 4.0,
		},
	}

	var buf bytes.Buffer
	PrintEvalText(&buf, result, "files")
	out := buf.String()
	if !strings.Contains(out, "Reference: example.md") {
		t.Errorf("expected ref header in files mode, got: %s", out)
	}

	buf.Reset()
	PrintEvalText(&buf, result, "aggregate")
	out = buf.String()
	if strings.Contains(out, "Reference: example.md") {
		t.Errorf("should not show individual refs in aggregate mode, got: %s", out)
	}
	if !strings.Contains(out, "Reference Scores (1 file)") {
		t.Errorf("expected aggregate ref header, got: %s", out)
	}
}

func TestFormatEvalResults_SingleJSON(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir:    "/tmp/test",
		SkillScores: &judge.SkillScores{Clarity: 4, Overall: 4.0},
	}

	var buf bytes.Buffer
	err := FormatEvalResults(&buf, []*evaluate.EvalResult{result}, "json", "aggregate")
	if err != nil {
		t.Fatalf("FormatEvalResults(json) error = %v", err)
	}
	if !strings.Contains(buf.String(), `"skill_dir"`) {
		t.Errorf("expected JSON output, got: %s", buf.String())
	}
}

func TestFormatEvalResults_SingleMarkdown(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir:    "/tmp/test",
		SkillScores: &judge.SkillScores{Clarity: 4, Overall: 4.0},
	}

	var buf bytes.Buffer
	err := FormatEvalResults(&buf, []*evaluate.EvalResult{result}, "markdown", "aggregate")
	if err != nil {
		t.Fatalf("FormatEvalResults(markdown) error = %v", err)
	}
	if !strings.Contains(buf.String(), "## Scoring skill:") {
		t.Errorf("expected markdown output, got: %s", buf.String())
	}
}

func TestFormatMultiEvalResults_Text(t *testing.T) {
	results := []*evaluate.EvalResult{
		{SkillDir: "/tmp/a", SkillScores: &judge.SkillScores{Overall: 4.0}},
		{SkillDir: "/tmp/b", SkillScores: &judge.SkillScores{Overall: 3.0}},
	}

	var buf bytes.Buffer
	err := FormatMultiEvalResults(&buf, results, "text", "aggregate")
	if err != nil {
		t.Fatalf("FormatMultiEvalResults(text) error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "/tmp/a") || !strings.Contains(out, "/tmp/b") {
		t.Errorf("expected both skills, got: %s", out)
	}
	if !strings.Contains(out, "━") {
		t.Errorf("expected separator, got: %s", out)
	}
}

func TestFormatMultiEvalResults_JSON(t *testing.T) {
	results := []*evaluate.EvalResult{
		{SkillDir: "/tmp/a", SkillScores: &judge.SkillScores{Overall: 4.0}},
		{SkillDir: "/tmp/b", SkillScores: &judge.SkillScores{Overall: 3.0}},
	}

	var buf bytes.Buffer
	err := FormatMultiEvalResults(&buf, results, "json", "aggregate")
	if err != nil {
		t.Fatalf("FormatMultiEvalResults(json) error = %v", err)
	}
	if !strings.Contains(buf.String(), "/tmp/a") {
		t.Errorf("expected skill dir in JSON, got: %s", buf.String())
	}
}

func TestFormatMultiEvalResults_Markdown(t *testing.T) {
	results := []*evaluate.EvalResult{
		{SkillDir: "/tmp/a", SkillScores: &judge.SkillScores{Overall: 4.0}},
		{SkillDir: "/tmp/b", SkillScores: &judge.SkillScores{Overall: 3.0}},
	}

	var buf bytes.Buffer
	err := FormatMultiEvalResults(&buf, results, "markdown", "aggregate")
	if err != nil {
		t.Fatalf("FormatMultiEvalResults(markdown) error = %v", err)
	}
	if !strings.Contains(buf.String(), "---") {
		t.Errorf("expected markdown separator, got: %s", buf.String())
	}
}

func TestFormatEvalResults_MultiDelegates(t *testing.T) {
	results := []*evaluate.EvalResult{
		{SkillDir: "/tmp/a"},
		{SkillDir: "/tmp/b"},
	}

	var buf bytes.Buffer
	err := FormatEvalResults(&buf, results, "text", "aggregate")
	if err != nil {
		t.Fatalf("FormatEvalResults with 2 results error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "/tmp/a") || !strings.Contains(out, "/tmp/b") {
		t.Errorf("expected both skills, got: %s", out)
	}
}

func TestPrintEvalMarkdown_WithRefsFiles(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir:    "/tmp/my-skill",
		SkillScores: &judge.SkillScores{Clarity: 4, Overall: 4.0},
		RefResults: []evaluate.RefEvalResult{
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
	PrintEvalMarkdown(&buf, result, "files")
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

func TestPrintEvalMarkdown_WithNovelInfo(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir: "/tmp/test",
		SkillScores: &judge.SkillScores{
			Clarity: 4, Overall: 4.0,
			BriefAssessment: "Assessment", NovelInfo: "Internal API",
		},
	}

	var buf bytes.Buffer
	PrintEvalMarkdown(&buf, result, "aggregate")
	out := buf.String()

	if !strings.Contains(out, "> Assessment") {
		t.Errorf("expected assessment blockquote, got: %s", out)
	}
	if !strings.Contains(out, "*Novel details: Internal API*") {
		t.Errorf("expected novel info, got: %s", out)
	}
}

func TestPrintEvalText_NovelInfo(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir: "/tmp/test",
		SkillScores: &judge.SkillScores{
			Clarity: 4, Overall: 4.0,
			NovelInfo: "Proprietary details",
		},
	}

	var buf bytes.Buffer
	PrintEvalText(&buf, result, "aggregate")
	out := buf.String()
	if !strings.Contains(out, "Novel details: Proprietary details") {
		t.Errorf("expected novel info in text, got: %s", out)
	}
}

func TestPrintEvalText_RefFilesWithNovelInfo(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir: "/tmp/test",
		RefResults: []evaluate.RefEvalResult{
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
	PrintEvalText(&buf, result, "files")
	out := buf.String()
	if !strings.Contains(out, "Novel details: Internal endpoint") {
		t.Errorf("expected ref novel info, got: %s", out)
	}
}

func TestPrintEvalJSON_WithRefs(t *testing.T) {
	result := &evaluate.EvalResult{
		SkillDir: "/tmp/test",
		RefResults: []evaluate.RefEvalResult{
			{File: "ref.md", Scores: &judge.RefScores{Clarity: 4, Overall: 4.0}},
		},
		RefAggregate: &judge.RefScores{Clarity: 4, Overall: 4.0},
	}

	var buf bytes.Buffer
	err := PrintEvalJSON(&buf, []*evaluate.EvalResult{result})
	if err != nil {
		t.Fatalf("PrintEvalJSON error = %v", err)
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
	// Test via PrintEvalText with scores that trigger different color thresholds
	highResult := &evaluate.EvalResult{
		SkillDir:    "/tmp/test",
		SkillScores: &judge.SkillScores{Clarity: 5, Overall: 5.0},
	}
	var buf bytes.Buffer
	PrintEvalText(&buf, highResult, "aggregate")
	if !strings.Contains(buf.String(), util.ColorGreen) {
		t.Errorf("score 5 should use green, got: %s", buf.String())
	}

	medResult := &evaluate.EvalResult{
		SkillDir:    "/tmp/test",
		SkillScores: &judge.SkillScores{Clarity: 3, Overall: 3.0},
	}
	buf.Reset()
	PrintEvalText(&buf, medResult, "aggregate")
	if !strings.Contains(buf.String(), util.ColorYellow) {
		t.Errorf("score 3 should use yellow, got: %s", buf.String())
	}

	lowResult := &evaluate.EvalResult{
		SkillDir:    "/tmp/test",
		SkillScores: &judge.SkillScores{Clarity: 2, Overall: 2.0},
	}
	buf.Reset()
	PrintEvalText(&buf, lowResult, "aggregate")
	if !strings.Contains(buf.String(), util.ColorRed) {
		t.Errorf("score 2 should use red, got: %s", buf.String())
	}
}
