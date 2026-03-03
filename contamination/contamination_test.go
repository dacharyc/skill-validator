package contamination

import (
	"testing"
)

func TestAnalyze_NoContamination(t *testing.T) {
	r := Analyze("my-skill", "Simple content about writing code.", nil)
	if r.ContaminationLevel != "low" {
		t.Errorf("expected low contamination, got %s", r.ContaminationLevel)
	}
	if r.ContaminationScore != 0 {
		t.Errorf("expected 0 contamination score, got %f", r.ContaminationScore)
	}
}

func TestAnalyze_MultiInterfaceTool(t *testing.T) {
	r := Analyze("mongodb-queries", "Use MongoDB to query data.", nil)
	if len(r.MultiInterfaceTools) == 0 {
		t.Error("expected multi-interface tool detected")
	}
	found := false
	for _, tool := range r.MultiInterfaceTools {
		if tool == "mongodb" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected mongodb in multi-interface tools, got %v", r.MultiInterfaceTools)
	}
	if r.ContaminationScore < 0.2 {
		t.Errorf("expected contamination score >= 0.2 with multi-interface tool, got %f", r.ContaminationScore)
	}
}

func TestAnalyze_LanguageMismatch(t *testing.T) {
	languages := []string{"python", "python", "bash", "javascript"}
	r := Analyze("my-skill", "Some content.", languages)
	if !r.LanguageMismatch {
		t.Error("expected language mismatch")
	}
	if r.PrimaryCategory != "python" {
		t.Errorf("expected primary category python, got %s", r.PrimaryCategory)
	}
	if len(r.MismatchedCategories) == 0 {
		t.Error("expected mismatched categories")
	}
}

func TestAnalyze_NoPrimaryCategory(t *testing.T) {
	r := Analyze("my-skill", "Content.", nil)
	if r.PrimaryCategory != "" {
		t.Errorf("expected empty primary category, got %s", r.PrimaryCategory)
	}
	if r.LanguageMismatch {
		t.Error("expected no language mismatch with no languages")
	}
}

func TestAnalyze_TechReferences(t *testing.T) {
	content := "Use Django and Node.js for this skill."
	r := Analyze("my-skill", content, nil)
	if len(r.TechReferences) < 2 {
		t.Errorf("expected at least 2 tech references, got %d: %v", len(r.TechReferences), r.TechReferences)
	}
}

func TestAnalyze_HighContamination(t *testing.T) {
	// Multi-interface tool + language mismatch + scope breadth
	content := "Use MongoDB with Node.js, Django, and Rails."
	languages := []string{"python", "javascript", "bash", "ruby"}
	r := Analyze("mongodb-skill", content, languages)
	if r.ContaminationLevel != "high" {
		t.Errorf("expected high contamination, got %s (score=%f)", r.ContaminationLevel, r.ContaminationScore)
	}
	if r.ContaminationScore < 0.5 {
		t.Errorf("expected contamination score >= 0.5, got %f", r.ContaminationScore)
	}
}

func TestAnalyze_ScopeBreadth(t *testing.T) {
	content := "Use Django, Node.js, and Rails."
	languages := []string{"python", "javascript", "ruby", "bash"}
	r := Analyze("my-skill", content, languages)
	if r.ScopeBreadth < 3 {
		t.Errorf("expected scope breadth >= 3, got %d", r.ScopeBreadth)
	}
}

func TestDetectMultiInterfaceTools(t *testing.T) {
	t.Run("in name", func(t *testing.T) {
		matches := detectMultiInterfaceTools("aws-deploy", "Deploy stuff.")
		found := false
		for _, m := range matches {
			if m == "aws" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected aws in matches, got %v", matches)
		}
	})

	t.Run("in content", func(t *testing.T) {
		matches := detectMultiInterfaceTools("my-skill", "Configure Redis for caching.")
		found := false
		for _, m := range matches {
			if m == "redis" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected redis in matches, got %v", matches)
		}
	})

	t.Run("none", func(t *testing.T) {
		matches := detectMultiInterfaceTools("my-skill", "Write some code.")
		if len(matches) != 0 {
			t.Errorf("expected no matches, got %v", matches)
		}
	})
}

func TestGetLanguageCategories(t *testing.T) {
	cats := getLanguageCategories([]string{"python", "bash", "javascript"})
	if !cats["python"] {
		t.Error("expected python category")
	}
	if !cats["shell"] {
		t.Error("expected shell category")
	}
	if !cats["javascript"] {
		t.Error("expected javascript category")
	}
}

func TestFindPrimaryCategory(t *testing.T) {
	t.Run("most frequent wins", func(t *testing.T) {
		primary := findPrimaryCategory([]string{"python", "python", "bash"})
		if primary != "python" {
			t.Errorf("expected python, got %s", primary)
		}
	})

	t.Run("empty", func(t *testing.T) {
		primary := findPrimaryCategory(nil)
		if primary != "" {
			t.Errorf("expected empty, got %s", primary)
		}
	})
}

func TestMismatchWeight(t *testing.T) {
	tests := []struct {
		cat1, cat2 string
		want       float64
	}{
		{"python", "javascript", 1.0},  // app ↔ app
		{"java", "dotnet", 1.0},        // app ↔ app
		{"python", "shell", 0.25},      // app ↔ aux
		{"javascript", "config", 0.25}, // app ↔ aux
		{"shell", "config", 0.1},       // aux ↔ aux
		{"query", "markup", 0.1},       // aux ↔ aux
	}
	for _, tt := range tests {
		got := mismatchWeight(tt.cat1, tt.cat2)
		if got != tt.want {
			t.Errorf("mismatchWeight(%s, %s) = %f, want %f", tt.cat1, tt.cat2, got, tt.want)
		}
	}
}

func TestAnalyze_AuxiliaryOnlyMismatches(t *testing.T) {
	// python + bash + yaml: auxiliary mismatches should score low
	languages := []string{"python", "python", "bash", "yaml"}
	r := Analyze("deploy-skill", "Deploy with bash and config.", languages)
	if !r.LanguageMismatch {
		t.Error("expected language mismatch")
	}
	// 2 auxiliary mismatches × 0.25 = 0.50 weighted → 0.4 × (0.50/3) ≈ 0.067
	// No multi-interface tool, scope breadth = 3 → factor3 = 0.3 * (1/4) = 0.075
	// Total ≈ 0.142, should be low
	if r.ContaminationLevel != "low" {
		t.Errorf("expected low contamination for python+bash+yaml, got %s (score=%f)", r.ContaminationLevel, r.ContaminationScore)
	}
	// Verify weights are populated
	if w, ok := r.MismatchWeights["shell"]; !ok || w != 0.25 {
		t.Errorf("expected shell weight 0.25, got %f (ok=%v)", w, ok)
	}
	if w, ok := r.MismatchWeights["config"]; !ok || w != 0.25 {
		t.Errorf("expected config weight 0.25, got %f (ok=%v)", w, ok)
	}
}

func TestAnalyze_ApplicationOnlyMismatches(t *testing.T) {
	// python + javascript + ruby: all application mismatches, score unchanged from old behavior
	languages := []string{"python", "python", "javascript", "ruby"}
	r := Analyze("multi-sdk", "Multi-SDK skill.", languages)
	if !r.LanguageMismatch {
		t.Error("expected language mismatch")
	}
	// 2 app mismatches × 1.0 = 2.0 weighted → 0.4 × (2.0/3) ≈ 0.267
	if r.ContaminationScore < 0.2 {
		t.Errorf("expected score >= 0.2 for app-only mismatches, got %f", r.ContaminationScore)
	}
	if w := r.MismatchWeights["javascript"]; w != 1.0 {
		t.Errorf("expected javascript weight 1.0, got %f", w)
	}
	if w := r.MismatchWeights["ruby"]; w != 1.0 {
		t.Errorf("expected ruby weight 1.0, got %f", w)
	}
}

func TestAnalyze_MixedMismatches(t *testing.T) {
	// java + config + shell + markup: 3 auxiliary mismatches
	languages := []string{"java", "java", "yaml", "bash", "html"}
	r := Analyze("spring-boot", "Spring Boot app with config.", languages)
	if !r.LanguageMismatch {
		t.Error("expected language mismatch")
	}
	// 3 auxiliary mismatches × 0.25 = 0.75 weighted → 0.4 × (0.75/3) = 0.1
	// Should be significantly lower than old score of 0.4 × (3/3) = 0.4
	if r.ContaminationScore >= 0.4 {
		t.Errorf("expected score < 0.4 for java+config+shell+markup, got %f", r.ContaminationScore)
	}
}

func TestContaminationLevels(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{0.0, "low"},
		{0.1, "low"},
		{0.19, "low"},
		{0.2, "medium"},
		{0.35, "medium"},
		{0.49, "medium"},
		{0.5, "high"},
		{0.8, "high"},
		{1.0, "high"},
	}
	for _, tt := range tests {
		level := "low"
		if tt.score >= 0.5 {
			level = "high"
		} else if tt.score >= 0.2 {
			level = "medium"
		}
		if level != tt.want {
			t.Errorf("score=%f → level=%s, want %s", tt.score, level, tt.want)
		}
	}
}
