# skill-validator

[![CI](https://github.com/dacharyc/skill-validator/actions/workflows/ci.yml/badge.svg)](https://github.com/dacharyc/skill-validator/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool that validates [Agent Skill](https://agentskills.io) packages.

Spec compliance is table stakes. `skill-validator` goes further: it checks that links actually resolve, flags files that shouldn't be in a skill directory, reports token counts so you can see how much of an agent's context window your skill will consume, analyzes content quality metrics, and detects cross-language contamination. A spec-compliant skill that has broken links or a 60k-token reference file will technically pass the spec but perform poorly in practice.

## Install

```
go install github.com/dacharyc/skill-validator@latest
```

Or build from source:

```
git clone https://github.com/dacharyc/skill-validator.git
cd skill-validator
go build -o skill-validator .
```

## Commands

Commands map to skill development lifecycle stages:

| Development stage | Command | What it answers |
|---|---|---|
| Scaffolding | `validate structure` | Does it conform to the spec and can agents use it? (structure, frontmatter, tokens, code fences, internal links) |
| Writing content | `analyze content` | Is the instruction quality good? (density, specificity, imperative ratio) |
| Adding examples | `analyze contamination` | Am I introducing cross-language contamination? |
| Review | `validate links` | Do external links still resolve? (HTTP/HTTPS) |
| Pre-publish | `check` | Run everything |

All commands accept `-o text` (default) or `-o json` for output format. Use `--version` to print the installed version.

Exit codes: `0` = passed, `1` = validation errors, `2` = usage/tool error.

### validate structure

```
skill-validator validate structure <path>
```

Checks spec compliance: directory structure, frontmatter fields, token limits, skill ratio, code fence integrity, and internal link validity.

```
Validating skill: my-skill/

Structure
  ✓ SKILL.md found

Frontmatter
  ✓ name: "my-skill" (valid)
  ✓ description: (54 chars)
  ✓ license: "MIT"

Markdown
  ✓ no unclosed code fences found

Tokens
  SKILL.md body:        1,250 tokens
  references/guide.md:    820 tokens
  ─────────────────────────────────────
  Total:                2,070 tokens

Result: passed
```

### validate links

```
skill-validator validate links <path>
```

Validates external (HTTP/HTTPS) links in SKILL.md. Internal (relative) links are checked by `validate structure`.

### analyze content

```
skill-validator analyze content <path>
skill-validator analyze content --per-file <path>
```

Computes content quality metrics for SKILL.md and reference markdown files:

```
Content Analysis
  Word count:               1,250
  Code block ratio:         0.32
  Imperative ratio:         0.45
  Information density:      0.39
  Instruction specificity:  0.78
  Sections: 6  |  List items: 23  |  Code blocks: 8

References Content Analysis
  Word count:               820
  ...

References Contamination Analysis
  Contamination level: low (score: 0.00)
  Scope breadth: 0
```

Metrics include word count, code block count/ratio, code languages, sentence count, imperative sentence ratio, information density, strong/weak language markers, instruction specificity, section count, and list item count. Reference files in `references/` are analyzed in aggregate. Use `--per-file` to see a breakdown by individual reference file.

### analyze contamination

```
skill-validator analyze contamination <path>
skill-validator analyze contamination --per-file <path>
```

Detects cross-language contamination — skills where code examples in one language could cause incorrect code generation in another context. Analyzes both SKILL.md and reference markdown files:

```
Contamination Analysis
  Contamination level: medium (score: 0.35)
  Primary language category: javascript
  ⚠ Language mismatch: python, shell (2 categories differ from primary)
  ℹ Multi-interface tool detected: mongodb
  Scope breadth: 4

References Contamination Analysis
  Contamination level: low (score: 0.00)
  Scope breadth: 0
```

Contamination scoring considers three factors: multi-interface tools (0.3 weight), language mismatch across code blocks (0.4 weight), and scope breadth (0.3 weight). Reference files in `references/` are analyzed in aggregate. Use `--per-file` to see a breakdown by individual reference file.

### check

```
skill-validator check <path>
skill-validator check --only structure,links <path>
skill-validator check --skip contamination <path>
skill-validator check --per-file <path>
```

Runs all checks (structure + links + content + contamination). Use `--only` or `--skip` to select specific check groups. The flags are mutually exclusive. Use `--per-file` to see per-file reference analysis alongside the aggregate.

Valid check groups: `structure`, `links`, `content`, `contamination`.

### JSON output

Use `-o json` for machine-readable output:

```
skill-validator check -o json my-skill/
```

```json
{
  "skill_dir": "/path/to/my-skill",
  "passed": true,
  "errors": 0,
  "warnings": 0,
  "results": [
    { "level": "pass", "category": "Structure", "message": "SKILL.md found" }
  ],
  "token_counts": {
    "files": [
      { "file": "SKILL.md body", "tokens": 1250 },
      { "file": "references/guide.md", "tokens": 820 }
    ],
    "total": 2070
  },
  "content_analysis": {
    "word_count": 1250,
    "code_block_count": 5,
    "code_block_ratio": 0.25,
    "code_languages": ["python", "bash"],
    "imperative_ratio": 0.35,
    "information_density": 0.30,
    "instruction_specificity": 0.78,
    "section_count": 4,
    "list_item_count": 12
  },
  "references_content_analysis": { "..." : "same shape as content_analysis" },
  "contamination_analysis": {
    "multi_interface_tools": ["mongodb"],
    "contamination_score": 0.35,
    "contamination_level": "medium",
    "language_mismatch": true,
    "scope_breadth": 4
  },
  "references_contamination_analysis": { "..." : "same shape as contamination_analysis" },
  "reference_reports": [
    {
      "file": "guide.md",
      "content_analysis": { "..." : "same shape" },
      "contamination_analysis": { "..." : "same shape" }
    }
  ]
}
```

The `passed` field is `true` when `errors` is `0`. Token count, content analysis, and contamination analysis sections are omitted when not computed. The `reference_reports` array is only included with `--per-file`. Pipe to `jq` for post-processing:

```
skill-validator check -o json my-skill/ | jq '.content_analysis'
skill-validator check -o json my-skill/ | jq '.results[] | select(.level == "error")'
```

### Multi-skill directories

If the given path does not contain a `SKILL.md` but has subdirectories that do, the validator automatically detects and validates each skill. This is useful when skills are organized as sibling directories (e.g. `skills/algorithmic-art/`, `skills/brand-guidelines/`). Symlinks are followed during detection.

```
skill-validator check skills/
```

Each skill is validated independently. The text output separates skills with a line and appends an overall summary. The JSON output wraps individual skill reports in a `skills` array:

```json
{
  "passed": false,
  "errors": 3,
  "warnings": 1,
  "skills": [
    { "skill_dir": "...", "passed": true, "errors": 0, "warnings": 0, "results": [...] },
    { "skill_dir": "...", "passed": false, "errors": 3, "warnings": 1, "results": [...] }
  ]
}
```

If no `SKILL.md` is found at the root or in any immediate subdirectory, the validator exits with code 2.

## What it checks

### Structure validation (`validate structure`)

These checks validate conformance with the [Agent Skills specification](https://agentskills.io/specification) and perform additional checks:

- **Structure**: `SKILL.md` exists; only recognized directories (`scripts/`, `references/`, `assets/`); no deep nesting
- **Frontmatter**: required fields (`name`, `description`) are present and valid; `name` is lowercase alphanumeric with hyphens (1-64 chars) and matches the directory name; optional fields (`license`, `compatibility`, `metadata`, `allowed-tools`) conform to expected types and lengths; unrecognized fields are flagged

**Extraneous file detection**
- Files like `README.md`, `CHANGELOG.md`, and `LICENSE` are flagged at the skill root -- these are for human readers, not agents, and may be loaded into the context window unnecessarily
- `AGENTS.md` gets a specific warning: it's for repo-level agent configuration, not skill content, and should live outside the skill directory
- Unknown files suggest moving content into `references/` or `assets/` as appropriate
- Unknown directories report how many files they contain and suggest standard alternatives (when applicable)
- Based on Anthropic's [skill-creator](https://github.com/anthropics/skills/blob/main/skills/skill-creator/SKILL.md): *"A skill should only contain essential files that directly support its functionality"*

**Keyword stuffing detection**
- Descriptions with 5+ quoted strings are flagged as likely trigger-phrase stuffing
- Descriptions with 8+ comma-separated short segments are flagged as keyword lists
- Per the spec, the description should concisely describe what the skill does and when to use it

**Token counting and limits**
- Reports per-file and total token counts (using `o200k_base` encoding)
- SKILL.md body: warns if over 5,000 tokens or 500 lines (per spec recommendation)
- Per reference file: warns at 10,000 tokens, errors at 25,000 tokens
- Total references: warns at 25,000 tokens, errors at 50,000 tokens
- Non-standard files (anything outside SKILL.md, references/, scripts/, assets/) are scanned separately and reported in an "Other files" section with per-file and total token counts
- Other files total: warns at 25,000 tokens, errors at 100,000 tokens

**Holistic structure check**
- If non-standard content exceeds 10x the standard structure content (and is over 25,000 tokens), the validator errors with a clear message that the directory doesn't appear to be structured as a skill

**Markdown validation**
- Checks SKILL.md and reference files for unclosed code fences (`` ``` `` or `~~~`)
- An unclosed fence causes agents to misinterpret everything after it as code
- Unclosed fences are reported as errors (not warnings) because they break agent usability

**Internal link validation**
- Relative links in SKILL.md are resolved against the skill directory and checked for existence
- A broken internal link means the skill references a file that doesn't exist in the package -- this is a structural problem, not a network issue, so it's checked here rather than in `validate links`
- Broken internal links are reported as errors

### Link validation (`validate links`)

- Checks external (HTTP/HTTPS) links only -- internal (relative) links are validated by `validate structure`
- HTTP/HTTPS links are verified with a HEAD request (10s timeout, concurrent checks)
- Template URLs using [RFC 6570](https://www.rfc-editor.org/rfc/rfc6570) syntax are skipped (e.g. `https://github.com/{OWNER}/{REPO}/pull/{PR}`)

> [!TIP]
> HTTP 403 responses are reported as `info` rather than errors, since many sites (e.g. doi.org, science.org, mathworks.com) block automated HEAD requests while working fine in browsers. A 403 doesn't necessarily mean the link is broken -- but it does mean the validator couldn't verify it. If your skill includes 403-flagged links, keep in mind that sites blocking the validator's requests may also block requests from LLM agents. If an agent can't access a linked resource, the link wastes context without providing value. Where possible, consider providing the content directly in `references/` rather than linking to it, or offer an alternate source that doesn't restrict automated access. If the links are for human readers rather than agent use, consider removing them from the skill entirely.

### Content analysis (`analyze content`)

Computes content quality metrics ported from the [agent-skill-analysis](https://github.com/dacharyc/agent-skill-analysis) research project. Analyzes SKILL.md and markdown files in `references/` (aggregate and per-file):

- **Word count**: total words in SKILL.md
- **Code block count / ratio**: number and proportion of fenced code blocks
- **Code languages**: language identifiers from code block markers
- **Sentence count**: approximate sentences (split on punctuation and blank lines, after stripping code)
- **Imperative count / ratio**: sentences starting with imperative verbs (use, run, create, configure, etc.)
- **Strong markers**: directive language count (must, always, never, required, ensure, etc.)
- **Weak markers**: advisory language count (may, consider, could, optional, suggested, etc.)
- **Instruction specificity**: strong / (strong + weak) — how directive vs advisory the language is
- **Information density**: (code_block_ratio * 0.5) + (imperative_ratio * 0.5)
- **Section count**: H2+ headers
- **List item count**: bullet and numbered list items

### Contamination analysis (`analyze contamination`)

Detects cross-language contamination — where code examples in one language could cause incorrect generation in another context. Analyzes SKILL.md and markdown files in `references/` (aggregate and per-file):

- **Multi-interface tools**: detects tools with many language bindings (MongoDB, AWS, Docker, Kubernetes, Redis, etc.) by scanning the skill name and content
- **Language categories**: maps code block languages to broad categories (shell, javascript, python, java, systems, config, etc.)
- **Language mismatch**: code blocks spanning different language categories
- **Technology references**: framework/runtime mentions (Node.js, Django, Flask, Spring, Rails, etc.)
- **Scope breadth**: number of distinct technology categories referenced
- **Contamination score**: 3-factor formula — multi_interface (0.3) + mismatch (0.4) + breadth (0.3), capped at 1.0
- **Contamination level**: high (≥0.5), medium (≥0.2), low (<0.2)

## Development

```
go test ./...
go vet ./...
```
