package display

import (
	"fmt"
	"os"
	"time"
)

// Spinner renders a live progress line with elapsed time to stderr.
type Spinner struct {
	frames  []string
	started time.Time
	frame   int

	// ANSI codes
	reset, bold, dim, cyan, green, red string
}

// NewSpinner creates a spinner respecting color/emoji preferences.
func NewSpinner(noColor, noEmoji bool) *Spinner {
	s := &Spinner{
		started: time.Now(),
	}

	if noEmoji {
		s.frames = []string{"|", "/", "-", "\\"}
	} else {
		s.frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	}

	if !noColor {
		s.reset = "\033[0m"
		s.bold = "\033[1m"
		s.dim = "\033[2m"
		s.cyan = "\033[36m"
		s.green = "\033[32m"
		s.red = "\033[31m"
	}

	return s
}

// Tick renders one spinner frame with a message and elapsed time.
func (s *Spinner) Tick(message string) {
	elapsed := time.Since(s.started).Truncate(time.Second)
	frame := s.frames[s.frame%len(s.frames)]
	s.frame++
	fmt.Fprintf(os.Stderr, "\r\033[K%s%s%s %s %s[%s]%s",
		s.cyan, frame, s.reset,
		message,
		s.dim, elapsed, s.reset,
	)
}

// ClearLine clears the current spinner line.
func (s *Spinner) ClearLine() {
	fmt.Fprint(os.Stderr, "\r\033[K")
}

// Success prints a final success line with terminal bell.
func (s *Spinner) Success(message string) {
	s.ClearLine()
	elapsed := time.Since(s.started).Truncate(time.Second)
	fmt.Fprintf(os.Stderr, "%s\u2705 %s%s %s[%s]%s\a\n",
		s.green, message, s.reset,
		s.dim, elapsed, s.reset,
	)
}

// Failure prints a final failure line with terminal bell.
func (s *Spinner) Failure(message string) {
	s.ClearLine()
	elapsed := time.Since(s.started).Truncate(time.Second)
	fmt.Fprintf(os.Stderr, "%s\u274c %s%s %s[%s]%s\a\n",
		s.red, message, s.reset,
		s.dim, elapsed, s.reset,
	)
}
