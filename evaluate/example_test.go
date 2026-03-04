package evaluate_test

import (
	"context"
	"fmt"
	"os"

	"github.com/agent-ecosystem/skill-validator/evaluate"
	"github.com/agent-ecosystem/skill-validator/judge"
)

// This example demonstrates scoring a skill directory with caching and
// progress reporting. It requires a valid API key, so it is not executed
// as a test.
func ExampleEvaluateSkill() {
	client, err := judge.NewClient(judge.ClientOptions{
		Provider: "anthropic",
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	result, err := evaluate.EvaluateSkill(context.Background(), "./my-skill", client, evaluate.Options{
		MaxLen: judge.DefaultMaxContentLen,
		Progress: func(event, detail string) {
			fmt.Printf("[%s] %s\n", event, detail)
		},
	})
	if err != nil {
		panic(err)
	}

	// SKILL.md scores
	if result.SkillScores != nil {
		fmt.Printf("Overall: %.2f/5\n", result.SkillScores.Overall)
		fmt.Printf("Assessment: %s\n", result.SkillScores.BriefAssessment)
	}

	// Reference file scores
	for _, ref := range result.RefResults {
		fmt.Printf("%s: %.2f/5\n", ref.File, ref.Scores.Overall)
	}

	// Aggregated reference scores
	if result.RefAggregate != nil {
		fmt.Printf("References average: %.2f/5\n", result.RefAggregate.Overall)
	}
}

// This example shows how to score only reference files, skipping SKILL.md.
func ExampleEvaluateSkill_refsOnly() {
	client, err := judge.NewClient(judge.ClientOptions{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "gpt-4o",
	})
	if err != nil {
		panic(err)
	}

	result, err := evaluate.EvaluateSkill(context.Background(), "./my-skill", client, evaluate.Options{
		RefsOnly: true,
		MaxLen:   judge.DefaultMaxContentLen,
	})
	if err != nil {
		panic(err)
	}

	for _, ref := range result.RefResults {
		fmt.Printf("%s: %.2f/5\n", ref.File, ref.Scores.Overall)
	}
}
