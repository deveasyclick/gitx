package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func askProceed() bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Ready to release? [y/N] ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		}
	}
}

func askGeneratePR() bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Generate PR description? [y/N] ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		}
	}
}

func askVersion(suggested, current string) string {
	reader := bufio.NewReader(os.Stdin)
	for {
		currentDisplay := current
		if currentDisplay == "" {
			currentDisplay = "(none)"
		}
		fmt.Printf("\nCurrent version: %s\n", currentDisplay)
		fmt.Printf("Suggested version: %s\n", suggested)
		fmt.Print("Enter version or press Enter to accept: v")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			return suggested
		}
		// Ensure v prefix
		if !strings.HasPrefix(input, "v") {
			input = "v" + input
		}
		return input
	}
}

func askReleaseBranch(version, currentBranch string) (bool, string) {
	releaseBranch := fmt.Sprintf("release/%s", version)
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Printf("Release branch options:\n")
	fmt.Printf("  1) Use current branch (%s)\n", currentBranch)
	fmt.Printf("  2) Create release branch (%s)\n", releaseBranch)
	fmt.Print("Choose [1/2] (default: 1): ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "2":
		return true, releaseBranch
	default:
		return false, currentBranch
	}
}

func askCheckoutBranch(branch string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\nCheckout into release branch '%s'? [y/N] ", branch)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// retryOrSkip is the outcome of askRetryOrSkip.
type retryOrSkip int

const (
	retryAction retryOrSkip = iota // try the step again
	skipAction                     // skip the step and continue
	quitAction                     // abort the release
)

func askRetryOrSkip(msg string) retryOrSkip {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [R] Retry  [S] Skip  [Q] Quit\n(default: Quit): ", msg)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "r", "retry":
			return retryAction
		case "s", "skip":
			return skipAction
		case "q", "quit", "":
			return quitAction
		}
	}
}
