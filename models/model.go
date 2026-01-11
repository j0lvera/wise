package models

import "context"

// Message represents a chat message.
type Message struct {
	Role    string
	Content string
}

// Model sends messages to an LLM and returns responses.
type Model interface {
	Query(ctx context.Context, messages []Message) (string, error)
}
