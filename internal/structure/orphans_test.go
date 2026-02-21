package structure

import (
	"testing"

	"github.com/dacharyc/skill-validator/internal/validator"
)

func TestCheckOrphanFiles(t *testing.T) {
	t.Run("all files referenced from SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/guide.md", "guide content")
		writeFile(t, dir, "scripts/setup.sh", "#!/bin/bash")
		writeFile(t, dir, "assets/logo.png", "fake image")

		body := "See references/guide.md and scripts/setup.sh and assets/logo.png"
		results := CheckOrphanFiles(dir, body)

		requireResult(t, results, validator.Pass, "all files in scripts/ are referenced")
		requireResult(t, results, validator.Pass, "all files in references/ are referenced")
		requireResult(t, results, validator.Pass, "all files in assets/ are referenced")
		requireNoLevel(t, results, validator.Warning)
	})

	t.Run("orphan in references", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/guide.md", "guide content")
		writeFile(t, dir, "references/unused.md", "unused content")

		body := "See references/guide.md for details."
		results := CheckOrphanFiles(dir, body)

		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: references/unused.md")
	})

	t.Run("orphan in scripts", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/setup.sh", "#!/bin/bash")

		body := "No references to scripts here."
		results := CheckOrphanFiles(dir, body)

		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: scripts/setup.sh")
	})

	t.Run("empty directories produce no results", func(t *testing.T) {
		dir := t.TempDir()
		// No files at all
		results := CheckOrphanFiles(dir, "some body")
		if len(results) != 0 {
			t.Errorf("expected 0 results for empty dirs, got %d", len(results))
		}
	})

	t.Run("no recognized directories", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "other/file.md", "content")

		results := CheckOrphanFiles(dir, "some body")
		if len(results) != 0 {
			t.Errorf("expected 0 results for unrecognized dirs, got %d", len(results))
		}
	})

	t.Run("binary file referenced but not scanned", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "assets/logo.png", "fake binary with references/secret.md inside")
		writeFile(t, dir, "references/secret.md", "secret content")

		body := "See assets/logo.png for the logo."
		results := CheckOrphanFiles(dir, body)

		// logo.png is reached (referenced from body) but not scanned for further refs
		// so references/secret.md should be an orphan
		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: references/secret.md")
		requireNoResultContaining(t, results, validator.Warning, "assets/logo.png")
	})

	t.Run("directory-relative reference from referenced file", func(t *testing.T) {
		dir := t.TempDir()
		// references/guide.md references images/diagram.png using a path
		// relative to its own directory, not the skill root.
		writeFile(t, dir, "references/guide.md", "See ![diagram](images/diagram.png) for details.")
		writeFile(t, dir, "references/images/diagram.png", "fake image")

		body := "Read the [guide](references/guide.md)."
		results := CheckOrphanFiles(dir, body)

		// The image should be reached (indirectly via guide.md), not flagged as orphan
		requireNoResultContaining(t, results, validator.Warning, "references/images/diagram.png")
		requireResult(t, results, validator.Pass, "all files in references/ are referenced")
	})

	t.Run("root-level file bridges SKILL.md to scripts", func(t *testing.T) {
		dir := t.TempDir()
		// SKILL.md mentions FORMS.md (root-level), which mentions the script
		writeFile(t, dir, "FORMS.md", "Run scripts/fill_form.py to fill the form.")
		writeFile(t, dir, "scripts/fill_form.py", "#!/usr/bin/env python3")

		body := "For form filling, read FORMS.md and follow its instructions."
		results := CheckOrphanFiles(dir, body)

		requireNoResultContaining(t, results, validator.Warning, "scripts/fill_form.py")
		requireResult(t, results, validator.Pass, "all files in scripts/ are referenced")
	})

	t.Run("package.json bridges SKILL.md to scripts when referenced", func(t *testing.T) {
		dir := t.TempDir()
		// SKILL.md mentions package.json, which maps to the script
		writeFile(t, dir, "package.json", `{"scripts":{"validate":"node scripts/validate.js"}}`)
		writeFile(t, dir, "scripts/validate.js", "// validator")

		body := "See package.json for available commands. Run `npm run validate` to check."
		results := CheckOrphanFiles(dir, body)

		// package.json is mentioned so it gets scanned, finding scripts/validate.js
		requireNoResultContaining(t, results, validator.Warning, "scripts/validate.js")
	})

	t.Run("package.json not scanned when SKILL.md only mentions npm commands", func(t *testing.T) {
		dir := t.TempDir()
		// SKILL.md says "npm run validate" but doesn't mention package.json
		writeFile(t, dir, "package.json", `{"scripts":{"validate":"node scripts/validate.js"}}`)
		writeFile(t, dir, "scripts/validate.js", "// validator")

		body := "Run `npm run validate` to check your component."
		results := CheckOrphanFiles(dir, body)

		// package.json is not mentioned, so scripts/validate.js stays orphaned
		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: scripts/validate.js")
	})

	t.Run("root file matched case-insensitively", func(t *testing.T) {
		dir := t.TempDir()
		// SKILL.md says "FORMS.md" but the file on disk is "forms.md"
		writeFile(t, dir, "forms.md", "Run scripts/fill_form.py to fill the form.")
		writeFile(t, dir, "scripts/fill_form.py", "#!/usr/bin/env python3")

		body := "For form filling, read FORMS.md and follow its instructions."
		results := CheckOrphanFiles(dir, body)

		requireNoResultContaining(t, results, validator.Warning, "scripts/fill_form.py")
		requireResult(t, results, validator.Pass, "all files in scripts/ are referenced")
	})

	t.Run("script referenced without extension gets specific warning", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/check_fields.py", "#!/usr/bin/env python3")

		body := "Run `python scripts/check_fields <file.pdf>` to check."
		results := CheckOrphanFiles(dir, body)

		requireResultContaining(t, results, validator.Warning,
			"file scripts/check_fields.py is referenced without its extension (as scripts/check_fields in SKILL.md) — include the .py extension so agents can reliably locate the file")
		// Should NOT also emit the generic orphan warning
		requireNoResultContaining(t, results, validator.Warning, "potentially unreferenced file: scripts/check_fields.py")
	})

	t.Run("extensionless match via intermediary file", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "forms.md", "Run `python scripts/check_fields <file>`.")
		writeFile(t, dir, "scripts/check_fields.py", "#!/usr/bin/env python3")

		body := "For form filling, read forms.md."
		results := CheckOrphanFiles(dir, body)

		requireResultContaining(t, results, validator.Warning,
			"file scripts/check_fields.py is referenced without its extension (as scripts/check_fields in forms.md)")
	})

	t.Run("__init__.py excluded from checks entirely", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/__init__.py", "")
		writeFile(t, dir, "scripts/run.py", "#!/usr/bin/env python3")

		body := "Run scripts/run.py to start."
		results := CheckOrphanFiles(dir, body)

		requireNoResultContaining(t, results, validator.Warning, "__init__.py")
		requireNoResultContaining(t, results, validator.Info, "__init__.py")
		requireResult(t, results, validator.Pass, "all files in scripts/ are referenced")
	})

	t.Run("__init__.py not flagged even when directory is orphaned", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/__init__.py", "")
		writeFile(t, dir, "scripts/run.py", "#!/usr/bin/env python3")

		body := "No references here."
		results := CheckOrphanFiles(dir, body)

		requireNoResultContaining(t, results, validator.Warning, "__init__.py")
		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: scripts/run.py")
	})

	t.Run("nested __init__.py excluded from checks", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/pkg/__init__.py", "")
		writeFile(t, dir, "scripts/pkg/helpers.py", "# helpers")

		body := "No references here."
		results := CheckOrphanFiles(dir, body)

		requireNoResultContaining(t, results, validator.Warning, "__init__.py")
		requireResultContaining(t, results, validator.Warning, "scripts/pkg/helpers.py")
	})

	t.Run("full extension match takes priority over extensionless", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/setup.sh", "#!/bin/bash")

		// Body references with full extension — should get normal treatment, not the extension warning
		body := "Run scripts/setup.sh to configure."
		results := CheckOrphanFiles(dir, body)

		requireResult(t, results, validator.Pass, "all files in scripts/ are referenced")
		requireNoResultContaining(t, results, validator.Warning, "referenced without its extension")
	})

	t.Run("unreferenced root file does not get scanned", func(t *testing.T) {
		dir := t.TempDir()
		// notes.md exists at root but SKILL.md doesn't mention it
		writeFile(t, dir, "notes.md", "Run scripts/secret.sh for setup.")
		writeFile(t, dir, "scripts/secret.sh", "#!/bin/bash")

		body := "This skill has no special setup."
		results := CheckOrphanFiles(dir, body)

		// notes.md is never mentioned, so it shouldn't be scanned, and the script stays orphaned
		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: scripts/secret.sh")
	})

	t.Run("Python import resolves sibling module", func(t *testing.T) {
		dir := t.TempDir()
		// SKILL.md references main.py, which imports helpers
		writeFile(t, dir, "scripts/main.py", "from helpers import merge\nmerge()")
		writeFile(t, dir, "scripts/helpers.py", "def merge(): pass")

		body := "Run scripts/main.py to start."
		results := CheckOrphanFiles(dir, body)

		requireNoResultContaining(t, results, validator.Warning, "scripts/helpers.py")
		requireResult(t, results, validator.Pass, "all files in scripts/ are referenced")
	})

	t.Run("Python import resolves dotted module path", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/main.py", "from helpers.merge_runs import merge\nmerge()")
		writeFile(t, dir, "scripts/helpers/__init__.py", "")
		writeFile(t, dir, "scripts/helpers/merge_runs.py", "def merge(): pass")

		body := "Run scripts/main.py to start."
		results := CheckOrphanFiles(dir, body)

		requireNoResultContaining(t, results, validator.Warning, "scripts/helpers/merge_runs.py")
		requireResult(t, results, validator.Pass, "all files in scripts/ are referenced")
	})

	t.Run("Python relative import resolves", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/pkg/__init__.py", "")
		writeFile(t, dir, "scripts/pkg/main.py", "from .utils import helper\nhelper()")
		writeFile(t, dir, "scripts/pkg/utils.py", "def helper(): pass")

		body := "Run scripts/pkg/main.py to start."
		results := CheckOrphanFiles(dir, body)

		requireNoResultContaining(t, results, validator.Warning, "scripts/pkg/utils.py")
		requireResult(t, results, validator.Pass, "all files in scripts/ are referenced")
	})

	t.Run("Python import does not match non-Python files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "scripts/main.py", "import data_loader\ndata_loader.run()")
		writeFile(t, dir, "scripts/data_loader.sh", "#!/bin/bash")

		body := "Run scripts/main.py to start."
		results := CheckOrphanFiles(dir, body)

		// .sh file should not be resolved by Python imports; it's matched
		// via the extensionless fallback since "data_loader" appears in the text
		requireResultContaining(t, results, validator.Warning,
			"file scripts/data_loader.sh is referenced without its extension")
	})

	t.Run("__init__.py bridges package imports to sibling modules", func(t *testing.T) {
		dir := t.TempDir()
		// pack.py imports from the validators package, which is a directory
		// with __init__.py that re-exports from sibling modules.
		writeFile(t, dir, "scripts/pack.py", "from validators import BaseValidator\nBaseValidator()")
		writeFile(t, dir, "scripts/validators/__init__.py", "from .base import BaseValidator")
		writeFile(t, dir, "scripts/validators/base.py", "class BaseValidator: pass")
		writeFile(t, dir, "scripts/validators/extra.py", "class ExtraValidator: pass")

		body := "Run scripts/pack.py to package."
		results := CheckOrphanFiles(dir, body)

		// base.py should be reached via: pack.py → __init__.py → .base
		requireNoResultContaining(t, results, validator.Warning, "scripts/validators/base.py")
		// extra.py is not imported by __init__.py, so it stays orphaned
		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: scripts/validators/extra.py")
	})

	t.Run("multiple orphans across directories", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "references/unused1.md", "content")
		writeFile(t, dir, "scripts/unused2.sh", "content")
		writeFile(t, dir, "assets/unused3.png", "content")

		body := "No references to any files."
		results := CheckOrphanFiles(dir, body)

		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: references/unused1.md")
		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: scripts/unused2.sh")
		requireResultContaining(t, results, validator.Warning, "potentially unreferenced file: assets/unused3.png")
	})
}
