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
	session := newClientSession("machine-01", func(msg []byte) error {
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

	err := handleCommandEnvelope(session, nil, "codex", false, protocol.Envelope{
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

	err := handleCommandEnvelope(session, mgr, "codex", false, protocol.Envelope{
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

func TestHandleCommandEnvelopeAsyncRuntimeKeepsTurnStartAsAckOnly(t *testing.T) {
	runtime := &notifyingRuntime{
		startTurnResult: agenttypes.StartTurnResult{
			TurnID:   "turn-async-1",
			ThreadID: "thread-01",
		},
	}
	registry := agentregistry.New()
	registry.Register("codex", runtime)
	mgr := manager.New(registry)
	session, sent := newRecordingSession()

	if !bindRuntimeTurnEvents(runtime, session) {
		t.Fatal("expected runtime turn event binding")
	}

	err := handleCommandEnvelope(session, mgr, "codex", true, protocol.Envelope{
		Name:      "turn.start",
		RequestID: "req-async-1",
		Payload:   []byte(`{"threadId":"thread-01","input":"run tests"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(*sent) != 1 {
		t.Fatalf("expected command ack only, got %d frames", len(*sent))
	}

	if decodeEnvelope(t, (*sent)[0]).Name != "command.completed" {
		t.Fatalf("unexpected ack envelope: %+v", decodeEnvelope(t, (*sent)[0]))
	}

	runtime.emit(agenttypes.RuntimeTurnEvent{
		Type:     agenttypes.RuntimeTurnEventTypeStarted,
		ThreadID: "thread-01",
		TurnID:   "turn-async-1",
	})
	runtime.emit(agenttypes.RuntimeTurnEvent{
		Type:     agenttypes.RuntimeTurnEventTypeDelta,
		ThreadID: "thread-01",
		TurnID:   "turn-async-1",
		Sequence: 1,
		Delta:    "hello",
	})
	runtime.emit(agenttypes.RuntimeTurnEvent{
		Type: agenttypes.RuntimeTurnEventTypeCompleted,
		Turn: domain.Turn{
			TurnID:   "turn-async-1",
			ThreadID: "thread-01",
			Status:   domain.TurnStatusCompleted,
		},
	})

	if len(*sent) != 4 {
		t.Fatalf("expected ack plus 3 runtime events, got %d frames", len(*sent))
	}
	if decodeEnvelope(t, (*sent)[1]).Name != "turn.started" {
		t.Fatalf("unexpected started envelope: %+v", decodeEnvelope(t, (*sent)[1]))
	}
	if decodeEnvelope(t, (*sent)[2]).Name != "turn.delta" {
		t.Fatalf("unexpected delta envelope: %+v", decodeEnvelope(t, (*sent)[2]))
	}
	if decodeEnvelope(t, (*sent)[3]).Name != "turn.completed" {
		t.Fatalf("unexpected completed envelope: %+v", decodeEnvelope(t, (*sent)[3]))
	}
}

func TestBindRuntimeApprovalEventsEmitsApprovalRequired(t *testing.T) {
	runtime := &notifyingRuntime{}
	session, sent := newRecordingSession()

	if !bindRuntimeApprovalEvents(runtime, session) {
		t.Fatal("expected runtime approval event binding")
	}

	runtime.emitApproval(agenttypes.RuntimeApprovalRequest{
		RequestID: "approval-1",
		ThreadID:  "thread-01",
		TurnID:    "turn-01",
		ItemID:    "item-01",
		Kind:      "command",
		Command:   "go test ./...",
	})

	if len(*sent) != 1 {
		t.Fatalf("expected 1 approval event, got %d", len(*sent))
	}

	envelope := decodeEnvelope(t, (*sent)[0])
	if envelope.Name != "approval.required" || envelope.RequestID != "approval-1" {
		t.Fatalf("unexpected approval envelope: %+v", envelope)
	}

	var payload protocol.ApprovalRequiredPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}
	if payload.RequestID != "approval-1" || payload.Command != "go test ./..." || payload.Kind != "command" {
		t.Fatalf("unexpected approval payload: %+v", payload)
	}
}

func TestHandleCommandEnvelopeRespondsToApprovalRequests(t *testing.T) {
	runtime := &notifyingRuntime{}
	registry := agentregistry.New()
	registry.Register("codex", runtime)
	mgr := manager.New(registry)
	session, sent := newRecordingSession()

	err := handleCommandEnvelope(session, mgr, "codex", false, protocol.Envelope{
		Name:      "approval.respond",
		RequestID: "req-approval-1",
		Payload:   []byte(`{"requestId":"approval-1","decision":"accept"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if runtime.lastApprovalDecision.requestID != "approval-1" || runtime.lastApprovalDecision.decision != "accept" {
		t.Fatalf("unexpected approval response: %+v", runtime.lastApprovalDecision)
	}

	if len(*sent) != 2 {
		t.Fatalf("expected command ack and approval event, got %d frames", len(*sent))
	}
	if decodeEnvelope(t, (*sent)[0]).Name != "command.completed" {
		t.Fatalf("unexpected ack envelope: %+v", decodeEnvelope(t, (*sent)[0]))
	}
	if decodeEnvelope(t, (*sent)[1]).Name != "approval.resolved" {
		t.Fatalf("unexpected resolved envelope: %+v", decodeEnvelope(t, (*sent)[1]))
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

func newRecordingSession() (*clientSession, *[][]byte) {
	sent := make([][]byte, 0, 1)
	session := newClientSession("machine-01", func(msg []byte) error {
		sent = append(sent, append([]byte(nil), msg...))
		return nil
	}, func() time.Time {
		return time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	})

	return session, &sent
}

func decodeEnvelope(t *testing.T, raw []byte) protocol.Envelope {
	t.Helper()

	var envelope protocol.Envelope
	if err := transport.Decode(raw, &envelope); err != nil {
		t.Fatalf("decode envelope failed: %v", err)
	}

	return envelope
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

type notifyingRuntime struct {
	startTurnResult      agenttypes.StartTurnResult
	handler              func(agenttypes.RuntimeTurnEvent)
	approvalHandler      func(agenttypes.RuntimeApprovalRequest)
	lastApprovalDecision struct {
		requestID string
		decision  string
	}
}

func (r *notifyingRuntime) ListThreads() ([]domain.Thread, error) {
	return nil, nil
}

func (r *notifyingRuntime) ListEnvironment() ([]domain.EnvironmentResource, error) {
	return nil, nil
}

func (r *notifyingRuntime) CreateThread(agenttypes.CreateThreadParams) (domain.Thread, error) {
	return domain.Thread{}, nil
}

func (r *notifyingRuntime) ReadThread(string) (domain.Thread, error) {
	return domain.Thread{}, nil
}

func (r *notifyingRuntime) ResumeThread(string) (domain.Thread, error) {
	return domain.Thread{}, nil
}

func (r *notifyingRuntime) ArchiveThread(string) error {
	return nil
}

func (r *notifyingRuntime) StartTurn(agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	return r.startTurnResult, nil
}

func (r *notifyingRuntime) SteerTurn(agenttypes.SteerTurnParams) (agenttypes.SteerTurnResult, error) {
	return agenttypes.SteerTurnResult{}, nil
}

func (r *notifyingRuntime) InterruptTurn(agenttypes.InterruptTurnParams) (domain.Turn, error) {
	return domain.Turn{}, nil
}

func (r *notifyingRuntime) SetTurnEventHandler(handler func(agenttypes.RuntimeTurnEvent)) {
	r.handler = handler
}

func (r *notifyingRuntime) SetApprovalHandler(handler func(agenttypes.RuntimeApprovalRequest)) {
	r.approvalHandler = handler
}

func (r *notifyingRuntime) RespondApproval(requestID string, decision string) error {
	r.lastApprovalDecision.requestID = requestID
	r.lastApprovalDecision.decision = decision
	return nil
}

func (r *notifyingRuntime) emit(event agenttypes.RuntimeTurnEvent) {
	if r.handler != nil {
		r.handler(event)
	}
}

func (r *notifyingRuntime) emitApproval(event agenttypes.RuntimeApprovalRequest) {
	if r.approvalHandler != nil {
		r.approvalHandler(event)
	}
}
