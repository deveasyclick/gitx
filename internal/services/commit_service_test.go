package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/user/gitx/internal/ai"
	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/git"
	"github.com/user/gitx/internal/prompts"
)

func TestCommitService_Generate_Success(t *testing.T) {
	svc := NewCommitService(
		&mockGit{
			status: git.StagedChanges{Files: []string{"main.go"}, IsEmpty: false},
			diff:   domain.Change{Files: []string{"main.go"}, Diff: "diff --git a/main.go b/main.go\n+new\n-old"},
			repo:   domain.RepoInfo{CurrentBranch: "main", IsClean: false},
		},
		&mockAI{text: "feat: add new feature"},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	result, err := svc.Generate(context.Background(), CommitModeStaged)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.Message.Title != "feat: add new feature" {
		t.Errorf("title = %q", result.Message.Title)
	}
	if result.Message.Style != "conventional" {
		t.Errorf("style = %q", result.Message.Style)
	}
}

func TestCommitService_Generate_NoStagedChanges(t *testing.T) {
	svc := NewCommitService(
		&mockGit{
			status: git.StagedChanges{IsEmpty: true},
		},
		&mockAI{},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	_, err := svc.Generate(context.Background(), CommitModeStaged)
	if !errors.Is(err, domain.ErrNoStagedChanges) {
		t.Errorf("expected ErrNoStagedChanges, got %v", err)
	}
}

func TestCommitService_Generate_ParseResponse(t *testing.T) {
	tests := []struct {
		name     string
		aiText   string
		want     string
		wantBody string
	}{
		{
			name:     "simple title",
			aiText:   "feat: add login",
			want:     "feat: add login",
			wantBody: "",
		},
		{
			name:     "title and body",
			aiText:   "feat(auth): add login\n\n- Added OAuth provider\n- Added token validation",
			want:     "feat(auth): add login",
			wantBody: "- Added OAuth provider\n- Added token validation",
		},
		{
			name:     "wrapped in markdown",
			aiText:   "```\nfeat: add login\n```",
			want:     "feat: add login",
			wantBody: "",
		},
		{
			name:     "with quotes",
			aiText:   "\"feat: add login\"",
			want:     "feat: add login",
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewCommitService(
				&mockGit{
					status: git.StagedChanges{Files: []string{"x.go"}, IsEmpty: false},
					diff:   domain.Change{Files: []string{"x.go"}, Diff: "diff --git a/x.go b/x.go\n"},
					repo:   domain.RepoInfo{CurrentBranch: "main"},
				},
				&mockAI{text: tt.aiText},
				prompts.NewCommitBuilder(),
				prompts.NewGroupBuilder(),
			)

			result, err := svc.Generate(context.Background(), CommitModeStaged)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			if result.Message.Title != tt.want {
				t.Errorf("title = %q, want %q", result.Message.Title, tt.want)
			}
			if result.Message.Body != tt.wantBody {
				t.Errorf("body = %q, want %q", result.Message.Body, tt.wantBody)
			}
		})
	}
}

func TestCleanResponse(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"  spaced  ", "spaced"},
		{"```\ncontent\n```", "content"},
		{"```\ncode block\n```\ntrailing", "code block"},
		{`"quoted text"`, "quoted text"},
		{"```go\npackage main\n```", "package main"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanResponse(tt.input)
			if got != tt.want {
				t.Errorf("cleanResponse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCommitService_GenerateGrouped_FolderFallback_MultipleDirectories(t *testing.T) {
	diff := `diff --git a/api/handler.go b/api/handler.go
index abc..def 100644
--- a/api/handler.go
+++ b/api/handler.go
@@ -1 +1 @@
-func old() {}
+func new() {}
diff --git a/ui/button.go b/ui/button.go
index 123..456 100644
--- a/ui/button.go
+++ b/ui/button.go
@@ -1 +1 @@
-old button
+new button`

	// The AI's response to the grouping call is not valid JSON, so the service
	// must fall back to grouping by directory.
	svc := NewCommitService(
		&mockGit{
			status:         git.StagedChanges{IsEmpty: true},
			unstagedStatus: git.StagedChanges{Files: []string{"api/handler.go", "ui/button.go"}, IsEmpty: false},
			diff:           domain.Change{Files: []string{"api/handler.go", "ui/button.go"}, Diff: diff},
			repo:           domain.RepoInfo{CurrentBranch: "main"},
		},
		&mockAI{
			texts: []string{"feat: update component", "feat: update component", "feat: update component"},
		},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	result, err := svc.GenerateGrouped(context.Background(), CommitModeAuto)
	if err != nil {
		t.Fatalf("GenerateGrouped: %v", err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result.Results))
	}

	// First group should be api
	if result.Results[0].Name != "api" {
		t.Errorf("group[0].Name = %q, want %q", result.Results[0].Name, "api")
	}
	if len(result.Results[0].Files) != 1 || result.Results[0].Files[0] != "api/handler.go" {
		t.Errorf("group[0].Files = %v, want [api/handler.go]", result.Results[0].Files)
	}
	if result.Results[0].Message.Title != "feat: update component" {
		t.Errorf("group[0].Message.Title = %q", result.Results[0].Message.Title)
	}

	// Second group should be ui
	if result.Results[1].Name != "ui" {
		t.Errorf("group[1].Name = %q, want %q", result.Results[1].Name, "ui")
	}
	if len(result.Results[1].Files) != 1 || result.Results[1].Files[0] != "ui/button.go" {
		t.Errorf("group[1].Files = %v, want [ui/button.go]", result.Results[1].Files)
	}
}

func TestCommitService_GenerateGrouped_AI_LogicalGroups(t *testing.T) {
	diff := `diff --git a/api/auth.go b/api/auth.go
index abc..def 100644
--- a/api/auth.go
+++ b/api/auth.go
@@ -1 +1 @@
-old auth
+new auth
diff --git a/ui/auth_modal.go b/ui/auth_modal.go
index 123..456 100644
--- a/ui/auth_modal.go
+++ b/ui/auth_modal.go
@@ -1 +1 @@
-old modal
+new modal
diff --git a/api/payment.go b/api/payment.go
index 111..222 100644
--- a/api/payment.go
+++ b/api/payment.go
@@ -1 +1 @@
-old payment
+new payment
diff --git a/ui/payment_form.go b/ui/payment_form.go
index 333..444 100644
--- a/ui/payment_form.go
+++ b/ui/payment_form.go
@@ -1 +1 @@
-old form
+new form`

	// The AI groups files across directories by concern: auth (api + ui) and
	// payment (api + ui).
	plan := `{"groups":[
		{"name":"add user authentication","files":["api/auth.go","ui/auth_modal.go"]},
		{"name":"add payment flow","files":["api/payment.go","ui/payment_form.go"]}
	]}`

	svc := NewCommitService(
		&mockGit{
			status: git.StagedChanges{
				Files:   []string{"api/auth.go", "ui/auth_modal.go", "api/payment.go", "ui/payment_form.go"},
				IsEmpty: false,
			},
			diff: domain.Change{
				Files: []string{"api/auth.go", "ui/auth_modal.go", "api/payment.go", "ui/payment_form.go"},
				Diff:  diff,
			},
			repo: domain.RepoInfo{CurrentBranch: "main"},
		},
		&mockAI{
			texts: []string{plan, "feat: add user authentication", "feat: add payment flow"},
		},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	result, err := svc.GenerateGrouped(context.Background(), CommitModeStaged)
	if err != nil {
		t.Fatalf("GenerateGrouped: %v", err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result.Results))
	}

	g0 := result.Results[0]
	if g0.Name != "add user authentication" {
		t.Errorf("group[0].Name = %q", g0.Name)
	}
	if len(g0.Files) != 2 || g0.Files[0] != "api/auth.go" || g0.Files[1] != "ui/auth_modal.go" {
		t.Errorf("group[0].Files = %v", g0.Files)
	}
	if g0.Message.Title != "feat: add user authentication" {
		t.Errorf("group[0].Message.Title = %q", g0.Message.Title)
	}
	// Each group's diff must contain only its own files.
	if !contains(g0.Diff, "api/auth.go") || contains(g0.Diff, "payment") {
		t.Errorf("group[0].Diff does not match its files:\n%s", g0.Diff)
	}

	g1 := result.Results[1]
	if g1.Name != "add payment flow" {
		t.Errorf("group[1].Name = %q", g1.Name)
	}
	if len(g1.Files) != 2 || g1.Files[0] != "api/payment.go" || g1.Files[1] != "ui/payment_form.go" {
		t.Errorf("group[1].Files = %v", g1.Files)
	}
	if g1.Message.Title != "feat: add payment flow" {
		t.Errorf("group[1].Message.Title = %q", g1.Message.Title)
	}
	if !contains(g1.Diff, "api/payment.go") || contains(g1.Diff, "auth") {
		t.Errorf("group[1].Diff does not match its files:\n%s", g1.Diff)
	}
}

func TestCommitService_GenerateGrouped_Fallback_InvalidJSON(t *testing.T) {
	diff := `diff --git a/api/handler.go b/api/handler.go
index abc..def 100644
--- a/api/handler.go
+++ b/api/handler.go
@@ -1 +1 @@
-func old() {}
+func new() {}
diff --git a/ui/button.go b/ui/button.go
index 123..456 100644
--- a/ui/button.go
+++ b/ui/button.go
@@ -1 +1 @@
-old button
+new button`

	svc := NewCommitService(
		&mockGit{
			status: git.StagedChanges{
				Files:   []string{"api/handler.go", "ui/button.go"},
				IsEmpty: false,
			},
			diff: domain.Change{Files: []string{"api/handler.go", "ui/button.go"}, Diff: diff},
			repo: domain.RepoInfo{CurrentBranch: "main"},
		},
		&mockAI{
			texts: []string{"sure, here is the plan", "feat: api change", "feat: ui change"},
		},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	result, err := svc.GenerateGrouped(context.Background(), CommitModeStaged)
	if err != nil {
		t.Fatalf("GenerateGrouped: %v", err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 fallback groups, got %d", len(result.Results))
	}
	if result.Results[0].Name != "api" || result.Results[1].Name != "ui" {
		t.Errorf("expected folder fallback groups, got %q and %q", result.Results[0].Name, result.Results[1].Name)
	}
	if result.Results[0].Message.Title != "feat: api change" {
		t.Errorf("group[0].Message.Title = %q", result.Results[0].Message.Title)
	}
}

func TestCommitService_GenerateGrouped_Fallback_AIError(t *testing.T) {
	diff := `diff --git a/api/handler.go b/api/handler.go
index abc..def 100644
--- a/api/handler.go
+++ b/api/handler.go
@@ -1 +1 @@
-func old() {}
+func new() {}
diff --git a/ui/button.go b/ui/button.go
index 123..456 100644
--- a/ui/button.go
+++ b/ui/button.go
@@ -1 +1 @@
-old button
+new button`

	svc := NewCommitService(
		&mockGit{
			status: git.StagedChanges{
				Files:   []string{"api/handler.go", "ui/button.go"},
				IsEmpty: false,
			},
			diff: domain.Change{Files: []string{"api/handler.go", "ui/button.go"}, Diff: diff},
			repo: domain.RepoInfo{CurrentBranch: "main"},
		},
		&mockAI{
			errs:  []error{errors.New("provider unavailable")},
			texts: []string{"", "feat: api change", "feat: ui change"},
		},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	result, err := svc.GenerateGrouped(context.Background(), CommitModeStaged)
	if err != nil {
		t.Fatalf("GenerateGrouped: %v", err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 fallback groups, got %d", len(result.Results))
	}
	if result.Results[0].Name != "api" || result.Results[1].Name != "ui" {
		t.Errorf("expected folder fallback groups, got %q and %q", result.Results[0].Name, result.Results[1].Name)
	}
}

func TestCommitService_GenerateGrouped_Fallback_IncompleteCoverage(t *testing.T) {
	diff := `diff --git a/api/handler.go b/api/handler.go
index abc..def 100644
--- a/api/handler.go
+++ b/api/handler.go
@@ -1 +1 @@
-func old() {}
+func new() {}
diff --git a/ui/button.go b/ui/button.go
index 123..456 100644
--- a/ui/button.go
+++ b/ui/button.go
@@ -1 +1 @@
-old button
+new button`

	// The AI plan omits ui/button.go; grouping must not drop it, so the service
	// falls back to grouping by directory.
	plan := `{"groups":[{"name":"only api","files":["api/handler.go"]}]}`

	svc := NewCommitService(
		&mockGit{
			status: git.StagedChanges{
				Files:   []string{"api/handler.go", "ui/button.go"},
				IsEmpty: false,
			},
			diff: domain.Change{Files: []string{"api/handler.go", "ui/button.go"}, Diff: diff},
			repo: domain.RepoInfo{CurrentBranch: "main"},
		},
		&mockAI{
			texts: []string{plan, "feat: api change", "feat: ui change"},
		},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	result, err := svc.GenerateGrouped(context.Background(), CommitModeStaged)
	if err != nil {
		t.Fatalf("GenerateGrouped: %v", err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 fallback groups, got %d", len(result.Results))
	}
	if result.Results[0].Name != "api" || result.Results[1].Name != "ui" {
		t.Errorf("expected folder fallback groups, got %q and %q", result.Results[0].Name, result.Results[1].Name)
	}
}

func TestCommitService_GenerateGrouped_NoChanges(t *testing.T) {
	svc := NewCommitService(
		&mockGit{
			status:         git.StagedChanges{IsEmpty: true},
			unstagedStatus: git.StagedChanges{IsEmpty: true},
		},
		&mockAI{},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	_, err := svc.GenerateGrouped(context.Background(), CommitModeAuto)
	if err == nil {
		t.Fatal("expected error for no changes")
	}
}

func TestCommitService_GenerateGrouped_RootFiles(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1 +1 @@
-old
+new`

	svc := NewCommitService(
		&mockGit{
			status: git.StagedChanges{Files: []string{"main.go"}, IsEmpty: false},
			diff:   domain.Change{Files: []string{"main.go"}, Diff: diff},
			repo:   domain.RepoInfo{CurrentBranch: "main"},
		},
		&mockAI{text: "feat: update main"},
		prompts.NewCommitBuilder(),
		prompts.NewGroupBuilder(),
	)

	result, err := svc.GenerateGrouped(context.Background(), CommitModeStaged)
	if err != nil {
		t.Fatalf("GenerateGrouped: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Results))
	}
	if result.Results[0].Name != "." {
		t.Errorf("Name = %q, want %q", result.Results[0].Name, ".")
	}
	if len(result.Results[0].Files) != 1 || result.Results[0].Files[0] != "main.go" {
		t.Errorf("Files = %v, want [main.go]", result.Results[0].Files)
	}
}

func TestParseGroupPlan(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []git.FileGroup
		wantErr bool
	}{
		{
			name:  "valid plan",
			input: `{"groups":[{"name":"add auth","files":["auth.go","auth_test.go"]},{"name":"fix ui","files":["button.go"]}]}`,
			want: []git.FileGroup{
				{Name: "add auth", Files: []string{"auth.go", "auth_test.go"}},
				{Name: "fix ui", Files: []string{"button.go"}},
			},
		},
		{
			name:  "wrapped in markdown fences",
			input: "```json\n{\"groups\":[{\"name\":\"add auth\",\"files\":[\"auth.go\"]}]}\n```",
			want: []git.FileGroup{
				{Name: "add auth", Files: []string{"auth.go"}},
			},
		},
		{
			name:  "surrounding commentary",
			input: `Here is my plan: {"groups":[{"name":"add auth","files":["auth.go"]}]} hope that helps`,
			want: []git.FileGroup{
				{Name: "add auth", Files: []string{"auth.go"}},
			},
		},
		{
			name:  "duplicate files deduped",
			input: `{"groups":[{"name":"add auth","files":["auth.go","auth.go"]}]}`,
			want: []git.FileGroup{
				{Name: "add auth", Files: []string{"auth.go"}},
			},
		},
		{
			name:    "no JSON object",
			input:   "not json at all",
			wantErr: true,
		},
		{
			name:    "empty groups",
			input:   `{"groups":[]}`,
			wantErr: true,
		},
		{
			name:    "group without name",
			input:   `{"groups":[{"files":["auth.go"]}]}`,
			wantErr: true,
		},
		{
			name:    "group without files",
			input:   `{"groups":[{"name":"empty group"}]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGroupPlan(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGroupPlan: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d groups, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("group[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
				if len(got[i].Files) != len(tt.want[i].Files) {
					t.Errorf("group[%d].Files = %v, want %v", i, got[i].Files, tt.want[i].Files)
					continue
				}
				for j := range tt.want[i].Files {
					if got[i].Files[j] != tt.want[i].Files[j] {
						t.Errorf("group[%d].Files = %v, want %v", i, got[i].Files, tt.want[i].Files)
						break
					}
				}
			}
		})
	}
}

func TestCut(t *testing.T) {
	before, after, found := cut("a\n\nb", "\n\n")
	if !found || before != "a" || after != "b" {
		t.Errorf("cut(a\\n\\nb) = (%q, %q, %v)", before, after, found)
	}

	before, after, found = cut("no-separator", "\n\n")
	if found {
		t.Errorf("expected not found")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// --- Mocks ---

type mockGit struct {
	status           git.StagedChanges
	unstagedStatus   git.StagedChanges
	diff             domain.Change
	repo             domain.RepoInfo
	log              []git.CommitLog
	tags             []string
	err              error
	logFrom, logTo   string
	diffFrom, diffTo string
}

func (m *mockGit) DiffCached(_ context.Context) (domain.Change, error)         { return m.diff, m.err }
func (m *mockGit) DiffUnstaged(_ context.Context) (domain.Change, error)       { return m.diff, m.err }
func (m *mockGit) Diff(_ context.Context, _ string) (domain.Change, error)     { return m.diff, m.err }
func (m *mockGit) DiffRange(_ context.Context, from, to string) (domain.Change, error) {
	m.diffFrom, m.diffTo = from, to
	return m.diff, m.err
}
func (m *mockGit) Log(_ context.Context, from, to string) ([]git.CommitLog, error) {
	m.logFrom, m.logTo = from, to
	return m.log, m.err
}
func (m *mockGit) RecentCommits(_ context.Context, _ int, _ string) ([]git.CommitLog, error) {
	return m.log, m.err
}
func (m *mockGit) Commit(_ context.Context, _ domain.CommitMessage) error { return m.err }
func (m *mockGit) Status(_ context.Context) (git.StagedChanges, error)    { return m.status, m.err }
func (m *mockGit) UnstagedStatus(_ context.Context) (git.StagedChanges, error) {
	return m.unstagedStatus, m.err
}
func (m *mockGit) Tags(_ context.Context) ([]string, error)            { return m.tags, m.err }
func (m *mockGit) RepoInfo(_ context.Context) (domain.RepoInfo, error) { return m.repo, m.err }
func (m *mockGit) Stage(_ context.Context, _ []string) error           { return m.err }
func (m *mockGit) StageAll(_ context.Context) error                    { return m.err }
func (m *mockGit) UnstageAll(_ context.Context) error                  { return m.err }

type mockAI struct {
	text           string   // fallback text returned when texts is empty
	err            error    // single error; returned when errs is empty
	texts          []string // queued responses; each Generate call pops the next
	errs           []error  // queued errors parallel to texts; nil entry means no error
	lastUserPrompt string
}

func (m *mockAI) Name() string { return "mock" }
func (m *mockAI) Generate(_ context.Context, req ai.Request) (ai.Response, error) {
	m.lastUserPrompt = req.UserPrompt
	if len(m.errs) > 0 {
		err := m.errs[0]
		m.errs = m.errs[1:]
		if err != nil {
			return ai.Response{}, err
		}
	} else if m.err != nil {
		return ai.Response{}, m.err
	}
	text := m.text
	if len(m.texts) > 0 {
		text = m.texts[0]
		m.texts = m.texts[1:]
	}
	return ai.Response{
		Text: text,
		Usage: ai.TokenUsage{
			InputTokens:  50,
			OutputTokens: 10,
			TotalTokens:  60,
		},
	}, nil
}
