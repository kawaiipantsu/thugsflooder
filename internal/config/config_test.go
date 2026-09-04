package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "targets.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadRejectsEmptyTargets(t *testing.T) {
	path := writeTemp(t, `
targets: []
rate_limit:
  max_pps: 100
audit_log: /tmp/thugsflooder-test-audit.jsonl
`)
	if _, _, err := Load(path); err == nil {
		t.Fatal("expected error for empty allowlist, got nil")
	}
}

func TestLoadRejectsNullAuditSink(t *testing.T) {
	path := writeTemp(t, `
targets:
  - host: 127.0.0.1
    ports: [80]
    protocols: [tcp]
rate_limit:
  max_pps: 100
audit_log: /dev/null
`)
	if _, _, err := Load(path); err == nil {
		t.Fatal("expected error for /dev/null audit_log, got nil")
	}
}

func TestLoadRejectsUnknownProtocol(t *testing.T) {
	path := writeTemp(t, `
targets:
  - host: 127.0.0.1
    ports: [80]
    protocols: [carrier-pigeon]
rate_limit:
  max_pps: 100
audit_log: /tmp/thugsflooder-test-audit.jsonl
`)
	if _, _, err := Load(path); err == nil {
		t.Fatal("expected error for unknown protocol, got nil")
	}
}

func TestLoadValidConfigAndAllowlist(t *testing.T) {
	path := writeTemp(t, `
targets:
  - host: 127.0.0.1
    ports: [80, 443]
    protocols: [tcp, http]
rate_limit:
  max_pps: 100
audit_log: /tmp/thugsflooder-test-audit.jsonl
`)
	cfg, allow, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RateLimit.WorkersPerTarget != DefaultWorkersPerTarget {
		t.Errorf("expected default workers_per_target=%d, got %d", DefaultWorkersPerTarget, cfg.RateLimit.WorkersPerTarget)
	}

	if !allow.Allowed("127.0.0.1", 80, ProtoTCP) {
		t.Error("expected 127.0.0.1:80 tcp to be allowed")
	}
	if !allow.Allowed("127.0.0.1", 443, ProtoHTTP) {
		t.Error("expected 127.0.0.1:443 http to be allowed")
	}
	if allow.Allowed("127.0.0.1", 80, ProtoUDP) {
		t.Error("127.0.0.1:80 udp was not configured and must not be allowed")
	}
	if allow.Allowed("10.0.0.9", 80, ProtoTCP) {
		t.Error("host not in config must not be allowed")
	}
}
