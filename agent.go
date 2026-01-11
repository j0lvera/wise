package wise

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/j0lvera/wise/environments"
	"github.com/j0lvera/wise/environments/local"
	"github.com/j0lvera/wise/models"

	"github.com/rs/zerolog"
)

// baseAgent implements the Agent interface (unexported).
type baseAgent struct {
	model    models.Model
	env      environments.Environment
	cfg      Config
	messages []Message
	step     int
}

// New creates an agent with required dependencies and optional config.
// Model and Environment are required, Config uses defaults if zero value.
func New(model models.Model, env environments.Environment, cfg Config) (Agent, error) {
	// Apply defaults for zero values
	if cfg.maxSteps == 0 {
		cfg.maxSteps = 25
	}
	if cfg.systemPrompt == "" {
		cfg.systemPrompt = DefaultSystemPrompt
	}
	if cfg.output == nil {
		cfg.output = io.Discard
	}
	if cfg.parser == nil {
		cfg.parser = NewBashParser()
	}
	if cfg.logger == nil {
		l := zerolog.Nop()
		cfg.logger = &l
	}

	return &baseAgent{
		model:    model,
		env:      env,
		cfg:      cfg,
		messages: []Message{},
	}, nil
}

// Run executes the agent loop with the given task.
func (a *baseAgent) Run(ctx context.Context, task string) (string, error) {
	// Initialize conversation
	a.messages = []Message{}
	a.addMessage(RoleSystem, a.cfg.systemPrompt)
	a.addMessage(RoleUser, task)

	a.cfg.logger.Info().
		Int("max_steps", a.cfg.maxSteps).
		Msg("agent loop starting")

	var lastResponse string

	// Main loop
	for a.step = 0; a.step < a.cfg.maxSteps; a.step++ {
		a.cfg.logger.Info().
			Int("step", a.step+1).
			Msg("step starting")

		response, err := a.Step(ctx)
		if err != nil {
			var termErr *TerminatingErr
			var procErr *ProcessErr

			if errors.As(err, &termErr) {
				// Clean exit - task complete or limit reached
				a.cfg.logger.Info().
					Str("reason", string(termErr.Reason)).
					Msg("agent terminated")
				return termErr.Output, nil
			}

			if errors.As(err, &procErr) {
				// Recoverable - add feedback and continue
				a.cfg.logger.Warn().
					Str("type", string(procErr.Type)).
					Str("message", procErr.Message).
					Msg("process error, continuing")
				a.addMessage(RoleUser, procErr.Message)
				continue
			}

			// Check for execution errors from the environment
			var execErr *local.ExecutionError
			if errors.As(err, &execErr) {
				a.cfg.logger.Warn().
					Str("type", string(execErr.Type)).
					Str("message", execErr.Message).
					Msg("execution error, continuing")
				a.addMessage(RoleUser, execErr.Message)
				continue
			}

			// Unrecoverable error
			a.cfg.logger.Error().Err(err).Msg("unrecoverable error")
			return "", err
		}
		lastResponse = response
	}

	// Step limit reached
	a.cfg.logger.Warn().
		Int("max_steps", a.cfg.maxSteps).
		Msg("step limit reached")
	return lastResponse, &TerminatingErr{Reason: ReasonStepLimit}
}

// Step performs a single iteration of the agent loop.
func (a *baseAgent) Step(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled: %w", err)
	}

	a.cfg.logger.Debug().Msg("querying model")

	// 1. Query the model
	response, err := a.model.Query(ctx, a.messages)
	if err != nil {
		a.cfg.logger.Error().Err(err).Msg("query failed")
		return "", fmt.Errorf("query failed: %w", err)
	}

	a.cfg.logger.Debug().
		Int("response_length", len(response)).
		Msg("got response")
	a.cfg.logger.Trace().
		Str("response", response).
		Msg("full response")

	// 2. Parse action from response
	action, err := a.cfg.parser.ParseAction(response)
	if err != nil {
		// Format error - will be added as feedback
		a.cfg.logger.Debug().Err(err).Msg("failed to parse action")
		return "", err
	}

	// 3. Add assistant message before execution
	a.addMessage(RoleAssistant, response)

	// 4. Execute the action and stream output
	fmt.Fprintf(a.cfg.output, "$ %s\n", action.Command)

	a.cfg.logger.Info().
		Str("command", action.Command).
		Msg("executing command")

	// Try custom action handler first
	if a.cfg.actionHandler != nil {
		output, handled, err := a.cfg.actionHandler(ctx, action)
		if handled {
			if err != nil {
				return "", err
			}
			return a.handleOutput(output)
		}
	}

	// Default execution via environment
	output, err := a.env.Execute(ctx, action)
	if err != nil {
		a.cfg.logger.Warn().Err(err).Msg("command execution failed")
		return "", err
	}

	return a.handleOutput(output)
}

// handleOutput processes command output and checks for completion.
func (a *baseAgent) handleOutput(output Output) (string, error) {
	// Print output (skip if it's just the completion marker)
	if !a.isTaskComplete(output) && strings.TrimSpace(output.Stdout) != "" {
		fmt.Fprintln(a.cfg.output, output.Stdout)
	}

	a.cfg.logger.Debug().
		Int("output_length", len(output.String())).
		Int("exit_code", output.ExitCode).
		Msg("command completed")
	a.cfg.logger.Trace().
		Str("output", output.String()).
		Msg("full output")

	// Check for completion signal in command output
	if a.isTaskComplete(output) {
		a.cfg.logger.Info().Msg("task complete signal in output")
		return a.extractFinalOutput(output), &TerminatingErr{
			Reason: ReasonComplete,
			Output: a.extractFinalOutput(output),
		}
	}

	// Add execution result as user message
	feedback := a.formatObservation(output)
	a.addMessage(RoleUser, feedback)

	return "", nil
}

const completionMarker = "TASK_COMPLETE"

// isTaskComplete checks if the command output starts with the completion signal.
func (a *baseAgent) isTaskComplete(output Output) bool {
	firstLine := strings.SplitN(strings.TrimSpace(output.Stdout), "\n", 2)[0]
	return strings.TrimSpace(firstLine) == completionMarker
}

// extractFinalOutput returns everything after the completion marker.
func (a *baseAgent) extractFinalOutput(output Output) string {
	parts := strings.SplitN(output.Stdout, "\n", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// formatObservation formats command output for the LLM.
func (a *baseAgent) formatObservation(output Output) string {
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
func (a *baseAgent) addMessage(role string, content string) {
	a.messages = append(a.messages, Message{
		Role:    role,
		Content: content,
	})
	a.cfg.logger.Debug().
		Str("role", role).
		Int("content_length", len(content)).
		Msg("message added")
}

// Messages returns the current conversation history (for debugging/testing).
func (a *baseAgent) Messages() []Message {
	return a.messages
}
