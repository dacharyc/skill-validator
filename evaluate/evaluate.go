// Package evaluate provides LLM-as-judge scoring orchestration for skills.
//
// It exposes the evaluation logic (caching, scoring, aggregation) as a library
// so that both the CLI and enterprise variants can reuse it.
package evaluate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dacharyc/skill-validator/judge"
	"github.com/dacharyc/skill-validator/skill"
	"github.com/dacharyc/skill-validator/skillcheck"
)

// ProgressFunc receives progress events during evaluation.
// event identifies the kind of event (e.g. "scoring", "cached", "warning", "error").
// detail provides human-readable context.
type ProgressFunc func(event string, detail string)

// Result holds the complete scoring output for one skill.
type Result struct {
	SkillDir     string
	SkillScores  *judge.SkillScores
	RefResults   []RefResult
	RefAggregate *judge.RefScores
}

// RefResult holds scoring output for a single reference file.
type RefResult struct {
	File   string
	Scores *judge.RefScores
}

// Options controls what gets scored.
type Options struct {
	Rescore   bool
	SkillOnly bool
	RefsOnly  bool
	MaxLen    int
	CacheDir  string       // Override cache directory; defaults to judge.CacheDir(skillDir) when empty
	Progress  ProgressFunc // Optional progress callback; nil means no output
}

// progress calls the progress callback if set.
func progress(opts Options, event, detail string) {
	if opts.Progress != nil {
		opts.Progress(event, detail)
	}
}

// resolveCacheDir returns the configured cache directory, falling back to the
// default .score_cache location inside skillDir.
func resolveCacheDir(opts Options, skillDir string) string {
	if opts.CacheDir != "" {
		return opts.CacheDir
	}
	return judge.CacheDir(skillDir)
}

// EvaluateSkill scores a skill directory (SKILL.md and/or reference files).
func EvaluateSkill(ctx context.Context, dir string, client judge.LLMClient, opts Options) (*Result, error) {
	result := &Result{SkillDir: dir}
	cacheDir := resolveCacheDir(opts, dir)
	skillName := filepath.Base(dir)

	// Load skill
	s, err := skill.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("loading skill: %w", err)
	}

	// Score SKILL.md
	if !opts.RefsOnly {
		progress(opts, "scoring", fmt.Sprintf("%s/SKILL.md", skillName))

		cacheKey := judge.CacheKey(client.Provider(), client.ModelName(), "skill", skillName, "SKILL.md")

		if !opts.Rescore {
			if cached, ok := judge.GetCached(cacheDir, cacheKey); ok {
				var scores judge.SkillScores
				if err := json.Unmarshal(cached.Scores, &scores); err == nil {
					result.SkillScores = &scores
					progress(opts, "cached", fmt.Sprintf("%s/SKILL.md", skillName))
				}
			}
		}

		if result.SkillScores == nil {
			scores, err := judge.ScoreSkill(ctx, s.RawContent, client, opts.MaxLen)
			if err != nil {
				return nil, fmt.Errorf("scoring SKILL.md: %w", err)
			}
			result.SkillScores = scores

			// Save to cache
			scoresJSON, _ := json.Marshal(scores)
			cacheResult := &judge.CachedResult{
				Provider:    client.Provider(),
				Model:       client.ModelName(),
				File:        "SKILL.md",
				Type:        "skill",
				ContentHash: judge.ContentHash(s.RawContent),
				ScoredAt:    time.Now().UTC(),
				Scores:      scoresJSON,
			}
			if err := judge.SaveCache(cacheDir, cacheKey, cacheResult); err != nil {
				progress(opts, "warning", fmt.Sprintf("could not save cache: %v", err))
			}
		}
	}

	// Score reference files
	if !opts.SkillOnly {
		refFiles := skillcheck.ReadReferencesMarkdownFiles(dir)
		if refFiles != nil {
			skillDesc := s.Frontmatter.Description

			// Sort for deterministic ordering
			names := make([]string, 0, len(refFiles))
			for name := range refFiles {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				content := refFiles[name]
				progress(opts, "scoring", fmt.Sprintf("%s/references/%s", skillName, name))

				cacheKey := judge.CacheKey(client.Provider(), client.ModelName(), "ref:"+name, skillName, name)
				var refScores *judge.RefScores

				if !opts.Rescore {
					if cached, ok := judge.GetCached(cacheDir, cacheKey); ok {
						var scores judge.RefScores
						if err := json.Unmarshal(cached.Scores, &scores); err == nil {
							refScores = &scores
							progress(opts, "cached", fmt.Sprintf("%s/references/%s", skillName, name))
						}
					}
				}

				if refScores == nil {
					scores, err := judge.ScoreReference(ctx, content, s.Frontmatter.Name, skillDesc, client, opts.MaxLen)
					if err != nil {
						progress(opts, "error", fmt.Sprintf("scoring %s: %v", name, err))
						continue
					}
					refScores = scores

					scoresJSON, _ := json.Marshal(scores)
					cacheResult := &judge.CachedResult{
						Provider:    client.Provider(),
						Model:       client.ModelName(),
						File:        name,
						Type:        "ref:" + name,
						ContentHash: judge.ContentHash(content),
						ScoredAt:    time.Now().UTC(),
						Scores:      scoresJSON,
					}
					if err := judge.SaveCache(cacheDir, cacheKey, cacheResult); err != nil {
						progress(opts, "warning", fmt.Sprintf("could not save cache: %v", err))
					}
				}

				result.RefResults = append(result.RefResults, RefResult{File: name, Scores: refScores})
			}

			// Aggregate
			if len(result.RefResults) > 0 {
				var allScores []*judge.RefScores
				for _, r := range result.RefResults {
					allScores = append(allScores, r.Scores)
				}
				result.RefAggregate = judge.AggregateRefScores(allScores)
			}
		}
	}

	return result, nil
}

// EvaluateSingleFile scores a single reference .md file.
func EvaluateSingleFile(ctx context.Context, absPath string, client judge.LLMClient, opts Options) (*Result, error) {
	if !strings.HasSuffix(strings.ToLower(absPath), ".md") {
		return nil, fmt.Errorf("single-file scoring only supports .md files: %s", absPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Walk up to find parent skill directory
	skillDir, err := FindParentSkillDir(absPath)
	if err != nil {
		return nil, err
	}

	// Load parent skill for context
	s, err := skill.Load(skillDir)
	if err != nil {
		return nil, fmt.Errorf("loading parent skill: %w", err)
	}

	fileName := filepath.Base(absPath)
	skillName := s.Frontmatter.Name
	if skillName == "" {
		skillName = filepath.Base(skillDir)
	}

	progress(opts, "scoring", fmt.Sprintf("%s (parent: %s)", fileName, skillName))

	cacheDir := resolveCacheDir(opts, skillDir)
	cacheKey := judge.CacheKey(client.Provider(), client.ModelName(), "ref:"+fileName, skillName, fileName)

	if !opts.Rescore {
		if cached, ok := judge.GetCached(cacheDir, cacheKey); ok {
			var scores judge.RefScores
			if err := json.Unmarshal(cached.Scores, &scores); err == nil {
				progress(opts, "cached", fileName)
				result := &Result{
					SkillDir:   skillDir,
					RefResults: []RefResult{{File: fileName, Scores: &scores}},
				}
				return result, nil
			}
		}
	}

	scores, err := judge.ScoreReference(ctx, string(content), skillName, s.Frontmatter.Description, client, opts.MaxLen)
	if err != nil {
		return nil, fmt.Errorf("scoring %s: %w", fileName, err)
	}

	// Save to cache
	scoresJSON, _ := json.Marshal(scores)
	cacheResult := &judge.CachedResult{
		Provider:    client.Provider(),
		Model:       client.ModelName(),
		File:        fileName,
		Type:        "ref:" + fileName,
		ContentHash: judge.ContentHash(string(content)),
		ScoredAt:    time.Now().UTC(),
		Scores:      scoresJSON,
	}
	if err := judge.SaveCache(cacheDir, cacheKey, cacheResult); err != nil {
		progress(opts, "warning", fmt.Sprintf("could not save cache: %v", err))
	}

	result := &Result{
		SkillDir:   skillDir,
		RefResults: []RefResult{{File: fileName, Scores: scores}},
	}
	return result, nil
}

// FindParentSkillDir walks up from filePath looking for a directory containing SKILL.md.
func FindParentSkillDir(filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	// Check up to 3 levels
	for range 3 {
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("could not find parent SKILL.md for %s (checked up to 3 directories)", filePath)
}
