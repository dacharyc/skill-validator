package structure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/validator"
)

// writeFile creates a file at dir/relPath with the given content, creating directories as needed.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// writeSkill creates a SKILL.md file in the given directory.
func writeSkill(t *testing.T, dir, content string) {
	t.Helper()
	writeFile(t, dir, "SKILL.md", content)
}

// dirName returns the base name of a directory path.
func dirName(dir string) string {
	return filepath.Base(dir)
}

// requireResult asserts that at least one result has the exact level and message.
func requireResult(t *testing.T, results []validator.Result, level validator.Level, message string) {
	t.Helper()
	for _, r := range results {
		if r.Level == level && r.Message == message {
			return
		}
	}
	t.Errorf("expected result with level=%d message=%q, got:", level, message)
	for _, r := range results {
		t.Logf("  level=%d category=%s message=%q", r.Level, r.Category, r.Message)
	}
}

// requireResultContaining asserts that at least one result has the given level and message containing substr.
func requireResultContaining(t *testing.T, results []validator.Result, level validator.Level, substr string) {
	t.Helper()
	for _, r := range results {
		if r.Level == level && strings.Contains(r.Message, substr) {
			return
		}
	}
	t.Errorf("expected result with level=%d message containing %q, got:", level, substr)
	for _, r := range results {
		t.Logf("  level=%d category=%s message=%q", r.Level, r.Category, r.Message)
	}
}

// requireNoLevel asserts that no result has the given level.
func requireNoLevel(t *testing.T, results []validator.Result, level validator.Level) {
	t.Helper()
	for _, r := range results {
		if r.Level == level {
			t.Errorf("unexpected result with level=%d: category=%s message=%q", level, r.Category, r.Message)
		}
	}
}

// requireNoResultContaining asserts no result has the given level with message containing substr.
func requireNoResultContaining(t *testing.T, results []validator.Result, level validator.Level, substr string) {
	t.Helper()
	for _, r := range results {
		if r.Level == level && strings.Contains(r.Message, substr) {
			t.Errorf("unexpected result with level=%d message containing %q: %q", level, substr, r.Message)
		}
	}
}

// requireContains asserts that s contains substr.
func requireContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
