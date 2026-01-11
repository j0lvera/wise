package main

import (
	"context"
	"fmt"
	"os"

	"github.com/j0lvera/wise"
	"github.com/j0lvera/wise/environments/local"
	"github.com/j0lvera/wise/models/openai"
)

func main() {
	// Create model with Builder pattern
	modelCfg := openai.NewConfig().
		WithAPIKey(os.Getenv("OPENROUTER_API_KEY")).
		WithBaseURL("https://openrouter.ai/api/v1")

	modelName := os.Getenv("OPENROUTER_MODEL")
	if modelName == "" {
		modelName = "anthropic/claude-3.5-sonnet" // default
	}

	model, err := openai.New(modelName, modelCfg)
	if err != nil {
		fmt.Printf("Failed to create model: %v\n", err)
		os.Exit(1)
	}

	// Create environment with defaults
	env := local.New(local.NewConfig())

	// Create agent with Builder pattern for config
	cfg := wise.NewConfig().WithOutput(os.Stdout)

	a, err := wise.New(model, env, cfg)
	if err != nil {
		fmt.Printf("Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	// Run task
	result, err := a.Run(context.Background(), "Create a file called hello.txt with 'Hello, World!' inside")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Result:", result)
}
