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
		enc.Encode(Result{
			Command: "commit",
			Title:   msg.Title,
			Body:    msg.Body,
		})
	default:
		fmt.Fprintln(Stdout)
		fmt.Fprintln(Stdout, "Generated commit:")
		fmt.Fprintln(Stdout)
		fmt.Fprintln(Stdout, msg.Title)
		if msg.Body != "" {
			fmt.Fprintln(Stdout)
			fmt.Fprintln(Stdout, msg.Body)
		}
		fmt.Fprintln(Stdout)
	}
}

// PrintInfo prints an informational message.
func PrintInfo(msg string) {
	fmt.Fprintln(Stdout, msg)
}

// PrintSuccess prints a success message.
func PrintSuccess(msg string) {
	fmt.Fprintln(Stdout, msg)
}

// PrintWarning prints a warning message.
func PrintWarning(msg string) {
	fmt.Fprintln(Stderr, msg)
}

// PrintError prints an error message.
func PrintError(msg string) {
	fmt.Fprintln(Stderr, "Error:", msg)
}

// PrintVerbose prints a message only in verbose mode.
func PrintVerbose(msg string, level OutputLevel) {
	if level >= OutputVerbose {
		fmt.Fprintln(Stderr, msg)
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
