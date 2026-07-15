package ui

import (
	"fmt"
	"os"
	"time"
)

// Spinner displays an animated spinner with a message.
type Spinner struct {
	msg    string
	done   chan bool
	frames []string
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(msg string) *Spinner {
	return &Spinner{
		msg:  msg,
		done: make(chan bool, 1),
		frames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
	}
}

// Start begins the spinner animation in a goroutine.
func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				// Clear the line
				fmt.Fprint(Stderr, "\r\033[K")
				return
			default:
				fmt.Fprintf(Stderr, "\r%s %s", s.frames[i%len(s.frames)], s.msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop halts the spinner and clears its line.
func (s *Spinner) Stop() {
	s.done <- true
}

// StopWithMessage halts the spinner and prints a completion message.
func (s *Spinner) StopWithMessage(doneMsg string) {
	s.done <- true
	// Small delay to ensure the last frame is cleared
	time.Sleep(10 * time.Millisecond)
	fmt.Fprintf(Stderr, "\r\033[K%s\n", doneMsg)
}

// IsInteractive returns true if stderr is a terminal (spinner support).
func IsInteractive() bool {
	stat, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
