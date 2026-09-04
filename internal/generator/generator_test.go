package generator

import (
	"context"
	"encoding/base64"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/thugsred/thugsflooder/internal/audit"
	"github.com/thugsred/thugsflooder/internal/config"
	"github.com/thugsred/thugsflooder/internal/recording"
	"github.com/thugsred/thugsflooder/internal/stats"
)

// TestReplayBlocksOutOfScopeTarget is a regression test for the
// defense-in-depth allowlist check: a recording entry addressed to a
// host that isn't in the config's allowlist must be blocked and must
// never be sent, even though the recording file itself contains it.
func TestReplayBlocksOutOfScopeTarget(t *testing.T) {
	// A real listener for the in-scope entry so "sent" is genuinely observed.
	udpConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer udpConn.Close()
	port := udpConn.LocalAddr().(*net.UDPAddr).Port

	go func() {
		buf := make([]byte, 1024)
		for {
			_, _, err := udpConn.ReadFrom(buf)
			if err != nil {
				return
			}
		}
	}()

	tmpDir := t.TempDir()
	auditPath := filepath.Join(tmpDir, "audit.jsonl")
	cfgPath := filepath.Join(tmpDir, "targets.yaml")
	cfgYAML := "targets:\n" +
		"  - host: 127.0.0.1\n" +
		"    ports: [" + strconv.Itoa(port) + "]\n" +
		"    protocols: [udp]\n" +
		"rate_limit:\n" +
		"  max_pps: 200\n" +
		"  workers_per_target: 1\n" +
		"audit_log: " + auditPath + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, allow, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	auditLogger, err := audit.Open(auditPath)
	if err != nil {
		t.Fatalf("audit.Open: %v", err)
	}
	defer auditLogger.Close()

	st := stats.New()
	sinks := &Sinks{Stats: st, Audit: auditLogger, Log: func(string, ...any) {}}

	inScopePayload := base64.StdEncoding.EncodeToString([]byte("in-scope"))
	outOfScopePayload := base64.StdEncoding.EncodeToString([]byte("out-of-scope"))
	entries := []recording.Entry{
		{Proto: "udp", Host: "127.0.0.1", Port: port, PayloadB64: inScopePayload, DelayMS: 5},
		{Proto: "udp", Host: "203.0.113.1", Port: 9999, PayloadB64: outOfScopePayload, DelayMS: 5},
	}

	mgr := NewManager(cfg, allow, sinks).
		WithReplay(entries, 20) // sped up so the short test deadline covers several cycles

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	mgr.Run(ctx)

	snap := st.Snapshot()
	if snap.Sent == 0 {
		t.Error("expected at least one in-scope send, got 0")
	}
	if snap.Blocked == 0 {
		t.Error("expected the out-of-scope entry to be blocked at least once, got 0")
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("reading audit log: %v", err)
	}
	if !strings.Contains(string(data), "203.0.113.1") {
		t.Error("expected audit log to mention the blocked out-of-scope host")
	}
	if !strings.Contains(string(data), `"result":"blocked"`) {
		t.Error("expected audit log to contain a blocked result")
	}
}
