package services

import (
	"context"

	"github.com/user/gitx/internal/ai"
)

// aiProvider is the interface the services layer needs from an AI provider.
// Uses the real ai types to avoid adapter boilerplate.
type aiProvider interface {
	Name() string
	Generate(ctx context.Context, req ai.Request) (ai.Response, error)
}
