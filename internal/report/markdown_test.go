package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/validator"
)

func TestPrintMarkdown_Passed(t *testing.T) {
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
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "## Validating skill: /tmp/my-skill") {
		t.Error("expected skill dir heading")
	}
	if !strings.Contains(output, "### Structure") {
		t.Error("expected Structure heading")
	}
	if !strings.Contains(output, "### Frontmatter") {
		t.Error("expected Frontmatter heading")
	}
	if !strings.Contains(output, "- **Pass:** SKILL.md found") {
		t.Error("expected pass result item")
	}
	if !strings.Contains(output, "**Result: passed**") {
		t.Error("expected passed result")
	}
}

func TestPrintMarkdown_WithErrors(t *testing.T) {
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
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "- **Error:** name is required") {
		t.Error("expected error item")
	}
	if !strings.Contains(output, "- **Warning:** unknown directory: extras/") {
		t.Error("expected warning item")
	}
	if !strings.Contains(output, "**Result: 1 error, 1 warning**") {
		t.Errorf("expected result summary, got:\n%s", output)
	}
}

func TestPrintMarkdown_TokenCounts(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		TokenCounts: []validator.TokenCount{
			{File: "SKILL.md body", Tokens: 1250},
			{File: "references/guide.md", Tokens: 820},
		},
	}

	var buf bytes.Buffer
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "### Tokens") {
		t.Error("expected Tokens heading")
	}
	if !strings.Contains(output, "| File | Tokens |") {
		t.Error("expected table header")
	}
	if !strings.Contains(output, "| SKILL.md body | 1,250 |") {
		t.Error("expected SKILL.md body row")
	}
	if !strings.Contains(output, "| references/guide.md | 820 |") {
		t.Error("expected references/guide.md row")
	}
	if !strings.Contains(output, "| **Total** | **2,070** |") {
		t.Error("expected total row")
	}
}

func TestPrintMarkdown_OtherTokenCounts(t *testing.T) {
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
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "### Other files") {
		t.Error("expected Other files heading")
	}
	if !strings.Contains(output, "| AGENTS.md | 45,000 |") {
		t.Error("expected AGENTS.md row")
	}
	if !strings.Contains(output, "| **Total (other)** | **45,850** |") {
		t.Error("expected total other row")
	}
}

func TestPrintMarkdown_ContentAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContentReport: &content.Report{
			WordCount:              1250,
			CodeBlockCount:         5,
			CodeBlockRatio:         0.25,
			ImperativeRatio:        0.45,
			InformationDensity:     0.35,
			InstructionSpecificity: 0.73,
			SectionCount:           4,
			ListItemCount:          12,
		},
	}

	var buf bytes.Buffer
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "### Content Analysis") {
		t.Error("expected Content Analysis heading")
	}
	if !strings.Contains(output, "| Metric | Value |") {
		t.Error("expected table header")
	}
	if !strings.Contains(output, "| Word count | 1,250 |") {
		t.Error("expected word count row")
	}
	if !strings.Contains(output, "| Code block ratio | 0.25 |") {
		t.Error("expected code block ratio row")
	}
	if !strings.Contains(output, "| Sections | 4 |") {
		t.Error("expected sections row")
	}
}

func TestPrintMarkdown_ContaminationAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContaminationReport: &contamination.Report{
			ContaminationLevel:   "high",
			ContaminationScore:   0.7,
			ScopeBreadth:         5,
			PrimaryCategory:      "python",
			MismatchedCategories: []string{"javascript", "ruby"},
			LanguageMismatch:     true,
			MultiInterfaceTools:  []string{"mongodb"},
		},
	}

	var buf bytes.Buffer
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "### Contamination Analysis") {
		t.Error("expected Contamination Analysis heading")
	}
	if !strings.Contains(output, "| Contamination level | high |") {
		t.Error("expected contamination level row")
	}
	if !strings.Contains(output, "| Contamination score | 0.70 |") {
		t.Error("expected contamination score row")
	}
	if !strings.Contains(output, "| Primary language category | python |") {
		t.Error("expected primary category row")
	}
	if !strings.Contains(output, "| Scope breadth | 5 |") {
		t.Error("expected scope breadth row")
	}
	if !strings.Contains(output, "**Warning: Language mismatch:**") {
		t.Error("expected language mismatch warning")
	}
	if !strings.Contains(output, "**Multi-interface tool detected:** mongodb") {
		t.Error("expected multi-interface tool warning")
	}
}

func TestPrintMarkdown_MinimalData(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/minimal",
		Results: []validator.Result{
			{Level: validator.Pass, Category: "Structure", Message: "ok"},
		},
	}

	var buf bytes.Buffer
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()

	if strings.Contains(output, "### Tokens") {
		t.Error("unexpected Tokens section")
	}
	if strings.Contains(output, "### Content Analysis") {
		t.Error("unexpected Content Analysis section")
	}
	if strings.Contains(output, "### Contamination Analysis") {
		t.Error("unexpected Contamination Analysis section")
	}
	if !strings.Contains(output, "**Result: passed**") {
		t.Error("expected passed result")
	}
}

func TestPrintMultiMarkdown(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/alpha",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
			},
			{
				SkillDir: "/tmp/beta",
				Results: []validator.Result{
					{Level: validator.Error, Category: "Frontmatter", Message: "name is required"},
				},
				Errors: 1,
			},
		},
		Errors: 1,
	}

	var buf bytes.Buffer
	if err := PrintMultiMarkdown(&buf, mr, false); err != nil {
		t.Fatalf("PrintMultiMarkdown error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "## Validating skill: /tmp/alpha") {
		t.Error("expected alpha heading")
	}
	if !strings.Contains(output, "## Validating skill: /tmp/beta") {
		t.Error("expected beta heading")
	}
	if !strings.Contains(output, "---") {
		t.Error("expected separator")
	}
	if !strings.Contains(output, "**2 skills validated: 1 passed, 1 failed**") {
		t.Errorf("expected summary, got:\n%s", output)
	}
	if !strings.Contains(output, "**Total: 1 error**") {
		t.Errorf("expected total, got:\n%s", output)
	}
}

func TestPrintMultiMarkdown_AllPassed(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/a",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
			},
			{
				SkillDir: "/tmp/b",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
			},
		},
	}

	var buf bytes.Buffer
	if err := PrintMultiMarkdown(&buf, mr, false); err != nil {
		t.Fatalf("PrintMultiMarkdown error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "all passed**") {
		t.Errorf("expected 'all passed', got:\n%s", output)
	}
	// No Total line when no errors or warnings
	if strings.Contains(output, "**Total:") {
		t.Error("unexpected Total line when no errors or warnings")
	}
}

func TestPrintMarkdown_NoAnsiCodes(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results: []validator.Result{
			{Level: validator.Pass, Category: "Structure", Message: "SKILL.md found"},
			{Level: validator.Error, Category: "Frontmatter", Message: "name is required"},
			{Level: validator.Warning, Category: "Structure", Message: "unknown dir"},
		},
		Errors:   1,
		Warnings: 1,
		TokenCounts: []validator.TokenCount{
			{File: "SKILL.md body", Tokens: 1250},
		},
		ContentReport: &content.Report{
			WordCount:              500,
			CodeBlockRatio:         0.2,
			ImperativeRatio:        0.5,
			InformationDensity:     0.35,
			InstructionSpecificity: 0.71,
			SectionCount:           3,
			ListItemCount:          8,
			CodeBlockCount:         2,
		},
		ContaminationReport: &contamination.Report{
			ContaminationLevel: "medium",
			ContaminationScore: 0.4,
			ScopeBreadth:       3,
			PrimaryCategory:    "python",
		},
	}

	var buf bytes.Buffer
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()

	if strings.Contains(output, "\033[") {
		t.Error("markdown output contains ANSI escape codes")
	}
}

func TestPrintMarkdown_PerFileReports(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
		ReferenceReports: []validator.ReferenceFileReport{
			{
				File: "guide.md",
				ContentReport: &content.Report{
					WordCount:              200,
					CodeBlockRatio:         0.1,
					ImperativeRatio:        0.3,
					InformationDensity:     0.2,
					InstructionSpecificity: 0.5,
					SectionCount:           2,
					ListItemCount:          4,
					CodeBlockCount:         1,
				},
				ContaminationReport: &contamination.Report{
					ContaminationLevel: "low",
					ContaminationScore: 0.0,
					ScopeBreadth:       1,
				},
			},
		},
	}

	// Without perFile, reference reports should not appear
	var buf bytes.Buffer
	if err := PrintMarkdown(&buf, r, false); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output := buf.String()
	if strings.Contains(output, "[guide.md]") {
		t.Error("per-file reports should not appear when perFile is false")
	}

	// With perFile, reference reports should appear
	buf.Reset()
	if err := PrintMarkdown(&buf, r, true); err != nil {
		t.Fatalf("PrintMarkdown error: %v", err)
	}
	output = buf.String()
	if !strings.Contains(output, "### [guide.md] Content Analysis") {
		t.Error("expected per-file content analysis heading")
	}
	if !strings.Contains(output, "### [guide.md] Contamination Analysis") {
		t.Error("expected per-file contamination analysis heading")
	}
}
