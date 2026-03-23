# LLM Scoring Flow

Provider-specific prerequisites and LLM scoring steps. Only follow this if the
user selected an LLM provider in Step 0.

## Provider Prerequisites

Complete after Step 1a (binary check) passes.

### Anthropic provider

Verify the API key is set:

```bash
test -n "$ANTHROPIC_API_KEY" && echo "Key is set" || echo "Key is NOT set"
```

If not set, tell the user to export it:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

The default model is `claude-sonnet-4-5-20250929`. The user can specify a
different Anthropic model with the `--model` flag.

### OpenAI provider

Verify the API key is set:

```bash
test -n "$OPENAI_API_KEY" && echo "Key is set" || echo "Key is NOT set"
```

If not set, tell the user to export it:

```bash
export OPENAI_API_KEY=sk-...
```

The default model is `gpt-5.2`. The user can specify a different model with the
`--model` flag. For applications where a frontier model is more appropriate,
the user can specify `--model gpt-5.4`, but this will increase scoring cost.

### Claude CLI provider

The Claude CLI provider shells out to the locally installed `claude` binary.
No API key is needed — it uses the CLI's existing authentication (e.g. a
company or team subscription).

Verify the CLI is available:

```bash
claude --version
```

If not found, see https://docs.anthropic.com/en/docs/claude-code for
installation instructions.

The default model is `sonnet`. The user can specify a different model with the
`--model` flag (e.g. `--model opus`).

### OpenAI-compatible provider

This uses the OpenAI provider with a custom `--base-url`. It supports any
OpenAI-compatible API: Ollama, Groq, Azure OpenAI, Together, vLLM, and others.

Verify the API key is set (some local providers like Ollama may not require one,
but the CLI still expects it):

```bash
test -n "$OPENAI_API_KEY" && echo "Key is set" || echo "Key is NOT set"
```

Ask the user for:

1. **Base URL** — the endpoint for their provider (e.g., `http://localhost:11434/v1`
   for Ollama, `https://api.groq.com/openai/v1` for Groq)
2. **Model name** — the model identifier their endpoint expects

For providers that do not require a real API key (e.g., Ollama), the user can
set a placeholder:

```bash
export OPENAI_API_KEY=not-needed
```

If the model uses the newer `max_completion_tokens` parameter instead of
`max_tokens` (common for reasoning models), the user can override detection
with `--max-tokens-style max_completion_tokens`. By default, skill-validator
auto-detects based on the model name.

Do NOT proceed until the API key check passes.

### Cross-model comparison

If `CROSS_MODEL=true`, verify that **both** API keys are set:

```bash
test -n "$ANTHROPIC_API_KEY" && echo "Anthropic key is set" || echo "Anthropic key is NOT set"
test -n "$OPENAI_API_KEY" && echo "OpenAI key is set" || echo "OpenAI key is NOT set"
```

Both keys must be present before proceeding. If one is missing, tell the user
to export it. Do NOT proceed until both keys are set.

## Run LLM Scoring (after structural validation passes)

Build the scoring command based on the provider selected in Step 0.

### Anthropic (default)

Check for cached scores:

```bash
skill-validator score report <path> -o json 2>/dev/null
```

If scored output exists, use `--rescore` to generate fresh scores:

```bash
skill-validator score evaluate <path> --full-content --display files -o json --rescore
```

If no cached scores exist, run without `--rescore`:

```bash
skill-validator score evaluate <path> --full-content --display files -o json
```

### OpenAI

Check for cached scores:

```bash
skill-validator score report <path> -o json 2>/dev/null
```

If scored output exists, use `--rescore` to generate fresh scores:

```bash
skill-validator score evaluate <path> --provider openai --full-content --display files -o json --rescore
```

If no cached scores exist, run without `--rescore`:

```bash
skill-validator score evaluate <path> --provider openai --full-content --display files -o json
```

Add `--model <name>` if the user specified a model other than gpt-5.2.

### Claude CLI

Check for cached scores:

```bash
skill-validator score report <path> -o json 2>/dev/null
```

If scored output exists, use `--rescore` to generate fresh scores:

```bash
skill-validator score evaluate <path> --provider claude-cli --full-content --display files -o json --rescore
```

If no cached scores exist, run without `--rescore`:

```bash
skill-validator score evaluate <path> --provider claude-cli --full-content --display files -o json
```

Add `--model <name>` if the user specified a model other than sonnet.

### OpenAI-compatible

```bash
skill-validator score evaluate <path> --provider openai --base-url <url> --model <model> --full-content --display files -o json
```

Add `--max-tokens-style max_completion_tokens` if needed for reasoning models.

### After scoring completes

If `CROSS_MODEL=true`, skip this step and proceed to the "Cross-Model
Comparison" section instead.

Run the report:

```bash
skill-validator score report <path> -o json
```

Capture the output for interpretation.

### If scoring fails

Read the error message and match it to one of these categories:

| Error pattern | Cause | Action |
|---------------|-------|--------|
| "invalid API key", "Incorrect API key", 401/403 status | Key is wrong or revoked | Tell the user to verify the key value and re-export it |
| "insufficient quota", "rate limit", 429 status | Account billing or rate limit | Tell the user to check their account billing/usage at their provider's dashboard |
| "model not found", "does not exist" | Wrong model name for this provider | Tell the user to verify the model name; show the `--model` flag |
| "connection refused", timeout, DNS errors | Endpoint unreachable (common with OpenAI-compatible) | Tell the user to verify the `--base-url` and that the server is running |
| Other errors | Transient or unexpected | Show the full error to the user and suggest retrying once |

For authentication errors (first row), invalidate saved state so prerequisite
checks re-run:

```bash
rm -f ~/.config/skill-validator/review-state.yaml
```

For other errors, do NOT invalidate saved state. The configuration is likely
correct; the issue is elsewhere.

## Cross-Model Comparison (if CROSS_MODEL=true)

After completing the primary scoring run, score the skill again with the other
provider. This produces a second set of scores from a different model family,
which is especially valuable for novelty assessment since each model has
different training data coverage.

If the primary provider was **Anthropic**, run the OpenAI scoring:

```bash
skill-validator score evaluate <path> --provider openai --full-content --display files -o json
```

If the primary provider was **OpenAI**, run the Anthropic scoring:

```bash
skill-validator score evaluate <path> --full-content --display files -o json
```

After both scoring runs complete, generate the comparison report:

```bash
skill-validator score report <path> --compare -o json
```

This shows side-by-side dimension scores across models. Use this to:

- **Validate novelty**: If both model families rate novelty below 3, the skill
  almost certainly restates common knowledge. If one rates it high and the other
  low, the skill may contain information that is novel to one model's training
  data but not the other. Flag this for the author to investigate.
- **Spot scoring bias**: Large divergences on other dimensions (more than 1
  point apart) suggest the content may be structured in a way that favors one
  model family. The author should review those dimensions.
- **Increase confidence**: When both models agree on scores, the assessment is
  more reliable than a single-model evaluation.

## Interpret LLM Scores

Read [../assets/report.md](../assets/report.md) for the full interpretation
framework, then present results to the user following that structure.

### Quality thresholds

There are no hard pass/fail gates on most dimensions. Use these guidelines:

- **Overall >= 3.5**: The skill is in good shape across most dimensions.
- **Overall 2.5-3.5**: The skill needs work in specific areas. Identify which
  dimensions are dragging the score down and advise accordingly.
- **Overall < 2.5**: The skill needs significant revision across multiple areas.
- **Any dimension at 2 or below**: Flag this specifically as an area needing
  attention. Explain what a low score on that dimension means and suggest
  concrete improvements.

### Novelty is the key differentiator

If mean novelty across all files is below 3, the skill may not justify its
context window cost. A low-novelty skill restates what models already know.
The critical question: **does this skill teach the agent something it genuinely
doesn't know?**

### Surface `novel_info` for author review

Each scored file includes a `novel_info` field describing what the LLM judge
identified as genuinely novel content. Present these details to the author for
each file, because:

- **Verification**: The LLM may misidentify something as novel or miss truly
  novel content. The author should confirm the claims are accurate.
- **Focus**: Items listed in `novel_info` represent the skill's highest-value
  content. If these details are wrong or missing, the novelty score is
  unreliable.
- **Trimming guidance**: Content NOT mentioned in `novel_info` is likely
  restating common knowledge and is a candidate for compression or removal.

If novelty is low, advise the author to:

1. Identify which sections contain information that is NOT available in public
   documentation or model training data (proprietary APIs, internal conventions,
   non-obvious gotchas, organization-specific workflows).
2. Cut or compress sections that merely restate common knowledge.
3. Focus the skill on the genuinely novel content.

For more context on why novelty matters and how to think about skill quality,
refer the author to: https://agentskillreport.com/

### Important scoring caveat

Novelty scores reflect what the scoring model knows from its training data.
Different models may produce different novelty scores based on their training
data coverage. If using a smaller or older model, novelty scores may be less
reliable. Cross-model comparison (when available) helps mitigate this by
providing a second perspective.

## Full Review Summary

When LLM scoring was performed, present the review summary with:

1. **Structural validation result**: Pass/fail, with any errors or warnings.
2. **LLM score summary**: Overall score and per-dimension breakdown for SKILL.md
   and references (if any). If cross-model comparison was performed, show the
   comparison table from `score report --compare` and note any significant
   divergences between model families.
3. **Areas to address**: Specific dimensions that need improvement, with concrete
   suggestions.
4. **Novelty assessment**: Whether the skill provides sufficient novel value,
   with specific guidance if it doesn't. Include the `novel_info` details for
   each file so the author can verify accuracy and identify what to keep or cut.
   If cross-model comparison was performed, note whether both models agree on
   the novelty assessment or diverge.
5. **Recommendation**: Whether the skill is ready to publish, needs minor
   revisions, or needs significant rework.
