package services

import (
	"context"
	"fmt"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

// ChangelogService orchestrates changelog entry generation.
type ChangelogService struct {
	git  git.Client
	ai   aiProvider
	pmpt prompts.Builder[prompts.ChangelogPromptInput]
}

// NewChangelogService creates a new changelog service.
func NewChangelogService(gitClient git.Client, aiProvider aiProvider, prompt prompts.Builder[prompts.ChangelogPromptInput]) *ChangelogService {
	return &ChangelogService{
		git:  gitClient,
		ai:   aiProvider,
		pmpt: prompt,
	}
}

// ChangelogResult contains the result of changelog generation.
type ChangelogResult struct {
	Entries      []domain.ChangelogEntry
	Provider     string
	InputTokens  int
	OutputTokens int
}

// GenerateLatest generates a changelog entry for the latest tag.
func (s *ChangelogService) GenerateLatest(ctx context.Context) (*ChangelogResult, error) {
	// Get tags and find the latest two
	tags, err := s.git.Tags(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading tags: %w", err)
	}

	if len(tags) == 0 {
		return nil, fmt.Errorf("no tags found: use --from to specify a range")
	}

	latest := tags[0]
	var from string
	if len(tags) >= 2 {
		from = tags[1]
	}

	return s.GenerateRange(ctx, from, latest)
}

// GenerateRange generates a changelog entry for a commit range.
func (s *ChangelogService) GenerateRange(ctx context.Context, from, to string) (*ChangelogResult, error) {
	// Get commits in range
	logs, err := s.git.Log(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("reading commit log: %w", err)
	}

	if len(logs) == 0 {
		return nil, fmt.Errorf("no commits found in range %s..%s", from, to)
	}

	system, userPrompt, err := s.pmpt.Build(prompts.ChangelogPromptInput{
		From:    from,
		To:      to,
		Commits: logs,
	})
	if err != nil {
		return nil, fmt.Errorf("building prompt: %w", err)
	}

	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("AI generation: %w", err)
	}

	// Parse response into a single entry
	entry := parseChangelogEntry(resp.Text, to)

	return &ChangelogResult{
		Entries:      []domain.ChangelogEntry{entry},
		Provider:     s.ai.Name(),
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}, nil
}

// parseChangelogEntry parses an AI response into a ChangelogEntry.
func parseChangelogEntry(text, version string) domain.ChangelogEntry {
	text = cleanResponse(text)

	entry := domain.ChangelogEntry{
		Version: version,
		Added:   extractBullets(text, "Added"),
		Fixed:   extractBullets(text, "Fixed"),
		Changed: extractBullets(text, "Changed"),
		Removed: extractBullets(text, "Removed"),
	}

	// If no sections found, treat the whole response as the body
	if entry.IsEmpty() {
		entry.Added = []string{text}
	}

	return entry
}
