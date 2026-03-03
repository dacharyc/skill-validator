package types

import "testing"

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{Pass, "pass"},
		{Info, "info"},
		{Warning, "warning"},
		{Error, "error"},
		{Level(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestTally(t *testing.T) {
	r := &Report{
		Results: []Result{
			{Level: Pass, Category: "A", Message: "ok"},
			{Level: Error, Category: "B", Message: "bad"},
			{Level: Warning, Category: "C", Message: "meh"},
			{Level: Error, Category: "D", Message: "also bad"},
			{Level: Info, Category: "E", Message: "fyi"},
		},
	}
	r.Tally()
	if r.Errors != 2 {
		t.Errorf("Errors = %d, want 2", r.Errors)
	}
	if r.Warnings != 1 {
		t.Errorf("Warnings = %d, want 1", r.Warnings)
	}
}

func TestTally_Empty(t *testing.T) {
	r := &Report{Errors: 5, Warnings: 3}
	r.Tally()
	if r.Errors != 0 {
		t.Errorf("Errors = %d, want 0", r.Errors)
	}
	if r.Warnings != 0 {
		t.Errorf("Warnings = %d, want 0", r.Warnings)
	}
}
