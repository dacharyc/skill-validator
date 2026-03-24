# Contributing

Thank you for your interest in contributing to skill-validator. This guide covers
how to set up your development environment, run checks, and submit changes.

## Getting started

### Prerequisites

- Go 1.25.5 or later
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

### Setup

1. Fork and clone the repository.
2. Install dependencies:

   ```bash
   go mod download
   ```

3. Verify everything works:

   ```bash
   go test -race ./... -count=1
   go build -o skill-validator ./cmd/skill-validator
   ```

## Development workflow

### Running tests

```bash
go test -race ./... -count=1
```

### Linting

The project uses golangci-lint with gofumpt formatting. Run it locally before
pushing:

```bash
golangci-lint run
```

CI runs both lint and test on every pull request. Your PR needs to pass both.

### Building

```bash
go build -o skill-validator ./cmd/skill-validator
```

## Making changes

### Bug fixes

If you've found a bug, check [existing issues](https://github.com/agent-ecosystem/skill-validator/issues)
first. If it hasn't been reported, open an issue describing the bug, then submit
a PR with the fix. Include a test that reproduces the bug where practical.

### New features

For anything beyond a small fix, open an issue first to discuss the approach.
This saves you from investing time in a direction that might not fit the project's
goals. Things worth discussing up front:

- New validation checks or scoring criteria
- New output formats or integrations
- Changes to existing check behavior
- New CLI commands or flags

### Code style

- Follow standard Go conventions. The linter enforces most of this.
- Use gofumpt for formatting (configured in `.golangci.yml`).
- Write tests for new functionality. The CI runs `go test -race`, so avoid
  data races.
- Keep commits focused. One logical change per commit is easier to review
  than a commit that mixes a bug fix with a refactor.

## Submitting a pull request

1. Create a branch from `main`.
2. Make your changes, ensuring tests pass and lint is clean.
3. Push your branch and open a PR against `main`.
4. Fill in the PR template. A clear description of what changed and why helps
   reviewers give useful feedback faster.

## Reporting issues

Use GitHub Issues. For bug reports, include:

- What you ran (command, flags, input)
- What you expected
- What happened instead
- Version info (`skill-validator --version`, Go version, OS)

## AI usage

We expect contributors to use AI tools. If you use AI to help write code, review
the output before submitting. Make sure tests pass, the code handles edge cases
you care about, and you understand what it does. The bar is the same whether you
wrote it by hand or not.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md).
By participating, you agree to uphold it.
