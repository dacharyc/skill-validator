package cmd

import "testing"

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		opts     exitOpts
		errors   int
		warnings int
		want     int
	}{
		{"clean", exitOpts{}, 0, 0, ExitClean},
		{"errors only", exitOpts{}, 3, 0, ExitError},
		{"warnings only", exitOpts{}, 0, 2, ExitWarning},
		{"errors and warnings", exitOpts{}, 1, 5, ExitError},
		{"strict clean", exitOpts{strict: true}, 0, 0, ExitClean},
		{"strict errors", exitOpts{strict: true}, 1, 0, ExitError},
		{"strict warnings", exitOpts{strict: true}, 0, 4, ExitError},
		{"strict errors and warnings", exitOpts{strict: true}, 2, 3, ExitError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.resolve(tt.errors, tt.warnings)
			if got != tt.want {
				t.Errorf("resolve(%d, %d) with strict=%v = %d, want %d",
					tt.errors, tt.warnings, tt.opts.strict, got, tt.want)
			}
		})
	}
}
