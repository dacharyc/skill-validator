package cmd

import (
	"fmt"
	"io"

	"github.com/dacharyc/skill-validator/internal/judge"
)

func printEvalResultMarkdown(w io.Writer, result *skillEvalResult) {
	_, _ = fmt.Fprintf(w, "## Scoring skill: %s\n", result.SkillDir)

	if result.SkillScores != nil {
		_, _ = fmt.Fprintf(w, "\n### SKILL.md Scores\n\n")
		_, _ = fmt.Fprintf(w, "| Dimension | Score |\n")
		_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
		_, _ = fmt.Fprintf(w, "| Clarity | %d/5 |\n", result.SkillScores.Clarity)
		_, _ = fmt.Fprintf(w, "| Actionability | %d/5 |\n", result.SkillScores.Actionability)
		_, _ = fmt.Fprintf(w, "| Token Efficiency | %d/5 |\n", result.SkillScores.TokenEfficiency)
		_, _ = fmt.Fprintf(w, "| Scope Discipline | %d/5 |\n", result.SkillScores.ScopeDiscipline)
		_, _ = fmt.Fprintf(w, "| Directive Precision | %d/5 |\n", result.SkillScores.DirectivePrecision)
		_, _ = fmt.Fprintf(w, "| Novelty | %d/5 |\n", result.SkillScores.Novelty)
		_, _ = fmt.Fprintf(w, "| **Overall** | **%.2f/5** |\n", result.SkillScores.Overall)

		if result.SkillScores.BriefAssessment != "" {
			_, _ = fmt.Fprintf(w, "\n> %s\n", result.SkillScores.BriefAssessment)
		}

		if result.SkillScores.NovelInfo != "" {
			_, _ = fmt.Fprintf(w, "\n*Novel details: %s*\n", result.SkillScores.NovelInfo)
		}
	}

	if evalDisplay == "files" && len(result.RefResults) > 0 {
		for _, ref := range result.RefResults {
			printRefScoresMarkdown(w, ref.File, ref.Scores)
		}
	}

	if result.RefAggregate != nil {
		_, _ = fmt.Fprintf(w, "\n### Reference Scores (%d file%s)\n\n", len(result.RefResults), pluralS(len(result.RefResults)))
		_, _ = fmt.Fprintf(w, "| Dimension | Score |\n")
		_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
		_, _ = fmt.Fprintf(w, "| Clarity | %d/5 |\n", result.RefAggregate.Clarity)
		_, _ = fmt.Fprintf(w, "| Instructional Value | %d/5 |\n", result.RefAggregate.InstructionalValue)
		_, _ = fmt.Fprintf(w, "| Token Efficiency | %d/5 |\n", result.RefAggregate.TokenEfficiency)
		_, _ = fmt.Fprintf(w, "| Novelty | %d/5 |\n", result.RefAggregate.Novelty)
		_, _ = fmt.Fprintf(w, "| Skill Relevance | %d/5 |\n", result.RefAggregate.SkillRelevance)
		_, _ = fmt.Fprintf(w, "| **Overall** | **%.2f/5** |\n", result.RefAggregate.Overall)
	}
}

func printMultiEvalResultsMarkdown(w io.Writer, results []skillEvalResult) {
	for i, r := range results {
		if i > 0 {
			_, _ = fmt.Fprintf(w, "\n---\n\n")
		}
		printEvalResultMarkdown(w, &r)
	}
}

func printRefScoresMarkdown(w io.Writer, file string, scores *judge.RefScores) {
	_, _ = fmt.Fprintf(w, "\n### Reference: %s\n\n", file)
	_, _ = fmt.Fprintf(w, "| Dimension | Score |\n")
	_, _ = fmt.Fprintf(w, "| --- | ---: |\n")
	_, _ = fmt.Fprintf(w, "| Clarity | %d/5 |\n", scores.Clarity)
	_, _ = fmt.Fprintf(w, "| Instructional Value | %d/5 |\n", scores.InstructionalValue)
	_, _ = fmt.Fprintf(w, "| Token Efficiency | %d/5 |\n", scores.TokenEfficiency)
	_, _ = fmt.Fprintf(w, "| Novelty | %d/5 |\n", scores.Novelty)
	_, _ = fmt.Fprintf(w, "| Skill Relevance | %d/5 |\n", scores.SkillRelevance)
	_, _ = fmt.Fprintf(w, "| **Overall** | **%.2f/5** |\n", scores.Overall)

	if scores.BriefAssessment != "" {
		_, _ = fmt.Fprintf(w, "\n> %s\n", scores.BriefAssessment)
	}

	if scores.NovelInfo != "" {
		_, _ = fmt.Fprintf(w, "\n*Novel details: %s*\n", scores.NovelInfo)
	}
}
