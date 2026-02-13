package validator

import (
	"strings"
	"testing"
)

func TestCheckTokens(t *testing.T) {
	t.Run("counts body tokens", func(t *testing.T) {
		dir := t.TempDir()
		body := "Hello world, this is a test body."
		results, counts := checkTokens(dir, body)
		requireNoLevel(t, results, Error)
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
		_, counts := checkTokens(dir, body)
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
		results, counts := checkTokens(dir, body)
		requireNoLevel(t, results, Error)
		if len(counts) != 1 {
			t.Fatalf("expected 1 token count (body only), got %d", len(counts))
		}
	})

	t.Run("skips hidden files in references", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/.hidden", "secret")
		writeFile(t, dir, "references/visible.md", "content")
		body := "Body."
		_, counts := checkTokens(dir, body)
		if len(counts) != 2 { // body + visible.md
			t.Fatalf("expected 2 token counts, got %d", len(counts))
		}
	})

	t.Run("skips subdirectories in references", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/subdir/file.md", "nested")
		writeFile(t, dir, "references/top.md", "top level")
		body := "Body."
		_, counts := checkTokens(dir, body)
		if len(counts) != 2 { // body + top.md
			t.Fatalf("expected 2 token counts, got %d", len(counts))
		}
	})

	t.Run("warns on large body", func(t *testing.T) {
		dir := t.TempDir()
		// Generate a body that exceeds 5000 tokens (~4 chars per token average)
		body := strings.Repeat("This is a test sentence for token counting purposes. ", 500)
		results, _ := checkTokens(dir, body)
		requireResultContaining(t, results, Warning, "spec recommends < 5000")
	})

	t.Run("warns on many lines", func(t *testing.T) {
		dir := t.TempDir()
		body := strings.Repeat("line\n", 501)
		results, _ := checkTokens(dir, body)
		requireResultContaining(t, results, Warning, "spec recommends < 500")
	})

	t.Run("no warning on small body", func(t *testing.T) {
		dir := t.TempDir()
		body := "Small body."
		results, _ := checkTokens(dir, body)
		requireNoLevel(t, results, Warning)
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
		results, _ := checkTokens(dir, "body")
		requireNoResultContaining(t, results, Warning, "references/small.md")
		requireNoResultContaining(t, results, Error, "references/small.md")
	})

	t.Run("reference file exceeds soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/medium.md", generateContent(11_000))
		results, _ := checkTokens(dir, "body")
		requireResultContaining(t, results, Warning, "references/medium.md")
		requireResultContaining(t, results, Warning, "consider splitting into smaller focused files")
	})

	t.Run("reference file exceeds hard limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/huge.md", generateContent(26_000))
		results, _ := checkTokens(dir, "body")
		requireResultContaining(t, results, Error, "references/huge.md")
		requireResultContaining(t, results, Error, "meaningfully degrade agent performance")
	})
}

func TestCheckTokens_AggregateRefLimits(t *testing.T) {
	t.Run("total under soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/a.md", generateContent(5_000))
		writeFile(t, dir, "references/b.md", generateContent(5_000))
		results, _ := checkTokens(dir, "body")
		requireNoResultContaining(t, results, Warning, "total reference files")
		requireNoResultContaining(t, results, Error, "total reference files")
	})

	t.Run("total exceeds soft limit", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/a.md", generateContent(9_000))
		writeFile(t, dir, "references/b.md", generateContent(9_000))
		writeFile(t, dir, "references/c.md", generateContent(9_000))
		results, _ := checkTokens(dir, "body")
		requireResultContaining(t, results, Warning, "total reference files")
		requireResultContaining(t, results, Warning, "consider whether all this content is essential")
	})

	t.Run("total exceeds hard limit", func(t *testing.T) {
		dir := t.TempDir()
		// 3 files at ~18k each ≈ 54k total, exceeding 50k hard limit
		writeFile(t, dir, "references/a.md", generateContent(18_000))
		writeFile(t, dir, "references/b.md", generateContent(18_000))
		writeFile(t, dir, "references/c.md", generateContent(18_000))
		results, _ := checkTokens(dir, "body")
		requireResultContaining(t, results, Error, "total reference files")
		requireResultContaining(t, results, Error, "25-40%")
	})
}
