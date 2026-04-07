package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"code-agent-gateway/common/transport"
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

	if decoded.Version != msg.Version {
		t.Fatalf("expected version %q, got %q", msg.Version, decoded.Version)
	}

	if decoded.Category != msg.Category {
		t.Fatalf("expected category %q, got %q", msg.Category, decoded.Category)
	}

	if decoded.RequestID != msg.RequestID {
		t.Fatalf("expected requestId %q, got %q", msg.RequestID, decoded.RequestID)
	}

	if decoded.MachineID != msg.MachineID {
		t.Fatalf("expected machineId %q, got %q", msg.MachineID, decoded.MachineID)
	}

	if decoded.Timestamp != msg.Timestamp {
		t.Fatalf("expected timestamp %q, got %q", msg.Timestamp, decoded.Timestamp)
	}

	if string(decoded.Payload) != string(msg.Payload) {
		t.Fatalf("expected payload %q, got %q", string(msg.Payload), string(decoded.Payload))
	}
}

func TestCommandEnvelopeRequiresRequestID(t *testing.T) {
	msg := Envelope{
		Version:   "v1",
		Category:  CategoryCommand,
		Name:      "thread.create",
		MachineID: "machine_01",
		Timestamp: "2026-04-07T10:00:00Z",
		Payload:   json.RawMessage(`{"title":"Investigate flaky test"}`),
	}

	_, err := transport.Encode(msg)
	if err == nil {
		t.Fatal("expected error when command envelope has empty requestId")
	}

	if !strings.Contains(err.Error(), "requestId") {
		t.Fatalf("expected requestId validation error, got %q", err.Error())
	}
}

func TestCommandEnvelopeMarshalRequiresRequestID(t *testing.T) {
	msg := Envelope{
		Version:   "v1",
		Category:  CategoryCommand,
		Name:      "thread.create",
		MachineID: "machine_01",
		Timestamp: "2026-04-07T10:00:00Z",
		Payload:   json.RawMessage(`{"title":"Investigate flaky test"}`),
	}

	_, err := json.Marshal(msg)
	if err == nil {
		t.Fatal("expected marshal to fail when command envelope has empty requestId")
	}

	if !strings.Contains(err.Error(), "requestId") {
		t.Fatalf("expected requestId validation error, got %q", err.Error())
	}
}
