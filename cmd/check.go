package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dacharyc/skill-validator/internal/contamination"
	"github.com/dacharyc/skill-validator/internal/content"
	"github.com/dacharyc/skill-validator/internal/quality"
	"github.com/dacharyc/skill-validator/internal/validator"
)

var (
	checkOnly string
	checkSkip string
)

var checkCmd = &cobra.Command{
	Use:   "check <path>",
	Short: "Run all checks (validate + analyze quality + content + contamination)",
	Long:  "Runs all validation and analysis checks. Use --only or --skip to select specific check groups.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCheck,
}

func init() {
	checkCmd.Flags().StringVar(&checkOnly, "only", "", "comma-separated list of check groups to run: validate,quality,content,contamination")
	checkCmd.Flags().StringVar(&checkSkip, "skip", "", "comma-separated list of check groups to skip: validate,quality,content,contamination")
	rootCmd.AddCommand(checkCmd)
}

var validGroups = map[string]bool{
	"validate":      true,
	"quality":       true,
	"content":       true,
	"contamination": true,
}

func runCheck(cmd *cobra.Command, args []string) error {
	if checkOnly != "" && checkSkip != "" {
		return fmt.Errorf("--only and --skip are mutually exclusive")
	}

	enabled, err := resolveCheckGroups(checkOnly, checkSkip)
	if err != nil {
		return err
	}

	_, mode, dirs, err := detectAndResolve(args)
	if err != nil {
		return err
	}

	switch mode {
	case validator.SingleSkill:
		r := runAllChecks(dirs[0], enabled)
		return outputReport(r)
	case validator.MultiSkill:
		mr := &validator.MultiReport{}
		for _, dir := range dirs {
			r := runAllChecks(dir, enabled)
			mr.Skills = append(mr.Skills, r)
			mr.Errors += r.Errors
			mr.Warnings += r.Warnings
		}
		return outputMultiReport(mr)
	}
	return nil
}

func resolveCheckGroups(only, skip string) (map[string]bool, error) {
	enabled := map[string]bool{
		"validate":      true,
		"quality":       true,
		"content":       true,
		"contamination": true,
	}

	if only != "" {
		// Reset all to false, enable only specified
		for k := range enabled {
			enabled[k] = false
		}
		for _, g := range strings.Split(only, ",") {
			g = strings.TrimSpace(g)
			if !validGroups[g] {
				return nil, fmt.Errorf("unknown check group %q (valid: validate, quality, content, contamination)", g)
			}
			enabled[g] = true
		}
	}

	if skip != "" {
		for _, g := range strings.Split(skip, ",") {
			g = strings.TrimSpace(g)
			if !validGroups[g] {
				return nil, fmt.Errorf("unknown check group %q (valid: validate, quality, content, contamination)", g)
			}
			enabled[g] = false
		}
	}

	return enabled, nil
}

func runAllChecks(dir string, enabled map[string]bool) *validator.Report {
	rpt := &validator.Report{SkillDir: dir}

	// Validate (spec compliance)
	if enabled["validate"] {
		vr := validator.Validate(dir)
		rpt.Results = append(rpt.Results, vr.Results...)
		rpt.TokenCounts = vr.TokenCounts
		rpt.OtherTokenCounts = vr.OtherTokenCounts
	}

	// Load skill for quality/content/contamination checks
	needsSkill := enabled["quality"] || enabled["content"] || enabled["contamination"]
	var rawContent, body string
	var skillLoaded bool
	if needsSkill {
		s, err := validator.LoadSkill(dir)
		if err != nil {
			if !enabled["validate"] {
				// Only add the error if validate didn't already catch it
				rpt.Results = append(rpt.Results, validator.Result{
					Level: validator.Error, Category: "Skill", Message: err.Error(),
				})
			}
			// Fall back to reading raw SKILL.md for content/contamination analysis
			rawContent = validator.ReadSkillRaw(dir)
		} else {
			rawContent = s.RawContent
			body = s.Body
			skillLoaded = true
		}

		// Quality checks require a fully parsed skill (links, code fences)
		if skillLoaded && enabled["quality"] {
			rpt.Results = append(rpt.Results, quality.CheckLinks(dir, body)...)
			rpt.Results = append(rpt.Results, quality.CheckMarkdown(dir, body)...)
		}

		// Content analysis works on raw content (no frontmatter parsing needed)
		if enabled["content"] && rawContent != "" {
			cr := content.Analyze(rawContent)
			rpt.ContentReport = cr
		}

		// Contamination analysis works on raw content
		if enabled["contamination"] && rawContent != "" {
			var codeLanguages []string
			if rpt.ContentReport != nil {
				codeLanguages = rpt.ContentReport.CodeLanguages
			} else {
				cr := content.Analyze(rawContent)
				codeLanguages = cr.CodeLanguages
			}
			skillName := filepath.Base(dir)
			rpt.ContaminationReport = contamination.Analyze(skillName, rawContent, codeLanguages)
		}
	}

	// Tally errors and warnings
	rpt.Errors = 0
	rpt.Warnings = 0
	for _, r := range rpt.Results {
		switch r.Level {
		case validator.Error:
			rpt.Errors++
		case validator.Warning:
			rpt.Warnings++
		}
	}

	return rpt
}
