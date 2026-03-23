# Installing skill-validator

## Installation methods

### Homebrew (recommended for macOS)

```bash
brew tap agent-ecosystem/homebrew-tap
brew install skill-validator
```

### From source (requires Go 1.25.5+)

```bash
go install github.com/agent-ecosystem/skill-validator/cmd/skill-validator@latest
```

Ensure `$GOPATH/bin` (usually `~/go/bin`) is on your PATH:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### From a pre-built binary

```bash
cp /path/to/skill-validator /usr/local/bin/ && chmod +x /usr/local/bin/skill-validator
```

## Verify installation

```bash
skill-validator --version
```

## Prerequisites for LLM scoring

LLM scoring requires one of:

- **Anthropic API key** — set `ANTHROPIC_API_KEY` environment variable
- **OpenAI API key** — set `OPENAI_API_KEY` environment variable
- **OpenAI-compatible endpoint** — set `OPENAI_API_KEY` and provide a `--base-url`
- **Claude CLI** — use `--provider claude-cli` (no API key needed; uses the locally authenticated `claude` binary)
