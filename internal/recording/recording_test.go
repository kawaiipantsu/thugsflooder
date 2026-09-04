package recording

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadValidEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rec.jsonl")
	payload := base64.StdEncoding.EncodeToString([]byte("hello"))
	contents := `{"proto":"udp","host":"127.0.0.1","port":9901,"payload_b64":"` + payload + `","delay_ms":150}
{"proto":"tcp","host":"127.0.0.1","port":9902,"payload_b64":"` + payload + `","delay_ms":0}
`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Delay() != 150*time.Millisecond {
		t.Errorf("Delay() = %v, want 150ms", entries[0].Delay())
	}
	got, err := entries[0].Payload()
	if err != nil || string(got) != "hello" {
		t.Errorf("Payload() = %q, %v, want %q, nil", got, err, "hello")
	}
}

func TestLoadRejectsBadPayload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rec.jsonl")
	contents := `{"proto":"udp","host":"127.0.0.1","port":9901,"payload_b64":"not-valid-base64!!","delay_ms":0}` + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for invalid payload_b64, got nil")
	}
}
