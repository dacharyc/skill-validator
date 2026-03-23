---
name: review-skill
description: >-
  Review a proposed Agent Skill for structural validity and content
  quality before publishing. Runs the skill-validator CLI to check for
  structural issues, scores the skill with an LLM judge, and interprets results
  to advise authors on what to address. Use when a user wants to review,
  validate, or quality-check an Agent Skill.
compatibility: Requires skill-validator CLI. LLM scoring requires an Anthropic or OpenAI API key, the Claude CLI, OR can be skipped for structural-only review.
metadata:
  author: agent-ecosystem
  version: "1.0"
---

# Review Skill Workflow

You are helping a skill author review an Agent Skill before publishing. This is
a multi-step process: determine environment, verify prerequisites, run
structural validation, review content, optionally run LLM scoring, and
interpret results. Follow every step in order.

## Step 0: Determine Environment

Check for saved configuration:

```bash
cat ~/.config/skill-validator/review-state.yaml 2>/dev/null
```

**If the state file exists** with `prereqs_passed: true`, offer:

> Found saved settings — configured for **[provider/structural-only]** reviews.
>
> 1. **Continue with saved settings** — skip to Step 2
> 2. **Re-run prerequisite checks**
> 3. **Change environment** — switch provider or between LLM and structural-only

Option 1: read `llm_scoring`, `provider`, and `cross_model` from the file and
skip to Step 2.
Options 2-3: continue below.

**If no state file exists**, or the user chose to re-check/change, ask:

> LLM scoring uses an Anthropic or OpenAI-compatible API, or the Claude CLI.
> Without an API key or CLI, we run structural validation only.
>
> 1. **Anthropic** — use Claude via the Anthropic API (requires `ANTHROPIC_API_KEY`)
> 2. **OpenAI** — use GPT via the OpenAI API (requires `OPENAI_API_KEY`)
> 3. **OpenAI-compatible** — use a custom endpoint (Ollama, Groq, Azure, Together, etc.)
> 4. **Claude CLI** — use the locally authenticated `claude` binary (no API key needed)
> 5. **Skip LLM scoring** — structural validation only

Options 1-4: set `LLM_SCORING=true` and record the provider choice.
Option 5: set `LLM_SCORING=false`. Run Step 1a only, then jump to Step 2.

**If the user chose option 1 or 2**, ask about cross-model comparison:

> Scoring with a second model family gives more robust novelty scores, since
> each model has different training data. This requires API keys for both
> Anthropic and OpenAI.
>
> 1. **Yes, compare across model families** — score with both Anthropic and OpenAI
> 2. **No, single provider is fine**

Option 1: set `CROSS_MODEL=true`. Option 2: set `CROSS_MODEL=false`.

Do not offer cross-model comparison for option 3 (OpenAI-compatible) or option 4
(Claude CLI), since the second provider would need a standard Anthropic or
OpenAI key.

After Step 1a, follow
[references/llm-scoring.md](references/llm-scoring.md) for API key checks
before Step 2.

## Step 1: Verify Prerequisites

### 1a. Check for `skill-validator` binary

```bash
skill-validator --version
```

If not found, search common locations (`/usr/local/bin`, `/opt/homebrew/bin`,
`~/go/bin`). If found but not on PATH, tell the user. If not found anywhere,
follow [references/install-skill-validator.md](references/install-skill-validator.md).

Do NOT proceed until this succeeds.

If `LLM_SCORING=true`, complete the provider checks in
[references/llm-scoring.md](references/llm-scoring.md) before continuing.

### Save state after prerequisites pass

Persist state so future runs skip this step. Replace placeholders with actual
values:

```bash
mkdir -p ~/.config/skill-validator
cat > ~/.config/skill-validator/review-state.yaml << 'EOF'
prereqs_passed: true
llm_scoring: <true or false>
provider: <anthropic, openai, openai-compatible, or claude-cli>
model: <model name if specified, or "default">
base_url: <custom base URL if openai-compatible, or omit>
cross_model: <true or false>
EOF
```

## Step 2: Locate the Skill

Ask the user for the path to the skill they want to review, unless they have
already provided it. Verify the path contains a `SKILL.md` file:

```bash
ls <path>/SKILL.md
```

If `SKILL.md` does not exist at the given path, tell the user this is not a
valid skill directory and ask them to provide the correct path.

## Step 3: Run Structural Validation

Run the full check suite:

```bash
skill-validator check <path>
```

Capture the exit code:

| Exit code | Meaning |
|-----------|---------|
| 0 | Clean — no errors or warnings |
| 1 | Errors found — must fix before publishing |
| 2 | Warnings only — review but not blocking |
| 3 | CLI/usage error — check the command |

Exit 0: proceed. Exit 2: note warnings, proceed. Exit 1: list errors — these
are blocking. The user must fix them before the skill can be published. Do NOT
proceed to LLM scoring if exit code is 1.

## Step 4: Content Review

Read the SKILL.md and any reference files, then evaluate each check below.
Report which checks pass and which do not, with specific details on what is
missing.

| Check | Criteria |
|-------|----------|
| Examples | Does the skill provide examples of expected inputs and outputs? |
| Edge cases | Does the skill document common edge cases or failure modes? |
| Scope-gating | Does the skill define when to stop/continue, prerequisites, and conditions for branching paths? |

Flag any failing checks as areas the author should address. These are not
blocking but should be resolved before publishing for best results.

## Step 5: LLM Scoring and Interpretation

If `LLM_SCORING=false`, skip to Step 6.

If `LLM_SCORING=true`, follow the "Run LLM Scoring" and "Interpret LLM Scores"
sections of
[references/llm-scoring.md](references/llm-scoring.md).

## Step 6: Present the Review Summary

If `LLM_SCORING=true`, follow the "Full Review Summary" section of
[references/llm-scoring.md](references/llm-scoring.md).
Include any failing content review checks from Step 4 in the action items.

If `LLM_SCORING=false`, present structural result, content review result,
areas to address, and a self-assessment checklist using the scoring dimensions
from [assets/report.md](assets/report.md). Note that LLM scoring was skipped;
advise re-running with an API key or self-assessing against the report
dimensions.

## Example Review Summary Structure

Structure the final summary with these sections in order:

1. **Structural validation** — pass/fail with errors or warnings
2. **SKILL.md scores** — overall and per-dimension table
3. **Reference scores** — per-file table with overall and lowest dimension
4. **Novelty assessment** — mean novelty vs threshold of 3; list `novel_info`
   per file for author verification
5. **Action items** — prioritized list of what to fix
6. **Recommendation** — ready to publish / minor revisions / significant rework
