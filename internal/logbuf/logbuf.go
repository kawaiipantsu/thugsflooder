// Package logbuf is a small thread-safe ring buffer of recent log lines,
// shared between the generator workers (writers) and the TUI/headless
// output (readers).
package logbuf

import (
	"fmt"
	"sync"
	"time"
)

// Line is one timestamped log entry.
type Line struct {
	Time time.Time
	Text string
}

// Buffer holds up to max recent lines.
type Buffer struct {
	mu    sync.Mutex
	lines []Line
	max   int
}

// New creates a Buffer capped at max lines.
func New(max int) *Buffer {
	return &Buffer{max: max}
}

// Logf formats and appends one line, evicting the oldest if over capacity.
// Matches the generator.Sinks.Log signature.
func (b *Buffer) Logf(format string, args ...any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, Line{Time: time.Now(), Text: fmt.Sprintf(format, args...)})
	if len(b.lines) > b.max {
		b.lines = b.lines[len(b.lines)-b.max:]
	}
}

// Snapshot returns a copy of all current lines, oldest first.
func (b *Buffer) Snapshot() []Line {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Line, len(b.lines))
	copy(out, b.lines)
	return out
}
