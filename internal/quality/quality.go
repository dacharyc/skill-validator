package quality

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dacharyc/skill-validator/internal/validator"
)

var (
	// Match [text](url) markdown links
	mdLinkPattern = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	// Match bare URLs
	bareURLPattern = regexp.MustCompile(`(?:^|\s)(https?://[^\s<>\)]+)`)
)

type linkResult struct {
	url    string
	result validator.Result
}

// CheckLinks validates all links in the skill body.
func CheckLinks(dir string, body string) []validator.Result {
	links := extractLinks(body)
	if len(links) == 0 {
		return nil
	}

	var (
		results   []validator.Result
		httpLinks []string
		mu        sync.Mutex
		wg        sync.WaitGroup
	)

	// Check relative and HTTP links
	for _, link := range links {
		// Skip template URLs containing {placeholder} variables (RFC 6570 URI Templates)
		if strings.Contains(link, "{") {
			continue
		}
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			httpLinks = append(httpLinks, link)
			continue
		}
		if strings.HasPrefix(link, "mailto:") || strings.HasPrefix(link, "#") {
			continue
		}
		// Relative link
		resolved := filepath.Join(dir, link)
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			results = append(results, validator.Result{Level: validator.Error, Category: "Links", Message: fmt.Sprintf("%s (file not found)", link)})
		} else {
			results = append(results, validator.Result{Level: validator.Pass, Category: "Links", Message: fmt.Sprintf("%s (exists)", link)})
		}
	}

	// Check HTTP links concurrently
	httpResults := make([]linkResult, len(httpLinks))
	for i, link := range httpLinks {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()
			r := checkHTTPLink(url)
			mu.Lock()
			httpResults[idx] = linkResult{url: url, result: r}
			mu.Unlock()
		}(i, link)
	}
	wg.Wait()

	for _, hr := range httpResults {
		results = append(results, hr.result)
	}

	return results
}

func extractLinks(body string) []string {
	seen := make(map[string]bool)
	var links []string

	// Markdown links
	for _, match := range mdLinkPattern.FindAllStringSubmatch(body, -1) {
		url := strings.TrimSpace(match[2])
		if !seen[url] {
			seen[url] = true
			links = append(links, url)
		}
	}

	// Bare URLs
	for _, match := range bareURLPattern.FindAllStringSubmatch(body, -1) {
		url := strings.TrimSpace(match[1])
		if !seen[url] {
			seen[url] = true
			links = append(links, url)
		}
	}

	return links
}

func checkHTTPLink(url string) validator.Result {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return validator.Result{Level: validator.Error, Category: "Links", Message: fmt.Sprintf("%s (invalid URL: %v)", url, err)}
	}
	req.Header.Set("User-Agent", "skill-validator/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return validator.Result{Level: validator.Error, Category: "Links", Message: fmt.Sprintf("%s (request failed: %v)", url, err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return validator.Result{Level: validator.Pass, Category: "Links", Message: fmt.Sprintf("%s (HTTP %d)", url, resp.StatusCode)}
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return validator.Result{Level: validator.Pass, Category: "Links", Message: fmt.Sprintf("%s (HTTP %d redirect)", url, resp.StatusCode)}
	}
	if resp.StatusCode == http.StatusForbidden {
		return validator.Result{Level: validator.Info, Category: "Links", Message: fmt.Sprintf("%s (HTTP 403 — may block automated requests)", url)}
	}
	return validator.Result{Level: validator.Error, Category: "Links", Message: fmt.Sprintf("%s (HTTP %d)", url, resp.StatusCode)}
}

// CheckMarkdown validates markdown structure in the skill.
func CheckMarkdown(dir string, body string) []validator.Result {
	var results []validator.Result

	// Check SKILL.md body
	if line, ok := FindUnclosedFence(body); ok {
		results = append(results, validator.Result{
			Level:    validator.Warning,
			Category: "Markdown",
			Message:  fmt.Sprintf("SKILL.md has an unclosed code fence starting at line %d — this may cause agents to misinterpret everything after it as code", line),
		})
	}

	// Check .md files in references/
	refsDir := filepath.Join(dir, "references")
	entries, err := os.ReadDir(refsDir)
	if err != nil {
		return results
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(refsDir, entry.Name()))
		if err != nil {
			continue
		}
		relPath := filepath.Join("references", entry.Name())
		if line, ok := FindUnclosedFence(string(data)); ok {
			results = append(results, validator.Result{
				Level:    validator.Warning,
				Category: "Markdown",
				Message:  fmt.Sprintf("%s has an unclosed code fence starting at line %d — this may cause agents to misinterpret everything after it as code", relPath, line),
			})
		}
	}

	return results
}

// FindUnclosedFence checks for unclosed code fences (``` or ~~~).
// Returns the line number of the unclosed opening fence and true, or 0 and false.
func FindUnclosedFence(content string) (int, bool) {
	lines := strings.Split(content, "\n")
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	fenceLine := 0

	for i, line := range lines {
		// Strip up to 3 leading spaces
		stripped := line
		for range 3 {
			if len(stripped) > 0 && stripped[0] == ' ' {
				stripped = stripped[1:]
			} else {
				break
			}
		}

		if !inFence {
			if char, n := fencePrefix(stripped); n >= 3 {
				inFence = true
				fenceChar = char
				fenceLen = n
				fenceLine = i + 1
			}
		} else {
			if char, n := fencePrefix(stripped); n >= fenceLen && char == fenceChar {
				// Closing fence: rest must be only whitespace
				rest := stripped[n:]
				if strings.TrimSpace(rest) == "" {
					inFence = false
				}
			}
		}
	}

	if inFence {
		return fenceLine, true
	}
	return 0, false
}

// fencePrefix returns the fence character and its count if the line starts
// with 3+ backticks or 3+ tildes. Returns (0, 0) otherwise.
func fencePrefix(line string) (byte, int) {
	if len(line) == 0 {
		return 0, 0
	}
	ch := line[0]
	if ch != '`' && ch != '~' {
		return 0, 0
	}
	n := 0
	for n < len(line) && line[n] == ch {
		n++
	}
	if n < 3 {
		return 0, 0
	}
	return ch, n
}
