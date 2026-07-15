package prompts

import (
	"fmt"
	"strings"

	"github.com/user/gitx/internal/domain"
)

// CommitPromptInput contains the information needed to generate a commit message.
type CommitPromptInput struct {
	Diff   domain.Change
	Branch string
	Style  string // "conventional" or "gitmoji"
}

// CommitBuilder generates prompts for commit message generation.
type CommitBuilder struct{}

// NewCommitBuilder creates a new commit prompt builder.
func NewCommitBuilder() *CommitBuilder {
	return &CommitBuilder{}
}

// Build returns system and user prompts for commit message generation.
func (b *CommitBuilder) Build(input CommitPromptInput) (string, string, error) {
	if input.Diff.IsEmpty() {
		return "", "", fmt.Errorf("empty diff: nothing to generate a commit message from")
	}

	system := `You are an expert software engineer generating git commit messages.

Rules:
- Follow conventional commits format: type(scope): description
- Use imperative tense (e.g. "add", "fix", "refactor")
- Keep the title under 72 characters
- Always include a body when there are multiple changes to describe
- Do not invent changes that are not in the diff
- Group related changes together in the body as bullet points`

	if input.Style == "gitmoji" {
		system += `
- Prefix the title with an appropriate emoji (e.g. ✨ feat:, 🐛 fix:, ♻️ refactor:)`
	}

	user := fmt.Sprintf(`Generate a commit message for the diff below.

Branch: %s
Style: %s

Diff:
%s`,
		input.Branch,
		input.Style,
		strings.TrimSpace(input.Diff.Diff),
	)

	return system, user, nil
}
