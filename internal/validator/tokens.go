package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tiktoken-go/tokenizer"
)

type TokenCount struct {
	File   string
	Tokens int
}

const (
	// Per-file thresholds for reference files
	refFileSoftLimit = 10_000
	refFileHardLimit = 25_000

	// Aggregate thresholds across all reference files
	refTotalSoftLimit = 25_000
	refTotalHardLimit = 50_000
)

func checkTokens(dir string, body string) ([]Result, []TokenCount) {
	var results []Result
	var counts []TokenCount

	enc, err := tokenizer.Get(tokenizer.O200kBase)
	if err != nil {
		results = append(results, Result{Level: Error, Category: "Tokens", Message: fmt.Sprintf("failed to initialize tokenizer: %v", err)})
		return results, counts
	}

	// Count SKILL.md body tokens
	bodyTokens, _, _ := enc.Encode(body)
	bodyCount := len(bodyTokens)
	counts = append(counts, TokenCount{File: "SKILL.md body", Tokens: bodyCount})

	// Warn if body exceeds 5000 tokens
	if bodyCount > 5000 {
		results = append(results, Result{Level: Warning, Category: "Tokens", Message: fmt.Sprintf("SKILL.md body is %d tokens (spec recommends < 5000)", bodyCount)})
	}

	// Warn if SKILL.md exceeds 500 lines
	lineCount := strings.Count(body, "\n") + 1
	if lineCount > 500 {
		results = append(results, Result{Level: Warning, Category: "Tokens", Message: fmt.Sprintf("SKILL.md body is %d lines (spec recommends < 500)", lineCount)})
	}

	// Count tokens for files in references/
	refTotal := 0
	refsDir := filepath.Join(dir, "references")
	if entries, err := os.ReadDir(refsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			path := filepath.Join(refsDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				results = append(results, Result{Level: Warning, Category: "Tokens", Message: fmt.Sprintf("could not read %s: %v", filepath.Join("references", entry.Name()), err)})
				continue
			}
			tokens, _, _ := enc.Encode(string(data))
			fileTokens := len(tokens)
			relPath := filepath.Join("references", entry.Name())
			counts = append(counts, TokenCount{
				File:   relPath,
				Tokens: fileTokens,
			})
			refTotal += fileTokens

			// Per-file limits
			if fileTokens > refFileHardLimit {
				results = append(results, Result{
					Level:    Error,
					Category: "Tokens",
					Message: fmt.Sprintf(
						"%s is %d tokens — this will consume 12-20%% of a typical context window "+
							"and meaningfully degrade agent performance; split into smaller focused files",
						relPath, fileTokens,
					),
				})
			} else if fileTokens > refFileSoftLimit {
				results = append(results, Result{
					Level:    Warning,
					Category: "Tokens",
					Message: fmt.Sprintf(
						"%s is %d tokens — consider splitting into smaller focused files "+
							"so agents load only what they need",
						relPath, fileTokens,
					),
				})
			}
		}
	}

	// Aggregate reference limits
	if refTotal > refTotalHardLimit {
		results = append(results, Result{
			Level:    Error,
			Category: "Tokens",
			Message: fmt.Sprintf(
				"total reference files: %d tokens — this will consume 25-40%% of a typical "+
					"context window; reduce content or split into a skill with fewer references",
				refTotal,
			),
		})
	} else if refTotal > refTotalSoftLimit {
		results = append(results, Result{
			Level:    Warning,
			Category: "Tokens",
			Message: fmt.Sprintf(
				"total reference files: %d tokens — agents may load multiple references "+
					"in one session, consider whether all this content is essential",
				refTotal,
			),
		})
	}

	return results, counts
}
