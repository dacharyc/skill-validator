package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/validator"
)

func TestPrint_Passed(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/my-skill",
		Results: []validator.Result{
			{Level: validator.Pass, Category: "Structure", Message: "SKILL.md found"},
			{Level: validator.Pass, Category: "Frontmatter", Message: `name: "my-skill" (valid)`},
		},
		Errors:   0,
		Warnings: 0,
	}

	var buf bytes.Buffer
	Print(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Validating skill: /tmp/my-skill") {
		t.Error("expected skill dir in output")
	}
	if !strings.Contains(output, "Structure") {
		t.Error("expected Structure category")
	}
	if !strings.Contains(output, "Frontmatter") {
		t.Error("expected Frontmatter category")
	}
	if !strings.Contains(output, "Result: passed") {
		t.Error("expected passed result")
	}
}

func TestPrint_WithErrors(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/bad-skill",
		Results: []validator.Result{
			{Level: validator.Pass, Category: "Structure", Message: "SKILL.md found"},
			{Level: validator.Error, Category: "Frontmatter", Message: "name is required"},
			{Level: validator.Warning, Category: "Structure", Message: "unknown directory: extras/"},
		},
		Errors:   1,
		Warnings: 1,
	}

	var buf bytes.Buffer
	Print(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "1 error") {
		t.Errorf("expected '1 error' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "1 warning") {
		t.Errorf("expected '1 warning' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "✗") {
		t.Error("expected error icon ✗")
	}
	if !strings.Contains(output, "⚠") {
		t.Error("expected warning icon ⚠")
	}
	if !strings.Contains(output, "✓") {
		t.Error("expected pass icon ✓")
	}
}

func TestPrint_Pluralization(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results: []validator.Result{
			{Level: validator.Error, Category: "A", Message: "err1"},
			{Level: validator.Error, Category: "A", Message: "err2"},
			{Level: validator.Warning, Category: "B", Message: "warn1"},
			{Level: validator.Warning, Category: "B", Message: "warn2"},
			{Level: validator.Warning, Category: "B", Message: "warn3"},
		},
		Errors:   2,
		Warnings: 3,
	}

	var buf bytes.Buffer
	Print(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "2 errors") {
		t.Errorf("expected '2 errors' in output")
	}
	if !strings.Contains(output, "3 warnings") {
		t.Errorf("expected '3 warnings' in output")
	}
}

func TestPrint_TokenCounts(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		TokenCounts: []validator.TokenCount{
			{File: "SKILL.md body", Tokens: 1250},
			{File: "references/guide.md", Tokens: 820},
		},
		Errors:   0,
		Warnings: 0,
	}

	var buf bytes.Buffer
	Print(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Tokens") {
		t.Error("expected Tokens section")
	}
	if !strings.Contains(output, "SKILL.md body:") {
		t.Error("expected SKILL.md body in token counts")
	}
	if !strings.Contains(output, "references/guide.md:") {
		t.Error("expected references/guide.md in token counts")
	}
	if !strings.Contains(output, "1,250") {
		t.Errorf("expected formatted number 1,250 in output")
	}
	if !strings.Contains(output, "Total:") {
		t.Error("expected Total in token counts")
	}
	if !strings.Contains(output, "2,070") {
		t.Errorf("expected formatted total 2,070 in output")
	}
}

func TestPrint_NoTokenCounts(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results: []validator.Result{
			{Level: validator.Error, Category: "Structure", Message: "SKILL.md not found"},
		},
		Errors: 1,
	}

	var buf bytes.Buffer
	Print(&buf, r)
	output := buf.String()

	if strings.Contains(output, "Tokens\n") {
		t.Error("unexpected Tokens section when no counts")
	}
}

func TestPrint_CategoryGrouping(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results: []validator.Result{
			{Level: validator.Pass, Category: "Structure", Message: "a"},
			{Level: validator.Pass, Category: "Frontmatter", Message: "b"},
			{Level: validator.Pass, Category: "Structure", Message: "c"},
		},
	}

	var buf bytes.Buffer
	Print(&buf, r)
	output := buf.String()

	// Structure should appear before Frontmatter (first-appearance order)
	structIdx := strings.Index(output, "Structure")
	fmIdx := strings.Index(output, "Frontmatter")
	if structIdx > fmIdx {
		t.Error("expected Structure before Frontmatter in output")
	}

	// Structure should appear only once as a header (with ANSI bold codes)
	count := strings.Count(output, "Structure")
	if count != 1 {
		t.Errorf("expected Structure to appear once, got %d", count)
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{1250, "1,250"},
		{12345, "12,345"},
		{1000000, "1,000,000"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.input)
		if got != tt.want {
			t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPrint_OtherTokenCounts(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		TokenCounts: []validator.TokenCount{
			{File: "SKILL.md body", Tokens: 1250},
		},
		OtherTokenCounts: []validator.TokenCount{
			{File: "AGENTS.md", Tokens: 45000},
			{File: "rules/rule1.md", Tokens: 850},
		},
	}

	var buf bytes.Buffer
	Print(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Other files (outside standard structure)") {
		t.Error("expected Other files section header")
	}
	if !strings.Contains(output, "AGENTS.md:") {
		t.Error("expected AGENTS.md in other token counts")
	}
	if !strings.Contains(output, "rules/rule1.md:") {
		t.Error("expected rules/rule1.md in other token counts")
	}
	if !strings.Contains(output, "45,000") {
		t.Error("expected formatted number 45,000")
	}
	if !strings.Contains(output, "Total (other):") {
		t.Error("expected Total (other) in output")
	}
	if !strings.Contains(output, "45,850") {
		t.Errorf("expected formatted total 45,850 in output")
	}
}

func TestPrint_NoOtherTokenCounts(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		TokenCounts: []validator.TokenCount{
			{File: "SKILL.md body", Tokens: 1250},
		},
	}

	var buf bytes.Buffer
	Print(&buf, r)
	output := buf.String()

	if strings.Contains(output, "Other files") {
		t.Error("unexpected Other files section when no other counts")
	}
}

func TestPluralize(t *testing.T) {
	if pluralize(0) != "s" {
		t.Error("pluralize(0) should be 's'")
	}
	if pluralize(1) != "" {
		t.Error("pluralize(1) should be ''")
	}
	if pluralize(2) != "s" {
		t.Error("pluralize(2) should be 's'")
	}
}
