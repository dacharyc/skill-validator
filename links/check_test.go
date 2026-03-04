package links

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agent-ecosystem/skill-validator/types"
)

// writeFile creates a file at dir/relPath with the given content, creating directories as needed.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// requireResultContaining asserts that at least one result has the given level and message containing substr.
func requireResultContaining(t *testing.T, results []types.Result, level types.Level, substr string) {
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

func requireContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

func TestCheckLinks_SkipsRelative(t *testing.T) {
	t.Run("relative-only links returns nil", func(t *testing.T) {
		dir := t.TempDir()
		body := "See [guide](references/guide.md)."
		results := CheckLinks(t.Context(), dir, body)
		if results != nil {
			t.Errorf("expected nil for relative-only links, got %v", results)
		}
	})

	t.Run("mailto and anchors are skipped", func(t *testing.T) {
		dir := t.TempDir()
		body := "[email](mailto:user@example.com) and [section](#heading)"
		results := CheckLinks(t.Context(), dir, body)
		if results != nil {
			t.Errorf("expected nil for mailto/anchor links, got %v", results)
		}
	})

	t.Run("template URLs are skipped", func(t *testing.T) {
		dir := t.TempDir()
		body := "[PR](https://github.com/{OWNER}/{REPO}/pull/{PR}) and https://api.example.com/{version}/users/{id}"
		results := CheckLinks(t.Context(), dir, body)
		if results != nil {
			t.Errorf("expected nil for template URLs, got %v", results)
		}
	})

	t.Run("no links returns nil", func(t *testing.T) {
		dir := t.TempDir()
		body := "No links here."
		results := CheckLinks(t.Context(), dir, body)
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
		results := CheckLinks(t.Context(), dir, body)
		requireResultContaining(t, results, types.Pass, "HTTP 200")
	})

	t.Run("404 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[missing](" + server.URL + "/not-found)"
		results := CheckLinks(t.Context(), dir, body)
		requireResultContaining(t, results, types.Error, "HTTP 404")
	})

	t.Run("403 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[blocked](" + server.URL + "/forbidden)"
		results := CheckLinks(t.Context(), dir, body)
		requireResultContaining(t, results, types.Info, "HTTP 403")
	})

	t.Run("500 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[error](" + server.URL + "/server-error)"
		results := CheckLinks(t.Context(), dir, body)
		requireResultContaining(t, results, types.Error, "HTTP 500")
	})

	t.Run("mixed relative and HTTP only checks HTTP", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/guide.md", "content")
		body := "[guide](references/guide.md) and [site](" + server.URL + "/ok)"
		results := CheckLinks(t.Context(), dir, body)
		if len(results) != 1 {
			t.Fatalf("expected 1 result (HTTP only), got %d", len(results))
		}
		requireResultContaining(t, results, types.Pass, "HTTP 200")
	})
}

func testHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}}
}

func TestCheckHTTPLink(t *testing.T) {
	client := testHTTPClient()

	t.Run("connection refused", func(t *testing.T) {
		result := checkHTTPLink(types.ResultContext{Category: "Links", File: "SKILL.md"}, client, "http://127.0.0.1:1")
		if result.Level != types.Error {
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

		result := checkHTTPLink(types.ResultContext{Category: "Links", File: "SKILL.md"}, client, server.URL+"/redirect")
		if result.Level != types.Pass {
			t.Errorf("expected Pass for followed redirect, got level=%d message=%q", result.Level, result.Message)
		}
	})

	t.Run("redirect without follow results in 3xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "http://127.0.0.1:1/nowhere")
			w.WriteHeader(http.StatusTemporaryRedirect)
		}))
		defer server.Close()

		result := checkHTTPLink(types.ResultContext{Category: "Links", File: "SKILL.md"}, client, server.URL)
		if result.Level != types.Error {
			t.Errorf("expected Error for broken redirect target, got level=%d message=%q", result.Level, result.Message)
		}
	})

	t.Run("too many redirects", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", r.URL.Path)
			w.WriteHeader(http.StatusFound)
		}))
		defer server.Close()

		result := checkHTTPLink(types.ResultContext{Category: "Links", File: "SKILL.md"}, client, server.URL+"/loop")
		if result.Level != types.Error {
			t.Errorf("expected Error for redirect loop, got level=%d message=%q", result.Level, result.Message)
		}
		requireContains(t, result.Message, "request failed")
	})

	t.Run("403 forbidden", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		result := checkHTTPLink(types.ResultContext{Category: "Links", File: "SKILL.md"}, client, server.URL)
		if result.Level != types.Info {
			t.Errorf("expected Info level for 403, got %d", result.Level)
		}
		requireContains(t, result.Message, "HTTP 403")
	})

	t.Run("invalid URL", func(t *testing.T) {
		result := checkHTTPLink(types.ResultContext{Category: "Links", File: "SKILL.md"}, client, "http://invalid host with spaces/")
		if result.Level != types.Error {
			t.Errorf("expected Error for invalid URL, got level=%d", result.Level)
		}
		requireContains(t, result.Message, "invalid URL")
	})
}
