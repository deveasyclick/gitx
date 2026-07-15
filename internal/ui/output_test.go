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

func TestPrintInfo(t *testing.T) {
	var buf bytes.Buffer
	ui.Stdout = &buf

	ui.PrintInfo("hello world")

	output := buf.String()
	if !strings.Contains(output, "hello world") {
		t.Errorf("output = %q, want to contain hello world", output)
	}
}

func TestPrintSuccess(t *testing.T) {
	var buf bytes.Buffer
	ui.Stdout = &buf

	ui.PrintSuccess("success!")

	output := buf.String()
	if !strings.Contains(output, "success!") {
		t.Errorf("output = %q, want to contain success!", output)
	}
}

func TestPrintWarning(t *testing.T) {
	var buf bytes.Buffer
	ui.Stderr = &buf

	ui.PrintWarning("warning message")

	output := buf.String()
	if !strings.Contains(output, "warning message") {
		t.Errorf("output = %q, want to contain warning", output)
	}
}

func TestPrintError(t *testing.T) {
	var buf bytes.Buffer
	ui.Stderr = &buf

	ui.PrintError("error message")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("output = %q, want to contain error", output)
	}
}

func TestPrintCommitMessage_Verbose(t *testing.T) {
	var buf bytes.Buffer
	ui.Stdout = &buf

	msg := domain.CommitMessage{
		Title: "feat: verbose test",
		Body:  "some body",
	}

	ui.PrintCommitMessage(msg, ui.OutputVerbose)

	output := buf.String()
	if !strings.Contains(output, "feat: verbose test") {
		t.Errorf("verbose output should contain title: %s", output)
	}
	if !strings.Contains(output, "some body") {
		t.Errorf("verbose output should contain body: %s", output)
	}
}

func TestPrintCommitMessage_TitleOnly(t *testing.T) {
	var buf bytes.Buffer
	ui.Stdout = &buf

	msg := domain.CommitMessage{
		Title: "fix: just a title",
		Style: "conventional",
	}

	ui.PrintCommitMessage(msg, ui.OutputNormal)

	output := buf.String()
	if !strings.Contains(output, "fix: just a title") {
		t.Errorf("output should contain title: %s", output)
	}
	if strings.Contains(output, "\n\n\n") && strings.Count(output, "\n") > 5 {
		t.Errorf("unexpected blank lines for title-only: %q", output)
	}
}

func TestPrintVerbose_Shown(t *testing.T) {
	var buf bytes.Buffer
	ui.Stderr = &buf

	ui.PrintVerbose("verbose detail", ui.OutputVerbose)

	output := buf.String()
	if !strings.Contains(output, "verbose detail") {
		t.Errorf("verbose message should be printed: %s", output)
	}
}

func TestPrintVerbose_Hidden(t *testing.T) {
	var buf bytes.Buffer
	ui.Stderr = &buf

	ui.PrintVerbose("should not appear", ui.OutputNormal)

	output := buf.String()
	if output != "" {
		t.Errorf("expected empty output in normal mode, got %q", output)
	}
}
