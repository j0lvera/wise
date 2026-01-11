package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/j0lvera/wise"
	"github.com/j0lvera/wise/environments/local"
	"github.com/j0lvera/wise/models/openai"
)

func main() {
	// Build model
	modelCfg := openai.NewConfig().
		WithAPIKey(os.Getenv("API_KEY")).
		WithBaseURL("https://openrouter.ai/api/v1")

	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "anthropic/claude-3.5-sonnet"
	}

	model, err := openai.New(modelName, modelCfg)
	if err != nil {
		fmt.Printf("Failed to create model: %v\n", err)
		os.Exit(1)
	}

	// Build environment
	env := local.New(local.NewConfig())

	// Custom action handler for Python functions
	pythonHandler := func(ctx context.Context, action wise.Action) (wise.Output, bool, error) {
		if !strings.HasPrefix(action.Command, "python_function") {
			return wise.Output{}, false, nil // Not handled, use default
		}

		// Parse and execute custom Python function
		result := executePythonFunction(action.Command)
		return wise.Output{Stdout: result}, true, nil
	}

	// Build agent config with custom handler
	cfg := wise.NewConfig().
		WithOutput(os.Stdout).
		WithActionHandler(pythonHandler)

	a, err := wise.New(model, env, cfg)
	if err != nil {
		fmt.Printf("Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	result, err := a.Run(context.Background(), "analyze the data")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Result:", result)
}

func executePythonFunction(cmd string) string {
	// Custom implementation - this is just a placeholder
	// In a real implementation, you would parse the command and execute Python code
	return "python function result"
}
