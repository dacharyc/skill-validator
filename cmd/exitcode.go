package cmd

// Exit codes used by the CLI.
const (
	ExitClean   = 0 // no errors, no warnings
	ExitError   = 1 // validation errors present
	ExitWarning = 2 // warnings present, no errors
	ExitCobra   = 3 // CLI/usage error (bad flags, missing args)
)

// exitOpts controls how validation results map to exit codes.
type exitOpts struct {
	strict bool // when true, warnings are treated as errors (exit 1)
}

// resolve returns the appropriate exit code given error and warning counts.
func (o exitOpts) resolve(errors, warnings int) int {
	if errors > 0 {
		return ExitError
	}
	if warnings > 0 {
		if o.strict {
			return ExitError
		}
		return ExitWarning
	}
	return ExitClean
}
