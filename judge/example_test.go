package judge_test

import (
	"context"
	"fmt"
	"os"

	"github.com/agent-ecosystem/skill-validator/judge"
)

func ExampleNewClient() {
	client, err := judge.NewClient(judge.ClientOptions{
		Provider: "anthropic",
		APIKey:   "your-api-key",
		// Model defaults to claude-sonnet-4-5-20250929
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Provider: %s, Model: %s\n", client.Provider(), client.ModelName())
	// Output:
	// Provider: anthropic, Model: claude-sonnet-4-5-20250929
}

func ExampleNewClient_openai() {
	client, err := judge.NewClient(judge.ClientOptions{
		Provider: "openai",
		APIKey:   "your-api-key",
		Model:    "gpt-4o",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Provider: %s, Model: %s\n", client.Provider(), client.ModelName())
	// Output:
	// Provider: openai, Model: gpt-4o
}

// This example shows how to score a SKILL.md file. It requires a valid
// API key, so it is not executed as a test.
func ExampleScoreSkill() {
	client, err := judge.NewClient(judge.ClientOptions{
		Provider: "anthropic",
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	skillContent := "# My Skill\n\nInstructions for the agent..."

	scores, err := judge.ScoreSkill(context.Background(), skillContent, client, judge.DefaultMaxContentLen)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Overall: %.2f/5\n", scores.Overall)
	fmt.Printf("Assessment: %s\n", scores.BriefAssessment)
	for _, d := range scores.DimensionScores() {
		fmt.Printf("  %s: %d/5\n", d.Label, d.Value)
	}
}

// This example shows how to score a reference file against its parent skill.
func ExampleScoreReference() {
	client, err := judge.NewClient(judge.ClientOptions{
		Provider: "anthropic",
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	refContent := "# API Reference\n\nDetailed API documentation..."

	scores, err := judge.ScoreReference(
		context.Background(),
		refContent,
		"my-skill",                 // parent skill name
		"A skill for doing things", // parent skill description
		client,
		judge.DefaultMaxContentLen,
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Overall: %.2f/5\n", scores.Overall)
	for _, d := range scores.DimensionScores() {
		fmt.Printf("  %s: %d/5\n", d.Label, d.Value)
	}
}
