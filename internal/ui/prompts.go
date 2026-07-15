package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ConfirmOption represents a choice in a confirmation prompt.
type ConfirmOption int

const (
	ConfirmYes ConfirmOption = iota
	ConfirmYesAll
	ConfirmNo
	ConfirmEdit
	ConfirmRegenerate
	ConfirmCopy
	ConfirmQuit
	ConfirmStage
)

// ConfirmCommit asks the user what to do with the generated commit message.
// Returns the selected option. Default is No (safe default).
func ConfirmCommit() ConfirmOption {
	reader := bufio.NewReader(os.Stdin)

	for attempts := 3; attempts > 0; attempts-- {
		fmt.Print("\nCommit this change?\n[Y] Yes   [N] No   [E] Edit   [R] Regenerate   [C] Copy\n(default: No): ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y", "yes":
			return ConfirmYes
		case "e", "edit":
			return ConfirmEdit
		case "r", "regenerate":
			return ConfirmRegenerate
		case "c", "copy":
			return ConfirmCopy
		case "n", "no", "":
			return ConfirmNo
		default:
			if attempts > 1 {
				fmt.Printf("Invalid option %q. ", input)
			}
		}
	}

	return ConfirmNo
}

// ConfirmCommitGrouped asks the user what to do with a grouped commit message.
// No skips this group, Quit stops the entire grouped flow, All commits all remaining.
func ConfirmCommitGrouped() ConfirmOption {
	reader := bufio.NewReader(os.Stdin)

	for attempts := 3; attempts > 0; attempts-- {
		fmt.Print("\nCommit this group?\n[Y] Yes   [A] All   [N] Skip   [E] Edit   [R] Regenerate   [C] Copy   [S] Stage   [Q] Quit\n(default: Skip): ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y", "yes":
			return ConfirmYes
		case "a", "all":
			return ConfirmYesAll
		case "s", "stage":
			return ConfirmStage
		case "e", "edit":
			return ConfirmEdit
		case "r", "regenerate":
			return ConfirmRegenerate
		case "c", "copy":
			return ConfirmCopy
		case "q", "quit":
			return ConfirmQuit
		case "n", "no", "":
			return ConfirmNo
		default:
			if attempts > 1 {
				fmt.Printf("Invalid option %q. ", input)
			}
		}
	}

	return ConfirmNo
}

// CopyToClipboard copies text to the system clipboard.
// Tries pbcopy (macOS), wl-copy (Wayland), xsel (Linux), then xclip (X11).
func CopyToClipboard(text string) error {
	for _, cmd := range []struct {
		name string
		args []string
	}{
		{"pbcopy", nil},
		{"wl-copy", nil},
		{"xsel", []string{"-b"}},
		{"xclip", []string{"-selection", "clipboard"}},
	} {
		if _, err := exec.LookPath(cmd.name); err == nil {
			execCmd := exec.Command(cmd.name, cmd.args...)
			execCmd.Stdin = strings.NewReader(text)
			return execCmd.Run()
		}
	}
	return fmt.Errorf("no clipboard tool found (install pbcopy, wl-copy, xsel, or xclip)")
}

// OpenEditor opens the system editor with the given initial content.
// Returns the edited content, or an error.
func OpenEditor(initialContent string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}

	tmpFile, err := os.CreateTemp("", "gitx-commit-*.txt")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(initialContent); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %s: %w", editor, err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("read temp file: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}
