// Package recording implements thugsflooder's simple "simulated legitimate
// traffic" replay format: JSON lines, no pcap/libpcap/cgo dependency, so
// cross-compilation to 386/amd64/arm/arm64 stays simple.
package recording

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Entry is one replayable send: a protocol, destination, payload, and the
// delay to wait before sending it (relative to the previous entry).
type Entry struct {
	Proto      string `json:"proto"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	PayloadB64 string `json:"payload_b64"`
	DelayMS    int    `json:"delay_ms"`
}

// Payload decodes the entry's base64 payload.
func (e Entry) Payload() ([]byte, error) {
	return base64.StdEncoding.DecodeString(e.PayloadB64)
}

// Delay returns the entry's inter-packet delay as a Duration.
func (e Entry) Delay() time.Duration {
	return time.Duration(e.DelayMS) * time.Millisecond
}

// Load reads every entry from a JSON-lines recording file.
func Load(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening recording %q: %w", path, err)
	}
	defer f.Close()

	var entries []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("recording %q line %d: %w", path, lineNo, err)
		}
		if _, err := e.Payload(); err != nil {
			return nil, fmt.Errorf("recording %q line %d: invalid payload_b64: %w", path, lineNo, err)
		}
		entries = append(entries, e)
	}
	if err := sc.Err(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading recording %q: %w", path, err)
	}
	return entries, nil
}
