package quality

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/validator"
)

// writeFile creates a file at dir/relPath with the given content, creating directories as needed.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// requireResult asserts that at least one result has the exact level and message.
func requireResult(t *testing.T, results []validator.Result, level validator.Level, message string) {
	t.Helper()
	for _, r := range results {
		if r.Level == level && r.Message == message {
			return
		}
	}
	t.Errorf("expected result with level=%d message=%q, got:", level, message)
	for _, r := range results {
		t.Logf("  level=%d category=%s message=%q", r.Level, r.Category, r.Message)
	}
}

// requireResultContaining asserts that at least one result has the given level and message containing substr.
func requireResultContaining(t *testing.T, results []validator.Result, level validator.Level, substr string) {
	t.Helper()
	for _, r := range results {
		if r.Level == level && strings.Contains(r.Message, substr) {
			return
		}
	}
	t.Errorf("expected result with level=%d message containing %q, got:", level, substr)
	for _, r := range results {
		t.Logf("  level=%d category=%s message=%q", r.Level, r.Category, r.Message)
	}
}

// requireNoLevel asserts that no result has the given level.
func requireNoLevel(t *testing.T, results []validator.Result, level validator.Level) {
	t.Helper()
	for _, r := range results {
		if r.Level == level {
			t.Errorf("unexpected result with level=%d: category=%s message=%q", level, r.Category, r.Message)
		}
	}
}

func requireContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

func TestExtractLinks(t *testing.T) {
	t.Run("markdown links", func(t *testing.T) {
		body := "See [guide](references/guide.md) and [docs](https://example.com/docs)."
		links := extractLinks(body)
		if len(links) != 2 {
			t.Fatalf("expected 2 links, got %d: %v", len(links), links)
		}
		if links[0] != "references/guide.md" {
			t.Errorf("links[0] = %q, want %q", links[0], "references/guide.md")
		}
		if links[1] != "https://example.com/docs" {
			t.Errorf("links[1] = %q, want %q", links[1], "https://example.com/docs")
		}
	})

	t.Run("bare URLs", func(t *testing.T) {
		body := "Visit https://example.com for details.\nAlso http://other.com/page"
		links := extractLinks(body)
		if len(links) != 2 {
			t.Fatalf("expected 2 links, got %d: %v", len(links), links)
		}
		if links[0] != "https://example.com" {
			t.Errorf("links[0] = %q, want %q", links[0], "https://example.com")
		}
		if links[1] != "http://other.com/page" {
			t.Errorf("links[1] = %q, want %q", links[1], "http://other.com/page")
		}
	})

	t.Run("deduplication", func(t *testing.T) {
		body := "[link1](https://example.com) and [link2](https://example.com) and https://example.com"
		links := extractLinks(body)
		if len(links) != 1 {
			t.Fatalf("expected 1 deduplicated link, got %d: %v", len(links), links)
		}
	})

	t.Run("no links", func(t *testing.T) {
		body := "Just plain text with no links at all."
		links := extractLinks(body)
		if len(links) != 0 {
			t.Fatalf("expected 0 links, got %d: %v", len(links), links)
		}
	})

	t.Run("mixed link types", func(t *testing.T) {
		body := "[file](scripts/run.sh)\n[site](https://example.com)\nmailto:user@example.com\n#anchor"
		links := extractLinks(body)
		if len(links) != 2 {
			t.Fatalf("expected 2 links (markdown only), got %d: %v", len(links), links)
		}
	})

	t.Run("empty link text", func(t *testing.T) {
		body := "[](references/empty.md)"
		links := extractLinks(body)
		if len(links) != 1 {
			t.Fatalf("expected 1 link, got %d: %v", len(links), links)
		}
		if links[0] != "references/empty.md" {
			t.Errorf("links[0] = %q, want %q", links[0], "references/empty.md")
		}
	})
}

func TestCheckLinks_Relative(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/guide.md", "content")
		body := "See [guide](references/guide.md)."
		results := CheckLinks(dir, body)
		requireResult(t, results, validator.Pass, "references/guide.md (exists)")
	})

	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		body := "See [guide](references/missing.md)."
		results := CheckLinks(dir, body)
		requireResult(t, results, validator.Error, "references/missing.md (file not found)")
	})

	t.Run("mailto and anchors are skipped", func(t *testing.T) {
		dir := t.TempDir()
		body := "[email](mailto:user@example.com) and [section](#heading)"
		results := CheckLinks(dir, body)
		if len(results) != 0 {
			t.Errorf("expected 0 results for mailto/anchor links, got %d", len(results))
		}
	})

	t.Run("template URLs are skipped", func(t *testing.T) {
		dir := t.TempDir()
		body := "[PR](https://github.com/{OWNER}/{REPO}/pull/{PR}) and https://api.example.com/{version}/users/{id}"
		results := CheckLinks(dir, body)
		if len(results) != 0 {
			t.Errorf("expected 0 results for template URLs, got %d", len(results))
		}
	})

	t.Run("no links returns nil", func(t *testing.T) {
		dir := t.TempDir()
		body := "No links here."
		results := CheckLinks(dir, body)
		if results != nil {
			t.Errorf("expected nil for no links, got %v", results)
		}
	})
}

func TestCheckLinks_HTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/not-found", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/forbidden", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/server-error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("successful HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[ok](" + server.URL + "/ok)"
		results := CheckLinks(dir, body)
		requireResultContaining(t, results, validator.Pass, "HTTP 200")
	})

	t.Run("404 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[missing](" + server.URL + "/not-found)"
		results := CheckLinks(dir, body)
		requireResultContaining(t, results, validator.Error, "HTTP 404")
	})

	t.Run("403 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[blocked](" + server.URL + "/forbidden)"
		results := CheckLinks(dir, body)
		requireResultContaining(t, results, validator.Info, "HTTP 403")
	})

	t.Run("500 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[error](" + server.URL + "/server-error)"
		results := CheckLinks(dir, body)
		requireResultContaining(t, results, validator.Error, "HTTP 500")
	})

	t.Run("mixed relative and HTTP", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/guide.md", "content")
		body := "[guide](references/guide.md) and [site](" + server.URL + "/ok)"
		results := CheckLinks(dir, body)
		requireResult(t, results, validator.Pass, "references/guide.md (exists)")
		requireResultContaining(t, results, validator.Pass, "HTTP 200")
	})
}

func TestCheckHTTPLink(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		result := checkHTTPLink("http://127.0.0.1:1")
		if result.Level != validator.Error {
			t.Errorf("expected Error level, got %d", result.Level)
		}
		requireContains(t, result.Message, "request failed")
	})

	t.Run("redirect 3xx", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "/dest")
			w.WriteHeader(http.StatusMovedPermanently)
		})
		mux.HandleFunc("/dest", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		result := checkHTTPLink(server.URL + "/redirect")
		if result.Level != validator.Pass {
			t.Errorf("expected Pass for followed redirect, got level=%d message=%q", result.Level, result.Message)
		}
	})

	t.Run("redirect without follow results in 3xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "http://127.0.0.1:1/nowhere")
			w.WriteHeader(http.StatusTemporaryRedirect)
		}))
		defer server.Close()

		result := checkHTTPLink(server.URL)
		if result.Level != validator.Error {
			t.Errorf("expected Error for broken redirect target, got level=%d message=%q", result.Level, result.Message)
		}
	})

	t.Run("too many redirects", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", r.URL.Path)
			w.WriteHeader(http.StatusFound)
		}))
		defer server.Close()

		result := checkHTTPLink(server.URL + "/loop")
		if result.Level != validator.Error {
			t.Errorf("expected Error for redirect loop, got level=%d message=%q", result.Level, result.Message)
		}
		requireContains(t, result.Message, "request failed")
	})

	t.Run("403 forbidden", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		result := checkHTTPLink(server.URL)
		if result.Level != validator.Info {
			t.Errorf("expected Info level for 403, got %d", result.Level)
		}
		requireContains(t, result.Message, "HTTP 403")
	})

	t.Run("invalid URL", func(t *testing.T) {
		result := checkHTTPLink("http://invalid host with spaces/")
		if result.Level != validator.Error {
			t.Errorf("expected Error for invalid URL, got level=%d", result.Level)
		}
		requireContains(t, result.Message, "invalid URL")
	})
}

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
		requireNoLevel(t, results, validator.Warning)
	})

	t.Run("unclosed fence in body", func(t *testing.T) {
		dir := t.TempDir()
		results := CheckMarkdown(dir, "# Body\n```\ncode without closing")
		requireResultContaining(t, results, validator.Warning, "SKILL.md has an unclosed code fence")
		requireResultContaining(t, results, validator.Warning, "line 2")
	})

	t.Run("unclosed fence in reference", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/broken.md", "# Ref\n```\nunclosed")
		results := CheckMarkdown(dir, "Clean body.")
		requireResultContaining(t, results, validator.Warning, "references/broken.md has an unclosed code fence")
	})

	t.Run("skips non-md reference files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/data.json", "```not markdown")
		results := CheckMarkdown(dir, "Clean body.")
		requireNoLevel(t, results, validator.Warning)
	})

	t.Run("skips hidden reference files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/.hidden.md", "```unclosed")
		results := CheckMarkdown(dir, "Clean body.")
		requireNoLevel(t, results, validator.Warning)
	})
}
