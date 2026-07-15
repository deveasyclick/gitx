package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/user/gitx/internal/domain"
	"github.com/user/gitx/internal/ui"
)

func TestPrintCommitMessage_Normal(t *testing.T) {
	var buf bytes.Buffer
	ui.Stdout = &buf

	msg := domain.CommitMessage{
		Title: "feat(auth): add login",
		Body:  "- Added OAuth\n- Added refresh",
		Style: "conventional",
	}

	ui.PrintCommitMessage(msg, ui.OutputNormal)

	output := buf.String()
	if !strings.Contains(output, "feat(auth): add login") {
		t.Errorf("output should contain title")
	}
	if !strings.Contains(output, "Added OAuth") {
		t.Errorf("output should contain body")
	}
}

func TestPrintCommitMessage_JSON(t *testing.T) {
	var buf bytes.Buffer
	ui.Stdout = &buf

	msg := domain.CommitMessage{
		Title: "fix: resolve panic",
		Style: "conventional",
	}

	ui.PrintCommitMessage(msg, ui.OutputJSON)

	output := buf.String()
	if !strings.Contains(output, `"title":"fix: resolve panic"`) {
		t.Errorf("output should contain JSON title: %s", output)
	}
}

func TestFormatDiffStat(t *testing.T) {
	got := ui.FormatDiffStat("10 files changed, 200 insertions(+)")
	if got != "  10 files changed, 200 insertions(+)" {
		t.Errorf("got = %q", got)
	}
}

func TestFormatDiffStatEmpty(t *testing.T) {
	got := ui.FormatDiffStat("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFormatCommitBullets(t *testing.T) {
	got := ui.FormatCommitBullets([]string{"add login", "fix timeout"})
	expected := "- add login\n- fix timeout\n"
	if got != expected {
		t.Errorf("got = %q, want %q", got, expected)
	}
}

func TestFormatCommitBulletsEmpty(t *testing.T) {
	got := ui.FormatCommitBullets(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
