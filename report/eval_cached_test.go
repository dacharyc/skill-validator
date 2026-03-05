package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/agent-ecosystem/skill-validator/judge"
)

// --- Test data helpers ---

func makeSkillScoresJSON(t *testing.T) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(judge.SkillScores{
		Clarity: 4, Actionability: 5, TokenEfficiency: 3,
		ScopeDiscipline: 4, DirectivePrecision: 4, Novelty: 2,
		Overall: 3.67, BriefAssessment: "Solid skill.",
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func makeRefScoresJSON(t *testing.T) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(judge.RefScores{
		Clarity: 3, InstructionalValue: 4, TokenEfficiency: 3,
		Novelty: 5, SkillRelevance: 4, Overall: 3.80,
		BriefAssessment: "Good ref.", NovelInfo: "Proprietary API.",
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func makeCachedTestResults(t *testing.T) []*judge.CachedResult {
	t.Helper()
	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	return []*judge.CachedResult{
		{Provider: "anthropic", Model: "claude-sonnet", File: "SKILL.md", ScoredAt: now, Scores: makeSkillScoresJSON(t)},
		{Provider: "anthropic", Model: "claude-sonnet", File: "ref.md", ScoredAt: now, Scores: makeRefScoresJSON(t)},
	}
}

// --- ReportList tests ---

func TestReportList_Text(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := List(&buf, results, "/tmp/skill", "text")
	if err != nil {
		t.Fatalf("ReportList text error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Cached scores for:") {
		t.Errorf("expected header, got: %s", out)
	}
	if !strings.Contains(out, "SKILL.md") {
		t.Errorf("expected SKILL.md, got: %s", out)
	}
	if !strings.Contains(out, "ref.md") {
		t.Errorf("expected ref.md, got: %s", out)
	}
	if !strings.Contains(out, "claude-sonnet") {
		t.Errorf("expected model name, got: %s", out)
	}
}

func TestReportList_JSON(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := List(&buf, results, "/tmp/skill", "json")
	if err != nil {
		t.Fatalf("ReportList json error = %v", err)
	}
	var parsed []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 entries, got %d", len(parsed))
	}
}

func TestReportList_Markdown(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := List(&buf, results, "/tmp/skill", "markdown")
	if err != nil {
		t.Fatalf("ReportList markdown error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "## Cached scores for:") {
		t.Errorf("expected markdown header, got: %s", out)
	}
	if !strings.Contains(out, "| File |") {
		t.Errorf("expected table header, got: %s", out)
	}
}

// --- ReportCompare tests ---

func TestReportCompare_Text(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := Compare(&buf, results, "/tmp/skill", "text")
	if err != nil {
		t.Fatalf("ReportCompare text error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Score comparison for:") {
		t.Errorf("expected header, got: %s", out)
	}
	if !strings.Contains(out, "Clarity") {
		t.Errorf("expected Clarity dimension, got: %s", out)
	}
}

func TestReportCompare_JSON(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := Compare(&buf, results, "/tmp/skill", "json")
	if err != nil {
		t.Fatalf("ReportCompare json error = %v", err)
	}
	var parsed []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestReportCompare_Markdown(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := Compare(&buf, results, "/tmp/skill", "markdown")
	if err != nil {
		t.Fatalf("ReportCompare markdown error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "## Score comparison for:") {
		t.Errorf("expected markdown header, got: %s", out)
	}
	if !strings.Contains(out, "| Dimension |") {
		t.Errorf("expected table header, got: %s", out)
	}
}

func TestReportCompare_MultiModel(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	results := []*judge.CachedResult{
		{Provider: "anthropic", Model: "claude-sonnet", File: "SKILL.md", ScoredAt: now, Scores: makeSkillScoresJSON(t)},
		{Provider: "anthropic", Model: "gpt-4o", File: "SKILL.md", ScoredAt: now, Scores: makeSkillScoresJSON(t)},
	}
	var buf bytes.Buffer
	err := Compare(&buf, results, "/tmp/skill", "text")
	if err != nil {
		t.Fatalf("ReportCompare multi-model error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "claude-sonnet") {
		t.Errorf("expected claude-sonnet, got: %s", out)
	}
	if !strings.Contains(out, "gpt-4o") {
		t.Errorf("expected gpt-4o, got: %s", out)
	}
}

// --- ReportDefault tests ---

func TestReportDefault_Text(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := Default(&buf, results, "/tmp/skill", "text")
	if err != nil {
		t.Fatalf("ReportDefault text error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "SKILL.md Scores") {
		t.Errorf("expected SKILL.md header, got: %s", out)
	}
	if !strings.Contains(out, "Reference: ref.md") {
		t.Errorf("expected ref header, got: %s", out)
	}
	if !strings.Contains(out, "3.67/5") {
		t.Errorf("expected overall score, got: %s", out)
	}
}

func TestReportDefault_JSON(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := Default(&buf, results, "/tmp/skill", "json")
	if err != nil {
		t.Fatalf("ReportDefault json error = %v", err)
	}
	var parsed []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestReportDefault_Markdown(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := Default(&buf, results, "/tmp/skill", "markdown")
	if err != nil {
		t.Fatalf("ReportDefault markdown error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "### SKILL.md Scores") {
		t.Errorf("expected markdown skill header, got: %s", out)
	}
	if !strings.Contains(out, "### Reference: ref.md") {
		t.Errorf("expected markdown ref header, got: %s", out)
	}
}

func TestReportDefault_Text_NovelInfo(t *testing.T) {
	results := makeCachedTestResults(t)
	var buf bytes.Buffer
	err := Default(&buf, results, "/tmp/skill", "text")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Solid skill.") {
		t.Errorf("expected assessment, got: %s", out)
	}
	if !strings.Contains(out, "Proprietary API.") {
		t.Errorf("expected novel info, got: %s", out)
	}
}

func TestTruncateModel(t *testing.T) {
	if got := truncateModel("short"); got != "short" {
		t.Errorf("truncateModel(short) = %q", got)
	}
	long := "claude-sonnet-4-5-20250929-v1:0"
	got := truncateModel(long)
	if len(got) > 24 {
		t.Errorf("truncateModel should truncate to <=24 chars, got %q (%d)", got, len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("truncated model should end with ..., got %q", got)
	}
}
