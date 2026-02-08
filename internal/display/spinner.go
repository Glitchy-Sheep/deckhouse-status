package display

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Spinner renders a live progress line with elapsed time to stderr.
// It runs an async render loop so the animation stays smooth between API polls.
type Spinner struct {
	frames  []string
	started time.Time

	mu      sync.Mutex
	frame   int
	message string
	done    bool
	stopCh  chan struct{}

	// ANSI codes
	reset, bold, dim, cyan, green, red string
}

// NewSpinner creates a spinner respecting color/emoji preferences.
// It starts the background render loop immediately.
func NewSpinner(noColor, noEmoji bool) *Spinner {
	s := &Spinner{
		started: time.Now(),
		stopCh:  make(chan struct{}),
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

	go s.loop()
	return s
}

func (s *Spinner) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.render()
		}
	}
}

func (s *Spinner) render() {
	s.mu.Lock()
	if s.done || s.message == "" {
		s.mu.Unlock()
		return
	}
	elapsed := time.Since(s.started).Truncate(time.Second)
	frame := s.frames[s.frame%len(s.frames)]
	msg := s.message
	s.frame++
	s.mu.Unlock()

	fmt.Fprintf(os.Stderr, "\r\033[K%s%s%s %s %s[%s]%s",
		s.cyan, frame, s.reset,
		msg,
		s.dim, elapsed, s.reset,
	)
}

// Tick updates the spinner message. The animation continues asynchronously.
func (s *Spinner) Tick(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// ClearLine clears the current spinner line.
func (s *Spinner) ClearLine() {
	s.mu.Lock()
	s.done = true
	s.mu.Unlock()

	select {
	case s.stopCh <- struct{}{}:
	default:
	}

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
