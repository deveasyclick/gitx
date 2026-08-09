package prompts

import (
	"fmt"
	"strings"
)

// GroupPromptInput contains the information needed to partition a diff
// into logical groups.
type GroupPromptInput struct {
	Diff   string // full unified diff text
	Branch string
}

// GroupBuilder generates prompts that ask the AI to partition changed files
// into logical groups (e.g. one group per feature, refactor, or fix).
type GroupBuilder struct{}

// NewGroupBuilder creates a new group prompt builder.
func NewGroupBuilder() *GroupBuilder {
	return &GroupBuilder{}
}

// Build returns system and user prompts for logical change grouping.
func (b *GroupBuilder) Build(input GroupPromptInput) (string, string, error) {
	if strings.TrimSpace(input.Diff) == "" {
		return "", "", fmt.Errorf("empty diff: nothing to group")
	}

	system := `You are an expert software engineer partitioning a set of file changes into logical groups for separate commits.

Rules:
- Group files that belong together logically (same feature, refactor, bug fix, or concern), regardless of directory.
- Each changed file must appear in exactly one group. Do not invent or omit files.
- Keep groups small and coherent — aim for 1-3 related commits for a typical change set.
- Respond ONLY with valid JSON in exactly this shape (no markdown, no commentary):
{"groups":[{"name":"short logical group description","files":["path/to/file.go"]}]}
- The "name" must be a concise description of what the group does (e.g. "add payment refund flow", "fix auth token expiry").`

	user := fmt.Sprintf(`Partition the changed files below into logical groups for separate commits.

Branch: %s

Diff:
%s`,
		input.Branch,
		strings.TrimSpace(input.Diff),
	)

	return system, user, nil
}
