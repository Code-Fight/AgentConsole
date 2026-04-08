package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"code-agent-gateway/client/internal/agent/codex"
	"code-agent-gateway/client/internal/agent/manager"
	agentregistry "code-agent-gateway/client/internal/agent/registry"
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/client/internal/config"
	"code-agent-gateway/client/internal/gateway"
	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
)

func TestNextBackoffStartsAtOneSecond(t *testing.T) {
	got := nextBackoff(0, 5*time.Second)
	if got != 1*time.Second {
		t.Fatalf("expected initial backoff to be 1s, got %s", got)
	}
}

func TestNextBackoffCapsAtMax(t *testing.T) {
	got := nextBackoff(4*time.Second, 5*time.Second)
	if got != 5*time.Second {
		t.Fatalf("expected capped backoff of 5s, got %s", got)
	}
}

func TestSendLiveSnapshotRebuildsRuntimeStateOnEveryCall(t *testing.T) {
	adapter := codex.NewFakeAdapter()
	adapter.SeedSnapshot(
		[]domain.Thread{
			{ThreadID: "thread-01", MachineID: "machine-01", Status: domain.ThreadStatusIdle, Title: "One"},
		},
		[]domain.EnvironmentResource{
			{
				ResourceID:      "skill-01",
				MachineID:       "machine-01",
				Kind:            domain.EnvironmentKindSkill,
				DisplayName:     "Skill A",
				Status:          domain.EnvironmentResourceStatusEnabled,
				LastObservedAt:  "2026-04-08T10:00:00Z",
				RestartRequired: false,
			},
		},
	)

	registry := agentregistry.New()
	registry.Register("codex", adapter)
	mgr := manager.New(registry)

	var sent [][]byte
	session := gateway.NewSession("machine-01", func(msg []byte) error {
		sent = append(sent, append([]byte(nil), msg...))
		return nil
	}, func() time.Time {
		return time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	})

	machine := domain.Machine{ID: "machine-01", Name: "machine-01", Status: domain.MachineStatusOnline}

	if err := sendLiveSnapshot(session, machine, mgr, "codex"); err != nil {
		t.Fatal(err)
	}

	if _, err := adapter.CreateThread(agenttypes.CreateThreadParams{Title: "Two"}); err != nil {
		t.Fatal(err)
	}

	if err := sendLiveSnapshot(session, machine, mgr, "codex"); err != nil {
		t.Fatal(err)
	}

	if len(sent) != 6 {
		t.Fatalf("expected 6 frames, got %d", len(sent))
	}

	threads := decodeThreadSnapshotPayload(t, sent[4])
	if len(threads.Threads) != 2 {
		t.Fatalf("expected refreshed snapshot to contain 2 threads, got %d", len(threads.Threads))
	}
}

func TestHandleCommandEnvelopeRejectsUnsupportedCommands(t *testing.T) {
	session, sent := newRecordingSession()

	err := handleCommandEnvelope(session, nil, "codex", protocol.Envelope{
		Name:      "unknown.command",
		RequestID: "req-1",
		Payload:   []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	rejection := decodeRejectedPayload(t, (*sent)[0])
	if rejection.CommandName != "unknown.command" || rejection.Reason != "unsupported command" {
		t.Fatalf("unexpected rejection: %+v", rejection)
	}
}

func TestHandleCommandEnvelopeRejectsFailedTurnStartWithoutDisconnecting(t *testing.T) {
	adapter := codex.NewFakeAdapter()
	registry := agentregistry.New()
	registry.Register("codex", adapter)
	mgr := manager.New(registry)
	session, sent := newRecordingSession()

	err := handleCommandEnvelope(session, mgr, "codex", protocol.Envelope{
		Name:      "turn.start",
		RequestID: "req-2",
		Payload:   []byte(`{"threadId":"thread-99","input":"run tests"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	rejection := decodeRejectedPayload(t, (*sent)[0])
	if rejection.CommandName != "turn.start" {
		t.Fatalf("commandName = %q", rejection.CommandName)
	}
	if rejection.ThreadID != "thread-99" {
		t.Fatalf("threadId = %q", rejection.ThreadID)
	}
	if rejection.Reason == "" {
		t.Fatal("expected rejection reason")
	}
}

func TestBuildRuntimeUsesFakeOnlyWhenConfigured(t *testing.T) {
	cfg := config.Config{MachineID: "machine-01", RuntimeMode: config.RuntimeModeFake}
	calledFake := false
	calledAppServer := false

	runtime, cleanup, err := buildRuntime(context.Background(), cfg, time.Now, runtimeFactories{
		newFake: func(config.Config, func() time.Time) agenttypes.Runtime {
			calledFake = true
			return codex.NewFakeAdapter()
		},
		newAppServer: func(context.Context, config.Config) (agenttypes.Runtime, func() error, error) {
			calledAppServer = true
			return codex.NewFakeAdapter(), func() error { return nil }, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if runtime == nil {
		t.Fatal("expected runtime")
	}
	if cleanup == nil {
		t.Fatal("expected cleanup")
	}
	if !calledFake || calledAppServer {
		t.Fatalf("unexpected selection fake=%v appserver=%v", calledFake, calledAppServer)
	}
}

func TestBuildRuntimeUsesAppServerByDefault(t *testing.T) {
	cfg := config.Config{MachineID: "machine-01", RuntimeMode: config.RuntimeModeAppServer}
	calledFake := false
	calledAppServer := false

	runtime, cleanup, err := buildRuntime(context.Background(), cfg, time.Now, runtimeFactories{
		newFake: func(config.Config, func() time.Time) agenttypes.Runtime {
			calledFake = true
			return codex.NewFakeAdapter()
		},
		newAppServer: func(context.Context, config.Config) (agenttypes.Runtime, func() error, error) {
			calledAppServer = true
			return codex.NewFakeAdapter(), func() error { return nil }, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if runtime == nil {
		t.Fatal("expected runtime")
	}
	if cleanup == nil {
		t.Fatal("expected cleanup")
	}
	if calledFake || !calledAppServer {
		t.Fatalf("unexpected selection fake=%v appserver=%v", calledFake, calledAppServer)
	}
}

func newRecordingSession() (*gateway.Session, *[][]byte) {
	sent := make([][]byte, 0, 1)
	session := gateway.NewSession("machine-01", func(msg []byte) error {
		sent = append(sent, append([]byte(nil), msg...))
		return nil
	}, func() time.Time {
		return time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	})

	return session, &sent
}

func decodeThreadSnapshotPayload(t *testing.T, raw []byte) protocol.ThreadSnapshotPayload {
	t.Helper()

	var envelope protocol.Envelope
	if err := transport.Decode(raw, &envelope); err != nil {
		t.Fatalf("decode envelope failed: %v", err)
	}

	var payload protocol.ThreadSnapshotPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}

	return payload
}

func decodeRejectedPayload(t *testing.T, raw []byte) protocol.CommandRejectedPayload {
	t.Helper()

	var envelope protocol.Envelope
	if err := transport.Decode(raw, &envelope); err != nil {
		t.Fatalf("decode envelope failed: %v", err)
	}

	var payload protocol.CommandRejectedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}

	return payload
}
