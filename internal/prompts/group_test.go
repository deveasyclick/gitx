package prompts

import (
	"strings"
	"testing"
)

func TestGroupBuilder_Build(t *testing.T) {
	b := NewGroupBuilder()
	system, user, err := b.Build(GroupPromptInput{
		Diff:   "diff --git a/auth.go b/auth.go\n-old\n+new",
		Branch: "feat/login",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if !strings.Contains(system, "logical groups") {
		t.Errorf("system prompt missing grouping instructions")
	}
	if !strings.Contains(system, `{"groups":[{"name":"short logical group description","files":["path/to/file.go"]}]}`) {
		t.Errorf("system prompt missing JSON schema")
	}
	if !strings.Contains(user, "feat/login") {
		t.Errorf("user prompt missing branch")
	}
	if !strings.Contains(user, "diff --git a/auth.go") {
		t.Errorf("user prompt missing diff")
	}
}

func TestGroupBuilder_Build_EmptyDiff(t *testing.T) {
	_, _, err := NewGroupBuilder().Build(GroupPromptInput{Diff: "   "})
	if err == nil {
		t.Fatal("expected error for empty diff")
	}
}
