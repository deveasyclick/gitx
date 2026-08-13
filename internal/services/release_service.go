package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

// ReleaseService orchestrates the release workflow.
type ReleaseService struct {
	git  git.Client
	ai   aiProvider
	pmpt *prompts.ReleaseBuilder
}

// NewReleaseService creates a new release service.
func NewReleaseService(gitClient git.Client, aiProvider aiProvider, prompt *prompts.ReleaseBuilder) *ReleaseService {
	return &ReleaseService{
		git:  gitClient,
		ai:   aiProvider,
		pmpt: prompt,
	}
}

// CompareResult holds the result of comparing two branches.
type CompareResult struct {
	Info domain.ReleaseInfo
	Diff domain.Change
}

// CompareBranches compares head branch against base and returns the differences.
func (s *ReleaseService) CompareBranches(ctx context.Context, base, head string) (*CompareResult, error) {
	// Get repo info
	repoInfo, err := s.git.RepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("repo info: %w", err)
	}

	// Resolve branches
	if base == "" {
		return nil, fmt.Errorf("base branch is required")
	}
	if head == "" {
		head = repoInfo.CurrentBranch
	}
	if head == base {
		return nil, fmt.Errorf("head branch %q is the same as base branch %q", head, base)
	}

	// Get commits in head that are not in base
	logs, err := s.git.Log(ctx, base, head)
	if err != nil {
		// Try the other direction
		return nil, fmt.Errorf("comparing %s..%s: %w", base, head, err)
	}

	// Get diff between base and head
	diff, err := s.git.Diff(ctx, base)
	if err != nil {
		// If diff fails, still proceed with just log info
		diff = domain.Change{}
	}

	// Determine current version from tags
	tags, err := s.git.Tags(ctx)
	if err != nil {
		tags = nil
	}

	currentVersion := ""
	var latestTag string
	if len(tags) > 0 {
		latestTag = tags[0]
		for _, t := range tags {
			currentVersion = t
			break
		}
	}

	// For the range, use base..head commits
	// Suggest next version based on latest tag
	nextVersion := SuggestNextVersion(currentVersion, "minor")

	// Convert git.CommitLog to domain.CommitSummary to avoid circular imports
	commitSummaries := toCommitSummaries(logs)

	info := domain.ReleaseInfo{
		BaseBranch:     base,
		HeadBranch:     head,
		CurrentVersion: currentVersion,
		NextVersion:    nextVersion,
		LatestTag:      latestTag,
		Commits:        commitSummaries,
		Diff:           diff,
	}

	return &CompareResult{
		Info: info,
		Diff: diff,
	}, nil
}

// CompareWithTag compares the current branch against a base version tag. It
// returns every commit after the tag up to the latest commit (the tip of the
// current branch) plus the cumulative diff since the tag.
// When tag is empty (or the sentinel "latest"), the newest tag is used.
func (s *ReleaseService) CompareWithTag(ctx context.Context, tag string) (*CompareResult, error) {
	if tag == "" || tag == "latest" {
		tags, err := s.git.Tags(ctx)
		if err != nil || len(tags) == 0 {
			return nil, fmt.Errorf("no version tags found — cannot use the latest tag (specify --tag <version> or use --base <branch>)")
		}
		tag = tags[0]
	}

	repoInfo, err := s.git.RepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("repo info: %w", err)
	}
	head := repoInfo.CurrentBranch
	if head == "" {
		head = "HEAD"
	}

	// Commits after the tag, up to the latest commit.
	logs, err := s.git.Log(ctx, tag, head)
	if err != nil {
		return nil, fmt.Errorf("comparing %s..%s (is %q a valid tag?): %w", tag, head, tag, err)
	}

	// Cumulative diff between the tag and the latest commit.
	diff, err := s.git.DiffRange(ctx, tag, head)
	if err != nil {
		// If the diff fails, still proceed with just log info.
		diff = domain.Change{}
	}

	// The explicitly requested tag is the base for this release, so it also
	// drives the version range and the next-version suggestion.
	nextVersion := SuggestNextVersion(tag, "minor")

	info := domain.ReleaseInfo{
		BaseBranch:     tag,
		HeadBranch:     head,
		CurrentVersion: tag,
		NextVersion:    nextVersion,
		LatestTag:      tag,
		Commits:        toCommitSummaries(logs),
		Diff:           diff,
	}

	return &CompareResult{
		Info: info,
		Diff: diff,
	}, nil
}

// SuggestVersion uses AI to suggest a version bump type.
func (s *ReleaseService) SuggestVersion(ctx context.Context, commits []domain.CommitSummary) (domain.VersionBump, error) {
	system, user, err := s.pmpt.BuildVersionSuggestion(commits)
	if err != nil {
		return "", fmt.Errorf("building version prompt: %w", err)
	}

	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   user,
	})
	if err != nil {
		return "", fmt.Errorf("AI suggestion: %w", err)
	}

	text := strings.TrimSpace(strings.ToLower(resp.Text))
	switch {
	case strings.Contains(text, "major"):
		return domain.BumpMajor, nil
	case strings.Contains(text, "minor"):
		return domain.BumpMinor, nil
	case strings.Contains(text, "patch"):
		return domain.BumpPatch, nil
	default:
		return domain.BumpMinor, nil
	}
}

// toCommitSummaries converts git.CommitLog entries to domain.CommitSummary.
func toCommitSummaries(logs []git.CommitLog) []domain.CommitSummary {
	commitSummaries := make([]domain.CommitSummary, len(logs))
	for i, l := range logs {
		commitSummaries[i] = domain.CommitSummary{
			Hash:    l.Hash,
			Author:  l.Author,
			Message: l.Message,
		}
	}
	return commitSummaries
}

// GenerateReleaseChangelog generates changelog text for a release.
func (s *ReleaseService) GenerateReleaseChangelog(ctx context.Context, info domain.ReleaseInfo, version string) (string, error) {
	// A release changelog should cover everything since the last release tag,
	// not just commits ahead of the base branch (base..head). base..head can
	// miss unreleased commits that landed on the base branch after the last
	// tag, which would otherwise silently drop real changes from the changelog.
	commits := info.Commits
	diff := info.Diff
	if s.git != nil && info.LatestTag != "" && info.HeadBranch != "" {
		if sinceTag, err := s.git.Log(ctx, info.LatestTag, info.HeadBranch); err == nil && len(sinceTag) > 0 {
			commits = toCommitSummaries(sinceTag)
		}
		// The diff is the source of truth for the changelog: use the
		// cumulative diff since the last release tag, not the base..head diff.
		if sinceTagDiff, err := s.git.DiffRange(ctx, info.LatestTag, info.HeadBranch); err == nil && !sinceTagDiff.IsEmpty() {
			diff = sinceTagDiff
		}
	}

	input := prompts.ReleaseChangelogInput{
		Version:       version,
		PreviousTag:   info.LatestTag,
		CurrentTag:    version,
		Commits:       commits,
		CommitDetails: diff.DiffStat,
		Diff:          diff.Diff,
	}

	system, user, err := s.pmpt.BuildReleaseChangelog(input)
	if err != nil {
		return "", fmt.Errorf("building changelog prompt: %w", err)
	}

	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   user,
	})
	if err != nil {
		return "", fmt.Errorf("AI changelog generation: %w", err)
	}

	return strings.TrimSpace(resp.Text), nil
}

// GeneratePRDescription generates a PR description for the release.
func (s *ReleaseService) GeneratePRDescription(ctx context.Context, info domain.ReleaseInfo, version, changelogText string) (string, error) {
	input := prompts.ReleasePRInput{
		Version:       version,
		BaseBranch:    info.BaseBranch,
		HeadBranch:    info.HeadBranch,
		ChangelogText: changelogText,
		Commits:       info.Commits,
		FilesChanged:  info.Diff.Files,
		DiffStat:      info.Diff.DiffStat,
	}

	system, user, err := s.pmpt.BuildPRDescription(input)
	if err != nil {
		return "", fmt.Errorf("building PR prompt: %w", err)
	}

	resp, err := s.ai.Generate(ctx, ai.Request{
		SystemPrompt: system,
		UserPrompt:   user,
	})
	if err != nil {
		return "", fmt.Errorf("AI PR description generation: %w", err)
	}

	return strings.TrimSpace(resp.Text), nil
}

// UpdateChangelogFile inserts or creates a CHANGELOG.md entry.
// The new entry is placed so the file keeps its existing version order
// (either newest-first or oldest-first, detected from the current entries).
// If an entry for the same version already exists (e.g. because the release
// command was run more than once), it is replaced in place instead of being
// duplicated.
func (s *ReleaseService) UpdateChangelogFile(changelogPath string, version string, newEntry string) error {
	dir := filepath.Dir(changelogPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	header := "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n"
	// Normalize the entry: "## <version>\n\n<entry>\n"
	entryBlock := fmt.Sprintf("## %s\n\n%s\n", version, formatChangelogEntry(newEntry))

	// Check if file exists
	existing, err := os.ReadFile(changelogPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", changelogPath, err)
		}
		// File doesn't exist — create it
		return os.WriteFile(changelogPath, []byte(header+entryBlock), 0o644)
	}

	text := string(existing)

	// If an entry for this version already exists, replace it instead of
	// prepending a duplicate.
	if updated, ok := replaceChangelogEntry(text, version, entryBlock); ok {
		return os.WriteFile(changelogPath, []byte(updated), 0o644)
	}

	// File exists without an entry for this version — insert it in the
	// position that keeps the file's existing version order.
	var content string
	if strings.HasPrefix(text, "# Changelog") {
		content = insertChangelogEntry(text, version, entryBlock)
	} else {
		// No standard header — add it and the entry.
		content = header + entryBlock + "\n" + text
	}

	return os.WriteFile(changelogPath, []byte(content), 0o644)
}

// versionHeadingRe matches "## v1.2.3" (or "## 1.2.3") section headers.
var versionHeadingRe = regexp.MustCompile(`(?m)^##\s+(v?\d+\.\d+\.\d+[^\s]*)\s*$`)

// insertChangelogEntry inserts entryBlock into text at the position that
// preserves the file's existing version order. The order convention
// (newest-first vs oldest-first) is detected from the entries already present;
// with fewer than two entries the new entry is prepended after the header.
func insertChangelogEntry(text, version, entryBlock string) string {
	type heading struct {
		version string
		start   int
	}
	var headings []heading
	for _, m := range versionHeadingRe.FindAllStringSubmatchIndex(text, -1) {
		headings = append(headings, heading{
			version: text[m[2]:m[3]],
			start:   m[0],
		})
	}

	if len(headings) < 2 {
		// Too few entries to detect order — prepend after the header block.
		return prependChangelogEntry(text, entryBlock)
	}

	// Detect order: ascending (oldest first) when the first entry is older
	// than the second; otherwise newest-first.
	ascending := compareVersions(headings[0].version, headings[1].version) < 0

	// Find the first heading the new entry sorts before.
	insertAt := len(headings)
	for i, h := range headings {
		cmp := compareVersions(h.version, version)
		if (ascending && cmp > 0) || (!ascending && cmp < 0) {
			insertAt = i
			break
		}
	}

	if insertAt == len(headings) {
		// New version sorts after every existing entry — append at the end.
		return strings.TrimRight(text, "\n") + "\n\n" + entryBlock
	}

	pos := headings[insertAt].start
	return text[:pos] + entryBlock + "\n" + text[pos:]
}

// prependChangelogEntry inserts entryBlock right after the header block, i.e.
// before the first "## " heading (or at the end when there is none).
func prependChangelogEntry(text, entryBlock string) string {
	// Find the first ## or end of file after header
	headerEnd := strings.Index(text, "\n## ")
	if headerEnd < 0 {
		// No existing entries — insert after the header block
		lines := strings.SplitN(text, "\n", 4)
		if len(lines) >= 3 {
			return lines[0] + "\n" + lines[1] + "\n" + lines[2] + "\n\n" + entryBlock + strings.Join(lines[3:], "\n")
		}
		return text + "\n" + entryBlock
	}
	// Insert after header but before first existing entry
	return text[:headerEnd] + "\n" + entryBlock + text[headerEnd:]
}

// compareVersions compares two version strings such as "v1.2.3" or "1.2.3",
// returning -1, 0, or 1. Non-semver strings fall back to lexical comparison.
func compareVersions(a, b string) int {
	amaj, amin, apat, aok := parseSemver(a)
	bmaj, bmin, bpat, bok := parseSemver(b)
	if !aok || !bok {
		return strings.Compare(a, b)
	}
	if amaj != bmaj {
		return sign(amaj - bmaj)
	}
	if amin != bmin {
		return sign(amin - bmin)
	}
	if apat != bpat {
		return sign(apat - bpat)
	}
	return 0
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

// parseSemver parses "v1.2.3" (or "1.2.3", optionally with a prerelease or
// build suffix) into its numeric parts.
func parseSemver(v string) (major, minor, patch int, ok bool) {
	v = strings.TrimPrefix(v, "v")
	// Strip prerelease/build suffix (e.g. "-beta.1" or "+build").
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0, 0, 0, false
		}
		nums[i] = n
	}
	return nums[0], nums[1], nums[2], true
}

// formatChangelogEntry normalizes a changelog entry so it satisfies common
// markdownlint rules (MD022 headings-surrounded-by-blank-lines, MD032
// lists-surrounded-by-blank-lines): each heading and each list block is
// separated from surrounding content by exactly one blank line, and the
// result ends with a single trailing newline.
func formatChangelogEntry(entry string) string {
	lines := strings.Split(strings.ReplaceAll(entry, "\r\n", "\n"), "\n")

	var out []string
	var block []string
	prevKind := kindOther
	flush := func() {
		if len(block) > 0 {
			out = append(out, strings.Join(block, "\n"))
			block = nil
		}
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		if strings.TrimSpace(line) == "" {
			flush()
			prevKind = kindOther
			continue
		}
		kind := classifyChangelogLine(line)
		// Each heading starts its own block (MD022: headings need blank lines
		// on both sides); list items group into one block.
		if len(block) > 0 && (kind != prevKind || kind == kindHeading) {
			flush()
		}
		if len(block) == 0 {
			// Start a new block; separate it from the previous block with a
			// blank line when needed (headings and lists both require one).
			if len(out) > 0 {
				out = append(out, "")
			}
		}
		block = append(block, line)
		prevKind = kind
	}
	flush()

	// Collapse the trailing blank line inserted between the last two blocks.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}

	return strings.Join(out, "\n")
}

type changelogLineKind int

const (
	kindOther changelogLineKind = iota
	kindHeading
	kindList
)

// classifyChangelogLine groups a line into heading, list item, or other.
func classifyChangelogLine(line string) changelogLineKind {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "#"):
		return kindHeading
	case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "),
		strings.HasPrefix(trimmed, "+ "), isNumberedListItem(trimmed):
		return kindList
	default:
		return kindOther
	}
}

// isNumberedListItem reports whether s looks like "1. item" or "1) item".
func isNumberedListItem(s string) bool {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(s) {
		return false
	}
	return (s[i] == '.' || s[i] == ')') && i+1 < len(s) && s[i+1] == ' '
}

// replaceChangelogEntry finds every existing "## <version>" section in text
// and swaps them for the single entryBlock, returning the updated text and
// true. If the version has no entry yet, it returns ok=false with text
// unchanged. Replacing in place (rather than always prepending) keeps
// re-running the release command from duplicating the entry, and repairs
// duplicates already present in the file.
func replaceChangelogEntry(text, version, entryBlock string) (string, bool) {
	re := regexp.MustCompile(`(?m)^##\s+` + regexp.QuoteMeta(version) + `\s*$`)

	// Each "## <version>" section runs from its header line to the next
	// "## " header (or the end of the file).
	var firstStart, lastEnd int
	found := false
	searchFrom := 0
	for {
		loc := re.FindStringIndex(text[searchFrom:])
		if loc == nil {
			break
		}
		start := searchFrom + loc[0]
		bodyStart := searchFrom + loc[1]
		end := len(text)
		if next := strings.Index(text[bodyStart:], "\n## "); next >= 0 {
			end = bodyStart + next
			// Drop the blank line left between this entry and the next header.
			for end > start && text[end-1] == '\n' {
				end--
			}
		}
		if !found {
			firstStart = start
			found = true
		}
		lastEnd = end
		searchFrom = end
	}

	if !found {
		return text, false
	}

	updated := text[:firstStart] + strings.TrimRight(entryBlock, "\n") + "\n" + text[lastEnd:]
	return updated, true
}

// BumpVersionInFiles updates version strings in known files.
// Currently supports: package.json, package-lock.json
// Returns list of files modified.
func (s *ReleaseService) BumpVersionInFiles(version string) ([]string, error) {
	version = strings.TrimPrefix(version, "v")
	var modified []string

	// package.json
	if _, err := os.Stat("package.json"); err == nil {
		if err := bumpJSONVersion("package.json", version); err != nil {
			return modified, fmt.Errorf("bumping package.json: %w", err)
		}
		modified = append(modified, "package.json")
	}

	// package-lock.json
	if _, err := os.Stat("package-lock.json"); err == nil {
		if err := bumpPackageLock("package-lock.json", version); err != nil {
			return modified, fmt.Errorf("bumping package-lock.json: %w", err)
		}
		modified = append(modified, "package-lock.json")
	}

	return modified, nil
}

// bumpJSONVersion reads a JSON file, updates only the top-level "version"
// field, and writes it back. Nested objects (e.g. every dependency in
// package-lock.json) keep their own versions untouched.
func bumpJSONVersion(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// The top-level "version" field is the first occurrence of the key at the
	// root of the document. Replace only that one; nested "version" fields
	// (package-lock.json dependencies) must stay as they are.
	re := regexp.MustCompile(`"version"\s*:\s*"[^"]+"`)
	loc := re.FindIndex(data)
	if loc == nil {
		return fmt.Errorf("no version field found in %s", path)
	}

	replacement := []byte(fmt.Sprintf(`"version": "%s"`, version))
	newData := make([]byte, 0, len(data))
	newData = append(newData, data[:loc[0]]...)
	newData = append(newData, replacement...)
	newData = append(newData, data[loc[1]:]...)

	if string(data) == string(newData) {
		return fmt.Errorf("no version field found in %s", path)
	}

	return os.WriteFile(path, newData, 0o644)
}

// bumpPackageLock updates the versions in a package-lock.json that mirror the
// app version: the top-level "version" field and the root package entry inside
// "packages" (the "" key, lockfile v2/v3). Every dependency entry under
// "node_modules/*" keeps its own version untouched.
func bumpPackageLock(path, version string) error {
	// Top-level "version" (first occurrence at the root of the document).
	if err := bumpJSONVersion(path, version); err != nil {
		return err
	}

	// Root package entry: "packages": { "": { ... "version": ... } }.
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Anchor at the start of the root entry object, then bump the first
	// "version" key inside it regardless of key order.
	rootRe := regexp.MustCompile(`"packages"\s*:\s*\{\s*"":\s*\{`)
	rootLoc := rootRe.FindIndex(data)
	if rootLoc == nil {
		return nil // no root package entry — nothing more to update
	}

	verRe := regexp.MustCompile(`"version"\s*:\s*"[^"]+"`)
	rel := rootLoc[1]
	verLoc := verRe.FindIndex(data[rel:])
	if verLoc == nil {
		return nil
	}

	start, end := rel+verLoc[0], rel+verLoc[1]
	replacement := []byte(fmt.Sprintf(`"version": "%s"`, version))
	newData := make([]byte, 0, len(data))
	newData = append(newData, data[:start]...)
	newData = append(newData, replacement...)
	newData = append(newData, data[end:]...)

	return os.WriteFile(path, newData, 0o644)
}

// SuggestNextVersion takes a current version (e.g., "v1.2.0" or "1.2.0") and a bump type,
// and returns the next version string with "v" prefix.
func SuggestNextVersion(current string, bump string) string {
	if current == "" {
		return "v0.1.0"
	}

	current = strings.TrimPrefix(current, "v")
	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "v" + current
	}

	major, minor, patch := parts[0], parts[1], parts[2]

	switch bump {
	case "major":
		return fmt.Sprintf("v%s.%s.%s", nextNum(major), "0", "0")
	case "minor":
		return fmt.Sprintf("v%s.%s.%s", major, nextNum(minor), "0")
	case "patch":
		return fmt.Sprintf("v%s.%s.%s", major, minor, nextNum(patch))
	default:
		return fmt.Sprintf("v%s.%s.%s", major, nextNum(minor), "0")
	}
}

func nextNum(s string) string {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return "1" // no leading number — treat the segment as 0
	}
	return fmt.Sprintf("%d", n+1)
}

// ComputeDiffStat returns a human-readable stat from the diff.
func ComputeDiffStat(diff domain.Change) string {
	if diff.DiffStat != "" {
		return diff.DiffStat
	}
	return fmt.Sprintf("%d file(s) changed", len(diff.Files))
}
