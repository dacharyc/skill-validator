// Package skillvalidator validates Agent Skill directories and scores their
// quality using LLM judges.
//
// This module provides both a CLI tool and importable library packages.
// Library consumers typically start with one of two entry points depending
// on their use case.
//
// # Validation (structure, content, contamination, links)
//
// The [github.com/agent-ecosystem/skill-validator/orchestrate] package coordinates
// all validation checks and returns a unified [github.com/agent-ecosystem/skill-validator/types.Report].
// Use [github.com/agent-ecosystem/skill-validator/orchestrate.AllGroups] to enable every check,
// or selectively enable only the groups you need.
//
// For single-purpose analysis, orchestrate also provides focused functions:
// [github.com/agent-ecosystem/skill-validator/orchestrate.RunContentAnalysis],
// [github.com/agent-ecosystem/skill-validator/orchestrate.RunContaminationAnalysis], and
// [github.com/agent-ecosystem/skill-validator/orchestrate.RunLinkChecks].
//
// # LLM Scoring
//
// The [github.com/agent-ecosystem/skill-validator/evaluate] package handles scoring
// orchestration with caching and progress reporting. It uses the
// [github.com/agent-ecosystem/skill-validator/judge] package for LLM API calls.
//
// The judge package includes built-in clients for Anthropic and OpenAI-compatible
// APIs. For other providers (AWS Bedrock, Azure OpenAI, local models), implement
// the [github.com/agent-ecosystem/skill-validator/judge.LLMClient] interface.
// See the "Custom LLM providers" section in the README for details.
//
// Note: the judge package is EXPERIMENTAL. Its API may change in minor releases.
// See the project README for the full stability policy.
//
// # Lower-Level Packages
//
// For fine-grained control, use the individual packages directly:
//
//   - [github.com/agent-ecosystem/skill-validator/structure] — directory layout, frontmatter, tokens, internal links
//   - [github.com/agent-ecosystem/skill-validator/content] — content quality metrics (density, specificity, imperative ratio)
//   - [github.com/agent-ecosystem/skill-validator/contamination] — cross-language contamination detection
//   - [github.com/agent-ecosystem/skill-validator/links] — external HTTP/HTTPS link validation
//   - [github.com/agent-ecosystem/skill-validator/skill] — SKILL.md parsing (frontmatter + body)
//   - [github.com/agent-ecosystem/skill-validator/skillcheck] — skill detection and reference file analysis
//   - [github.com/agent-ecosystem/skill-validator/report] — output formatting (text, JSON, markdown, GitHub annotations)
//   - [github.com/agent-ecosystem/skill-validator/types] — shared data types (Report, Result, Level, etc.)
package skillvalidator
