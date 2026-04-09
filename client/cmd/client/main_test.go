package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	err := handleCommandEnvelope(session, nil, "codex", false, nil, protocol.Envelope{
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

	err := handleCommandEnvelope(session, mgr, "codex", false, nil, protocol.Envelope{
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

	if !bindRuntimeTurnEvents(runtime, session, mgr, "codex") {
		t.Fatal("expected runtime turn event binding")
	}

	err := handleCommandEnvelope(session, mgr, "codex", true, nil, protocol.Envelope{
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

	if len(*sent) != 6 {
		t.Fatalf("expected ack, 3 runtime events, and 2 thread snapshots, got %d frames", len(*sent))
	}
	if decodeEnvelope(t, (*sent)[1]).Name != "turn.started" {
		t.Fatalf("unexpected started envelope: %+v", decodeEnvelope(t, (*sent)[1]))
	}
	if decodeEnvelope(t, (*sent)[2]).Name != "thread.snapshot" {
		t.Fatalf("unexpected first snapshot envelope: %+v", decodeEnvelope(t, (*sent)[2]))
	}
	if decodeEnvelope(t, (*sent)[3]).Name != "turn.delta" {
		t.Fatalf("unexpected delta envelope: %+v", decodeEnvelope(t, (*sent)[3]))
	}
	if decodeEnvelope(t, (*sent)[4]).Name != "turn.completed" {
		t.Fatalf("unexpected completed envelope: %+v", decodeEnvelope(t, (*sent)[4]))
	}
	if decodeEnvelope(t, (*sent)[5]).Name != "thread.snapshot" {
		t.Fatalf("unexpected second snapshot envelope: %+v", decodeEnvelope(t, (*sent)[5]))
	}
}

func TestBindRuntimeTurnEventsRefreshesThreadSnapshotOnStartedAndCompleted(t *testing.T) {
	runtime := &notifyingRuntime{
		startTurnResult: agenttypes.StartTurnResult{
			TurnID:   "turn-async-1",
			ThreadID: "thread-01",
		},
		threadSnapshots: [][]domain.Thread{
			{
				{ThreadID: "thread-01", MachineID: "machine-01", Status: domain.ThreadStatusActive, Title: "One"},
			},
			{
				{ThreadID: "thread-01", MachineID: "machine-01", Status: domain.ThreadStatusIdle, Title: "One"},
			},
		},
	}
	registry := agentregistry.New()
	registry.Register("codex", runtime)
	mgr := manager.New(registry)
	session, sent := newRecordingSession()

	if !bindRuntimeTurnEvents(runtime, session, mgr, "codex") {
		t.Fatal("expected runtime turn event binding")
	}

	runtime.emit(agenttypes.RuntimeTurnEvent{
		Type:     agenttypes.RuntimeTurnEventTypeStarted,
		ThreadID: "thread-01",
		TurnID:   "turn-async-1",
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
		t.Fatalf("expected turn events plus 2 thread snapshots, got %d frames", len(*sent))
	}
	if decodeEnvelope(t, (*sent)[0]).Name != "turn.started" {
		t.Fatalf("unexpected first envelope: %+v", decodeEnvelope(t, (*sent)[0]))
	}
	if decodeEnvelope(t, (*sent)[1]).Name != "thread.snapshot" {
		t.Fatalf("unexpected second envelope: %+v", decodeEnvelope(t, (*sent)[1]))
	}
	if decodeEnvelope(t, (*sent)[2]).Name != "turn.completed" {
		t.Fatalf("unexpected third envelope: %+v", decodeEnvelope(t, (*sent)[2]))
	}
	if decodeEnvelope(t, (*sent)[3]).Name != "thread.snapshot" {
		t.Fatalf("unexpected fourth envelope: %+v", decodeEnvelope(t, (*sent)[3]))
	}

	firstSnapshot := decodeThreadSnapshotPayload(t, (*sent)[1])
	if len(firstSnapshot.Threads) != 1 || firstSnapshot.Threads[0].Status != domain.ThreadStatusActive {
		t.Fatalf("expected active thread snapshot after turn start, got %+v", firstSnapshot.Threads)
	}

	secondSnapshot := decodeThreadSnapshotPayload(t, (*sent)[3])
	if len(secondSnapshot.Threads) != 1 || secondSnapshot.Threads[0].Status != domain.ThreadStatusIdle {
		t.Fatalf("expected idle thread snapshot after turn completion, got %+v", secondSnapshot.Threads)
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

	if err := session.ApprovalRequired(protocol.ApprovalRequiredPayload{
		RequestID: "approval-1",
		ThreadID:  "thread-01",
		TurnID:    "turn-01",
		ItemID:    "item-01",
		Kind:      "command",
		Command:   "go test ./...",
	}); err != nil {
		t.Fatal(err)
	}

	err := handleCommandEnvelope(session, mgr, "codex", false, nil, protocol.Envelope{
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

	if len(*sent) != 3 {
		t.Fatalf("expected approval required, command ack, and approval event, got %d frames", len(*sent))
	}
	if decodeEnvelope(t, (*sent)[1]).Name != "command.completed" {
		t.Fatalf("unexpected ack envelope: %+v", decodeEnvelope(t, (*sent)[1]))
	}
	if decodeEnvelope(t, (*sent)[2]).Name != "approval.resolved" {
		t.Fatalf("unexpected resolved envelope: %+v", decodeEnvelope(t, (*sent)[2]))
	}

	var resolved protocol.ApprovalResolvedPayload
	if err := json.Unmarshal(decodeEnvelope(t, (*sent)[2]).Payload, &resolved); err != nil {
		t.Fatalf("decode resolved payload failed: %v", err)
	}
	if resolved.RequestID != "approval-1" || resolved.ThreadID != "thread-01" || resolved.Decision != "accept" {
		t.Fatalf("unexpected resolved payload: %+v", resolved)
	}
}

func TestHandleCommandEnvelopeApprovalResolvedPreservesThreadContextAfterReconnect(t *testing.T) {
	runtime := &notifyingRuntime{}
	registry := agentregistry.New()
	registry.Register("codex", runtime)
	mgr := manager.New(registry)
	session, sent := newRecordingSession()

	err := handleCommandEnvelope(session, mgr, "codex", false, nil, protocol.Envelope{
		Name:      "approval.respond",
		RequestID: "req-approval-1",
		Payload:   []byte(`{"requestId":"approval-1","threadId":"thread-01","turnId":"turn-01","decision":"accept"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(*sent) != 2 {
		t.Fatalf("expected command ack and resolved event, got %d frames", len(*sent))
	}

	var resolved protocol.ApprovalResolvedPayload
	if err := json.Unmarshal(decodeEnvelope(t, (*sent)[1]).Payload, &resolved); err != nil {
		t.Fatalf("decode resolved payload failed: %v", err)
	}
	if resolved.RequestID != "approval-1" || resolved.ThreadID != "thread-01" || resolved.TurnID != "turn-01" || resolved.Decision != "accept" {
		t.Fatalf("unexpected resolved payload: %+v", resolved)
	}
}

func TestHandleCommandEnvelopeRuntimeStartStopChangesAvailability(t *testing.T) {
	initialRuntime := &notifyingRuntime{
		threadSnapshots: [][]domain.Thread{
			{
				{ThreadID: "thread-01", MachineID: "machine-01", Status: domain.ThreadStatusIdle, Title: "Initial"},
			},
		},
		environment: []domain.EnvironmentResource{
			{
				ResourceID:     "skill-01",
				MachineID:      "machine-01",
				Kind:           domain.EnvironmentKindSkill,
				DisplayName:    "Skill A",
				Status:         domain.EnvironmentResourceStatusEnabled,
				LastObservedAt: "2026-04-08T10:00:00Z",
			},
		},
	}
	startedRuntime := &notifyingRuntime{
		startTurnResult: agenttypes.StartTurnResult{
			TurnID:   "turn-02",
			ThreadID: "thread-01",
		},
		threadSnapshots: [][]domain.Thread{
			{
				{ThreadID: "thread-01", MachineID: "machine-01", Status: domain.ThreadStatusIdle, Title: "Restarted"},
			},
		},
	}

	registry := agentregistry.New()
	registry.Register("codex", initialRuntime)
	mgr := manager.New(registry)
	controller := newRuntimeController(
		context.Background(),
		config.Config{MachineID: "machine-01", RuntimeMode: config.RuntimeModeFake},
		time.Now,
		runtimeFactories{
			newFake: func(config.Config, func() time.Time) agenttypes.Runtime {
				return startedRuntime
			},
		},
		registry,
		"codex",
		initialRuntime,
		initialRuntime.cleanup,
	)
	session, sent := newRecordingSession()
	controller.bindSession(session, mgr, "codex")

	if err := handleCommandEnvelope(session, mgr, "codex", false, controller, protocol.Envelope{
		Name:      "runtime.stop",
		RequestID: "req-stop-1",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	if !initialRuntime.cleanupCalled {
		t.Fatal("expected initial runtime cleanup to run on stop")
	}

	machineSnapshot := decodeMachineSnapshotPayload(t, (*sent)[1])
	if machineSnapshot.Machine.Status != domain.MachineStatusOffline {
		t.Fatalf("expected offline machine snapshot after stop, got %+v", machineSnapshot.Machine)
	}
	stoppedThreads := decodeThreadSnapshotPayload(t, (*sent)[2])
	if len(stoppedThreads.Threads) != 0 {
		t.Fatalf("expected empty thread snapshot after stop, got %+v", stoppedThreads.Threads)
	}

	if err := handleCommandEnvelope(session, mgr, "codex", false, controller, protocol.Envelope{
		Name:      "turn.start",
		RequestID: "req-turn-1",
		Payload:   []byte(`{"threadId":"thread-01","input":"run tests"}`),
	}); err != nil {
		t.Fatal(err)
	}

	rejection := decodeRejectedPayload(t, (*sent)[4])
	if rejection.CommandName != "turn.start" {
		t.Fatalf("expected turn.start rejection while stopped, got %+v", rejection)
	}

	if err := handleCommandEnvelope(session, mgr, "codex", false, controller, protocol.Envelope{
		Name:      "runtime.start",
		RequestID: "req-start-1",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	machineSnapshot = decodeMachineSnapshotPayload(t, (*sent)[6])
	if machineSnapshot.Machine.Status != domain.MachineStatusOnline {
		t.Fatalf("expected online machine snapshot after start, got %+v", machineSnapshot.Machine)
	}

	if err := handleCommandEnvelope(session, mgr, "codex", false, controller, protocol.Envelope{
		Name:      "turn.start",
		RequestID: "req-turn-2",
		Payload:   []byte(`{"threadId":"thread-01","input":"run tests"}`),
	}); err != nil {
		t.Fatal(err)
	}

	var completed protocol.CommandCompletedPayload
	if err := json.Unmarshal(decodeEnvelope(t, (*sent)[9]).Payload, &completed); err != nil {
		t.Fatalf("decode command completed payload failed: %v", err)
	}
	if completed.CommandName != "turn.start" {
		t.Fatalf("unexpected final command ack: %+v", completed)
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

func TestRunClientReportsBootstrapFailureAndReturnsNonZero(t *testing.T) {
	var stderr bytes.Buffer

	exitCode := runClient(context.Background(), &stderr, config.Config{
		MachineID:   "machine-01",
		RuntimeMode: config.RuntimeModeAppServer,
	}, time.Now, runtimeFactories{
		newAppServer: func(context.Context, config.Config) (agenttypes.Runtime, func() error, error) {
			return nil, nil, errors.New("bootstrap boom")
		},
	})

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code on bootstrap failure")
	}
	if got := stderr.String(); got == "" || !bytes.Contains([]byte(got), []byte("bootstrap boom")) {
		t.Fatalf("expected bootstrap failure to be written to stderr, got %q", got)
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

func decodeMachineSnapshotPayload(t *testing.T, raw []byte) protocol.MachineSnapshotPayload {
	t.Helper()

	var envelope protocol.Envelope
	if err := transport.Decode(raw, &envelope); err != nil {
		t.Fatalf("decode envelope failed: %v", err)
	}

	var payload protocol.MachineSnapshotPayload
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
	threadSnapshots      [][]domain.Thread
	environment          []domain.EnvironmentResource
	listThreadsCalls     int
	cleanupCalled        bool
	lastApprovalDecision struct {
		requestID string
		decision  string
	}
}

func (r *notifyingRuntime) ListThreads() ([]domain.Thread, error) {
	if len(r.threadSnapshots) == 0 {
		return nil, nil
	}

	idx := r.listThreadsCalls
	if idx >= len(r.threadSnapshots) {
		idx = len(r.threadSnapshots) - 1
	}
	r.listThreadsCalls++
	return append([]domain.Thread(nil), r.threadSnapshots[idx]...), nil
}

func (r *notifyingRuntime) ListEnvironment() ([]domain.EnvironmentResource, error) {
	return append([]domain.EnvironmentResource(nil), r.environment...), nil
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

func (r *notifyingRuntime) cleanup() error {
	r.cleanupCalled = true
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
