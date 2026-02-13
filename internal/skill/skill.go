package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter represents the parsed YAML frontmatter of a SKILL.md file.
type Frontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Metadata      map[string]string `yaml:"metadata"`
	AllowedTools  string            `yaml:"allowed-tools"`
}

// Skill represents a parsed skill package.
type Skill struct {
	Dir           string
	Frontmatter   Frontmatter
	RawFrontmatter map[string]interface{}
	Body          string
	RawContent    string
}

var knownFrontmatterFields = map[string]bool{
	"name":          true,
	"description":   true,
	"license":       true,
	"compatibility": true,
	"metadata":      true,
	"allowed-tools": true,
}

// Load reads and parses a SKILL.md file from the given directory.
func Load(dir string) (*Skill, error) {
	path := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading SKILL.md: %w", err)
	}

	content := string(data)
	skill := &Skill{
		Dir:        dir,
		RawContent: content,
	}

	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	skill.Body = body

	if fm != "" {
		if err := yaml.Unmarshal([]byte(fm), &skill.Frontmatter); err != nil {
			return nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
		}
		if err := yaml.Unmarshal([]byte(fm), &skill.RawFrontmatter); err != nil {
			return nil, fmt.Errorf("parsing raw frontmatter: %w", err)
		}
	}

	return skill, nil
}

// UnrecognizedFields returns frontmatter field names not in the spec.
func (s *Skill) UnrecognizedFields() []string {
	var unknown []string
	for k := range s.RawFrontmatter {
		if !knownFrontmatterFields[k] {
			unknown = append(unknown, k)
		}
	}
	return unknown
}

// splitFrontmatter separates YAML frontmatter (between --- delimiters) from the body.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	if !strings.HasPrefix(content, "---") {
		return "", content, nil
	}

	// Find the closing ---
	rest := content[3:]
	// Skip the newline after opening ---
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}

	// Handle empty frontmatter (closing --- immediately)
	if strings.HasPrefix(rest, "---") {
		frontmatter = ""
		body = rest[3:]
		if len(body) > 0 && body[0] == '\n' {
			body = body[1:]
		} else if len(body) > 1 && body[0] == '\r' && body[1] == '\n' {
			body = body[2:]
		}
		return frontmatter, body, nil
	}

	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return "", "", fmt.Errorf("unterminated frontmatter: missing closing ---")
	}

	frontmatter = strings.TrimRight(rest[:idx], "\r")
	body = rest[idx+4:] // skip \n---
	// Strip leading newline from body
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	} else if len(body) > 1 && body[0] == '\r' && body[1] == '\n' {
		body = body[2:]
	}

	return frontmatter, body, nil
}
