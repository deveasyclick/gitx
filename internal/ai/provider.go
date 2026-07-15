package ai

import "context"

// Provider generates text using an AI model.
type Provider interface {
	// Name returns the provider identifier (e.g. "openai", "deepseek").
	Name() string

	// Generate sends a prompt and returns the generated text.
	Generate(ctx context.Context, req Request) (Response, error)
}
