# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] — Unreleased

First stable release. Includes the complete CLI and importable library packages.

### CLI

- `validate structure` — spec compliance, frontmatter, token counts, code fence
  integrity, internal link validation, orphan file detection, keyword stuffing
- `validate links` — external HTTP/HTTPS link validation with template URL support
- `analyze content` — content quality metrics (density, specificity, imperative ratio)
- `analyze contamination` — cross-language contamination detection and scoring
- `check` — run all deterministic checks with `--only`/`--skip` filtering
- `score evaluate` — LLM-as-judge scoring (Anthropic and OpenAI-compatible providers)
- `score report` — view and compare cached LLM scores across models
- Output formats: text, JSON, markdown
- GitHub Actions annotations via `--emit-annotations`
- `--strict` mode for CI (treats warnings as errors)
- Multi-skill directory detection
- Pre-commit hook support for all major agent platforms
- Homebrew install via `agent-ecosystem/tap`

### Library

- `orchestrate` — high-level validation coordination
- `evaluate` — LLM scoring orchestration with caching and progress reporting
- `judge` — LLM client abstraction and scoring (EXPERIMENTAL)
- `structure`, `content`, `contamination`, `links` — individual analysis packages
- `skill` — SKILL.md parsing (frontmatter + body)
- `skillcheck` — skill detection and reference file analysis
- `report` — output formatting (text, JSON, markdown, GitHub annotations)
- `types` — shared data types (`Report`, `Result`, `Level`, etc.)
- `judge.LLMClient` interface for custom LLM providers
