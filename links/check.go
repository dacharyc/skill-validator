package links

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/agent-ecosystem/skill-validator/types"
)

type linkResult struct {
	url    string
	result types.Result
}

// CheckLinks validates external (HTTP/HTTPS) links in the skill body.
func CheckLinks(ctx context.Context, dir, body string) []types.Result {
	rctx := types.ResultContext{Category: "Links", File: "SKILL.md"}
	allLinks := ExtractLinks(body)
	if len(allLinks) == 0 {
		return nil
	}

	var (
		results   []types.Result
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

	// Shared client for connection reuse across concurrent checks.
	// The client uses a safe transport that blocks requests to private IPs.
	client := newHTTPClient()

	// Check HTTP links concurrently
	httpResults := make([]linkResult, len(httpLinks))
	for i, link := range httpLinks {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()
			r := checkHTTPLink(rctx, client, url)
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

func checkHTTPLink(rctx types.ResultContext, client *http.Client, url string) types.Result {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return rctx.Errorf("%s (invalid URL: %v)", url, err)
	}
	req.Header.Set("User-Agent", "skill-validator/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return rctx.Errorf("%s (request failed: %v)", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return rctx.Passf("%s (HTTP %d)", url, resp.StatusCode)
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return rctx.Passf("%s (HTTP %d redirect)", url, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusForbidden {
		return rctx.Infof("%s (HTTP 403 — may block automated requests)", url)
	}
	return rctx.Errorf("%s (HTTP %d)", url, resp.StatusCode)
}
