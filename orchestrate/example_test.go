package orchestrate_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agent-ecosystem/skill-validator/orchestrate"
	"github.com/agent-ecosystem/skill-validator/report"
	"github.com/agent-ecosystem/skill-validator/skillcheck"
	"github.com/agent-ecosystem/skill-validator/structure"
	"github.com/agent-ecosystem/skill-validator/types"
)

func ExampleRunAllChecks() {
	dir, _ := filepath.Abs(filepath.Join("..", "testdata", "valid-skill"))

	opts := orchestrate.Options{
		Enabled:    orchestrate.AllGroups(),
		StructOpts: structure.Options{},
	}

	rpt := orchestrate.RunAllChecks(context.Background(), dir, opts)

	// Print human-readable output
	report.Print(os.Stdout, rpt, false)

	if rpt.Errors > 0 {
		fmt.Fprintf(os.Stderr, "%d error(s) found\n", rpt.Errors)
	}
}

func ExampleRunAllChecks_structureOnly() {
	dir, _ := filepath.Abs(filepath.Join("..", "testdata", "valid-skill"))

	opts := orchestrate.Options{
		Enabled: map[orchestrate.CheckGroup]bool{
			orchestrate.GroupStructure: true,
		},
		StructOpts: structure.Options{
			SkipOrphans: true,
		},
	}

	rpt := orchestrate.RunAllChecks(context.Background(), dir, opts)

	for _, r := range rpt.Results {
		if r.Level == types.Error {
			fmt.Printf("ERROR: %s\n", r.Message)
		}
	}
}

func ExampleRunContentAnalysis() {
	dir, _ := filepath.Abs(filepath.Join("..", "testdata", "valid-skill"))

	rpt := orchestrate.RunContentAnalysis(dir)

	if rpt.ContentReport != nil {
		fmt.Printf("Word count: %d\n", rpt.ContentReport.WordCount)
		fmt.Printf("Imperative ratio: %.2f\n", rpt.ContentReport.ImperativeRatio)
	}
}

func ExampleRunAllChecks_multiSkill() {
	dir, _ := filepath.Abs(filepath.Join("..", "testdata", "multi-skill"))

	mode, dirs := skillcheck.DetectSkills(dir)
	if mode != types.MultiSkill {
		return
	}

	opts := orchestrate.Options{
		Enabled:    orchestrate.AllGroups(),
		StructOpts: structure.Options{},
	}

	for _, d := range dirs {
		rpt := orchestrate.RunAllChecks(context.Background(), d, opts)
		report.Print(os.Stdout, rpt, false)
	}
}
