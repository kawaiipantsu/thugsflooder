// Package audit provides a mandatory, append-only, JSON-lines record of
// every send attempt thugsflooder makes. This is what keeps the tool
// self-documenting for an authorized exercise rather than anti-forensic:
// every packet it puts on the wire is permanently accounted for locally.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Result classifies the outcome of one send attempt.
type Result string

const (
	ResultSent    Result = "sent"
	ResultBlocked Result = "blocked" // rejected by allowlist check
	ResultDropped Result = "dropped" // send/connect error
)

// Entry is one audit record.
type Entry struct {
	Time     time.Time `json:"time"`
	Proto    string    `json:"proto"`
	Host     string    `json:"host"`
	Port     int       `json:"port"`
	Bytes    int       `json:"bytes"`
	Result   Result    `json:"result"`
	Detail   string    `json:"detail,omitempty"`
}

// Logger writes audit entries to an append-only file, safe for concurrent use.
type Logger struct {
	mu   sync.Mutex
	f    *os.File
	enc  *json.Encoder
	path string
}

// Open opens (creating if needed) the audit log at path for appending.
// It fails closed: if the file cannot be opened for writing, Open errors
// and the caller must not start any generators.
func Open(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating audit log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening audit log %q: %w", path, err)
	}
	return &Logger{f: f, enc: json.NewEncoder(f), path: path}, nil
}

// Record writes one entry, filling in the timestamp.
func (l *Logger) Record(e Entry) {
	e.Time = time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.enc.Encode(e)
}

// Path returns the audit log's file path.
func (l *Logger) Path() string { return l.path }

// Close flushes and closes the audit log.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.f.Close()
}
