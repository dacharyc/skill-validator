package validator

import "testing"

func TestResultContext_DefaultFile(t *testing.T) {
	ctx := ResultContext{Category: "Frontmatter", File: "SKILL.md"}

	r := ctx.Error("name is required")
	if r.Level != Error {
		t.Errorf("expected Error level, got %v", r.Level)
	}
	if r.Category != "Frontmatter" {
		t.Errorf("expected category Frontmatter, got %q", r.Category)
	}
	if r.File != "SKILL.md" {
		t.Errorf("expected file SKILL.md, got %q", r.File)
	}
	if r.Line != 0 {
		t.Errorf("expected line 0, got %d", r.Line)
	}
	if r.Message != "name is required" {
		t.Errorf("unexpected message: %q", r.Message)
	}
}

func TestResultContext_OverrideFile(t *testing.T) {
	ctx := ResultContext{Category: "Tokens", File: "SKILL.md"}

	r := ctx.WarnFile("references/guide.md", "too large")
	if r.File != "references/guide.md" {
		t.Errorf("expected file override, got %q", r.File)
	}
	if r.Level != Warning {
		t.Errorf("expected Warning level, got %v", r.Level)
	}
}

func TestResultContext_ErrorAtLine(t *testing.T) {
	ctx := ResultContext{Category: "Markdown"}

	r := ctx.ErrorAtLinef("SKILL.md", 42, "unclosed fence at line %d", 42)
	if r.File != "SKILL.md" {
		t.Errorf("expected SKILL.md, got %q", r.File)
	}
	if r.Line != 42 {
		t.Errorf("expected line 42, got %d", r.Line)
	}
	if r.Level != Error {
		t.Errorf("expected Error level, got %v", r.Level)
	}
}

func TestResultContext_NoDefaultFile(t *testing.T) {
	ctx := ResultContext{Category: "Structure"}

	r := ctx.Warn("something")
	if r.File != "" {
		t.Errorf("expected empty file, got %q", r.File)
	}
}

func TestResultContext_Formatters(t *testing.T) {
	ctx := ResultContext{Category: "Test", File: "test.md"}

	tests := []struct {
		name  string
		r     Result
		level Level
	}{
		{"Passf", ctx.Passf("count: %d", 5), Pass},
		{"Infof", ctx.Infof("info: %s", "x"), Info},
		{"Warnf", ctx.Warnf("warn: %d", 3), Warning},
		{"Errorf", ctx.Errorf("err: %s", "y"), Error},
		{"Pass", ctx.Pass("ok"), Pass},
		{"Info", ctx.Info("note"), Info},
		{"PassFile", ctx.PassFile("other.md", "ok"), Pass},
		{"WarnFilef", ctx.WarnFilef("other.md", "w: %d", 1), Warning},
		{"ErrorFile", ctx.ErrorFile("other.md", "e"), Error},
		{"ErrorFilef", ctx.ErrorFilef("other.md", "e: %d", 2), Error},
		{"ErrorAtLine", ctx.ErrorAtLine("f.md", 10, "err"), Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.r.Level != tt.level {
				t.Errorf("expected level %v, got %v", tt.level, tt.r.Level)
			}
			if tt.r.Category != "Test" {
				t.Errorf("expected category Test, got %q", tt.r.Category)
			}
		})
	}
}
