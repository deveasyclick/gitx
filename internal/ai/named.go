package ai

import "context"

// namedProvider wraps a provider with a custom name.
// This lets us reuse openAICompatibleClient for both OpenAI and DeepSeek
// while returning the correct Name() for each.
type namedProvider struct {
	name     string
	delegate Provider
}

func (p *namedProvider) Name() string {
	return p.name
}

func (p *namedProvider) Generate(ctx context.Context, req Request) (Response, error) {
	return p.delegate.Generate(ctx, req)
}
