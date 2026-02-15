package structure

import (
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/validator"
)

func TestCheckTokens(t *testing.T) {
	t.Run("counts body tokens", func(t *testing.T) {
		dir := t.TempDir()
		body := "Hello world, this is a test body."
		results, counts, _ := CheckTokens(dir, body)
		requireNoLevel(t, results, validator.Error)
		if len(counts) == 0 {
			t.Fatal("expected at least one token count")
		}
		if counts[0].File != "SKILL.md body" {
			t.Errorf("first count file = %q, want %q", counts[0].File, "SKILL.md body")
		}
		if counts[0].Tokens <= 0 {
			t.Errorf("expected positive token count, got %d", counts[0].Tokens)
		}
	})

	t.Run("counts reference file tokens", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/guide.md", "# Guide\n\nSome reference content here.")
		writeFile(t, dir, "references/api.md", "# API\n\nAPI documentation.")
		body := "Body text."
		_, counts, _ := CheckTokens(dir, body)
		if len(counts) != 3 { // body + 2 references
			t.Fatalf("expected 3 token counts, got %d", len(counts))
		}
		// Verify reference files are counted
		refFiles := map[string]bool{}
		for _, c := range counts[1:] {
			refFiles[c.File] = true
			if c.Tokens <= 0 {
				t.Errorf("expected positive tokens for %s, got %d", c.File, c.Tokens)
			}
		}
		if !refFiles["references/guide.md"] {
			t.Error("expected references/guide.md in counts")
		}
		if !refFiles["references/api.md"] {
			t.Error("expected references/api.md in counts")
		}
	})

	t.Run("no references directory", func(t *testing.T) {
		dir := t.TempDir()
		body := "Short body."
		results, counts, _ := CheckTokens(dir, body)
		requireNoLevel(t, results, validator.Error)
		if len(counts) != 1 {
			t.Fatalf("expected 1 token count (body only), got %d", len(counts))
		}
	})

	t.Run("skips hidden files in references", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/.hidden", "secret")
		writeFile(t, dir, "references/visible.md", "content")
		body := "Body."
		_, counts, _ := CheckTokens(dir, body)
		if len(counts) != 2 { // body + visible.md
			t.Fatalf("expected 2 token counts, got %d", len(counts))
		}
	})

	t.Run("skips subdirectories in references", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/subdir/file.md", "nested")
		writeFile(t, dir, "references/top.md", "top level")
		body := "Body."
		_, counts, _ := CheckTokens(dir, body)
		if len(counts) != 2 { // body + top.md
			t.Fatalf("expected 2 token counts, got %d", len(counts))
		}
	})

	t.Run("warns on large body", func(t *testing.T) {
		dir := t.TempDir()
		// Generate a body that exceeds 5000 tokens (~4 chars per token average)
		body := strings.Repeat("This is a test sentence for token counting purposes. ", 500)
		results, _, _ := CheckTokens(dir, body)
		requireResultContaining(t, results, validator.Warning, "spec recommends < 5000")
	})

	t.Run("warns on many lines", func(t *testing.T) {
		dir := t.TempDir()
		body := strings.Repeat("line\n", 501)
		results, _, _ := CheckTokens(dir, body)
		requireResultContaining(t, results, validator.Warning, "spec recommends < 500")
	})

	t.Run("no warning on small body", func(t *testing.T) {
		dir := t.TempDir()
		body := "Small body."
		results, _, _ := CheckTokens(dir, body)
		requireNoLevel(t, results, validator.Warning)
	})
}

// generateContent creates a string of approximately the target token count.
// Uses repetitive sentences (~10 tokens each).
func generateContent(approxTokens int) string {
	// "The quick brown fox jumps over the lazy sleeping dog today. " ≈ 10 tokens
	sentence := "The quick brown fox jumps over the lazy sleeping dog today. "
	reps := approxTokens / 10
	return strings.Repeat(sentence, reps)
}

func TestCheckTokens_PerFileRefLimits(t *testing.T) {
	t.Run("reference file under soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/small.md", "A small reference file.")
		results, _, _ := CheckTokens(dir, "body")
		requireNoResultContaining(t, results, validator.Warning, "references/small.md")
		requireNoResultContaining(t, results, validator.Error, "references/small.md")
	})

	t.Run("reference file exceeds soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/medium.md", generateContent(11_000))
		results, _, _ := CheckTokens(dir, "body")
		requireResultContaining(t, results, validator.Warning, "references/medium.md")
		requireResultContaining(t, results, validator.Warning, "consider splitting into smaller focused files")
	})

	t.Run("reference file exceeds hard limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/huge.md", generateContent(26_000))
		results, _, _ := CheckTokens(dir, "body")
		requireResultContaining(t, results, validator.Error, "references/huge.md")
		requireResultContaining(t, results, validator.Error, "meaningfully degrade agent performance")
	})
}

func TestCheckTokens_AggregateRefLimits(t *testing.T) {
	t.Run("total under soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/a.md", generateContent(5_000))
		writeFile(t, dir, "references/b.md", generateContent(5_000))
		results, _, _ := CheckTokens(dir, "body")
		requireNoResultContaining(t, results, validator.Warning, "total reference files")
		requireNoResultContaining(t, results, validator.Error, "total reference files")
	})

	t.Run("total exceeds soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/a.md", generateContent(9_000))
		writeFile(t, dir, "references/b.md", generateContent(9_000))
		writeFile(t, dir, "references/c.md", generateContent(9_000))
		results, _, _ := CheckTokens(dir, "body")
		requireResultContaining(t, results, validator.Warning, "total reference files")
		requireResultContaining(t, results, validator.Warning, "consider whether all this content is essential")
	})

	t.Run("total exceeds hard limit", func(t *testing.T) {
		dir := t.TempDir()
		// 3 files at ~18k each ≈ 54k total, exceeding 50k hard limit
		writeFile(t, dir, "references/a.md", generateContent(18_000))
		writeFile(t, dir, "references/b.md", generateContent(18_000))
		writeFile(t, dir, "references/c.md", generateContent(18_000))
		results, _, _ := CheckTokens(dir, "body")
		requireResultContaining(t, results, validator.Error, "total reference files")
		requireResultContaining(t, results, validator.Error, "25-40%")
	})
}

func TestCountOtherFiles(t *testing.T) {
	t.Run("counts extra root files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "---\nname: test\n---\nbody")
		writeFile(t, dir, "AGENTS.md", "Some agent content here.")
		writeFile(t, dir, "metadata.json", `{"key": "value"}`)
		_, _, otherCounts := CheckTokens(dir, "body")
		if len(otherCounts) != 2 {
			t.Fatalf("expected 2 other counts, got %d", len(otherCounts))
		}
		files := map[string]bool{}
		for _, c := range otherCounts {
			files[c.File] = true
			if c.Tokens <= 0 {
				t.Errorf("expected positive tokens for %s, got %d", c.File, c.Tokens)
			}
		}
		if !files["AGENTS.md"] {
			t.Error("expected AGENTS.md in other counts")
		}
		if !files["metadata.json"] {
			t.Error("expected metadata.json in other counts")
		}
	})

	t.Run("counts files in unknown directories", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "rules/rule1.md", "Rule one content.")
		writeFile(t, dir, "rules/rule2.md", "Rule two content.")
		_, _, otherCounts := CheckTokens(dir, "body")
		if len(otherCounts) != 2 {
			t.Fatalf("expected 2 other counts, got %d", len(otherCounts))
		}
		files := map[string]bool{}
		for _, c := range otherCounts {
			files[c.File] = true
		}
		if !files["rules/rule1.md"] {
			t.Error("expected rules/rule1.md in other counts")
		}
		if !files["rules/rule2.md"] {
			t.Error("expected rules/rule2.md in other counts")
		}
	})

	t.Run("skips binary files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "image.png", "fake png data")
		writeFile(t, dir, "archive.zip", "fake zip data")
		writeFile(t, dir, "notes.txt", "text content")
		_, _, otherCounts := CheckTokens(dir, "body")
		if len(otherCounts) != 1 {
			t.Fatalf("expected 1 other count (notes.txt only), got %d", len(otherCounts))
		}
		if otherCounts[0].File != "notes.txt" {
			t.Errorf("expected notes.txt, got %s", otherCounts[0].File)
		}
	})

	t.Run("skips hidden files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, ".hidden", "secret")
		writeFile(t, dir, "visible.txt", "visible content")
		_, _, otherCounts := CheckTokens(dir, "body")
		if len(otherCounts) != 1 {
			t.Fatalf("expected 1 other count, got %d", len(otherCounts))
		}
		if otherCounts[0].File != "visible.txt" {
			t.Errorf("expected visible.txt, got %s", otherCounts[0].File)
		}
	})

	t.Run("skips standard directories", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "references/ref.md", "reference content")
		writeFile(t, dir, "scripts/run.sh", "#!/bin/bash")
		writeFile(t, dir, "assets/logo.txt", "logo")
		_, _, otherCounts := CheckTokens(dir, "body")
		if len(otherCounts) != 0 {
			t.Fatalf("expected 0 other counts, got %d", len(otherCounts))
		}
	})

	t.Run("no other files returns empty", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		_, _, otherCounts := CheckTokens(dir, "body")
		if len(otherCounts) != 0 {
			t.Fatalf("expected 0 other counts, got %d", len(otherCounts))
		}
	})
}

func TestCheckTokens_OtherFilesLimits(t *testing.T) {
	t.Run("other files under soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "extra.md", generateContent(5_000))
		results, _, _ := CheckTokens(dir, "body")
		requireNoResultContaining(t, results, validator.Warning, "non-standard files total")
		requireNoResultContaining(t, results, validator.Error, "non-standard files total")
	})

	t.Run("other files exceed soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "extra1.md", generateContent(15_000))
		writeFile(t, dir, "extra2.md", generateContent(15_000))
		results, _, _ := CheckTokens(dir, "body")
		requireResultContaining(t, results, validator.Warning, "non-standard files total")
		requireResultContaining(t, results, validator.Warning, "could consume a significant portion")
	})

	t.Run("other files exceed hard limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "rules/a.md", generateContent(40_000))
		writeFile(t, dir, "rules/b.md", generateContent(40_000))
		writeFile(t, dir, "rules/c.md", generateContent(25_000))
		results, _, _ := CheckTokens(dir, "body")
		requireResultContaining(t, results, validator.Error, "non-standard files total")
		requireResultContaining(t, results, validator.Error, "severely degrade performance")
	})
}

// assetCounts filters token counts to only those with an "assets/" prefix.
func assetCounts(counts []validator.TokenCount) []validator.TokenCount {
	var out []validator.TokenCount
	for _, c := range counts {
		if strings.HasPrefix(c.File, "assets/") {
			out = append(out, c)
		}
	}
	return out
}

func TestCountAssetFiles(t *testing.T) {
	t.Run("counts text-based asset files in token_counts", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "assets/template.md", "# Template\n\nFill this out.")
		writeFile(t, dir, "assets/report.tex", "\\documentclass{article}\n\\begin{document}\nHello\n\\end{document}")
		writeFile(t, dir, "assets/analysis.py", "import pandas as pd\n\ndef analyze():\n    pass")
		_, counts, _ := CheckTokens(dir, "body")
		ac := assetCounts(counts)
		if len(ac) != 3 {
			t.Fatalf("expected 3 asset counts, got %d", len(ac))
		}
		files := map[string]bool{}
		for _, c := range ac {
			files[c.File] = true
			if c.Tokens <= 0 {
				t.Errorf("expected positive tokens for %s, got %d", c.File, c.Tokens)
			}
		}
		if !files["assets/template.md"] {
			t.Error("expected assets/template.md in counts")
		}
		if !files["assets/report.tex"] {
			t.Error("expected assets/report.tex in counts")
		}
		if !files["assets/analysis.py"] {
			t.Error("expected assets/analysis.py in counts")
		}
	})

	t.Run("counts all supported extensions", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "assets/guide.md", "guide content")
		writeFile(t, dir, "assets/report.tex", "tex content")
		writeFile(t, dir, "assets/script.py", "py content")
		writeFile(t, dir, "assets/config.yaml", "yaml content")
		writeFile(t, dir, "assets/config2.yml", "yml content")
		writeFile(t, dir, "assets/component.tsx", "tsx content")
		writeFile(t, dir, "assets/util.ts", "ts content")
		writeFile(t, dir, "assets/widget.jsx", "jsx content")
		writeFile(t, dir, "assets/style.sty", "sty content")
		writeFile(t, dir, "assets/plot.mplstyle", "mplstyle content")
		writeFile(t, dir, "assets/notebook.ipynb", "ipynb content")
		_, counts, _ := CheckTokens(dir, "body")
		ac := assetCounts(counts)
		if len(ac) != 11 {
			t.Fatalf("expected 11 asset counts, got %d", len(ac))
		}
	})

	t.Run("skips non-text asset files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "assets/logo.png", "fake png")
		writeFile(t, dir, "assets/photo.jpg", "fake jpg")
		writeFile(t, dir, "assets/icon.svg", "fake svg")
		writeFile(t, dir, "assets/data.csv", "a,b,c")
		writeFile(t, dir, "assets/template.md", "# Template")
		_, counts, _ := CheckTokens(dir, "body")
		ac := assetCounts(counts)
		if len(ac) != 1 {
			t.Fatalf("expected 1 asset count (template.md only), got %d", len(ac))
		}
		if ac[0].File != "assets/template.md" {
			t.Errorf("expected assets/template.md, got %s", ac[0].File)
		}
	})

	t.Run("counts files in asset subdirectories", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "assets/templates/report.tex", "tex content")
		writeFile(t, dir, "assets/templates/plan.md", "plan content")
		writeFile(t, dir, "assets/styles/custom.sty", "sty content")
		_, counts, _ := CheckTokens(dir, "body")
		ac := assetCounts(counts)
		if len(ac) != 3 {
			t.Fatalf("expected 3 asset counts, got %d", len(ac))
		}
	})

	t.Run("skips hidden files in assets", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "assets/.hidden.md", "hidden")
		writeFile(t, dir, "assets/visible.md", "visible")
		_, counts, _ := CheckTokens(dir, "body")
		ac := assetCounts(counts)
		if len(ac) != 1 {
			t.Fatalf("expected 1 asset count, got %d", len(ac))
		}
	})

	t.Run("no assets directory returns empty", func(t *testing.T) {
		dir := t.TempDir()
		_, counts, _ := CheckTokens(dir, "body")
		ac := assetCounts(counts)
		if len(ac) != 0 {
			t.Fatalf("expected 0 asset counts, got %d", len(ac))
		}
	})

	t.Run("assets excluded from other counts and included in token counts", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "SKILL.md", "content")
		writeFile(t, dir, "assets/template.md", "template content")
		_, counts, otherCounts := CheckTokens(dir, "body")
		if len(otherCounts) != 0 {
			t.Fatalf("expected 0 other counts, got %d", len(otherCounts))
		}
		ac := assetCounts(counts)
		if len(ac) != 1 {
			t.Fatalf("expected 1 asset in token counts, got %d", len(ac))
		}
	})
}
