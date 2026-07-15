package ai

// Request is sent to an AI provider for generation.
type Request struct {
	Model        string
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
}

// Response from an AI provider.
type Response struct {
	Text  string
	Usage TokenUsage
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}
