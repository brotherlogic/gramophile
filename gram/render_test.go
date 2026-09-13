package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func (s *safeBuffer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
}

func TestFailureStateFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"Failure 1", 1, "Loading [Cache Failure #1]..."},
		{"Failure 2", 2, "Loading [Cache Failure #2]..."},
		{"Failure 10", 10, "Loading [Cache Failure #10]..."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatFailureState(tc.input)
			if got != tc.expected {
				t.Fatalf("expected FormatFailureState(%d) = %q, got %q", tc.input, tc.expected, got)
			}
		})
	}

	if got := FormatLoadingState(); got != "Loading..." {
		t.Fatalf("expected FormatLoadingState() = \"Loading...\", got %q", got)
	}

	if got := FormatLoadingState(0); got != "Loading..." {
		t.Fatalf("expected FormatLoadingState(0) = \"Loading...\", got %q", got)
	}

	if got := FormatLoadingState(3); got != "Loading [Cache Failure #3]..." {
		t.Fatalf("expected FormatLoadingState(3) = \"Loading [Cache Failure #3]...\", got %q", got)
	}

	if got := FormatUnresolved(12345); got != "[Unresolved: 12345]" {
		t.Fatalf("expected FormatUnresolved(12345) = \"[Unresolved: 12345]\", got %q", got)
	}
}

func TestInteractiveTTYRendering(t *testing.T) {
	buf := &safeBuffer{}
	renderer := NewTerminalRenderer(
		buf,
		WithTTY(true),
		WithInterval(10*time.Millisecond),
	)

	if !renderer.IsTTY() {
		t.Fatalf("expected renderer.IsTTY() to be true")
	}

	prefix := "0. [Shelf-1] "
	session := renderer.StartResolution(prefix)

	// Allow some spinner frames to be rendered
	time.Sleep(25 * time.Millisecond)

	// Transition to retry failure state
	session.SetFailureCount(1)
	time.Sleep(25 * time.Millisecond)

	finalRecord := "The Beatles - Revolver"
	session.Finish(finalRecord)

	out := buf.String()

	// Verify ANSI escape sequences were used
	if !strings.Contains(out, AnsiClearLine) {
		t.Fatalf("expected interactive output to contain ANSI clear line sequence %q, got: %q", AnsiClearLine, out)
	}

	// Verify failure indicator was rendered
	if !strings.Contains(out, "Loading [Cache Failure #1]...") {
		t.Fatalf("expected interactive output to contain retry indicator \"Loading [Cache Failure #1]...\", got: %q", out)
	}

	// Verify finalized line ends cleanly with newline
	expectedFinalLine := prefix + finalRecord + "\n"
	if !strings.HasSuffix(out, expectedFinalLine) {
		t.Fatalf("expected interactive output to finalize with %q, got: %q", expectedFinalLine, out)
	}
}

func TestInteractiveTTYUnresolvedFallback(t *testing.T) {
	buf := &safeBuffer{}
	renderer := NewTerminalRenderer(
		buf,
		WithTTY(true),
		WithInterval(10*time.Millisecond),
	)

	prefix := "1. [Shelf-1] "
	session := renderer.StartResolution(prefix)
	time.Sleep(15 * time.Millisecond)

	fallback := FormatUnresolved(98765)
	session.Finish(fallback)

	out := buf.String()
	expectedFinalLine := prefix + "[Unresolved: 98765]\n"
	if !strings.HasSuffix(out, expectedFinalLine) {
		t.Fatalf("expected interactive output to finalize with %q, got: %q", expectedFinalLine, out)
	}
}

func TestNonInteractiveFallback(t *testing.T) {
	buf := &safeBuffer{}
	renderer := NewTerminalRenderer(
		buf,
		WithTTY(false),
		WithInterval(10*time.Millisecond),
	)

	if renderer.IsTTY() {
		t.Fatalf("expected renderer.IsTTY() to be false")
	}

	prefix := "2. [Shelf-A] "
	session := renderer.StartResolution(prefix)

	// Verify no output was emitted during active resolution
	if current := buf.String(); current != "" {
		t.Fatalf("expected empty buffer during non-interactive resolution, got %q", current)
	}

	// Retries should also not emit ANSI codes or text
	session.SetFailureCount(1)
	session.SetFailureCount(2)
	time.Sleep(30 * time.Millisecond)

	if current := buf.String(); current != "" {
		t.Fatalf("expected empty buffer after non-interactive retries, got %q", current)
	}

	finalRecord := "Miles Davis - Kind of Blue"
	session.Finish(finalRecord)

	out := buf.String()
	expected := prefix + finalRecord + "\n"

	// Output must be exactly the clean line with no ANSI escapes
	if out != expected {
		t.Fatalf("expected exact non-interactive output %q, got %q", expected, out)
	}

	// Double check no escape sequences or carriage returns exist
	if strings.Contains(out, "\033") || strings.Contains(out, "\r") {
		t.Fatalf("non-interactive output must not contain ANSI escape codes or carriage returns, got %q", out)
	}
}

func TestIsTerminalDetection(t *testing.T) {
	buf := &bytes.Buffer{}
	if IsTerminal(buf) {
		t.Fatalf("bytes.Buffer should not be recognized as a TTY terminal")
	}

	// Testing with os.Stdout should execute without panics
	_ = IsTerminal(os.Stdout)
}

func TestResolveWrapperHelper(t *testing.T) {
	buf := &safeBuffer{}
	renderer := NewTerminalRenderer(
		buf,
		WithTTY(true),
		WithInterval(10*time.Millisecond),
	)

	ctx := context.Background()
	prefix := "3. [Box-1] "
	res, err := renderer.Resolve(ctx, prefix, func(ctx context.Context, notifyRetry func(int)) (string, error) {
		notifyRetry(1)
		time.Sleep(20 * time.Millisecond)
		return "Pink Floyd - Animals", nil
	})

	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %v", err)
	}

	if res != "Pink Floyd - Animals" {
		t.Fatalf("unexpected Resolve return value %q", res)
	}

	out := buf.String()
	expectedEnd := prefix + "Pink Floyd - Animals\n"
	if !strings.HasSuffix(out, expectedEnd) {
		t.Fatalf("expected output ending with %q, got %q", expectedEnd, out)
	}
}

func TestInteractiveCancellation(t *testing.T) {
	buf := &safeBuffer{}
	renderer := NewTerminalRenderer(
		buf,
		WithTTY(true),
		WithInterval(10*time.Millisecond),
	)

	session := renderer.StartResolution("4. [Shelf-C] ")
	time.Sleep(25 * time.Millisecond)
	session.Cancel()

	// Cancellation should clear line and not print newline
	out := buf.String()
	if !strings.HasSuffix(out, AnsiClearLine) {
		t.Fatalf("expected cancelled output to end with clear line sequence, got %q", out)
	}
}

func TestIdempotentFinish(t *testing.T) {
	buf := &safeBuffer{}
	renderer := NewTerminalRenderer(
		buf,
		WithTTY(false),
	)

	session := renderer.StartResolution("5. [Shelf-D] ")
	session.Finish("Record Title")
	// Second finish call should be a no-op
	session.Finish("Record Title Second")
	session.Cancel()

	out := buf.String()
	expected := "5. [Shelf-D] Record Title\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}
