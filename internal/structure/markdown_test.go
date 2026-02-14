package structure

import (
	"testing"

	"github.com/dacharyc/skill-validator/internal/validator"
)

func TestFindUnclosedFence(t *testing.T) {
	t.Run("no fences", func(t *testing.T) {
		_, found := FindUnclosedFence("Just regular text.\nNo fences here.")
		if found {
			t.Error("expected no unclosed fence")
		}
	})

	t.Run("balanced backtick fences", func(t *testing.T) {
		content := "Before\n```\ncode\n```\nAfter"
		_, found := FindUnclosedFence(content)
		if found {
			t.Error("expected no unclosed fence")
		}
	})

	t.Run("balanced tilde fences", func(t *testing.T) {
		content := "Before\n~~~\ncode\n~~~\nAfter"
		_, found := FindUnclosedFence(content)
		if found {
			t.Error("expected no unclosed fence")
		}
	})

	t.Run("balanced fence with info string", func(t *testing.T) {
		content := "Before\n```python\nprint('hi')\n```\nAfter"
		_, found := FindUnclosedFence(content)
		if found {
			t.Error("expected no unclosed fence")
		}
	})

	t.Run("unclosed backtick fence", func(t *testing.T) {
		content := "Before\n```\ncode\nmore code"
		line, found := FindUnclosedFence(content)
		if !found {
			t.Fatal("expected unclosed fence")
		}
		if line != 2 {
			t.Errorf("expected fence at line 2, got %d", line)
		}
	})

	t.Run("unclosed tilde fence", func(t *testing.T) {
		content := "Before\n~~~\ncode"
		line, found := FindUnclosedFence(content)
		if !found {
			t.Fatal("expected unclosed fence")
		}
		if line != 2 {
			t.Errorf("expected fence at line 2, got %d", line)
		}
	})

	t.Run("mismatched fence characters", func(t *testing.T) {
		content := "```\ncode\n~~~"
		line, found := FindUnclosedFence(content)
		if !found {
			t.Fatal("expected unclosed fence")
		}
		if line != 1 {
			t.Errorf("expected fence at line 1, got %d", line)
		}
	})

	t.Run("closing fence must be at least as long", func(t *testing.T) {
		content := "````\ncode\n```"
		line, found := FindUnclosedFence(content)
		if !found {
			t.Fatal("expected unclosed fence")
		}
		if line != 1 {
			t.Errorf("expected fence at line 1, got %d", line)
		}
	})

	t.Run("longer closing fence is fine", func(t *testing.T) {
		content := "```\ncode\n````"
		_, found := FindUnclosedFence(content)
		if found {
			t.Error("expected no unclosed fence")
		}
	})

	t.Run("indented fence up to 3 spaces", func(t *testing.T) {
		content := "   ```\ncode\n   ```"
		_, found := FindUnclosedFence(content)
		if found {
			t.Error("expected no unclosed fence with 3-space indent")
		}
	})

	t.Run("multiple balanced fences", func(t *testing.T) {
		content := "```\nblock1\n```\ntext\n```\nblock2\n```"
		_, found := FindUnclosedFence(content)
		if found {
			t.Error("expected no unclosed fence")
		}
	})

	t.Run("second fence unclosed", func(t *testing.T) {
		content := "```\nblock1\n```\ntext\n```\nblock2"
		line, found := FindUnclosedFence(content)
		if !found {
			t.Fatal("expected unclosed fence")
		}
		if line != 5 {
			t.Errorf("expected fence at line 5, got %d", line)
		}
	})

	t.Run("closing fence with trailing spaces", func(t *testing.T) {
		content := "```\ncode\n```   "
		_, found := FindUnclosedFence(content)
		if found {
			t.Error("expected no unclosed fence with trailing spaces on closer")
		}
	})

	t.Run("closing fence with trailing text is not a close", func(t *testing.T) {
		content := "```\ncode\n``` not closed"
		_, found := FindUnclosedFence(content)
		if !found {
			t.Fatal("expected unclosed fence when closer has trailing text")
		}
	})
}

func TestCheckMarkdown(t *testing.T) {
	t.Run("clean body and references", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/guide.md", "# Guide\n```go\nfmt.Println()\n```\n")
		results := CheckMarkdown(dir, "# Body\nSome text.")
		requireNoLevel(t, results, validator.Error)
	})

	t.Run("unclosed fence in body", func(t *testing.T) {
		dir := t.TempDir()
		results := CheckMarkdown(dir, "# Body\n```\ncode without closing")
		requireResultContaining(t, results, validator.Error, "SKILL.md has an unclosed code fence")
		requireResultContaining(t, results, validator.Error, "line 2")
	})

	t.Run("unclosed fence in reference", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/broken.md", "# Ref\n```\nunclosed")
		results := CheckMarkdown(dir, "Clean body.")
		requireResultContaining(t, results, validator.Error, "references/broken.md has an unclosed code fence")
	})

	t.Run("skips non-md reference files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/data.json", "```not markdown")
		results := CheckMarkdown(dir, "Clean body.")
		requireNoLevel(t, results, validator.Error)
	})

	t.Run("skips hidden reference files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/.hidden.md", "```unclosed")
		results := CheckMarkdown(dir, "Clean body.")
		requireNoLevel(t, results, validator.Error)
	})
}
