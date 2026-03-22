---
name: allowed-dirs-skill
description: A skill with development directories that require --allow-dirs.
license: MIT
compatibility: Works with all major LLM providers.
metadata:
  author: test
  version: "1.0"
allowed-tools: Bash, Read, Write
---
# Allowed Dirs Skill

This skill demonstrates using non-standard directories alongside the
standard skill structure. The `evals/` and `testing/` directories are
development artifacts that require `--allow-dirs` to suppress warnings.

## Usage

Follow the instructions in the [reference guide](references/guide.md)
to get started.

Run scripts/validate.sh to check your work.

## Notes

The evals/ directory contains evaluation test cases and the testing/
directory contains integration test fixtures. Neither is part of the
standard skill structure, but both are useful during development.
