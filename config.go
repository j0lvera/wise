package wise

import (
	"io"

	"github.com/rs/zerolog"
)

// DefaultSystemPrompt is the default system prompt for the agent.
const DefaultSystemPrompt = `You are an autonomous agent that executes bash commands to complete tasks.

RULES:
1. You can ONLY execute bash commands by wrapping them in a markdown code block with the 'bash' language tag
2. Execute ONE command at a time and wait for the output
3. Use the command output to inform your next action
4. When the task is complete, output "TASK_COMPLETE" followed by a summary on the next line

Example command format:
` + "```bash" + `
ls -la
` + "```" + `

Example completion:
` + "```bash" + `
echo "TASK_COMPLETE"
echo "Summary: Created hello.txt with the requested content"
` + "```"

// Config holds the agent configuration (optional settings only).
type Config struct {
	parser        Parser
	logger        *zerolog.Logger
	output        io.Writer
	maxSteps      int
	systemPrompt  string
	actionHandler ActionHandler
}

// NewConfig creates a new Config with sensible defaults.
func NewConfig() Config {
	return Config{
		maxSteps:     25,
		systemPrompt: DefaultSystemPrompt,
		output:       io.Discard,
	}
}

// WithParser sets a custom response parser.
func (c Config) WithParser(p Parser) Config {
	c.parser = p
	return c
}

// WithLogger sets the logger.
func (c Config) WithLogger(l *zerolog.Logger) Config {
	c.logger = l
	return c
}

// WithOutput sets the output writer for streaming results.
func (c Config) WithOutput(w io.Writer) Config {
	c.output = w
	return c
}

// WithMaxSteps sets the maximum number of agent steps.
func (c Config) WithMaxSteps(n int) Config {
	c.maxSteps = n
	return c
}

// WithSystemPrompt sets the system prompt.
func (c Config) WithSystemPrompt(p string) Config {
	c.systemPrompt = p
	return c
}

// WithActionHandler sets a custom action handler for extensibility.
func (c Config) WithActionHandler(h ActionHandler) Config {
	c.actionHandler = h
	return c
}
