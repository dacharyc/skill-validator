package structure

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dacharyc/skill-validator/internal/skill"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func CheckFrontmatter(s *skill.Skill) []validator.Result {
	var results []validator.Result

	// Check name
	name := s.Frontmatter.Name
	if name == "" {
		results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: "name is required"})
	} else {
		if len(name) > 64 {
			results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: fmt.Sprintf("name exceeds 64 characters (%d)", len(name))})
		}
		if !namePattern.MatchString(name) {
			results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: fmt.Sprintf("name %q must be lowercase alphanumeric with hyphens, no leading/trailing/consecutive hyphens", name)})
		}
		// Check that name matches directory name
		dirName := filepath.Base(s.Dir)
		if name != dirName {
			results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: fmt.Sprintf("name does not match directory name (expected %q, got %q)", dirName, name)})
		}
		if len(results) == 0 || (name != "" && namePattern.MatchString(name)) {
			results = append(results, validator.Result{Level: validator.Pass, Category: "Frontmatter", Message: fmt.Sprintf("name: %q (valid)", name)})
		}
	}

	// Check description
	desc := s.Frontmatter.Description
	if desc == "" {
		results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: "description is required"})
	} else if len(desc) > 1024 {
		results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: fmt.Sprintf("description exceeds 1024 characters (%d)", len(desc))})
	} else if strings.TrimSpace(desc) == "" {
		results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: "description must not be empty/whitespace-only"})
	} else {
		results = append(results, validator.Result{Level: validator.Pass, Category: "Frontmatter", Message: fmt.Sprintf("description: (%d chars)", len(desc))})
		results = append(results, checkDescriptionKeywordStuffing(desc)...)
	}

	// Check optional license
	if s.Frontmatter.License != "" {
		results = append(results, validator.Result{Level: validator.Pass, Category: "Frontmatter", Message: fmt.Sprintf("license: %q", s.Frontmatter.License)})
	}

	// Check optional compatibility
	if s.Frontmatter.Compatibility != "" {
		if len(s.Frontmatter.Compatibility) > 500 {
			results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: fmt.Sprintf("compatibility exceeds 500 characters (%d)", len(s.Frontmatter.Compatibility))})
		} else {
			results = append(results, validator.Result{Level: validator.Pass, Category: "Frontmatter", Message: fmt.Sprintf("compatibility: (%d chars)", len(s.Frontmatter.Compatibility))})
		}
	}

	// Check optional metadata
	if s.RawFrontmatter["metadata"] != nil {
		// Verify it's a map[string]string
		if m, ok := s.RawFrontmatter["metadata"].(map[string]interface{}); ok {
			allStrings := true
			for k, v := range m {
				if _, ok := v.(string); !ok {
					results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: fmt.Sprintf("metadata[%q] value must be a string", k)})
					allStrings = false
				}
			}
			if allStrings {
				results = append(results, validator.Result{Level: validator.Pass, Category: "Frontmatter", Message: fmt.Sprintf("metadata: (%d entries)", len(m))})
			}
		} else {
			results = append(results, validator.Result{Level: validator.Error, Category: "Frontmatter", Message: "metadata must be a map of string keys to string values"})
		}
	}

	// Check optional allowed-tools
	if !s.Frontmatter.AllowedTools.IsEmpty() {
		results = append(results, validator.Result{Level: validator.Pass, Category: "Frontmatter", Message: fmt.Sprintf("allowed-tools: %q", s.Frontmatter.AllowedTools.Value)})
		if s.Frontmatter.AllowedTools.WasList {
			results = append(results, validator.Result{Level: validator.Info, Category: "Frontmatter", Message: "allowed-tools is a YAML list; the spec defines this as a space-delimited string — both are accepted, but a string is more portable across agent implementations"})
		}
	}

	// Warn on unrecognized fields
	for _, field := range s.UnrecognizedFields() {
		results = append(results, validator.Result{Level: validator.Warning, Category: "Frontmatter", Message: fmt.Sprintf("unrecognized field: %q", field)})
	}

	return results
}

var quotedStringPattern = regexp.MustCompile(`"[^"]*"`)

func checkDescriptionKeywordStuffing(desc string) []validator.Result {
	// Heuristic 1: Many quoted strings suggest keyword/trigger stuffing
	quotes := quotedStringPattern.FindAllString(desc, -1)
	if len(quotes) >= 5 {
		return []validator.Result{{
			Level:    validator.Warning,
			Category: "Frontmatter",
			Message: fmt.Sprintf(
				"description contains %d quoted strings — this looks like keyword stuffing; "+
					"per the spec, the description should concisely describe what the skill does "+
					"and when to use it, not list trigger phrases",
				len(quotes),
			),
		}}
	}

	// Heuristic 2: Many comma-separated short segments suggest a keyword list
	segments := strings.Split(desc, ",")
	if len(segments) >= 8 {
		shortCount := 0
		for _, seg := range segments {
			words := strings.Fields(strings.TrimSpace(seg))
			if len(words) <= 3 {
				shortCount++
			}
		}
		if shortCount*100/len(segments) >= 60 {
			return []validator.Result{{
				Level:    validator.Warning,
				Category: "Frontmatter",
				Message: fmt.Sprintf(
					"description has %d comma-separated segments, most very short — "+
						"this looks like a keyword list; per the spec, the description should "+
						"concisely describe what the skill does and when to use it",
					len(segments),
				),
			}}
		}
	}

	return nil
}
