package environments

import "context"

// Action represents a command to execute.
type Action struct {
	Type    string
	Command string
}

// Output represents command execution results.
type Output struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
}

// String returns a combined string of stdout and stderr.
func (o Output) String() string {
	if o.Stderr != "" {
		return o.Stdout + "\n" + o.Stderr
	}
	return o.Stdout
}

// Environment executes actions and returns results.
type Environment interface {
	Execute(ctx context.Context, action Action) (Output, error)
}

// CommandValidator checks if a command is safe to execute.
type CommandValidator interface {
	Validate(command string) error
}
