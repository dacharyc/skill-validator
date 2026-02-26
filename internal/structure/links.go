package structure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dacharyc/skill-validator/internal/links"
	"github.com/dacharyc/skill-validator/internal/validator"
)

// CheckInternalLinks validates relative (internal) links in the skill body.
// Broken internal links indicate a structural problem: the skill references
// files that don't exist in the package.
func CheckInternalLinks(dir, body string) []validator.Result {
	allLinks := links.ExtractLinks(body)
	if len(allLinks) == 0 {
		return nil
	}

	var results []validator.Result

	for _, link := range allLinks {
		// Skip template URLs containing {placeholder} variables (RFC 6570 URI Templates)
		if strings.Contains(link, "{") {
			continue
		}
		// Skip HTTP(S) links — those are external
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			continue
		}
		// Skip mailto and anchor links
		if strings.HasPrefix(link, "mailto:") || strings.HasPrefix(link, "#") {
			continue
		}
		// Strip fragment identifier (e.g. "guide.md#heading" → "guide.md")
		link, _, _ = strings.Cut(link, "#")
		if link == "" {
			continue
		}
		// Relative link — check file existence
		resolved := filepath.Join(dir, link)
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			results = append(results, validator.Result{Level: validator.Error, Category: "Structure", Message: fmt.Sprintf("broken internal link: %s (file not found)", link)})
		} else {
			results = append(results, validator.Result{Level: validator.Pass, Category: "Structure", Message: fmt.Sprintf("internal link: %s (exists)", link)})
		}
	}

	return results
}
