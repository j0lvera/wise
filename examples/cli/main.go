package main

import (
	"fmt"
	"os"
	"time"

	"github.com/j0lvera/wise"
	"github.com/j0lvera/wise/environments/local"
	"github.com/j0lvera/wise/models/openai"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "myagent",
		Short: "An LLM-powered command execution agent",
	}

	runCmd := &cobra.Command{
		Use:   "run [task]",
		Short: "Run the agent with a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task := args[0]

			// Build model config
			modelCfg := openai.NewConfig().
				WithAPIKey(os.Getenv("API_KEY")).
				WithBaseURL(os.Getenv("BASE_URL"))

			modelName := os.Getenv("MODEL")
			if modelName == "" {
				modelName = "anthropic/claude-sonnet-4-5-20250929"
			}

			model, err := openai.New(modelName, modelCfg)
			if err != nil {
				return fmt.Errorf("failed to create model: %w", err)
			}

			// Build environment config
			workingDir, _ := cmd.Flags().GetString("working-dir")
			timeout, _ := cmd.Flags().GetDuration("timeout")

			envCfg := local.NewConfig().
				WithWorkingDir(workingDir).
				WithTimeout(timeout)

			env := local.New(envCfg)

			// Build agent config
			maxSteps, _ := cmd.Flags().GetInt("max-steps")
			cfg := wise.NewConfig().
				WithOutput(os.Stdout).
				WithMaxSteps(maxSteps)

			a, err := wise.New(model, env, cfg)
			if err != nil {
				return fmt.Errorf("failed to create agent: %w", err)
			}

			_, err = a.Run(cmd.Context(), task)
			return err
		},
	}

	runCmd.Flags().String("working-dir", ".", "Working directory for commands")
	runCmd.Flags().Duration("timeout", 30*time.Second, "Command timeout")
	runCmd.Flags().Int("max-steps", 25, "Maximum number of agent steps")

	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
