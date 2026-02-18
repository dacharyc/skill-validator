# skill-validator

[![CI](https://github.com/dacharyc/skill-validator/actions/workflows/ci.yml/badge.svg)](https://github.com/dacharyc/skill-validator/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool that validates and scores [Agent Skill](https://agentskills.io) packages.

Spec compliance is table stakes. `skill-validator` goes further: it checks that links actually resolve, flags files that shouldn't be in a skill directory, reports token counts so you can see how much of an agent's context window your skill will consume, analyzes content quality metrics, detects cross-language contamination, and offers LLM-as-judge scoring to evaluate skill quality across dimensions like clarity, actionability, and novelty. A spec-compliant skill that has broken links or a 60k-token reference file will technically pass the spec but perform poorly in practice.

## Table of Contents

- [Install](#install)
  - [Homebrew](#homebrew)
  - [Using Go](#using-go)
  - [Pre-commit hook](#pre-commit-hook)
- [Command Usage](#command-usage)
  - [validate structure](#validate-structure)
  - [validate links](#validate-links)
  - [analyze content](#analyze-content)
  - [analyze contamination](#analyze-contamination)
  - [check](#check)
  - [score evaluate](#score-evaluate)
  - [score report](#score-report)
  - [JSON output](#json-output)
  - [Multi-skill directories](#multi-skill-directories)
- [What it checks & why](#what-it-checks)
  - [Structure validation](#structure-validation-validate-structure)
  - [Link validation](#link-validation-validate-links)
  - [Content analysis](#content-analysis-analyze-content)
  - [Contamination analysis](#contamination-analysis-analyze-contamination)
  - [LLM scoring](#llm-scoring-score-evaluate)
- [Development](#development)

## Install

You can install in three ways:

- [Homebrew](#homebrew)
- [Using Go](#using-go)
- [Pre-commit hook](#pre-commit-hook)

### Homebrew

```
brew tap dacharyc/tap
brew install skill-validator
```

### Using Go

```
go install github.com/dacharyc/skill-validator@latest
```

Or build from source:

```
git clone https://github.com/dacharyc/skill-validator.git
cd skill-validator
go build -o skill-validator .
```

### Pre-commit hook

`skill-validator` supports [pre-commit](https://pre-commit.com). Platform-specific hooks are provided for all major agent platforms, so the correct skills directory is used automatically. For example, the following configuration runs the skill-validator [`check`](#check) command on the `".claude/skills/"` path:

```yaml
repos:
  - repo: https://github.com/dacharyc/skill-validator
    rev: v0.1.0
    hooks:
      - id: skill-validator-claude
```

Available platform hooks: `skill-validator-amp`, `skill-validator-cline`, `skill-validator-claude`, `skill-validator-codex`, `skill-validator-copilot`, `skill-validator-cursor`, `skill-validator-gemini`, `skill-validator-goose`, `skill-validator-kiro`, `skill-validator-mistral-vibe`, `skill-validator-roo-code`, `skill-validator-trae`, `skill-validator-windsurf`.

A generic `skill-validator` hook is also available if you want to specify a custom command override and/or custom path — supply the command and path via `args`:

```yaml
hooks:
  - id: skill-validator
    args: ["check", "path/to/skills/"]
```

## Command Usage

Commands map to skill development lifecycle stages:

| Development stage | Command | What it answers |
|---|---|---|
| Scaffolding | [`validate structure`](#validate-structure) | Does it conform to the spec and can agents use it? (structure, frontmatter, tokens, code fences, internal links) |
| Writing content | [`analyze content`](#analyze-content) | Is the instruction quality good? (density, specificity, imperative ratio) |
| Adding examples | [`analyze contamination`](#analyze-contamination) | Am I introducing cross-language contamination? |
| Review | [`validate links`](#validate-links) | Do external links still resolve? (HTTP/HTTPS) |
| Quality scoring | [`score evaluate`](#score-evaluate) | How does an LLM judge rate this skill? (clarity, actionability, novelty, etc.) |
| Comparing models | [`score report`](#score-report) | How do scores compare across different LLM providers/models? |
| Pre-publish | [`check`](#check) | Run everything (except LLM scoring) |

All commands accept `-o text` (default) or `-o json` for output format. Use `--version` to print the installed version.

Exit codes: `0` = passed, `1` = validation errors, `2` = usage/tool error.

For more details about how the commands are implemented and what they provide, refer to [What it Checks](#what-it-checks).

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

### score evaluate

Uses an LLM-as-judge approach to score skill quality across multiple dimensions. This is based on findings from the [agent-skill-analysis](https://github.com/dacharyc/agent-skill-analysis) research project, which identified **novelty** as a key predictor of skill value — skills that provide genuinely novel information are more likely to improve LLM outputs, while skills that restate common knowledge can potentially degrade performance.

```
export ANTHROPIC_API_KEY=your-key-here
skill-validator score evaluate <path>
skill-validator score evaluate --skill-only <path>
skill-validator score evaluate --refs-only <path>
skill-validator score evaluate --display files <path>
skill-validator score evaluate path/to/references/api-guide.md
```

**Provider support**: Requires an API key via environment variable. Use `--provider` to select the backend:

| Provider | Env var | Default model | Covers |
|---|---|---|---|
| `anthropic` (default) | `ANTHROPIC_API_KEY` | `claude-sonnet-4-5-20250929` | Anthropic |
| `openai` | `OPENAI_API_KEY` | `gpt-4o` | OpenAI, Ollama, Together, Groq, Azure, etc. |

Use `--model` to override the default model and `--base-url` to point at any OpenAI-compatible endpoint (e.g. `http://localhost:11434/v1` for Ollama).

```
Scoring skill: my-skill/

SKILL.md Scores
  Clarity:              4/5
  Actionability:        5/5
  Token Efficiency:     3/5
  Scope Discipline:     4/5
  Directive Precision:  4/5
  Novelty:              2/5
  ──────────────────────────────
  Overall:              3.67/5

  "Clear instructions but mostly restates common React patterns."

Reference Scores (2 files)
  Clarity:              4/5
  Instructional Value:  4/5
  Token Efficiency:     4/5
  Novelty:              3/5
  Skill Relevance:      5/5
  ──────────────────────────────
  Overall:              4.00/5
```

**Targeting**:
- Pass a skill directory to score everything (SKILL.md + references)
- Use `--skill-only` to score just SKILL.md, `--refs-only` for just references
- Pass a specific file path (e.g. `path/to/references/api-guide.md`) to score a single reference file — useful for iterating on one file without burning API calls on everything else

**Content truncation**: By default, file content is truncated to 8,000 characters before sending to the LLM. Use `--full-content` to send the entire file — useful for large reference files where the scoring should account for all content, at the cost of higher token usage.

**Caching**: Results are cached in `.score_cache/` inside the skill directory. Cache keys are based on provider, model, and file path, so different models produce separate cache entries while editing a file and re-running overwrites the previous result for that file. Use `--rescore` to force re-scoring and overwrite cached results.

### score report

```
skill-validator score report <path>
skill-validator score report --list <path>
skill-validator score report --compare <path>
skill-validator score report --model claude-sonnet-4-5-20250929 <path>
```

Views and compares cached LLM scores without making API calls.

- **Default** (no flags): shows the most recent scores for each file
- `--list`: tabular summary of all cached entries with metadata (model, timestamp, provider)
- `--compare`: side-by-side comparison of dimension scores across different models
- `--model`: filter to scores from a specific model

The `--compare` flag is useful for understanding how different models perceive your skill's quality. For example, scoring with both Claude and GPT-4o can reveal whether novelty ratings are consistent across model families, or whether one model finds your instructions clearer than another.

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

- [Structure validation](#structure-validation-validate-structure)
- [Link validation](#link-validation-validate-links)
- [Content analysis](#content-analysis-analyze-content)
- [Contamination analysis](#contamination-analysis-analyze-contamination)
- [LLM scoring](#llm-scoring-score-evaluate)

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

> [!TIP]
> Extraneous file detection and recognized directories are based on the Agent Skills specification. Platform support for the spec may vary; some platforms show using different directory structures and additional files at skill root. Adhering to the spec is the best way to validate skill content is portable across platforms, so skill-validator checks against the spec.

**Keyword stuffing detection**
- Descriptions with 5+ quoted strings are flagged when the surrounding prose has fewer words than the number of quoted strings — a prose sentence followed by a supplementary trigger list (e.g., `Triggers: "term1", "term2"`) is fine
- Descriptions with 8+ comma-separated short segments (after excluding quoted strings) are flagged as keyword lists
- Per the spec, the description should concisely describe what the skill does and when to use it

**Token counting and limits**
- Reports per-file and total token counts (using `o200k_base` encoding)
- SKILL.md body: warns if over 5,000 tokens or 500 lines (per spec recommendation)
- Per reference file: warns at 10,000 tokens, errors at 25,000 tokens
- Total references: warns at 25,000 tokens, errors at 50,000 tokens
- Asset files: text-based files in `assets/` (`.md`, `.tex`, `.py`, `.yaml`, `.yml`, `.tsx`, `.ts`, `.jsx`, `.sty`, `.mplstyle`, `.ipynb`) are counted and reported in an "Asset files" section — these are templates, guides, and configs that LLMs load into context; non-text assets (images, binaries) are ignored
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

### LLM scoring (`score evaluate`)

Uses an LLM-as-judge approach ported from the [agent-skill-analysis](https://github.com/dacharyc/agent-skill-analysis) research project. The scoring prompts instruct the LLM to evaluate skill content on specific quality dimensions, returning structured JSON scores.

**SKILL.md** is scored on 6 dimensions (1-5 each):
- **Clarity**: How clear and unambiguous are the instructions?
- **Actionability**: Can an agent follow them step-by-step?
- **Token Efficiency**: Does every token earn its place in the context window?
- **Scope Discipline**: Does it stay focused on its stated purpose?
- **Directive Precision**: Does it use precise directives (must, always, never) vs vague suggestions?
- **Novelty**: How much content goes beyond what an LLM already knows from training data?

**Reference files** are scored on 5 dimensions (1-5 each):
- **Clarity**, **Token Efficiency**, **Novelty** (same as above)
- **Instructional Value**: Does it provide concrete, directly-applicable examples?
- **Skill Relevance**: Does every section support the parent skill's purpose?

## Development

```
go test ./...
go vet ./...
```
