package report

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/validator"
)

func TestPrintJSON_Passed(t *testing.T) {
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
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out["skill_dir"] != "/tmp/my-skill" {
		t.Errorf("skill_dir = %v, want /tmp/my-skill", out["skill_dir"])
	}
	if out["passed"] != true {
		t.Errorf("passed = %v, want true", out["passed"])
	}
	if out["errors"].(float64) != 0 {
		t.Errorf("errors = %v, want 0", out["errors"])
	}
	if out["warnings"].(float64) != 0 {
		t.Errorf("warnings = %v, want 0", out["warnings"])
	}

	results := out["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
	}

	first := results[0].(map[string]any)
	if first["level"] != "pass" {
		t.Errorf("first result level = %v, want pass", first["level"])
	}
	if first["category"] != "Structure" {
		t.Errorf("first result category = %v, want Structure", first["category"])
	}
}

func TestPrintJSON_Failed(t *testing.T) {
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
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out["passed"] != false {
		t.Errorf("passed = %v, want false", out["passed"])
	}
	if out["errors"].(float64) != 1 {
		t.Errorf("errors = %v, want 1", out["errors"])
	}
	if out["warnings"].(float64) != 1 {
		t.Errorf("warnings = %v, want 1", out["warnings"])
	}

	results := out["results"].([]any)
	second := results[1].(map[string]any)
	if second["level"] != "error" {
		t.Errorf("second result level = %v, want error", second["level"])
	}
	third := results[2].(map[string]any)
	if third["level"] != "warning" {
		t.Errorf("third result level = %v, want warning", third["level"])
	}
}

func TestPrintJSON_LevelStrings(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results: []validator.Result{
			{Level: validator.Pass, Category: "A", Message: "p"},
			{Level: validator.Info, Category: "A", Message: "i"},
			{Level: validator.Warning, Category: "A", Message: "w"},
			{Level: validator.Error, Category: "A", Message: "e"},
		},
		Errors:   1,
		Warnings: 1,
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	results := out["results"].([]any)
	levels := []string{"pass", "info", "warning", "error"}
	for i, want := range levels {
		got := results[i].(map[string]any)["level"]
		if got != want {
			t.Errorf("result[%d] level = %v, want %v", i, got, want)
		}
	}
}

func TestPrintJSON_TokenCounts(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		TokenCounts: []validator.TokenCount{
			{File: "SKILL.md body", Tokens: 1250},
			{File: "references/guide.md", Tokens: 820},
		},
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	tc := out["token_counts"].(map[string]any)
	if tc["total"].(float64) != 2070 {
		t.Errorf("token_counts.total = %v, want 2070", tc["total"])
	}

	files := tc["files"].([]any)
	if len(files) != 2 {
		t.Fatalf("token_counts.files length = %d, want 2", len(files))
	}
	first := files[0].(map[string]any)
	if first["file"] != "SKILL.md body" {
		t.Errorf("first file = %v, want SKILL.md body", first["file"])
	}
	if first["tokens"].(float64) != 1250 {
		t.Errorf("first tokens = %v, want 1250", first["tokens"])
	}
}

func TestPrintJSON_NoTokenCounts(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results: []validator.Result{
			{Level: validator.Error, Category: "Structure", Message: "SKILL.md not found"},
		},
		Errors: 1,
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := out["token_counts"]; ok {
		t.Error("token_counts should be omitted when empty")
	}
	if _, ok := out["other_token_counts"]; ok {
		t.Error("other_token_counts should be omitted when empty")
	}
}

func TestPrintJSON_OtherTokenCounts(t *testing.T) {
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
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	otc := out["other_token_counts"].(map[string]any)
	if otc["total"].(float64) != 45850 {
		t.Errorf("other_token_counts.total = %v, want 45850", otc["total"])
	}

	files := otc["files"].([]any)
	if len(files) != 2 {
		t.Fatalf("other_token_counts.files length = %d, want 2", len(files))
	}
}

func TestPrintJSON_SpecialCharacters(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results: []validator.Result{
			{Level: validator.Error, Category: "Frontmatter", Message: `field contains "quotes" and <angle> & ampersand`},
		},
		Errors: 1,
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	// Verify it's valid JSON
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON with special characters: %v", err)
	}

	results := out["results"].([]any)
	msg := results[0].(map[string]any)["message"].(string)
	want := `field contains "quotes" and <angle> & ampersand`
	if msg != want {
		t.Errorf("message = %q, want %q", msg, want)
	}
}

func TestPrintMultiJSON_AllPassed(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/alpha",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
			},
			{
				SkillDir: "/tmp/beta",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
			},
		},
	}

	var buf bytes.Buffer
	if err := PrintMultiJSON(&buf, mr, false); err != nil {
		t.Fatalf("PrintMultiJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out["passed"] != true {
		t.Errorf("passed = %v, want true", out["passed"])
	}
	if out["errors"].(float64) != 0 {
		t.Errorf("errors = %v, want 0", out["errors"])
	}
	if out["warnings"].(float64) != 0 {
		t.Errorf("warnings = %v, want 0", out["warnings"])
	}

	skills := out["skills"].([]any)
	if len(skills) != 2 {
		t.Fatalf("skills length = %d, want 2", len(skills))
	}

	first := skills[0].(map[string]any)
	if first["skill_dir"] != "/tmp/alpha" {
		t.Errorf("first skill_dir = %v, want /tmp/alpha", first["skill_dir"])
	}
	if first["passed"] != true {
		t.Errorf("first passed = %v, want true", first["passed"])
	}
}

func TestPrintMultiJSON_SomeFailed(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/good",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
			},
			{
				SkillDir: "/tmp/bad",
				Results: []validator.Result{
					{Level: validator.Error, Category: "Frontmatter", Message: "name is required"},
					{Level: validator.Warning, Category: "Structure", Message: "unknown dir"},
				},
				Errors:   1,
				Warnings: 1,
			},
		},
		Errors:   1,
		Warnings: 1,
	}

	var buf bytes.Buffer
	if err := PrintMultiJSON(&buf, mr, false); err != nil {
		t.Fatalf("PrintMultiJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out["passed"] != false {
		t.Errorf("passed = %v, want false", out["passed"])
	}
	if out["errors"].(float64) != 1 {
		t.Errorf("errors = %v, want 1", out["errors"])
	}
	if out["warnings"].(float64) != 1 {
		t.Errorf("warnings = %v, want 1", out["warnings"])
	}

	skills := out["skills"].([]any)
	bad := skills[1].(map[string]any)
	if bad["passed"] != false {
		t.Errorf("bad skill passed = %v, want false", bad["passed"])
	}
}

func TestPrintMultiJSON_IncludesTokenCounts(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/with-tokens",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
				TokenCounts: []validator.TokenCount{
					{File: "SKILL.md body", Tokens: 500},
					{File: "references/ref.md", Tokens: 300},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := PrintMultiJSON(&buf, mr, false); err != nil {
		t.Fatalf("PrintMultiJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	skills := out["skills"].([]any)
	skill := skills[0].(map[string]any)
	tc := skill["token_counts"].(map[string]any)
	if tc["total"].(float64) != 800 {
		t.Errorf("token_counts.total = %v, want 800", tc["total"])
	}
	files := tc["files"].([]any)
	if len(files) != 2 {
		t.Fatalf("token_counts.files length = %d, want 2", len(files))
	}
}

func TestPrintJSON_ContaminationAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContaminationReport: &contamination.Report{
			MultiInterfaceTools:  []string{"mongodb"},
			CodeLanguages:        []string{"python", "javascript", "bash"},
			LanguageCategories:   []string{"python", "javascript", "shell"},
			PrimaryCategory:      "python",
			MismatchedCategories: []string{"javascript", "shell"},
			LanguageMismatch:     true,
			TechReferences:       []string{"javascript", "python"},
			ScopeBreadth:         4,
			ContaminationScore:   0.55,
			ContaminationLevel:   "high",
		},
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	ca, ok := out["contamination_analysis"].(map[string]any)
	if !ok {
		t.Fatal("expected contamination_analysis in JSON")
	}

	if ca["contamination_level"].(string) != "high" {
		t.Errorf("contamination_level = %v, want high", ca["contamination_level"])
	}
	if ca["contamination_score"].(float64) != 0.55 {
		t.Errorf("contamination_score = %v, want 0.55", ca["contamination_score"])
	}
	if ca["language_mismatch"] != true {
		t.Error("expected language_mismatch=true")
	}
	if ca["primary_category"].(string) != "python" {
		t.Errorf("primary_category = %v, want python", ca["primary_category"])
	}
	if int(ca["scope_breadth"].(float64)) != 4 {
		t.Errorf("scope_breadth = %v, want 4", ca["scope_breadth"])
	}

	tools := ca["multi_interface_tools"].([]any)
	if len(tools) != 1 || tools[0].(string) != "mongodb" {
		t.Errorf("multi_interface_tools = %v, want [mongodb]", tools)
	}

	mismatched := ca["mismatched_categories"].([]any)
	if len(mismatched) != 2 {
		t.Errorf("mismatched_categories length = %d, want 2", len(mismatched))
	}
}

func TestPrintJSON_NoContaminationAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := out["contamination_analysis"]; ok {
		t.Error("contamination_analysis should be omitted when nil")
	}
}

func TestPrintJSON_ContentAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
		ContentReport: &content.Report{
			WordCount:              500,
			CodeBlockCount:         3,
			CodeBlockRatio:         0.2,
			CodeLanguages:          []string{"python", "bash"},
			SentenceCount:          20,
			ImperativeCount:        10,
			ImperativeRatio:        0.5,
			InformationDensity:     0.35,
			StrongMarkers:          5,
			WeakMarkers:            2,
			InstructionSpecificity: 0.71,
			SectionCount:           3,
			ListItemCount:          8,
		},
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	ca, ok := out["content_analysis"].(map[string]any)
	if !ok {
		t.Fatal("expected content_analysis in JSON")
	}

	if ca["word_count"].(float64) != 500 {
		t.Errorf("word_count = %v, want 500", ca["word_count"])
	}
	if ca["code_block_count"].(float64) != 3 {
		t.Errorf("code_block_count = %v, want 3", ca["code_block_count"])
	}
	if ca["imperative_ratio"].(float64) != 0.5 {
		t.Errorf("imperative_ratio = %v, want 0.5", ca["imperative_ratio"])
	}
	if ca["instruction_specificity"].(float64) != 0.71 {
		t.Errorf("instruction_specificity = %v, want 0.71", ca["instruction_specificity"])
	}

	langs := ca["code_languages"].([]any)
	if len(langs) != 2 {
		t.Errorf("code_languages length = %d, want 2", len(langs))
	}
}

func TestPrintJSON_NoContentAnalysis(t *testing.T) {
	r := &validator.Report{
		SkillDir: "/tmp/test",
		Results:  []validator.Result{},
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, r, false); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := out["content_analysis"]; ok {
		t.Error("content_analysis should be omitted when nil")
	}
}

func TestPrintMultiJSON_WithContamination(t *testing.T) {
	mr := &validator.MultiReport{
		Skills: []*validator.Report{
			{
				SkillDir: "/tmp/skill-a",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
				ContaminationReport: &contamination.Report{
					ContaminationLevel: "low",
					ContaminationScore: 0.0,
					ScopeBreadth:       1,
				},
			},
			{
				SkillDir: "/tmp/skill-b",
				Results:  []validator.Result{{Level: validator.Pass, Category: "Structure", Message: "ok"}},
				ContaminationReport: &contamination.Report{
					ContaminationLevel: "high",
					ContaminationScore: 0.6,
					ScopeBreadth:       5,
					LanguageMismatch:   true,
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := PrintMultiJSON(&buf, mr, false); err != nil {
		t.Fatalf("PrintMultiJSON error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	skills := out["skills"].([]any)
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	first := skills[0].(map[string]any)
	ca1 := first["contamination_analysis"].(map[string]any)
	if ca1["contamination_level"].(string) != "low" {
		t.Errorf("first skill contamination_level = %v, want low", ca1["contamination_level"])
	}

	second := skills[1].(map[string]any)
	ca2 := second["contamination_analysis"].(map[string]any)
	if ca2["contamination_level"].(string) != "high" {
		t.Errorf("second skill contamination_level = %v, want high", ca2["contamination_level"])
	}
	if ca2["contamination_score"].(float64) != 0.6 {
		t.Errorf("second skill contamination_score = %v, want 0.6", ca2["contamination_score"])
	}
}
