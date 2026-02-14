# skill-validator

[![CI](https://github.com/dacharyc/skill-validator/actions/workflows/ci.yml/badge.svg)](https://github.com/dacharyc/skill-validator/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool that validates [Agent Skill](https://agentskills.io) packages.

Spec compliance is table stakes. `skill-validator` goes further: it checks that links actually resolve, flags files that shouldn't be in a skill directory, and reports token counts so you can see how much of an agent's context window your skill will consume. A spec-compliant skill that has broken links or a 60k-token reference file will technically pass the spec but perform poorly in practice.

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

## Usage

```
skill-validator [-o format] <path-to-skill-directory>
```

Flags:

- `-o`, `--output` — output format: `text` (default) or `json`

Exit codes: `0` = passed, `1` = validation errors, `2` = usage/tool error.

### Example output

```
Validating skill: my-skill/

Structure
  ✓ SKILL.md found

Frontmatter
  ✓ name: "my-skill" (valid)
  ✓ description: (54 chars)
  ✓ license: "MIT"

Links
  ✓ references/guide.md (exists)

Tokens
  SKILL.md body:        1,250 tokens
  references/guide.md:    820 tokens
  ─────────────────────────────────────
  Total:                2,070 tokens

Result: passed
```

### JSON output

Use `-o json` for machine-readable output:

```
skill-validator -o json my-skill/
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
  }
}
```

The `passed` field is `true` when `errors` is `0`. Token count sections are omitted when empty. Pipe to `jq` for post-processing:

```
skill-validator -o json my-skill/ | jq '.results[] | select(.level == "error")'
```

### Multi-skill directories

If the given path does not contain a `SKILL.md` but has subdirectories that do, the validator automatically detects and validates each skill. This is useful when skills are organized as sibling directories (e.g. `skills/algorithmic-art/`, `skills/brand-guidelines/`). Symlinks are followed during detection.

```
skill-validator skills/
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

### Spec compliance

These checks validate conformance with the [Agent Skills specification](https://agentskills.io/specification):

- **Structure**: `SKILL.md` exists; only recognized directories (`scripts/`, `references/`, `assets/`); no deep nesting
- **Frontmatter**: required fields (`name`, `description`) are present and valid; `name` is lowercase alphanumeric with hyphens (1-64 chars) and matches the directory name; optional fields (`license`, `compatibility`, `metadata`, `allowed-tools`) conform to expected types and lengths; unrecognized fields are flagged

### Quality checks

These checks go beyond the spec. A skill can be spec-compliant and still perform poorly if an agent wastes context on broken links, irrelevant files, or oversized references.

**Link validation**
- Relative links are resolved against the skill directory and checked for existence
- HTTP/HTTPS links are verified with a HEAD request (10s timeout, concurrent checks)
- Template URLs using [RFC 6570](https://www.rfc-editor.org/rfc/rfc6570) syntax are skipped (e.g. `https://github.com/{OWNER}/{REPO}/pull/{PR}`)
- Broken links mean an agent will either fail silently or waste context on error handling

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

**Markdown validation**
- Checks SKILL.md and reference files for unclosed code fences (`` ``` `` or `~~~`)
- An unclosed fence causes agents to misinterpret everything after it as code, which can silently break comprehension of the rest of the file

**Token counting and limits**
- Reports per-file and total token counts (using `o200k_base` encoding)
- SKILL.md body: warns if over 5,000 tokens or 500 lines (per spec recommendation)
- Per reference file: warns at 10,000 tokens, errors at 25,000 tokens
- Total references: warns at 25,000 tokens, errors at 50,000 tokens
- Non-standard files (anything outside SKILL.md, references/, scripts/, assets/) are scanned separately and reported in an "Other files" section with per-file and total token counts
- Other files total: warns at 25,000 tokens, errors at 100,000 tokens
- Individual other-file counts are color-coded: yellow over 10k tokens, red over 25k

The reference file limits reflect practical context window budgets. A single 25,000-token reference consumes 12-20% of a typical context window (128k-200k). At 50,000 tokens across all references, you're using 25-40% of the window before the agent has even started working on the actual task. The context window is shared with the system prompt, conversation history, tool output, and the agent's own reasoning -- large reference files crowd all of that out and degrade tool performance.

**Holistic structure check**
- If non-standard content exceeds 10x the standard structure content (and is over 25,000 tokens), the validator errors with a clear message that the directory doesn't appear to be structured as a skill
- This catches build pipeline issues, documentation repos with a SKILL.md tacked on, and other cases where the content fundamentally isn't a skill

## Development

```
go test ./...
go vet ./...
```
