package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// buildBinary compiles the CLI to a temp directory and returns the path.
func buildBinary(t *testing.T) string {
	t.Helper()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	bin := filepath.Join(t.TempDir(), "skill-validator"+ext)
	cmd := exec.Command("go", "build", "-o", bin, "..")
	cmd.Dir = filepath.Join(moduleRoot(t), "cmd")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// moduleRoot returns the project root (parent of cmd/).
func moduleRoot(t *testing.T) string {
	t.Helper()
	// This file lives in cmd/, so the module root is one level up.
	dir, err := filepath.Abs(filepath.Join("..", "."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("cannot find module root: %v", err)
	}
	return dir
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(moduleRoot(t), "testdata", name)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("fixture %q not found: %v", name, err)
	}
	return p
}

func TestExitCodes(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{
			name:     "clean skill exits 0",
			args:     []string{"check", fixture(t, "valid-skill")},
			wantCode: 0,
		},
		{
			name:     "errors exit 1",
			args:     []string{"check", fixture(t, "invalid-skill")},
			wantCode: 1,
		},
		{
			name:     "warnings-only exits 2",
			args:     []string{"check", fixture(t, "warnings-only-skill")},
			wantCode: 2,
		},
		{
			name:     "strict with warnings exits 1",
			args:     []string{"check", "--strict", fixture(t, "warnings-only-skill")},
			wantCode: 1,
		},
		{
			name:     "bad flag exits 3",
			args:     []string{"check", "--bogus"},
			wantCode: 3,
		},
		{
			name:     "validate structure strict with warnings exits 1",
			args:     []string{"validate", "structure", "--strict", fixture(t, "warnings-only-skill")},
			wantCode: 1,
		},
		{
			name:     "validate structure warnings-only exits 2",
			args:     []string{"validate", "structure", fixture(t, "warnings-only-skill")},
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(bin, tt.args...)
			_ = cmd.Run()
			got := cmd.ProcessState.ExitCode()
			if got != tt.wantCode {
				t.Errorf("exit code = %d, want %d (args: %v)", got, tt.wantCode, tt.args)
			}
		})
	}
}
