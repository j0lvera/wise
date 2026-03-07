package main

import (
	"context"
	"fmt"
	"os"

	"github.com/j0lvera/wise"
	"github.com/j0lvera/wise/executor/local"
	"github.com/j0lvera/wise/models/openai"
)

func main() {
	// Create model — falls back to OPENAI_API_KEY and OPENAI_BASE_URL env vars
	modelCfg := openai.NewConfig()

	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "anthropic/claude-sonnet-4-5-20250929"
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
