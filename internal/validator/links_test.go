package validator

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
		results := checkLinks(dir, body)
		requireResult(t, results, Pass, "references/guide.md (exists)")
	})

	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		body := "See [guide](references/missing.md)."
		results := checkLinks(dir, body)
		requireResult(t, results, Error, "references/missing.md (file not found)")
	})

	t.Run("mailto and anchors are skipped", func(t *testing.T) {
		dir := t.TempDir()
		body := "[email](mailto:user@example.com) and [section](#heading)"
		results := checkLinks(dir, body)
		if len(results) != 0 {
			t.Errorf("expected 0 results for mailto/anchor links, got %d", len(results))
		}
	})

	t.Run("template URLs are skipped", func(t *testing.T) {
		dir := t.TempDir()
		body := "[PR](https://github.com/{OWNER}/{REPO}/pull/{PR}) and https://api.example.com/{version}/users/{id}"
		results := checkLinks(dir, body)
		if len(results) != 0 {
			t.Errorf("expected 0 results for template URLs, got %d", len(results))
		}
	})

	t.Run("no links returns nil", func(t *testing.T) {
		dir := t.TempDir()
		body := "No links here."
		results := checkLinks(dir, body)
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
		results := checkLinks(dir, body)
		requireResultContaining(t, results, Pass, "HTTP 200")
	})

	t.Run("404 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[missing](" + server.URL + "/not-found)"
		results := checkLinks(dir, body)
		requireResultContaining(t, results, Error, "HTTP 404")
	})

	t.Run("403 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[blocked](" + server.URL + "/forbidden)"
		results := checkLinks(dir, body)
		requireResultContaining(t, results, Info, "HTTP 403")
	})

	t.Run("500 HTTP link", func(t *testing.T) {
		dir := t.TempDir()
		body := "[error](" + server.URL + "/server-error)"
		results := checkLinks(dir, body)
		requireResultContaining(t, results, Error, "HTTP 500")
	})

	t.Run("mixed relative and HTTP", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/guide.md", "content")
		body := "[guide](references/guide.md) and [site](" + server.URL + "/ok)"
		results := checkLinks(dir, body)
		requireResult(t, results, Pass, "references/guide.md (exists)")
		requireResultContaining(t, results, Pass, "HTTP 200")
	})
}

func TestCheckHTTPLink(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		result := checkHTTPLink("http://127.0.0.1:1")
		if result.Level != Error {
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

		// The client follows redirects, so this should resolve to 200
		result := checkHTTPLink(server.URL + "/redirect")
		if result.Level != Pass {
			t.Errorf("expected Pass for followed redirect, got level=%d message=%q", result.Level, result.Message)
		}
	})

	t.Run("redirect without follow results in 3xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "http://127.0.0.1:1/nowhere")
			w.WriteHeader(http.StatusTemporaryRedirect)
		}))
		defer server.Close()

		// Redirect target is unreachable, so client.Do will error on the redirect
		result := checkHTTPLink(server.URL)
		// The redirect target fails, so we get a request error
		if result.Level != Error {
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
		if result.Level != Error {
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
		if result.Level != Info {
			t.Errorf("expected Info level for 403, got %d", result.Level)
		}
		requireContains(t, result.Message, "HTTP 403")
	})

	t.Run("invalid URL", func(t *testing.T) {
		result := checkHTTPLink("http://invalid host with spaces/")
		if result.Level != Error {
			t.Errorf("expected Error for invalid URL, got level=%d", result.Level)
		}
		requireContains(t, result.Message, "invalid URL")
	})
}
