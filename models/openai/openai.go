package openai

import (
	"context"
	"fmt"

	"github.com/j0lvera/wise/models"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Config holds the model configuration.
type Config struct {
	apiKey  string
	baseURL string
}

// NewConfig creates a new Config with defaults.
func NewConfig() Config {
	return Config{}
}

// WithAPIKey sets the API key.
func (c Config) WithAPIKey(key string) Config {
	c.apiKey = key
	return c
}

// WithBaseURL sets the base URL for the API.
func (c Config) WithBaseURL(url string) Config {
	c.baseURL = url
	return c
}

// model implements the Model interface (unexported).
type model struct {
	cfg    Config
	name   string
	client llms.Model
}

// New creates a new OpenAI-compatible model.
func New(modelName string, cfg Config) (models.Model, error) {
	if cfg.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	clientOpts := []openai.Option{
		openai.WithToken(cfg.apiKey),
		openai.WithModel(modelName),
	}
	if cfg.baseURL != "" {
		clientOpts = append(clientOpts, openai.WithBaseURL(cfg.baseURL))
	}

	client, err := openai.New(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &model{cfg: cfg, name: modelName, client: client}, nil
}

// Query sends messages to the LLM and returns the response.
func (m *model) Query(ctx context.Context, messages []models.Message) (string, error) {
	llmMessages := make([]llms.MessageContent, 0, len(messages))

	for _, msg := range messages {
		var msgType llms.ChatMessageType
		switch msg.Role {
		case "system":
			msgType = llms.ChatMessageTypeSystem
		case "user":
			msgType = llms.ChatMessageTypeHuman
		case "assistant":
			msgType = llms.ChatMessageTypeAI
		default:
			continue
		}
		llmMessages = append(llmMessages, llms.TextParts(msgType, msg.Content))
	}

	resp, err := m.client.GenerateContent(ctx, llmMessages)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from model")
	}

	return resp.Choices[0].Content, nil
}
