package wise

import (
	"context"

	"github.com/j0lvera/wise/environments"
	"github.com/j0lvera/wise/models"
)

// Re-export types for convenience.
type (
	Message = models.Message
	Action  = environments.Action
	Output  = environments.Output
)

// Role constants.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Agent defines the contract for an LLM-powered agent.
type Agent interface {
	Run(ctx context.Context, task string) (string, error)
	Step(ctx context.Context) (string, error)
}

// Parser extracts actions from LLM responses.
type Parser interface {
	ParseAction(response string) (Action, error)
}

// ActionHandler processes custom action types.
// Returns (output, handled, error) - if handled is false, default processing is used.
type ActionHandler func(ctx context.Context, action Action) (Output, bool, error)
