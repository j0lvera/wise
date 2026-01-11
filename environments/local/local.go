package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/j0lvera/wise/environments"
)

// ActionType for bash commands.
const ActionTypeBash = "bash"

// Config holds the environment configuration.
type Config struct {
	timeout    time.Duration
	workingDir string
	validator  environments.CommandValidator
}

// NewConfig creates a new Config with sensible defaults.
func NewConfig() Config {
	return Config{
		timeout:   30 * time.Second,
		validator: NewDefaultValidator(),
	}
}

// WithTimeout sets the command timeout.
func (c Config) WithTimeout(d time.Duration) Config {
	c.timeout = d
	return c
}

// WithWorkingDir sets the working directory for commands.
func (c Config) WithWorkingDir(dir string) Config {
	c.workingDir = dir
	return c
}

// WithValidator sets a custom command validator.
func (c Config) WithValidator(v environments.CommandValidator) Config {
	c.validator = v
	return c
}

// WithoutValidation disables command validation (use with caution).
func (c Config) WithoutValidation() Config {
	c.validator = nil
	return c
}

// environment implements the Environment interface (unexported).
type environment struct {
	cfg Config
}

// New creates a new local environment.
func New(cfg Config) environments.Environment {
	// Apply defaults if zero values
	if cfg.timeout == 0 {
		cfg.timeout = 30 * time.Second
	}
	return &environment{cfg: cfg}
}

// Execute runs a bash command and returns the output.
func (e *environment) Execute(ctx context.Context, action environments.Action) (environments.Output, error) {
	if action.Type != ActionTypeBash {
		return environments.Output{}, fmt.Errorf("unsupported action type: %s", action.Type)
	}

	// Validate command before execution
	if e.cfg.validator != nil {
		if err := e.cfg.validator.Validate(action.Command); err != nil {
			return environments.Output{}, err
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, e.cfg.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", action.Command)

	if e.cfg.workingDir != "" {
		cmd.Dir = e.cfg.workingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := environments.Output{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		// Check if it was a timeout
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			output.TimedOut = true
			return output, &ExecutionError{
				Type:    ErrTimeout,
				Message: fmt.Sprintf("Command timed out after %s. Partial output:\n%s", e.cfg.timeout, output.String()),
			}
		}

		// Get exit code if available
		if exitErr, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitErr.ExitCode()
		}

		return output, &ExecutionError{
			Type:    ErrExecution,
			Message: fmt.Sprintf("Command failed: %s\nOutput:\n%s", err.Error(), output.String()),
		}
	}

	return output, nil
}

// ExecutionErrorType indicates the type of execution error.
type ExecutionErrorType string

const (
	ErrTimeout   ExecutionErrorType = "timeout"
	ErrExecution ExecutionErrorType = "execution"
	ErrBlocked   ExecutionErrorType = "blocked"
)

// ExecutionError represents an error during command execution.
type ExecutionError struct {
	Type    ExecutionErrorType
	Message string
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("execution error [%s]: %s", e.Type, e.Message)
}
