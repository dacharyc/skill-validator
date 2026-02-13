# skill-validator

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
skill-validator <path-to-skill-directory>
skill-validator validate <path-to-skill-directory>
```

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

## What it checks

### Spec compliance

These checks validate conformance with the [Agent Skills specification](https://agentskills.io/specification.md):

- **Structure**: `SKILL.md` exists; only recognized directories (`scripts/`, `references/`, `assets/`); no deep nesting
- **Frontmatter**: required fields (`name`, `description`) are present and valid; `name` is lowercase alphanumeric with hyphens (1-64 chars) and matches the directory name; optional fields (`license`, `compatibility`, `metadata`, `allowed-tools`) conform to expected types and lengths; unrecognized fields are flagged

### Quality checks

These checks go beyond the spec. A skill can be spec-compliant and still perform poorly if an agent wastes context on broken links, irrelevant files, or oversized references.

**Link validation**
- Relative links are resolved against the skill directory and checked for existence
- HTTP/HTTPS links are verified with a HEAD request (10s timeout, concurrent checks)
- Broken links mean an agent will either fail silently or waste context on error handling

**Extraneous file detection**
- Files like `README.md`, `CHANGELOG.md`, and `LICENSE` are flagged at the skill root -- these are for human readers, not agents, and may be loaded into the context window unnecessarily
- Unknown files get a softer warning in case they are intentional
- Based on [Anthropic best practices](https://github.com/anthropics/skills): *"A skill should only contain essential files that directly support its functionality"*

**Token counting and limits**
- Reports per-file and total token counts (using `o200k_base` encoding)
- SKILL.md body: warns if over 5,000 tokens or 500 lines (per spec recommendation)
- Per reference file: warns at 10,000 tokens, errors at 25,000 tokens
- Total references: warns at 25,000 tokens, errors at 50,000 tokens

The reference file limits reflect practical context window budgets. A single 25,000-token reference consumes 12-20% of a typical context window (128k-200k). At 50,000 tokens across all references, you're using 25-40% of the window before the agent has even started working on the actual task. The context window is shared with the system prompt, conversation history, tool output, and the agent's own reasoning -- large reference files crowd all of that out and degrade tool performance.

## Development

```
go test ./...
go vet ./...
```
