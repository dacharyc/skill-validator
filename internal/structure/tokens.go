package structure

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dacharyc/skill-validator/internal/validator"
	"github.com/tiktoken-go/tokenizer"
)

const (
	// Per-file thresholds for reference files
	refFileSoftLimit = 10_000
	refFileHardLimit = 25_000

	// Aggregate thresholds across all reference files
	refTotalSoftLimit = 25_000
	refTotalHardLimit = 50_000

	// Aggregate thresholds for non-standard files
	otherTotalSoftLimit = 25_000
	otherTotalHardLimit = 100_000
)

func CheckTokens(dir, body string) ([]validator.Result, []validator.TokenCount, []validator.TokenCount) {
	ctx := validator.ResultContext{Category: "Tokens"}
	var results []validator.Result
	var counts []validator.TokenCount

	enc, err := tokenizer.Get(tokenizer.O200kBase)
	if err != nil {
		results = append(results, ctx.Errorf("failed to initialize tokenizer: %v", err))
		return results, counts, nil
	}

	// Count SKILL.md body tokens
	bodyTokens, _, _ := enc.Encode(body)
	bodyCount := len(bodyTokens)
	counts = append(counts, validator.TokenCount{File: "SKILL.md body", Tokens: bodyCount})

	// Warn if body exceeds 5000 tokens
	if bodyCount > 5000 {
		results = append(results, ctx.WarnFilef("SKILL.md", "SKILL.md body is %d tokens (spec recommends < 5000)", bodyCount))
	}

	// Warn if SKILL.md exceeds 500 lines
	lineCount := strings.Count(body, "\n") + 1
	if lineCount > 500 {
		results = append(results, ctx.WarnFilef("SKILL.md", "SKILL.md body is %d lines (spec recommends < 500)", lineCount))
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
				relPath := filepath.Join("references", entry.Name())
				results = append(results, ctx.WarnFilef(relPath, "could not read %s: %v", relPath, err))
				continue
			}
			tokens, _, _ := enc.Encode(string(data))
			fileTokens := len(tokens)
			relPath := filepath.Join("references", entry.Name())
			counts = append(counts, validator.TokenCount{
				File:   relPath,
				Tokens: fileTokens,
			})
			refTotal += fileTokens

			// Per-file limits
			if fileTokens > refFileHardLimit {
				results = append(results, ctx.ErrorFilef(relPath,
					"%s is %d tokens — this will consume 12-20%% of a typical context window "+
						"and meaningfully degrade agent performance; split into smaller focused files",
					relPath, fileTokens,
				))
			} else if fileTokens > refFileSoftLimit {
				results = append(results, ctx.WarnFilef(relPath,
					"%s is %d tokens — consider splitting into smaller focused files "+
						"so agents load only what they need",
					relPath, fileTokens,
				))
			}
		}
	}

	// Aggregate reference limits
	if refTotal > refTotalHardLimit {
		results = append(results, ctx.Errorf(
			"total reference files: %d tokens — this will consume 25-40%% of a typical "+
				"context window; reduce content or split into a skill with fewer references",
			refTotal,
		))
	} else if refTotal > refTotalSoftLimit {
		results = append(results, ctx.Warnf(
			"total reference files: %d tokens — agents may load multiple references "+
				"in one session, consider whether all this content is essential",
			refTotal,
		))
	}

	// Count tokens in non-standard files
	otherCounts := countOtherFiles(dir, enc)

	// Check other-files aggregate limits
	otherTotal := 0
	for _, c := range otherCounts {
		otherTotal += c.Tokens
	}
	if otherTotal > otherTotalHardLimit {
		results = append(results, ctx.Errorf(
			"non-standard files total %d tokens — if an agent loads these, "+
				"they will consume most of the context window and severely degrade performance; "+
				"move essential content into references/ or remove unnecessary files",
			otherTotal,
		))
	} else if otherTotal > otherTotalSoftLimit {
		results = append(results, ctx.Warnf(
			"non-standard files total %d tokens — if an agent loads these, "+
				"they could consume a significant portion of the context window; "+
				"consider moving essential content into references/ or removing unnecessary files",
			otherTotal,
		))
	}

	// Count tokens in text-based asset files
	assetCounts := countAssetFiles(dir, enc)
	counts = append(counts, assetCounts...)

	return results, counts, otherCounts
}

// binaryExtensions lists file extensions that should be skipped for token counting.
var binaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".ico": true, ".svg": true, ".webp": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".7z": true, ".rar": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
}

// standardRootFiles are files that are already counted in the main token table.
var standardRootFiles = map[string]bool{
	"skill.md": true,
}

// standardDirs are directories already handled by the standard structure.
var standardDirs = map[string]bool{
	"references": true,
	"scripts":    true,
	"assets":     true,
}

// textAssetExtensions lists file extensions in assets/ that are text-based
// and likely loaded into LLM context (templates, guides, configs).
var textAssetExtensions = map[string]bool{
	".md":       true,
	".tex":      true,
	".py":       true,
	".yaml":     true,
	".yml":      true,
	".tsx":      true,
	".ts":       true,
	".jsx":      true,
	".sty":      true,
	".mplstyle": true,
	".ipynb":    true,
}

func countAssetFiles(dir string, enc tokenizer.Codec) []validator.TokenCount {
	var counts []validator.TokenCount
	assetsDir := filepath.Join(dir, "assets")

	_ = filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && path != assetsDir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !textAssetExtensions[ext] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		tokens, _, _ := enc.Encode(string(data))
		counts = append(counts, validator.TokenCount{File: rel, Tokens: len(tokens)})
		return nil
	})

	return counts
}

func countOtherFiles(dir string, enc tokenizer.Codec) []validator.TokenCount {
	var counts []validator.TokenCount

	entries, err := os.ReadDir(dir)
	if err != nil {
		return counts
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			if standardDirs[strings.ToLower(name)] {
				continue
			}
			// Walk files in unknown directory
			counts = append(counts, countFilesInDir(dir, name, enc)...)
		} else {
			if standardRootFiles[strings.ToLower(name)] {
				continue
			}
			if binaryExtensions[strings.ToLower(filepath.Ext(name))] {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			tokens, _, _ := enc.Encode(string(data))
			counts = append(counts, validator.TokenCount{File: name, Tokens: len(tokens)})
		}
	}

	return counts
}

func countFilesInDir(rootDir, dirName string, enc tokenizer.Codec) []validator.TokenCount {
	var counts []validator.TokenCount
	fullDir := filepath.Join(rootDir, dirName)

	_ = filepath.Walk(fullDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && path != fullDir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		if binaryExtensions[strings.ToLower(filepath.Ext(info.Name()))] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(rootDir, path)
		tokens, _, _ := enc.Encode(string(data))
		counts = append(counts, validator.TokenCount{File: rel, Tokens: len(tokens)})
		return nil
	})

	return counts
}
