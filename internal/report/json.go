package report

import (
	"encoding/json"
	"io"

	"github.com/dacharyc/skill-validator/internal/validator"
)

type jsonReport struct {
	SkillDir         string           `json:"skill_dir"`
	Passed           bool             `json:"passed"`
	Errors           int              `json:"errors"`
	Warnings         int              `json:"warnings"`
	Results          []jsonResult     `json:"results"`
	TokenCounts      *jsonTokenCounts `json:"token_counts,omitempty"`
	OtherTokenCounts *jsonTokenCounts `json:"other_token_counts,omitempty"`
}

type jsonResult struct {
	Level    string `json:"level"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

type jsonTokenCounts struct {
	Files []jsonTokenCount `json:"files"`
	Total int              `json:"total"`
}

type jsonTokenCount struct {
	File   string `json:"file"`
	Tokens int    `json:"tokens"`
}

type jsonMultiReport struct {
	Passed   bool         `json:"passed"`
	Errors   int          `json:"errors"`
	Warnings int          `json:"warnings"`
	Skills   []jsonReport `json:"skills"`
}

func buildJSONReport(r *validator.Report) jsonReport {
	out := jsonReport{
		SkillDir: r.SkillDir,
		Passed:   r.Errors == 0,
		Errors:   r.Errors,
		Warnings: r.Warnings,
		Results:  make([]jsonResult, len(r.Results)),
	}

	for i, res := range r.Results {
		out.Results[i] = jsonResult{
			Level:    res.Level.String(),
			Category: res.Category,
			Message:  res.Message,
		}
	}

	if len(r.TokenCounts) > 0 {
		tc := &jsonTokenCounts{
			Files: make([]jsonTokenCount, len(r.TokenCounts)),
		}
		for i, c := range r.TokenCounts {
			tc.Files[i] = jsonTokenCount{File: c.File, Tokens: c.Tokens}
			tc.Total += c.Tokens
		}
		out.TokenCounts = tc
	}

	if len(r.OtherTokenCounts) > 0 {
		otc := &jsonTokenCounts{
			Files: make([]jsonTokenCount, len(r.OtherTokenCounts)),
		}
		for i, c := range r.OtherTokenCounts {
			otc.Files[i] = jsonTokenCount{File: c.File, Tokens: c.Tokens}
			otc.Total += c.Tokens
		}
		out.OtherTokenCounts = otc
	}

	return out
}

// PrintJSON writes the report as JSON to the given writer.
func PrintJSON(w io.Writer, r *validator.Report) error {
	out := buildJSONReport(r)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// PrintMultiJSON writes the multi-skill report as JSON to the given writer.
func PrintMultiJSON(w io.Writer, mr *validator.MultiReport) error {
	out := jsonMultiReport{
		Passed:   mr.Errors == 0,
		Errors:   mr.Errors,
		Warnings: mr.Warnings,
		Skills:   make([]jsonReport, len(mr.Skills)),
	}
	for i, r := range mr.Skills {
		out.Skills[i] = buildJSONReport(r)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
