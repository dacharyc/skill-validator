package validator

import (
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/skill"
)

func makeSkill(dir, name, desc string) *skill.Skill {
	s := &skill.Skill{
		Dir: dir,
		Frontmatter: skill.Frontmatter{
			Name:        name,
			Description: desc,
		},
		RawFrontmatter: map[string]interface{}{},
	}
	if name != "" {
		s.RawFrontmatter["name"] = name
	}
	if desc != "" {
		s.RawFrontmatter["description"] = desc
	}
	return s
}

func TestCheckFrontmatter_Name(t *testing.T) {
	t.Run("missing name", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "", "A description")
		results := checkFrontmatter(s)
		requireResult(t, results, Error, "name is required")
	})

	t.Run("valid name matching dir", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "A description")
		results := checkFrontmatter(s)
		requireResult(t, results, Pass, `name: "my-skill" (valid)`)
		requireNoResultContaining(t, results, Error, "name")
	})

	t.Run("name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 65)
		s := makeSkill("/tmp/"+longName, longName, "A description")
		results := checkFrontmatter(s)
		requireResult(t, results, Error, "name exceeds 64 characters (65)")
	})

	t.Run("name with uppercase", func(t *testing.T) {
		s := makeSkill("/tmp/My-Skill", "My-Skill", "A description")
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Error, "must be lowercase alphanumeric")
	})

	t.Run("name with consecutive hyphens", func(t *testing.T) {
		s := makeSkill("/tmp/my--skill", "my--skill", "A description")
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Error, "must be lowercase alphanumeric")
	})

	t.Run("name with leading hyphen", func(t *testing.T) {
		s := makeSkill("/tmp/-my-skill", "-my-skill", "A description")
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Error, "must be lowercase alphanumeric")
	})

	t.Run("name with trailing hyphen", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill-", "my-skill-", "A description")
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Error, "must be lowercase alphanumeric")
	})

	t.Run("name does not match directory", func(t *testing.T) {
		s := makeSkill("/tmp/other-dir", "my-skill", "A description")
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Error, "name does not match directory name")
	})

	t.Run("single char name", func(t *testing.T) {
		s := makeSkill("/tmp/a", "a", "A description")
		results := checkFrontmatter(s)
		requireResult(t, results, Pass, `name: "a" (valid)`)
	})

	t.Run("numeric name", func(t *testing.T) {
		s := makeSkill("/tmp/123", "123", "A description")
		results := checkFrontmatter(s)
		requireResult(t, results, Pass, `name: "123" (valid)`)
	})
}

func TestCheckFrontmatter_Description(t *testing.T) {
	t.Run("missing description", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "")
		results := checkFrontmatter(s)
		requireResult(t, results, Error, "description is required")
	})

	t.Run("valid description", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "A valid description")
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Pass, "description: (19 chars)")
	})

	t.Run("description too long", func(t *testing.T) {
		longDesc := strings.Repeat("x", 1025)
		s := makeSkill("/tmp/my-skill", "my-skill", longDesc)
		results := checkFrontmatter(s)
		requireResult(t, results, Error, "description exceeds 1024 characters (1025)")
	})

	t.Run("whitespace-only description", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "   \t\n  ")
		results := checkFrontmatter(s)
		requireResult(t, results, Error, "description must not be empty/whitespace-only")
	})
}

func TestCheckFrontmatter_Compatibility(t *testing.T) {
	t.Run("valid compatibility", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.Compatibility = "Works with GPT-4"
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Pass, "compatibility:")
	})

	t.Run("compatibility too long", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.Compatibility = strings.Repeat("x", 501)
		results := checkFrontmatter(s)
		requireResult(t, results, Error, "compatibility exceeds 500 characters (501)")
	})
}

func TestCheckFrontmatter_Metadata(t *testing.T) {
	t.Run("valid string metadata", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.RawFrontmatter["metadata"] = map[string]interface{}{
			"author":  "alice",
			"version": "1.0",
		}
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Pass, "metadata: (2 entries)")
	})

	t.Run("metadata with non-string value", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.RawFrontmatter["metadata"] = map[string]interface{}{
			"count": 42,
		}
		results := checkFrontmatter(s)
		requireResultContaining(t, results, Error, "metadata[\"count\"] value must be a string")
	})

	t.Run("metadata not a map", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.RawFrontmatter["metadata"] = "not a map"
		results := checkFrontmatter(s)
		requireResult(t, results, Error, "metadata must be a map of string keys to string values")
	})
}

func TestCheckFrontmatter_OptionalFields(t *testing.T) {
	t.Run("license present", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.License = "MIT"
		results := checkFrontmatter(s)
		requireResult(t, results, Pass, `license: "MIT"`)
	})

	t.Run("allowed-tools present", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.AllowedTools = "Bash, Read"
		results := checkFrontmatter(s)
		requireResult(t, results, Pass, `allowed-tools: "Bash, Read"`)
	})
}

func TestCheckFrontmatter_UnrecognizedFields(t *testing.T) {
	s := makeSkill("/tmp/my-skill", "my-skill", "desc")
	s.RawFrontmatter["custom"] = "value"
	results := checkFrontmatter(s)
	requireResult(t, results, Warning, `unrecognized field: "custom"`)
}
