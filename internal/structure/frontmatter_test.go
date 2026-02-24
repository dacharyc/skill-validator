package structure

import (
	"strings"
	"testing"

	"github.com/dacharyc/skill-validator/internal/skill"
	"github.com/dacharyc/skill-validator/internal/validator"
)

func makeSkill(dir, name, desc string) *skill.Skill {
	s := &skill.Skill{
		Dir: dir,
		Frontmatter: skill.Frontmatter{
			Name:        name,
			Description: desc,
		},
		RawFrontmatter: map[string]any{},
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
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Error, "name is required")
	})

	t.Run("valid name matching dir", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "A description")
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Pass, `name: "my-skill" (valid)`)
		requireNoResultContaining(t, results, validator.Error, "name")
	})

	t.Run("name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 65)
		s := makeSkill("/tmp/"+longName, longName, "A description")
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Error, "name exceeds 64 characters (65)")
	})

	t.Run("name with uppercase", func(t *testing.T) {
		s := makeSkill("/tmp/My-Skill", "My-Skill", "A description")
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Error, "must be lowercase alphanumeric")
	})

	t.Run("name with consecutive hyphens", func(t *testing.T) {
		s := makeSkill("/tmp/my--skill", "my--skill", "A description")
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Error, "must be lowercase alphanumeric")
	})

	t.Run("name with leading hyphen", func(t *testing.T) {
		s := makeSkill("/tmp/-my-skill", "-my-skill", "A description")
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Error, "must be lowercase alphanumeric")
	})

	t.Run("name with trailing hyphen", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill-", "my-skill-", "A description")
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Error, "must be lowercase alphanumeric")
	})

	t.Run("name does not match directory", func(t *testing.T) {
		s := makeSkill("/tmp/other-dir", "my-skill", "A description")
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Error, "name does not match directory name")
	})

	t.Run("single char name", func(t *testing.T) {
		s := makeSkill("/tmp/a", "a", "A description")
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Pass, `name: "a" (valid)`)
	})

	t.Run("numeric name", func(t *testing.T) {
		s := makeSkill("/tmp/123", "123", "A description")
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Pass, `name: "123" (valid)`)
	})
}

func TestCheckFrontmatter_Description(t *testing.T) {
	t.Run("missing description", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "")
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Error, "description is required")
	})

	t.Run("valid description", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "A valid description")
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Pass, "description: (19 chars)")
	})

	t.Run("description too long", func(t *testing.T) {
		longDesc := strings.Repeat("x", 1025)
		s := makeSkill("/tmp/my-skill", "my-skill", longDesc)
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Error, "description exceeds 1024 characters (1025)")
	})

	t.Run("whitespace-only description", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "   \t\n  ")
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Error, "description must not be empty/whitespace-only")
	})
}

func TestCheckFrontmatter_KeywordStuffing(t *testing.T) {
	t.Run("normal description no warning", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "A skill for building MongoDB vector search applications with best practices.")
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "keyword")
	})

	t.Run("description with a few quoted terms is fine", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", `Use when you see "vector search" or "embeddings" in a query.`)
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "keyword")
	})

	t.Run("description with many quoted strings and little prose", func(t *testing.T) {
		desc := `MongoDB vector search. Triggers on "vector search", "vector index", "$vectorSearch", "embedding", "semantic search", "RAG", "numCandidates".`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Warning, "quoted strings")
		requireResultContaining(t, results, validator.Warning, "what the skill does and when to use it")
	})

	t.Run("prose with supplementary trigger list is fine", func(t *testing.T) {
		desc := `Azure Identity SDK for Python authentication. Use for DefaultAzureCredential, managed identity, service principals, and token caching. Triggers: "azure-identity", "DefaultAzureCredential", "authentication", "managed identity", "service principal", "credential".`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "quoted strings")
		requireNoResultContaining(t, results, validator.Warning, "keyword")
	})

	t.Run("docx skill with trigger examples is fine", func(t *testing.T) {
		desc := `Use this skill whenever the user wants to create, read, edit, or manipulate Word documents (.docx files). Triggers include: any mention of "Word doc", "word document", ".docx", "resume", "cover letter", or requests to produce professional documents with formatting.`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "quoted strings")
		requireNoResultContaining(t, results, validator.Warning, "keyword")
	})

	t.Run("comma-separated keyword list", func(t *testing.T) {
		desc := "MongoDB, Atlas, Vector Search, embeddings, RAG, retrieval, indexing, HNSW, quantization, similarity"
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Warning, "comma-separated segments")
		requireResultContaining(t, results, validator.Warning, "what the skill does and when to use it")
	})

	t.Run("legitimate list of features is fine", func(t *testing.T) {
		desc := "Helps with creating indexes, writing queries, and building applications."
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "keyword")
		requireNoResultContaining(t, results, validator.Warning, "comma-separated")
	})

	t.Run("only one warning when both heuristics match", func(t *testing.T) {
		desc := `Triggers on "a", "b", "c", "d", "e", "f", "g", "h", "i", "j".`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Warning, "quoted strings")
		requireNoResultContaining(t, results, validator.Warning, "comma-separated segments")
	})

	t.Run("prose words equal to quote count is fine", func(t *testing.T) {
		// 5 quotes, 5 prose words (Manage identity tokens using SDK) — boundary: equal should pass
		desc := `Manage identity tokens using SDK. Triggers: "azure", "identity", "token", "credential", "auth".`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "quoted strings")
	})

	t.Run("all quoted strings no prose warns", func(t *testing.T) {
		desc := `"vector search" "embeddings" "RAG" "similarity" "indexing"`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Warning, "quoted strings")
	})

	t.Run("four quoted strings is fine", func(t *testing.T) {
		desc := `Use for "vector search", "embeddings", "RAG", and "similarity" queries.`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "quoted strings")
	})

	t.Run("bare keyword list with some quoted terms still warns", func(t *testing.T) {
		desc := `MongoDB, Atlas, "Vector Search", embeddings, RAG, retrieval, indexing, HNSW, "quantization", similarity`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Warning, "comma-separated segments")
	})

	t.Run("segments below threshold after empty filtering is fine", func(t *testing.T) {
		// Raw commas from quoted strings create empty segments; after filtering, only 4 real segments remain
		desc := `Use this skill for Python authentication and credential management. Triggers: "azure", "identity", "token", "credential", "auth", "login".`
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "comma-separated")
	})

	t.Run("many commas but long segments is fine", func(t *testing.T) {
		desc := "Use when creating vector indexes for search, writing complex aggregation queries with multiple stages, building RAG applications with retrieval patterns, implementing hybrid search with rank fusion, storing AI agent memory in collections, optimizing search performance with explain plans, configuring HNSW index parameters for your workload, tuning numCandidates for recall versus latency tradeoffs"
		s := makeSkill("/tmp/my-skill", "my-skill", desc)
		results := CheckFrontmatter(s)
		requireNoResultContaining(t, results, validator.Warning, "comma-separated segments")
	})
}

func TestCheckFrontmatter_Compatibility(t *testing.T) {
	t.Run("valid compatibility", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.Compatibility = "Works with GPT-4"
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Pass, "compatibility:")
	})

	t.Run("compatibility too long", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.Compatibility = strings.Repeat("x", 501)
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Error, "compatibility exceeds 500 characters (501)")
	})
}

func TestCheckFrontmatter_Metadata(t *testing.T) {
	t.Run("valid string metadata", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.RawFrontmatter["metadata"] = map[string]any{
			"author":  "alice",
			"version": "1.0",
		}
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Pass, "metadata: (2 entries)")
	})

	t.Run("metadata with non-string value", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.RawFrontmatter["metadata"] = map[string]any{
			"count": 42,
		}
		results := CheckFrontmatter(s)
		requireResultContaining(t, results, validator.Error, "metadata[\"count\"] value must be a string")
	})

	t.Run("metadata not a map", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.RawFrontmatter["metadata"] = "not a map"
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Error, "metadata must be a map of string keys to string values")
	})
}

func TestCheckFrontmatter_OptionalFields(t *testing.T) {
	t.Run("license present", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.License = "MIT"
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Pass, `license: "MIT"`)
	})

	t.Run("allowed-tools string", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.AllowedTools = skill.AllowedTools{Value: "Bash Read", WasList: false}
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Pass, `allowed-tools: "Bash Read"`)
		requireNoResultContaining(t, results, validator.Info, "YAML list")
	})

	t.Run("allowed-tools list emits info", func(t *testing.T) {
		s := makeSkill("/tmp/my-skill", "my-skill", "desc")
		s.Frontmatter.AllowedTools = skill.AllowedTools{Value: "Read Bash Grep", WasList: true}
		results := CheckFrontmatter(s)
		requireResult(t, results, validator.Pass, `allowed-tools: "Read Bash Grep"`)
		requireResultContaining(t, results, validator.Info, "YAML list")
		requireResultContaining(t, results, validator.Info, "space-delimited string")
	})
}

func TestCheckFrontmatter_UnrecognizedFields(t *testing.T) {
	s := makeSkill("/tmp/my-skill", "my-skill", "desc")
	s.RawFrontmatter["custom"] = "value"
	results := CheckFrontmatter(s)
	requireResult(t, results, validator.Warning, `unrecognized field: "custom"`)
}
