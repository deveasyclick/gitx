package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/user/gitx/internal/domain"
)

// OutputLevel controls how much detail is shown.
type OutputLevel int

const (
	OutputNormal OutputLevel = iota
	OutputVerbose
	OutputJSON
)

// Overridable output writers (swapped in tests).
var Stdout io.Writer = os.Stdout
var Stderr io.Writer = os.Stderr

// printLine writes a line to w. Write errors are ignored: display output is
// best-effort and there is nothing sensible a caller can do with the error.
func printLine(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}

// Result is a structured result for JSON output.
type Result struct {
	Command string `json:"command"`
	Title   string `json:"title,omitempty"`
	Body    string `json:"body,omitempty"`
}

// PrintCommitMessage displays a generated commit message.
func PrintCommitMessage(msg domain.CommitMessage, level OutputLevel) {
	switch level {
	case OutputJSON:
		enc := json.NewEncoder(Stdout)
		_ = enc.Encode(Result{
			Command: "commit",
			Title:   msg.Title,
			Body:    msg.Body,
		})
	default:
		printLine(Stdout)
		printLine(Stdout, "Generated commit:")
		printLine(Stdout)
		printLine(Stdout, msg.Title)
		if msg.Body != "" {
			printLine(Stdout)
			printLine(Stdout, msg.Body)
		}
		printLine(Stdout)
	}
}

// PrintInfo prints an informational message.
func PrintInfo(msg string) {
	printLine(Stdout, msg)
}

// PrintSuccess prints a success message.
func PrintSuccess(msg string) {
	printLine(Stdout, msg)
}

// PrintWarning prints a warning message.
func PrintWarning(msg string) {
	printLine(Stderr, msg)
}

// PrintError prints an error message.
func PrintError(msg string) {
	printLine(Stderr, "Error:", msg)
}

// PrintVerbose prints a message only in verbose mode.
func PrintVerbose(msg string, level OutputLevel) {
	if level >= OutputVerbose {
		printLine(Stderr, msg)
	}
}

// FormatDiffStat formats a diff stat for display.
func FormatDiffStat(stat string) string {
	if stat == "" {
		return ""
	}
	return "  " + stat
}

// FormatCommitBullets formats a list of changes as bullet points.
func FormatCommitBullets(items []string) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	return b.String()
}
