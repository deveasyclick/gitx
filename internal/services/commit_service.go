package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
	"github.com/user/gitx/internal/security"
)

// GroupedResult contains one generated commit message per group.
type GroupedResult struct {
	Results []GroupedGenerateResult
}

// GroupedGenerateResult is a single group's generated message.
type GroupedGenerateResult struct {
	Dir     string
	Files   []string
	Message domain.CommitMessage
	Diff    string
}

// CommitMode controls which changes to use for commit message generation.
type CommitMode int

const (
	// CommitModeStaged uses staged changes only.
	CommitModeStaged CommitMode = iota
	// CommitModeUnstaged uses unstaged changes (tracked modified files) only.
	CommitModeUnstaged
	// CommitModeAuto tries staged first, falls back to unstaged.
	CommitModeAuto
)

// CommitService orchestrates commit message generation.
type CommitService struct {
	git  git.Client
	ai   aiProvider
	pmpt prompts.Builder[prompts.CommitPromptInput]
	scan bool
}

// NewCommitService creates a new commit service.
func NewCommitService(gitClient git.Client, aiProvider aiProvider, prompt prompts.Builder[prompts.CommitPromptInput]) *CommitService {
	return &CommitService{
		git:  gitClient,
		ai:   aiProvider,
		pmpt: prompt,
		scan: true,
	}
}

// GenerateResult contains the result of commit message generation.
type GenerateResult struct {
	Message      domain.CommitMessage
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
}

// Generate generates a commit message from changes based on the given mode.
func (s *CommitService) Generate(ctx context.Context, mode CommitMode) (*GenerateResult, error) {
	var change domain.Change
	var source string

	switch mode {
	case CommitModeStaged:
		if err := s.checkStaged(ctx); err != nil {
			return nil, err
		}
		var err error
		change, err = s.git.DiffCached(ctx)
		if err != nil {
			return nil, fmt.Errorf("reading staged diff: %w", err)
		}
		source = "staged"

	case CommitModeUnstaged:
		if err := s.checkUnstaged(ctx); err != nil {
			return nil, err
		}
		var err error
		change, err = s.git.DiffUnstaged(ctx)
		if err != nil {
			return nil, fmt.Errorf("reading unstaged diff: %w", err)
		}
		source = "unstaged"

	case CommitModeAuto:
		// Try staged first
		stagedStatus, err := s.git.Status(ctx)
		if err != nil {
			return nil, fmt.Errorf("checking staged changes: %w", err)
		}
		if !stagedStatus.IsEmpty {
			change, err = s.git.DiffCached(ctx)
			if err != nil {
				return nil, fmt.Errorf("reading staged diff: %w", err)
			}
			source = "staged"
		} else {
			// Fall back to unstaged
			unstagedStatus, err := s.git.UnstagedStatus(ctx)
			if err != nil {
				return nil, fmt.Errorf("checking unstaged changes: %w", err)
			}
			if unstagedStatus.IsEmpty {
				return nil, errors.New("no staged or unstaged changes found")
			}
			change, err = s.git.DiffUnstaged(ctx)
			if err != nil {
				return nil, fmt.Errorf("reading unstaged diff: %w", err)
			}
			source = "unstaged"
		}
	}

	// Scan for secrets
	if s.scan {
		cleaned, found := security.Scan(change.Diff)
		if found {
			change.Diff = cleaned
		}
	}

	// Get repo info for branch name
	info, err := s.git.RepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading repo info: %w", err)
	}

	// Build prompt
	_, userPrompt, err := s.pmpt.Build(prompts.CommitPromptInput{
		Diff:   change,
		Branch: info.CurrentBranch,
		Style:  "conventional",
	})
	if err != nil {
		return nil, fmt.Errorf("building prompt: %w", err)
	}

	// Generate
	system := "You are an expert software engineer. Generate a conventional commit message."
	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("AI generation: %w", err)
	}

	// Parse response into commit message
	msg := parseCommitMessage(resp.Text)
	_ = source // available for future logging

	return &GenerateResult{
		Message:      msg,
		Provider:     s.ai.Name(),
		Model:        "",
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}, nil
}

// GenerateGrouped splits changes by directory and generates one commit message per group.
func (s *CommitService) GenerateGrouped(ctx context.Context, mode CommitMode) (*GroupedResult, error) {
	var change domain.Change

	switch mode {
	case CommitModeStaged:
		if err := s.checkStaged(ctx); err != nil {
			return nil, err
		}
		var err error
		change, err = s.git.DiffCached(ctx)
		if err != nil {
			return nil, fmt.Errorf("reading staged diff: %w", err)
		}

	case CommitModeUnstaged:
		if err := s.checkUnstaged(ctx); err != nil {
			return nil, err
		}
		var err error
		change, err = s.git.DiffUnstaged(ctx)
		if err != nil {
			return nil, fmt.Errorf("reading unstaged diff: %w", err)
		}

	case CommitModeAuto:
		stagedStatus, err := s.git.Status(ctx)
		if err != nil {
			return nil, fmt.Errorf("checking staged changes: %w", err)
		}
		if !stagedStatus.IsEmpty {
			var err error
			change, err = s.git.DiffCached(ctx)
			if err != nil {
				return nil, fmt.Errorf("reading staged diff: %w", err)
			}
		} else {
			unstagedStatus, err := s.git.UnstagedStatus(ctx)
			if err != nil {
				return nil, fmt.Errorf("checking unstaged changes: %w", err)
			}
			if unstagedStatus.IsEmpty {
				return nil, errors.New("no staged or unstaged changes found")
			}
			change, err = s.git.DiffUnstaged(ctx)
			if err != nil {
				return nil, fmt.Errorf("reading unstaged diff: %w", err)
			}
		}
	}

	// Scan for secrets on the full diff
	if s.scan {
		change.Diff, _ = security.Scan(change.Diff)
	}

	info, err := s.git.RepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading repo info: %w", err)
	}

	// Group diffs by directory
	groups := git.GroupDiffsByDir(change)
	if len(groups) == 0 {
		return nil, errors.New("no changes to group")
	}

	var results []GroupedGenerateResult
	for _, g := range groups {
		groupChange := domain.Change{
			Files: g.Files,
			Diff:  g.Diff,
		}

		_, userPrompt, err := s.pmpt.Build(prompts.CommitPromptInput{
			Diff:   groupChange,
			Branch: info.CurrentBranch,
			Style:  "conventional",
		})
		if err != nil {
			return nil, fmt.Errorf("building prompt for %s: %w", g.Dir, err)
		}

		system := "You are an expert software engineer. Generate a conventional commit message."
		resp, err := s.ai.Generate(ctx, ai.Request{
			SystemPrompt: system,
			UserPrompt:   userPrompt,
		})
		if err != nil {
			return nil, fmt.Errorf("AI generation for %s: %w", g.Dir, err)
		}

		msg := parseCommitMessage(resp.Text)
		results = append(results, GroupedGenerateResult{
			Dir:     g.Dir,
			Files:   g.Files,
			Message: msg,
			Diff:    g.Diff,
		})
	}

	return &GroupedResult{Results: results}, nil
}

func (s *CommitService) checkStaged(ctx context.Context) error {
	status, err := s.git.Status(ctx)
	if err != nil {
		return fmt.Errorf("checking staged changes: %w", err)
	}
	if status.IsEmpty {
		return domain.ErrNoStagedChanges
	}
	return nil
}

func (s *CommitService) checkUnstaged(ctx context.Context) error {
	status, err := s.git.UnstagedStatus(ctx)
	if err != nil {
		return fmt.Errorf("checking unstaged changes: %w", err)
	}
	if status.IsEmpty {
		return errors.New("no unstaged changes found (try staging changes first)")
	}
	return nil
}

// parseCommitMessage parses an AI response into a CommitMessage.
// The AI typically returns "type(scope): description\n\nbody".
func parseCommitMessage(text string) domain.CommitMessage {
	text = cleanResponse(text)

	title, body, _ := cut(text, "\n\n")
	if title == "" {
		title = text
	}

	if body == "" {
		// Try single newline
		title, body, _ = cut(text, "\n")
	}

	return domain.CommitMessage{
		Title: title,
		Body:  strings.TrimSpace(body),
		Style: "conventional",
	}
}

// cleanResponse removes markdown code fences and trims whitespace.
func cleanResponse(text string) string {
	text = strings.TrimSpace(text)

	// Remove ``` (with optional language) ... ``` if present
	if strings.HasPrefix(text, "```") {
		// Find the end of the opening fence line
		nl := strings.Index(text, "\n")
		if nl >= 0 {
			text = text[nl+1:] // skip past the ```lang\n
		} else {
			text = ""
		}
		// Remove closing fence
		if idx := strings.LastIndex(text, "```"); idx >= 0 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}

	// Remove leading/trailing quotes
	text = strings.Trim(text, `"'`)
	return strings.TrimSpace(text)
}

// cut splits s at the first sep, returning before and after.
func cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
