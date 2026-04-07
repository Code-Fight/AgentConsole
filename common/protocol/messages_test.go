package protocol

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	msg := Envelope{
		Version:   "v1",
		Category:  CategoryCommand,
		Name:      "thread.create",
		RequestID: "req_123",
		MachineID: "machine_01",
		Timestamp: "2026-04-07T10:00:00Z",
		Payload:   json.RawMessage(`{"title":"Investigate flaky test"}`),
	}

	blob, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Envelope
	if err := json.Unmarshal(blob, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Name != "thread.create" {
		t.Fatalf("expected thread.create, got %q", decoded.Name)
	}
}
