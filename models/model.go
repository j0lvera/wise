package models

import "context"

// Message represents a chat message.
type Message struct {
	Role    string
	Content string
}

// TokenUsage holds token counts from a model query.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Model sends messages to an LLM and returns responses.
type Model interface {
	Query(ctx context.Context, messages []Message) (string, TokenUsage, error)
}
