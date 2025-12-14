package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"
)

// Agent defines the contract for an LLM-powered command execution agent.
type Agent interface {
	// Run executes the agent loop until completion or error.
	// Returns the final LLM response on success.
	Run(ctx context.Context) (string, error)

	// Step performs a single iteration of the agent loop.
	Step(ctx context.Context) (string, error)
}

// BaseAgent implements the Agent interface with composable components.
type BaseAgent struct {
	config   *Config
	querier  Querier
	parser   Parser
	executor Executor
	logger   *zerolog.Logger
	output   io.Writer

	messages []Message
	step     int
}

// New creates a new agent with default components.
func New() (*BaseAgent, error) {
	config, err := LoadConfig(".")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewWithConfig(config)
}

// NewWithConfig creates a new agent with a pre-built config.
// This is useful when config is loaded externally (e.g., CLI with templating).
//
// Example:
//
//	cfg := &agent.Config{
//	    APIKey:       os.Getenv("OPENROUTER_API_KEY"),
//	    BaseURL:      "https://openrouter.ai/api/v1",
//	    Model:        "anthropic/claude-3.5-sonnet",
//	    SystemPrompt: "You are a helpful assistant...",
//	    UserPrompt:   "Create a file called hello.txt",
//	    MaxSteps:     25,
//	}
//	a, err := agent.NewWithConfig(cfg)
//	if err != nil {
//	    return err
//	}
//	return a.Run(ctx)
func NewWithConfig(config *Config) (*BaseAgent, error) {
	logger := NewLogger(config.LogLevel)

	querier, err := NewOpenAIQuerier(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create querier: %w", err)
	}

	// Default to discarding output if not set
	output := config.Output
	if output == nil {
		output = io.Discard
	}

	logger.Info().Msg("Agent initialized")

	return &BaseAgent{
		config:   config,
		querier:  querier,
		parser:   NewBashParser(),
		executor: NewBashExecutor(config.CommandTimeout, config.WorkingDir),
		logger:   &logger,
		output:   output,
		messages: []Message{},
	}, nil
}

// NewWithComponents creates a new agent with custom components (useful for testing).
func NewWithComponents(config *Config, querier Querier, parser Parser, executor Executor, logger *zerolog.Logger) *BaseAgent {
	if logger == nil {
		l := NewLogger(config.LogLevel)
		logger = &l
	}
	return &BaseAgent{
		config:   config,
		querier:  querier,
		parser:   parser,
		executor: executor,
		logger:   logger,
		messages: []Message{},
	}
}

// Run executes the agent loop until completion or error.
// Returns the final LLM response on success.
func (a *BaseAgent) Run(ctx context.Context) (string, error) {
	// Initialize conversation
	a.messages = []Message{}
	a.addMessage(RoleSystem, a.config.SystemPrompt)
	a.addMessage(RoleUser, a.config.UserPrompt)

	a.logger.Info().
		Int("max_steps", a.config.MaxSteps).
		Msg("Starting agent loop")

	var lastResponse string

	// Main loop
	for a.step = 0; a.step < a.config.MaxSteps; a.step++ {
		a.logger.Info().
			Int("step", a.step+1).
			Msg("Starting step")

		response, err := a.Step(ctx)
		if err != nil {
			var termErr *TerminatingErr
			var procErr *ProcessErr

			if errors.As(err, &termErr) {
				// Clean exit - task complete or limit reached
				a.logger.Info().
					Str("reason", string(termErr.Reason)).
					Msg("Agent terminated")
				return termErr.Output, nil
			}

			if errors.As(err, &procErr) {
				// Recoverable - add feedback and continue
				a.logger.Warn().
					Str("type", string(procErr.Type)).
					Str("message", procErr.Message).
					Msg("Process error, continuing")
				a.addMessage(RoleUser, procErr.Message)
				continue
			}

			// Unrecoverable error
			a.logger.Error().Err(err).Msg("Unrecoverable error")
			return "", err
		}
		lastResponse = response
	}

	// Step limit reached
	a.logger.Warn().
		Int("max_steps", a.config.MaxSteps).
		Msg("Step limit reached")
	return lastResponse, &TerminatingErr{Reason: ReasonStepLimit}
}

// Step performs a single iteration of the agent loop.
func (a *BaseAgent) Step(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled: %w", err)
	}

	a.logger.Debug().Msg("Querying model")

	// 1. Query the model
	response, err := a.querier.Query(ctx, a.messages)
	if err != nil {
		a.logger.Error().Err(err).Msg("Query failed")
		return "", fmt.Errorf("query failed: %w", err)
	}

	a.logger.Debug().
		Int("response_length", len(response)).
		Msg("Got response")
	a.logger.Trace().
		Str("response", response).
		Msg("Full response")

	// 2. Parse action from response
	action, err := a.parser.ParseAction(response)
	if err != nil {
		// Format error - will be added as feedback
		a.logger.Debug().Err(err).Msg("Failed to parse action")
		return "", err
	}

	// 3. Add assistant message before execution
	a.addMessage(RoleAssistant, response)

	// 4. Execute the action and stream output
	fmt.Fprintf(a.output, "$ %s\n", action.Command)

	a.logger.Info().
		Str("command", action.Command).
		Msg("Executing command")

	output, err := a.executor.Execute(ctx, action)
	if err != nil {
		// Timeout or execution error - will be added as feedback
		a.logger.Warn().Err(err).Msg("Command execution failed")
		return "", err
	}

	// Print output (skip if it's just the completion marker)
	if !a.isTaskComplete(output) && strings.TrimSpace(output.Stdout) != "" {
		fmt.Fprintln(a.output, output.Stdout)
	}

	a.logger.Debug().
		Int("output_length", len(output.String())).
		Int("exit_code", output.ExitCode).
		Msg("Command completed")
	a.logger.Trace().
		Str("output", output.String()).
		Msg("Full output")

	// 5. Check for completion signal in command output
	if a.isTaskComplete(output) {
		a.logger.Info().Msg("Task complete signal in output")
		// Return everything after the completion marker as final output
		return a.extractFinalOutput(output), &TerminatingErr{
			Reason: ReasonComplete,
			Output: a.extractFinalOutput(output),
		}
	}

	// 6. Add execution result as user message
	feedback := a.formatObservation(output)
	a.addMessage(RoleUser, feedback)

	return response, nil
}

const completionMarker = "TASK_COMPLETE"

// isTaskComplete checks if the command output starts with the completion signal.
func (a *BaseAgent) isTaskComplete(output Output) bool {
	firstLine := strings.SplitN(strings.TrimSpace(output.Stdout), "\n", 2)[0]
	return strings.TrimSpace(firstLine) == completionMarker
}

// extractFinalOutput returns everything after the completion marker.
func (a *BaseAgent) extractFinalOutput(output Output) string {
	parts := strings.SplitN(output.Stdout, "\n", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// formatObservation formats command output for the LLM.
func (a *BaseAgent) formatObservation(output Output) string {
	if strings.TrimSpace(output.Stdout) == "" && output.ExitCode == 0 {
		return "(no output)"
	}

	result := output.Stdout

	// Truncate long output
	const maxLen = 10000
	if len(result) > maxLen {
		head := result[:maxLen/2]
		tail := result[len(result)-maxLen/2:]
		result = head + "\n\n[... output truncated ...]\n\n" + tail
	}

	// Add exit code if non-zero
	if output.ExitCode != 0 {
		result = fmt.Sprintf("[exit code: %d]\n%s", output.ExitCode, result)
	}

	return result
}

// addMessage appends a message to the conversation history.
func (a *BaseAgent) addMessage(role Role, content string) {
	a.messages = append(a.messages, Message{
		Role:    role,
		Content: content,
	})
	a.logger.Debug().
		Str("role", string(role)).
		Int("content_length", len(content)).
		Msg("Message added")
}

// Messages returns the current conversation history (for debugging/testing).
func (a *BaseAgent) Messages() []Message {
	return a.messages
}
