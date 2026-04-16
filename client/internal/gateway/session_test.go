package gateway

import (
	"encoding/json"
	"testing"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
)

func TestSessionFrames(t *testing.T) {
	var sent [][]byte
	now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)

	session := NewSession("machine-01", "Workstation Alpha", func(msg []byte) error {
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

func TestSessionSnapshotFramesUseEnvelopeShape(t *testing.T) {
	var sent [][]byte
	now := time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)

	session := NewSession("machine-01", "Workstation Alpha", func(msg []byte) error {
		copyMsg := append([]byte(nil), msg...)
		sent = append(sent, copyMsg)
		return nil
	}, func() time.Time {
		return now
	})

	if err := session.MachineSnapshot(domain.Machine{
		ID:     "machine-01",
		Name:   "Dev Mac",
		Status: domain.MachineStatusOnline,
	}); err != nil {
		t.Fatal(err)
	}
	if err := session.ThreadSnapshot([]domain.Thread{
		{ThreadID: "thread-01", MachineID: "machine-01", Status: domain.ThreadStatusIdle, Title: "One"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := session.EnvironmentSnapshot([]domain.EnvironmentResource{
		{
			ResourceID:      "skill-01",
			MachineID:       "machine-01",
			Kind:            domain.EnvironmentKindSkill,
			DisplayName:     "Skill A",
			Status:          domain.EnvironmentResourceStatusEnabled,
			RestartRequired: false,
			LastObservedAt:  "2026-04-08T11:00:00Z",
		},
	}); err != nil {
		t.Fatal(err)
	}

	if len(sent) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(sent))
	}

	assertSnapshotFrame(t, sent[0], "machine.snapshot", now, []byte(`{"machine":{"id":"machine-01","name":"Dev Mac","status":"online"}}`))
	assertSnapshotFrame(t, sent[1], "thread.snapshot", now, []byte(`{"threads":[{"threadId":"thread-01","machineId":"machine-01","status":"idle","title":"One"}]}`))
	assertSnapshotFrame(t, sent[2], "environment.snapshot", now, []byte(`{"environment":[{"resourceId":"skill-01","machineId":"machine-01","kind":"skill","displayName":"Skill A","status":"enabled","restartRequired":false,"lastObservedAt":"2026-04-08T11:00:00Z"}]}`))
}

func TestSessionCommandRejectedFrameUsesEnvelopeShape(t *testing.T) {
	var sent [][]byte
	now := time.Date(2026, 4, 8, 11, 5, 0, 0, time.UTC)

	session := NewSession("machine-01", "Workstation Alpha", func(msg []byte) error {
		copyMsg := append([]byte(nil), msg...)
		sent = append(sent, copyMsg)
		return nil
	}, func() time.Time {
		return now
	})

	if err := session.CommandRejected("req-1", "turn.start", "thread not found", "thread-01"); err != nil {
		t.Fatal(err)
	}

	if len(sent) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(sent))
	}

	assertEventFrame(t, sent[0], "command.rejected", "req-1", now, []byte(`{"commandName":"turn.start","reason":"thread not found","threadId":"thread-01"}`))
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
	if expectedName == "client.register" {
		if payload["name"] != "Workstation Alpha" {
			t.Fatalf("expected register payload to include machine name, got %v", payload)
		}
		return
	}
	if len(payload) != 0 {
		t.Fatalf("expected empty payload, got %v", payload)
	}
}

func assertSnapshotFrame(t *testing.T, raw []byte, expectedName string, now time.Time, expectedPayload []byte) {
	t.Helper()

	assertEnvelopeShape(t, raw, protocol.CategorySnapshot, expectedName, "", now, expectedPayload)
}

func assertEventFrame(t *testing.T, raw []byte, expectedName string, expectedRequestID string, now time.Time, expectedPayload []byte) {
	t.Helper()

	assertEnvelopeShape(t, raw, protocol.CategoryEvent, expectedName, expectedRequestID, now, expectedPayload)
}

func assertEnvelopeShape(t *testing.T, raw []byte, expectedCategory protocol.Category, expectedName string, expectedRequestID string, now time.Time, expectedPayload []byte) {
	t.Helper()

	var envelope protocol.Envelope
	if err := transport.Decode(raw, &envelope); err != nil {
		t.Fatalf("decode frame failed: %v", err)
	}

	if envelope.Version != version.CurrentProtocolVersion {
		t.Fatalf("expected version %q, got %q", version.CurrentProtocolVersion, envelope.Version)
	}
	if envelope.Category != expectedCategory {
		t.Fatalf("expected category %q, got %q", expectedCategory, envelope.Category)
	}
	if envelope.Name != expectedName {
		t.Fatalf("expected name %q, got %q", expectedName, envelope.Name)
	}
	if envelope.RequestID != expectedRequestID {
		t.Fatalf("expected requestId %q, got %q", expectedRequestID, envelope.RequestID)
	}
	if envelope.MachineID != "machine-01" {
		t.Fatalf("expected machineId %q, got %q", "machine-01", envelope.MachineID)
	}
	if envelope.Timestamp != now.Format(time.RFC3339) {
		t.Fatalf("expected timestamp %q, got %q", now.Format(time.RFC3339), envelope.Timestamp)
	}

	var gotPayload any
	if err := json.Unmarshal(envelope.Payload, &gotPayload); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}

	var wantPayload any
	if err := json.Unmarshal(expectedPayload, &wantPayload); err != nil {
		t.Fatalf("decode expected payload failed: %v", err)
	}

	if !payloadEqual(gotPayload, wantPayload) {
		t.Fatalf("expected payload %s, got %s", string(expectedPayload), string(envelope.Payload))
	}
}

func payloadEqual(a any, b any) bool {
	encodedA, err := json.Marshal(a)
	if err != nil {
		return false
	}
	encodedB, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(encodedA) == string(encodedB)
}
