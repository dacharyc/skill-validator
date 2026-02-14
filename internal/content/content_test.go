package content

import (
	"math"
	"testing"
)

func TestAnalyze_EmptyContent(t *testing.T) {
	r := Analyze("")
	if r.WordCount != 0 {
		t.Errorf("expected 0 words, got %d", r.WordCount)
	}
}

func TestAnalyze_WordCount(t *testing.T) {
	r := Analyze("one two three four five")
	if r.WordCount != 5 {
		t.Errorf("expected 5 words, got %d", r.WordCount)
	}
}

func TestAnalyze_CodeBlocks(t *testing.T) {
	content := "Some text.\n\n```python\nprint('hello')\nprint('world')\n```\n\nMore text.\n\n```bash\necho hi\n```\n"
	r := Analyze(content)
	if r.CodeBlockCount != 2 {
		t.Errorf("expected 2 code blocks, got %d", r.CodeBlockCount)
	}
	if r.CodeBlockRatio <= 0 {
		t.Errorf("expected positive code block ratio, got %f", r.CodeBlockRatio)
	}
}

func TestAnalyze_CodeLanguages(t *testing.T) {
	content := "```python\ncode\n```\n\n```javascript\ncode\n```\n\n```python\ncode\n```\n"
	r := Analyze(content)
	if len(r.CodeLanguages) != 3 {
		t.Errorf("expected 3 code languages, got %d: %v", len(r.CodeLanguages), r.CodeLanguages)
	}
	if r.CodeLanguages[0] != "python" {
		t.Errorf("expected first language python, got %s", r.CodeLanguages[0])
	}
	if r.CodeLanguages[1] != "javascript" {
		t.Errorf("expected second language javascript, got %s", r.CodeLanguages[1])
	}
}

func TestAnalyze_ImperativeSentences(t *testing.T) {
	content := "Use the CLI tool. Run the tests. This is a description. Create a new file."
	r := Analyze(content)
	if r.ImperativeCount != 3 {
		t.Errorf("expected 3 imperative sentences, got %d", r.ImperativeCount)
	}
	if r.ImperativeRatio <= 0 {
		t.Errorf("expected positive imperative ratio, got %f", r.ImperativeRatio)
	}
}

func TestAnalyze_StrongMarkers(t *testing.T) {
	content := "You must always use this. Never do that. This is required."
	r := Analyze(content)
	if r.StrongMarkers < 4 {
		t.Errorf("expected at least 4 strong markers (must, always, never, required), got %d", r.StrongMarkers)
	}
}

func TestAnalyze_WeakMarkers(t *testing.T) {
	content := "You may consider this. It could work. It might be optional."
	r := Analyze(content)
	if r.WeakMarkers < 4 {
		t.Errorf("expected at least 4 weak markers (may, consider, could, might, optional), got %d", r.WeakMarkers)
	}
}

func TestAnalyze_InstructionSpecificity(t *testing.T) {
	content := "You must do this. You must always do that. Never skip it."
	r := Analyze(content)
	// All strong markers, no weak ones → specificity = 1.0
	if r.InstructionSpecificity != 1.0 {
		t.Errorf("expected specificity 1.0 with only strong markers, got %f", r.InstructionSpecificity)
	}
}

func TestAnalyze_Sections(t *testing.T) {
	content := "# Title\n\n## Section 1\n\nText.\n\n### Subsection\n\nMore text.\n\n## Section 2\n"
	r := Analyze(content)
	// H2+ headers: ## Section 1, ### Subsection, ## Section 2 = 3
	if r.SectionCount != 3 {
		t.Errorf("expected 3 sections, got %d", r.SectionCount)
	}
}

func TestAnalyze_ListItems(t *testing.T) {
	content := "- item 1\n- item 2\n* item 3\n1. numbered\n2. also numbered\n"
	r := Analyze(content)
	if r.ListItemCount != 5 {
		t.Errorf("expected 5 list items, got %d", r.ListItemCount)
	}
}

func TestAnalyze_InformationDensity(t *testing.T) {
	t.Run("with code blocks", func(t *testing.T) {
		content := "Use the tool.\n\n```bash\necho hello\n```\n\nRun the command. Build the project."
		r := Analyze(content)
		if r.InformationDensity <= 0 {
			t.Errorf("expected positive information density, got %f", r.InformationDensity)
		}
	})

	t.Run("without code blocks", func(t *testing.T) {
		// Prose-only skill with imperative sentences should not be penalized
		content := "Use the tool. Run the command. Build the project."
		r := Analyze(content)
		if r.CodeBlockCount != 0 {
			t.Errorf("expected 0 code blocks, got %d", r.CodeBlockCount)
		}
		if r.ImperativeRatio <= 0 {
			t.Fatalf("expected positive imperative ratio, got %f", r.ImperativeRatio)
		}
		// Without code blocks, information density should equal imperative ratio
		if r.InformationDensity != r.ImperativeRatio {
			t.Errorf("expected information density (%f) to equal imperative ratio (%f) when no code blocks",
				r.InformationDensity, r.ImperativeRatio)
		}
	})
}

func TestAnalyze_FullContent(t *testing.T) {
	content := `# My Skill

## Usage

Use the CLI to validate skills. You must always run tests before publishing.

` + "```bash\nskill-validator validate ./my-skill\n```" + `

## Configuration

Create a config file. Set the output format. You may consider using JSON.

- Step 1: Install
- Step 2: Configure
- Step 3: Run

Never skip validation. Ensure all checks pass.
`

	r := Analyze(content)

	if r.WordCount <= 0 {
		t.Error("expected positive word count")
	}
	if r.CodeBlockCount != 1 {
		t.Errorf("expected 1 code block, got %d", r.CodeBlockCount)
	}
	if r.SectionCount != 2 {
		t.Errorf("expected 2 sections, got %d", r.SectionCount)
	}
	if r.ListItemCount != 3 {
		t.Errorf("expected 3 list items, got %d", r.ListItemCount)
	}
	if r.StrongMarkers < 3 {
		t.Errorf("expected at least 3 strong markers, got %d", r.StrongMarkers)
	}
	if r.WeakMarkers < 1 {
		t.Errorf("expected at least 1 weak marker, got %d", r.WeakMarkers)
	}
	if r.InstructionSpecificity <= 0 || r.InstructionSpecificity > 1.0 {
		t.Errorf("expected specificity in (0, 1], got %f", r.InstructionSpecificity)
	}
}

func TestRoundTo(t *testing.T) {
	tests := []struct {
		val    float64
		places int
		want   float64
	}{
		{0.12345, 4, 0.1235},
		{0.5, 2, 0.5},
		{1.0, 4, 1.0},
		{0.0, 4, 0.0},
	}
	for _, tt := range tests {
		got := roundTo(tt.val, tt.places)
		if math.Abs(got-tt.want) > 1e-10 {
			t.Errorf("roundTo(%f, %d) = %f, want %f", tt.val, tt.places, got, tt.want)
		}
	}
}
