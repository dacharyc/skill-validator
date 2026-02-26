package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/validator"
)

func TestPrintAnnotations_ErrorAndWarning(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/workspace/skills/my-skill",
		Results: []validator.Result{
			{Level: validator.Error, Category: "Frontmatter", Message: "name is required", File: "SKILL.md"},
			{Level: validator.Warning, Category: "Structure", Message: "extraneous file", File: "README.md"},
		},
	}

	var buf bytes.Buffer
	PrintAnnotations(&buf, r, "/workspace")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}

	// Path should be relative to workDir
	if !strings.Contains(lines[0], "file=skills/my-skill/SKILL.md") {
		t.Errorf("expected relative path skills/my-skill/SKILL.md, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "title=Frontmatter") {
		t.Errorf("expected title=Frontmatter, got %q", lines[0])
	}
	if !strings.HasSuffix(lines[0], "::name is required") {
		t.Errorf("expected message suffix, got %q", lines[0])
	}

	if !strings.HasPrefix(lines[1], "::warning file=") {
		t.Errorf("expected ::warning prefix, got %q", lines[1])
	}
}

func TestPrintAnnotations_SkipsPassAndInfo(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/workspace/skills/my-skill",
		Results: []validator.Result{
			{Level: validator.Pass, Category: "Structure", Message: "SKILL.md found", File: "SKILL.md"},
			{Level: validator.Info, Category: "Links", Message: "HTTP 403", File: "SKILL.md"},
		},
	}

	var buf bytes.Buffer
	PrintAnnotations(&buf, r, "/workspace")

	if buf.Len() != 0 {
		t.Errorf("expected no output for Pass/Info, got %q", buf.String())
	}
}

func TestPrintAnnotations_WithLineNumber(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/workspace/skills/my-skill",
		Results: []validator.Result{
			{Level: validator.Error, Category: "Markdown", Message: "unclosed fence", File: "SKILL.md", Line: 42},
		},
	}

	var buf bytes.Buffer
	PrintAnnotations(&buf, r, "/workspace")

	line := strings.TrimSpace(buf.String())
	if !strings.Contains(line, "file=skills/my-skill/SKILL.md") {
		t.Errorf("expected relative path, got %q", line)
	}
	if !strings.Contains(line, "line=42") {
		t.Errorf("expected line=42, got %q", line)
	}
}

func TestPrintAnnotations_NoFile(t *testing.T) {
	r := &validator.Report{
		SkillDir: "skills/my-skill",
		Results: []validator.Result{
			{Level: validator.Error, Category: "Overall", Message: "not a skill"},
		},
	}

	var buf bytes.Buffer
	PrintAnnotations(&buf, r, ".")

	line := strings.TrimSpace(buf.String())
	expected := "::error title=Overall::not a skill"
	if line != expected {
		t.Errorf("expected %q, got %q", expected, line)
	}
}

func TestPrintMultiAnnotations(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/workspace/skills/a",
				Results: []validator.Result{
					{Level: validator.Error, Category: "Structure", Message: "missing", File: "SKILL.md"},
				},
			},
			{
				SkillDir: "/workspace/skills/b",
				Results: []validator.Result{
					{Level: validator.Warning, Category: "Tokens", Message: "too large", File: "references/big.md"},
				},
			},
		},
	}

	var buf bytes.Buffer
	PrintMultiAnnotations(&buf, mr, "/workspace")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "file=skills/a/SKILL.md") {
		t.Errorf("expected skills/a/SKILL.md path, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "file=skills/b/references/big.md") {
		t.Errorf("expected skills/b/references/big.md path, got %q", lines[1])
	}
}
