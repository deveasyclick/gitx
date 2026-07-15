// Package prompts provides template builders that construct
// system and user prompts for AI generation.
//
// Each builder implements Builder[T] for a specific input type,
// ensuring consistent prompt formatting across commands.
package prompts

// Builder constructs system and user prompts from structured input.
type Builder[T any] interface {
	// Build returns the system prompt and user prompt.
	Build(input T) (systemPrompt, userPrompt string, err error)
}
