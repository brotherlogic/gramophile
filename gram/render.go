package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/mattn/go-isatty"
)

// AnsiClearLine is the ANSI escape sequence to move the cursor to the beginning of the line
// and clear to the end of the line.
const AnsiClearLine = "\r\033[K"

// FormatLoadingState returns the loading message, optionally incorporating a retry failure count.
func FormatLoadingState(failureCount ...int) string {
	if len(failureCount) > 0 && failureCount[0] > 0 {
		return FormatFailureState(failureCount[0])
	}
	return "Loading..."
}

// FormatFailureState formats the dynamic retry failure indicator string.
func FormatFailureState(attempt int) string {
	return fmt.Sprintf("Loading [Cache Failure #%d]...", attempt)
}

// FormatUnresolved formats a fallback string for an unresolved record ID.
func FormatUnresolved(id int64) string {
	return fmt.Sprintf("[Unresolved: %d]", id)
}

// FormatResolved formats an artist and title string.
func FormatResolved(artist, title string) string {
	return fmt.Sprintf("%s - %s", artist, title)
}

// IsTerminal checks if the given writer is an interactive TTY terminal.
func IsTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return false
}

// TerminalRenderer provides terminal output formatting with support for
// interactive in-place spinner animations in TTY sessions and clean output in non-interactive/piped environments.
type TerminalRenderer struct {
	out      io.Writer
	isTTY    bool
	frames   []string
	interval time.Duration
	mu       sync.Mutex
}

// RendererOption configures a TerminalRenderer.
type RendererOption func(*TerminalRenderer)

// WithTTY overrides the automatic TTY detection.
func WithTTY(isTTY bool) RendererOption {
	return func(r *TerminalRenderer) {
		r.isTTY = isTTY
	}
}

// WithFrames sets custom animation frames for the spinner.
func WithFrames(frames []string) RendererOption {
	return func(r *TerminalRenderer) {
		if len(frames) > 0 {
			r.frames = frames
		}
	}
}

// WithInterval sets the update interval for the spinner animation.
func WithInterval(d time.Duration) RendererOption {
	return func(r *TerminalRenderer) {
		if d > 0 {
			r.interval = d
		}
	}
}

// NewTerminalRenderer initializes a new TerminalRenderer with options.
func NewTerminalRenderer(w io.Writer, opts ...RendererOption) *TerminalRenderer {
	if w == nil {
		w = os.Stdout
	}
	r := &TerminalRenderer{
		out:      w,
		isTTY:    IsTerminal(w),
		frames:   spinner.MiniDot.Frames,
		interval: 100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewRenderer is a convenience constructor taking an optional isTTY boolean.
func NewRenderer(w io.Writer, isTTY ...bool) *TerminalRenderer {
	if len(isTTY) > 0 {
		return NewTerminalRenderer(w, WithTTY(isTTY[0]))
	}
	return NewTerminalRenderer(w)
}

// IsTTY reports whether the renderer is operating in interactive TTY mode.
func (r *TerminalRenderer) IsTTY() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isTTY
}

// SetTTY enables or disables interactive TTY mode.
func (r *TerminalRenderer) SetTTY(isTTY bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isTTY = isTTY
}

// ResolutionSession coordinates an active in-place resolution line.
type ResolutionSession struct {
	renderer *TerminalRenderer
	prefix   string
	status   string
	done     chan struct{}
	stopped  chan struct{}
	mu       sync.Mutex
	closed   bool
}

// StartResolution begins an interactive resolution line.
// In TTY mode, it immediately displays the spinner and status and animates glyphs in the background.
// In non-interactive mode, it suppresses all output until Finish is invoked.
func (r *TerminalRenderer) StartResolution(prefix string) *ResolutionSession {
	session := &ResolutionSession{
		renderer: r,
		prefix:   prefix,
		status:   FormatLoadingState(),
		done:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}

	if !r.isTTY {
		close(session.stopped)
		return session
	}

	// In interactive mode, render initial frame
	r.mu.Lock()
	frame := r.frames[0]
	fmt.Fprintf(r.out, "%s%s%s %s", AnsiClearLine, prefix, frame, session.status)
	r.mu.Unlock()

	go func() {
		defer close(session.stopped)
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		frameIdx := 1

		for {
			select {
			case <-session.done:
				return
			case <-ticker.C:
				session.mu.Lock()
				currentStatus := session.status
				session.mu.Unlock()

				r.mu.Lock()
				f := r.frames[frameIdx%len(r.frames)]
				frameIdx++
				fmt.Fprintf(r.out, "%s%s%s %s", AnsiClearLine, prefix, f, currentStatus)
				r.mu.Unlock()
			}
		}
	}()

	return session
}

// Start is an alias for StartResolution.
func (r *TerminalRenderer) Start(prefix string) *ResolutionSession {
	return r.StartResolution(prefix)
}

// SetFailureCount updates the status message to indicate a retry attempt.
func (s *ResolutionSession) SetFailureCount(n int) {
	s.mu.Lock()
	s.status = FormatFailureState(n)
	currentStatus := s.status
	s.mu.Unlock()

	if s.renderer.isTTY {
		s.renderer.mu.Lock()
		f := s.renderer.frames[0]
		fmt.Fprintf(s.renderer.out, "%s%s%s %s", AnsiClearLine, s.prefix, f, currentStatus)
		s.renderer.mu.Unlock()
	}
}

// SetRetry is an alias for SetFailureCount.
func (s *ResolutionSession) SetRetry(n int) {
	s.SetFailureCount(n)
}

// UpdateStatus updates the resolution status text.
func (s *ResolutionSession) UpdateStatus(status string) {
	s.mu.Lock()
	s.status = status
	currentStatus := s.status
	s.mu.Unlock()

	if s.renderer.isTTY {
		s.renderer.mu.Lock()
		f := s.renderer.frames[0]
		fmt.Fprintf(s.renderer.out, "%s%s%s %s", AnsiClearLine, s.prefix, f, currentStatus)
		s.renderer.mu.Unlock()
	}
}

// Finish finalizes the resolution line in-place.
// In TTY mode, it clears the spinner line and outputs prefix + finalText with a newline.
// In non-interactive mode, it outputs prefix + finalText with a newline without any ANSI escapes.
func (s *ResolutionSession) Finish(finalText string) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.done)
	s.mu.Unlock()

	if s.renderer.isTTY {
		<-s.stopped
		s.renderer.mu.Lock()
		fmt.Fprintf(s.renderer.out, "%s%s%s\n", AnsiClearLine, s.prefix, finalText)
		s.renderer.mu.Unlock()
	} else {
		s.renderer.mu.Lock()
		fmt.Fprintf(s.renderer.out, "%s%s\n", s.prefix, finalText)
		s.renderer.mu.Unlock()
	}
}

// Cancel terminates the spinner without finalizing a result.
// In TTY mode, it clears the current line.
func (s *ResolutionSession) Cancel() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.done)
	s.mu.Unlock()

	if s.renderer.isTTY {
		<-s.stopped
		s.renderer.mu.Lock()
		fmt.Fprintf(s.renderer.out, "%s", AnsiClearLine)
		s.renderer.mu.Unlock()
	}
}

// Resolve wraps an asynchronous or blocking resolution function with terminal spinner handling.
func (r *TerminalRenderer) Resolve(ctx context.Context, prefix string, resolveFn func(ctx context.Context, notifyRetry func(int)) (string, error)) (string, error) {
	session := r.StartResolution(prefix)
	result, err := resolveFn(ctx, func(attempt int) {
		session.SetFailureCount(attempt)
	})
	if err != nil {
		session.Cancel()
		return "", err
	}
	session.Finish(result)
	return result, nil
}
