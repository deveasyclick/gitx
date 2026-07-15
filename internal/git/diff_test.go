package git

import (
	"strings"
	"testing"

	"github.com/user/gitx/internal/domain"
)

func TestGroupDiffsByDir(t *testing.T) {
	diff := `diff --git a/api/handler.go b/api/handler.go
index abc..def 100644
--- a/api/handler.go
+++ b/api/handler.go
@@ -1,3 +1,4 @@
 package api
+// new handler
 func Handler() {}

diff --git a/api/router.go b/api/router.go
index 123..456 100644
--- a/api/router.go
+++ b/api/router.go
@@ -5,3 +5,4 @@
+// new route
 func Setup() {}

diff --git a/ui/button.go b/ui/button.go
index 789..012 100644
--- a/ui/button.go
+++ b/ui/button.go
@@ -10,3 +10,4 @@
+// new button
 func Button() {}

diff --git a/main.go b/main.go
index 345..678 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
+// new main
 func main() {}`

	change := domain.Change{Diff: diff}
	groups := GroupDiffsByDir(change)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups (api, ui, .), got %d", len(groups))
	}

	// Check api group
	if groups[0].Dir != "api" {
		t.Errorf("group[0].Dir = %q, want %q", groups[0].Dir, "api")
	}
	if len(groups[0].Files) != 2 {
		t.Errorf("expected 2 files in api group, got %d: %v", len(groups[0].Files), groups[0].Files)
	}

	// Check ui group
	if groups[1].Dir != "ui" {
		t.Errorf("group[1].Dir = %q, want %q", groups[1].Dir, "ui")
	}

	// Check root group
	if groups[2].Dir != "." {
		t.Errorf("group[2].Dir = %q, want %q", groups[2].Dir, ".")
	}

	// Each group should have valid diff content
	for _, g := range groups {
		if g.Diff == "" {
			t.Errorf("group %q has empty diff", g.Dir)
		}
		if !strings.Contains(g.Diff, "+//") {
			t.Errorf("group %q diff missing content", g.Dir)
		}
	}
}

func TestGroupDiffsByDir_SingleFile(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
+// change`

	change := domain.Change{Diff: diff}
	groups := GroupDiffsByDir(change)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Dir != "." {
		t.Errorf("Dir = %q, want %q", groups[0].Dir, ".")
	}
}

func TestGroupDiffsByDir_EmptyDiff(t *testing.T) {
	change := domain.Change{Diff: ""}
	groups := GroupDiffsByDir(change)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for empty diff, got %d", len(groups))
	}
}
