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
	if groups[0].Name != "api" {
		t.Errorf("group[0].Name = %q, want %q", groups[0].Name, "api")
	}
	if len(groups[0].Files) != 2 {
		t.Errorf("expected 2 files in api group, got %d: %v", len(groups[0].Files), groups[0].Files)
	}

	// Check ui group
	if groups[1].Name != "ui" {
		t.Errorf("group[1].Name = %q, want %q", groups[1].Name, "ui")
	}

	// Check root group
	if groups[2].Name != "." {
		t.Errorf("group[2].Name = %q, want %q", groups[2].Name, ".")
	}

	// Each group should have valid diff content
	for _, g := range groups {
		if g.Diff == "" {
			t.Errorf("group %q has empty diff", g.Name)
		}
		if !strings.Contains(g.Diff, "+//") {
			t.Errorf("group %q diff missing content", g.Name)
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
	if groups[0].Name != "." {
		t.Errorf("Name = %q, want %q", groups[0].Name, ".")
	}
}

func TestGroupDiffsByDir_EmptyDiff(t *testing.T) {
	change := domain.Change{Diff: ""}
	groups := GroupDiffsByDir(change)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for empty diff, got %d", len(groups))
	}
}

func TestGroupDiffsByFileGroups(t *testing.T) {
	diff := `diff --git a/api/handler.go b/api/handler.go
index abc..def 100644
--- a/api/handler.go
+++ b/api/handler.go
@@ -1 +1 @@
-func old() {}
+func new() {}
diff --git a/api/payment.go b/api/payment.go
index 111..222 100644
--- a/api/payment.go
+++ b/api/payment.go
@@ -1 +1 @@
-old payment
+new payment
diff --git a/ui/button.go b/ui/button.go
index 123..456 100644
--- a/ui/button.go
+++ b/ui/button.go
@@ -1 +1 @@
-old button
+new button`

	change := domain.Change{Diff: diff}
	groups := GroupDiffsByFileGroups(change, []FileGroup{
		{Name: "payment backend", Files: []string{"api/payment.go"}},
		{Name: "ui polish", Files: []string{"ui/button.go", "api/handler.go"}},
	})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	if groups[0].Name != "payment backend" {
		t.Errorf("group[0].Name = %q", groups[0].Name)
	}
	if len(groups[0].Files) != 1 || groups[0].Files[0] != "api/payment.go" {
		t.Errorf("group[0].Files = %v", groups[0].Files)
	}
	if !strings.Contains(groups[0].Diff, "api/payment.go") || strings.Contains(groups[0].Diff, "button") {
		t.Errorf("group[0].Diff mismatch:\n%s", groups[0].Diff)
	}

	if groups[1].Name != "ui polish" {
		t.Errorf("group[1].Name = %q", groups[1].Name)
	}
	if len(groups[1].Files) != 2 {
		t.Errorf("group[1].Files = %v", groups[1].Files)
	}
	if !strings.Contains(groups[1].Diff, "ui/button.go") || !strings.Contains(groups[1].Diff, "api/handler.go") {
		t.Errorf("group[1].Diff mismatch:\n%s", groups[1].Diff)
	}
}

func TestGroupDiffsByFileGroups_SkipsUnknownAndDuplicates(t *testing.T) {
	diff := `diff --git a/a.go b/a.go
index abc..def 100644
--- a/a.go
+++ b/a.go
@@ -1 +1 @@
-old
+new`

	change := domain.Change{Diff: diff}
	groups := GroupDiffsByFileGroups(change, []FileGroup{
		{Name: "g", Files: []string{"a.go", "a.go", "missing.go"}},
	})

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Files) != 1 || groups[0].Files[0] != "a.go" {
		t.Errorf("Files = %v, want [a.go]", groups[0].Files)
	}
	// Empty partition yields no groups.
	if got := GroupDiffsByFileGroups(change, nil); len(got) != 0 {
		t.Errorf("expected 0 groups for nil partition, got %d", len(got))
	}
}

func TestFilesInDiff(t *testing.T) {
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

	files := FilesInDiff(domain.Change{Diff: diff})
	if len(files) != 2 || files[0] != "api/handler.go" || files[1] != "ui/button.go" {
		t.Errorf("FilesInDiff = %v, want [api/handler.go ui/button.go]", files)
	}

	if got := FilesInDiff(domain.Change{Diff: ""}); len(got) != 0 {
		t.Errorf("expected 0 files for empty diff, got %d", len(got))
	}
}
