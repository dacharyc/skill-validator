package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
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
	Print(&buf, r, false)
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
	Print(&buf, r, false)
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

func TestPrint_InfoLevel(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/info-skill",
		Results: []validator.Result{
			{Level: validator.Pass, Category: "Structure", Message: "SKILL.md found"},
			{Level: validator.Info, Category: "Links", Message: "https://example.com (HTTP 403 — may block automated requests)"},
		},
		Errors:   0,
		Warnings: 0,
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	if !strings.Contains(output, "ℹ") {
		t.Error("expected info icon ℹ")
	}
	if !strings.Contains(output, "HTTP 403") {
		t.Error("expected HTTP 403 message")
	}
	if !strings.Contains(output, "Result: passed") {
		t.Error("expected passed result (info should not block passing)")
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
	Print(&buf, r, false)
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
	Print(&buf, r, false)
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
	Print(&buf, r, false)
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
	Print(&buf, r, false)
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
	Print(&buf, r, false)
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

func TestPrint_OtherTokenCountsColors(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		OtherTokenCounts: []validator.TokenCount{
			{File: "small.md", Tokens: 500},
			{File: "medium.md", Tokens: 15000},
			{File: "large.md", Tokens: 40000},
		},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	// small.md (500 tokens) should have no warning/error color on the count
	// Find the line with small.md and check it doesn't have yellow or red before "500"
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "small.md") {
			if strings.Contains(line, "\033[33m500") || strings.Contains(line, "\033[31m500") {
				t.Error("small.md count should not be colored")
			}
		}
		// medium.md (15,000 tokens) should be yellow
		if strings.Contains(line, "medium.md") {
			if !strings.Contains(line, "\033[33m15,000") {
				t.Error("medium.md count should be yellow")
			}
		}
		// large.md (40,000 tokens) should be red
		if strings.Contains(line, "large.md") {
			if !strings.Contains(line, "\033[31m40,000") {
				t.Error("large.md count should be red")
			}
		}
		// Total (55,500) should be yellow (over 25k, under 100k)
		if strings.Contains(line, "Total (other)") {
			if !strings.Contains(line, "\033[33m55,500") {
				t.Errorf("total should be yellow, got line: %q", line)
			}
		}
	}
}

func TestPrint_OtherTokenCountsTotalRed(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		OtherTokenCounts: []validator.TokenCount{
			{File: "huge1.md", Tokens: 60000},
			{File: "huge2.md", Tokens: 50000},
		},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Total (other)") {
			if !strings.Contains(line, "\033[31m110,000") {
				t.Errorf("total over 100k should be red, got line: %q", line)
			}
		}
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
	Print(&buf, r, false)
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

func TestPrintMulti_AllPassed(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/alpha",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "SKILL.md found"}},
			},
			{
				SkillDir: "/tmp/beta",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "SKILL.md found"}},
			},
		},
	}

	var buf bytes.Buffer
	PrintMulti(&buf, mr, false)
	output := buf.String()

	if !strings.Contains(output, "Validating skill: /tmp/alpha") {
		t.Error("expected alpha in output")
	}
	if !strings.Contains(output, "Validating skill: /tmp/beta") {
		t.Error("expected beta in output")
	}
	if !strings.Contains(output, "━") {
		t.Error("expected separator line")
	}
	if !strings.Contains(output, "2 skills validated") {
		t.Error("expected '2 skills validated' summary")
	}
	if !strings.Contains(output, "all passed") {
		t.Error("expected 'all passed' in summary")
	}
	// No Total line when there are no errors or warnings
	if strings.Contains(output, "Total:") {
		t.Error("unexpected Total line when no errors or warnings")
	}
}

func TestPrintMulti_SomeFailed(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/good",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
			},
			{
				SkillDir: "/tmp/bad",
				Results: []validator.Result{
					{Level: validator.Error, Category: "Structure", Message: "fail"},
					{Level: validator.Warning, Category: "Structure", Message: "warn"},
				},
				Errors:   1,
				Warnings: 1,
			},
		},
		Errors:   1,
		Warnings: 1,
	}

	var buf bytes.Buffer
	PrintMulti(&buf, mr, false)
	output := buf.String()

	if !strings.Contains(output, "2 skills validated") {
		t.Error("expected '2 skills validated'")
	}
	if !strings.Contains(output, "1 passed") {
		t.Error("expected '1 passed'")
	}
	if !strings.Contains(output, "1 failed") {
		t.Error("expected '1 failed'")
	}
	if !strings.Contains(output, "Total:") {
		t.Error("expected 'Total:' line with error/warning counts")
	}
	if !strings.Contains(output, "1 error") {
		t.Errorf("expected '1 error' in total line, got:\n%s", output)
	}
	if !strings.Contains(output, "1 warning") {
		t.Errorf("expected '1 warning' in total line, got:\n%s", output)
	}
}

func TestPrintMulti_SingleSkill(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/only",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
			},
		},
	}

	var buf bytes.Buffer
	PrintMulti(&buf, mr, false)
	output := buf.String()

	// Singular: "1 skill validated"
	if !strings.Contains(output, "1 skill validated") {
		t.Errorf("expected '1 skill validated' (singular), got:\n%s", output)
	}
}

func TestPrint_ContentAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContentReport: &content.Report{
			WordCount:              1250,
			CodeBlockCount:         5,
			CodeBlockRatio:         0.25,
			CodeLanguages:          []string{"python", "bash"},
			SentenceCount:          40,
			ImperativeCount:        18,
			ImperativeRatio:        0.45,
			InformationDensity:     0.35,
			StrongMarkers:          8,
			WeakMarkers:            3,
			InstructionSpecificity: 0.73,
			SectionCount:           4,
			ListItemCount:          12,
		},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	if !strings.Contains(output, "Content Analysis") {
		t.Error("expected Content Analysis heading")
	}
	if !strings.Contains(output, "Word count:") {
		t.Error("expected Word count line")
	}
	if !strings.Contains(output, "1,250") {
		t.Error("expected formatted word count 1,250")
	}
	if !strings.Contains(output, "Code block ratio:") {
		t.Error("expected Code block ratio line")
	}
	if !strings.Contains(output, "Imperative ratio:") {
		t.Error("expected Imperative ratio line")
	}
	if !strings.Contains(output, "Information density:") {
		t.Error("expected Information density line")
	}
	if !strings.Contains(output, "Instruction specificity:") {
		t.Error("expected Instruction specificity line")
	}
	if !strings.Contains(output, "Sections: 4") {
		t.Error("expected Sections: 4")
	}
	if !strings.Contains(output, "List items: 12") {
		t.Error("expected List items: 12")
	}
	if !strings.Contains(output, "Code blocks: 5") {
		t.Error("expected Code blocks: 5")
	}
}

func TestPrint_NoContentAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	if strings.Contains(output, "Content Analysis") {
		t.Error("unexpected Content Analysis when ContentReport is nil")
	}
}

func TestPrint_ContaminationAnalysis_Low(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContaminationReport: &contamination.Report{
			ContaminationLevel: "low",
			ContaminationScore: 0.0,
			ScopeBreadth:       1,
			CodeLanguages:      []string{"python"},
			LanguageCategories: []string{"python"},
			PrimaryCategory:    "python",
		},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	if !strings.Contains(output, "Contamination Analysis") {
		t.Error("expected Contamination Analysis heading")
	}
	if !strings.Contains(output, "Contamination level:") {
		t.Error("expected Contamination level line")
	}
	if !strings.Contains(output, "low") {
		t.Error("expected 'low' level")
	}
	// Low level should use green
	if !strings.Contains(output, colorGreen+"low") {
		t.Error("expected green color for low level")
	}
	if !strings.Contains(output, "Primary language category: python") {
		t.Error("expected primary language category")
	}
	if !strings.Contains(output, "Scope breadth: 1") {
		t.Error("expected scope breadth")
	}
	// Should NOT show mismatch or multi-interface tool warnings
	if strings.Contains(output, "Language mismatch") {
		t.Error("unexpected language mismatch for low contamination")
	}
	if strings.Contains(output, "Multi-interface tool") {
		t.Error("unexpected multi-interface tool for simple skill")
	}
}

func TestPrint_ContaminationAnalysis_Medium(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContaminationReport: &contamination.Report{
			ContaminationLevel:   "medium",
			ContaminationScore:   0.35,
			ScopeBreadth:         3,
			CodeLanguages:        []string{"python", "bash"},
			LanguageCategories:   []string{"python", "shell"},
			PrimaryCategory:      "python",
			MismatchedCategories: []string{"shell"},
			LanguageMismatch:     true,
		},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	if !strings.Contains(output, colorYellow+"medium") {
		t.Error("expected yellow color for medium level")
	}
	if !strings.Contains(output, "Language mismatch: shell") {
		t.Error("expected language mismatch warning with shell")
	}
	if !strings.Contains(output, "1 category differ from primary") {
		t.Error("expected singular category count")
	}
}

func TestPrint_ContaminationAnalysis_High(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContaminationReport: &contamination.Report{
			ContaminationLevel:   "high",
			ContaminationScore:   0.7,
			ScopeBreadth:         5,
			CodeLanguages:        []string{"python", "javascript", "bash", "ruby"},
			LanguageCategories:   []string{"python", "javascript", "shell", "ruby"},
			PrimaryCategory:      "python",
			MismatchedCategories: []string{"javascript", "ruby", "shell"},
			LanguageMismatch:     true,
			MultiInterfaceTools:  []string{"mongodb"},
		},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	if !strings.Contains(output, colorRed+"high") {
		t.Error("expected red color for high level")
	}
	if !strings.Contains(output, "Multi-interface tool detected: mongodb") {
		t.Error("expected multi-interface tool warning")
	}
	if !strings.Contains(output, "3 categories differ") {
		t.Error("expected plural categories count")
	}
	if !strings.Contains(output, "Scope breadth: 5") {
		t.Error("expected scope breadth 5")
	}
}

func TestPrint_NoContaminationAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	if strings.Contains(output, "Contamination Analysis") {
		t.Error("unexpected Contamination Analysis when ContaminationReport is nil")
	}
}

func TestPrint_ContaminationAnalysis_NoPrimaryCategory(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContaminationReport: &contamination.Report{
			ContaminationLevel: "low",
			ContaminationScore: 0.0,
			ScopeBreadth:       0,
		},
	}

	var buf bytes.Buffer
	Print(&buf, r, false)
	output := buf.String()

	if strings.Contains(output, "Primary language category:") {
		t.Error("unexpected Primary language category when none set")
	}
}

func TestPrintMulti_AggregatedCounts(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/a",
				Results: []validator.Result{
					{Level: validator.Error, Category: "A", Message: "e1"},
					{Level: validator.Error, Category: "A", Message: "e2"},
					{Level: validator.Warning, Category: "A", Message: "w1"},
				},
				Errors:   2,
				Warnings: 1,
			},
			{
				SkillDir: "/tmp/b",
				Results: []validator.Result{
					{Level: validator.Error, Category: "A", Message: "e3"},
					{Level: validator.Warning, Category: "A", Message: "w2"},
					{Level: validator.Warning, Category: "A", Message: "w3"},
				},
				Errors:   1,
				Warnings: 2,
			},
		},
		Errors:   3,
		Warnings: 3,
	}

	var buf bytes.Buffer
	PrintMulti(&buf, mr, false)
	output := buf.String()

	if !strings.Contains(output, "3 errors") {
		t.Errorf("expected '3 errors' in total, got:\n%s", output)
	}
	if !strings.Contains(output, "3 warnings") {
		t.Errorf("expected '3 warnings' in total, got:\n%s", output)
	}
}
