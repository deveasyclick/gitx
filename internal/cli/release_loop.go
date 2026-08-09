package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/user/gitx/internal/ui"
)

// reviewLoopOptions configures reviewLoop.
type reviewLoopOptions struct {
	prompt    string
	copiedMsg string
	editedMsg string
	// regenerate re-generates the content. nil disables the Regenerate option.
	regenerate func() (string, error)
}

// reviewLoop lets the user Accept, Regenerate (optional), Copy, or Edit
// AI-generated text. It returns the final accepted text.
func reviewLoop(opts reviewLoopOptions, current string) string {
	reader := bufio.NewReader(os.Stdin)
	text := current

	for {
		fmt.Print(opts.prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "accept", "":
			return text
		case "r", "regenerate":
			if opts.regenerate == nil {
				fmt.Printf("Invalid option %q.\n", input)
				continue
			}
			newText, err := opts.regenerate()
			if err != nil {
				ui.PrintError(fmt.Sprintf("Regeneration failed: %s", err.Error()))
				continue
			}
			text = newText
			fmt.Println()
			fmt.Println(text)
			fmt.Println()
		case "c", "copy":
			if err := ui.CopyToClipboard(text); err != nil {
				ui.PrintWarning(fmt.Sprintf("Copy failed: %s", err.Error()))
			} else {
				ui.PrintSuccess(opts.copiedMsg)
			}
		case "e", "edit":
			edited, err := ui.OpenEditor(text)
			if err != nil {
				ui.PrintWarning(fmt.Sprintf("Edit failed: %s", err.Error()))
				continue
			}
			if edited != "" && edited != text {
				text = edited
				ui.PrintSuccess(opts.editedMsg)
			}
		default:
			fmt.Printf("Invalid option %q.\n", input)
		}
	}
}
