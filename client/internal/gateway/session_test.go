package gateway

import (
	"encoding/json"
	"testing"
	"time"

	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
)

func TestSessionFrames(t *testing.T) {
	var sent [][]byte
	now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)

	session := NewSession("machine-01", func(msg []byte) error {
		copyMsg := append([]byte(nil), msg...)
		sent = append(sent, copyMsg)
		return nil
	}, func() time.Time {
		return now
	})

	if err := session.Register(); err != nil {
		t.Fatal(err)
	}
	if err := session.Heartbeat(); err != nil {
		t.Fatal(err)
	}

	if len(sent) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(sent))
	}

	assertFrame(t, sent[0], "client.register", now)
	assertFrame(t, sent[1], "client.heartbeat", now)
}

func assertFrame(t *testing.T, raw []byte, expectedName string, now time.Time) {
	t.Helper()

	var envelope protocol.Envelope
	if err := transport.Decode(raw, &envelope); err != nil {
		t.Fatalf("decode frame failed: %v", err)
	}

	if envelope.Version != version.CurrentProtocolVersion {
		t.Fatalf("expected version %q, got %q", version.CurrentProtocolVersion, envelope.Version)
	}
	if envelope.Category != protocol.CategorySystem {
		t.Fatalf("expected category %q, got %q", protocol.CategorySystem, envelope.Category)
	}
	if envelope.Name != expectedName {
		t.Fatalf("expected name %q, got %q", expectedName, envelope.Name)
	}
	if envelope.MachineID != "machine-01" {
		t.Fatalf("expected machineId %q, got %q", "machine-01", envelope.MachineID)
	}
	if envelope.Timestamp != now.Format(time.RFC3339) {
		t.Fatalf("expected timestamp %q, got %q", now.Format(time.RFC3339), envelope.Timestamp)
	}

	var payload map[string]any
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}
	if len(payload) != 0 {
		t.Fatalf("expected empty payload, got %v", payload)
	}
}
