package links

import (
	"fmt"
	"net/http"
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

// CheckLinks validates external (HTTP/HTTPS) links in the skill body.
func CheckLinks(dir string, body string) []validator.Result {
	allLinks := ExtractLinks(body)
	if len(allLinks) == 0 {
		return nil
	}

	var (
		results   []validator.Result
		httpLinks []string
		mu        sync.Mutex
		wg        sync.WaitGroup
	)

	// Collect HTTP links only
	for _, link := range allLinks {
		// Skip template URLs containing {placeholder} variables (RFC 6570 URI Templates)
		if strings.Contains(link, "{") {
			continue
		}
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			httpLinks = append(httpLinks, link)
		}
	}

	if len(httpLinks) == 0 {
		return nil
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

// ExtractLinks extracts all unique links from a markdown body.
func ExtractLinks(body string) []string {
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
