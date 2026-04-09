package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"code-agent-gateway/client/internal/agent/codex"
	"code-agent-gateway/client/internal/agent/manager"
	agentregistry "code-agent-gateway/client/internal/agent/registry"
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/client/internal/config"
	"code-agent-gateway/client/internal/gateway"
	"code-agent-gateway/client/internal/snapshot"
	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
	cws "github.com/coder/websocket"
)

func main() {
	exitCode := runClient(context.Background(), os.Stderr, config.Read(), time.Now, defaultRuntimeFactories())
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func runClient(parentCtx context.Context, stderr io.Writer, cfg config.Config, now func() time.Time, factories runtimeFactories) int {
	const heartbeatInterval = 30 * time.Second
	const connectTimeout = 5 * time.Second
	const reconnectMaxBackoff = 5 * time.Second
	const runtimeName = "codex"

	if stderr == nil {
		stderr = io.Discard
	}
	if now == nil {
		now = time.Now
	}
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	shutdownCtx, stop := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	runtime, cleanupRuntime, err := buildRuntime(shutdownCtx, cfg, now, factories)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "runtime bootstrap failed: %v\n", err)
		return 1
	}

	runtimeRegistry := agentregistry.New()
	runtimeRegistry.Register(runtimeName, runtime)
	agentManager := manager.New(runtimeRegistry)
	runtimeController := newRuntimeController(shutdownCtx, cfg, now, factories, runtimeRegistry, runtimeName, runtime, cleanupRuntime)
	defer func() {
		_ = runtimeController.Close()
	}()

	machine := domain.Machine{
		ID:   cfg.MachineID,
		Name: cfg.MachineID,
	}

	backoff := time.Duration(0)
	for {
		if shutdownCtx.Err() != nil {
			return 0
		}

		dialCtx, cancelDial := context.WithTimeout(shutdownCtx, connectTimeout)
		conn, err := gateway.Dial(dialCtx, cfg.GatewayURL)
		cancelDial()
		if err != nil {
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return 0
			}
			continue
		}

		session := newClientSession(cfg.MachineID, func(msg []byte) error {
			return conn.Write(shutdownCtx, cws.MessageText, msg)
		}, now)
		runtimeStreamsTurnEvents := runtimeController.bindSession(session, agentManager, runtimeName)
		if err := session.Register(); err != nil {
			_ = conn.Close(cws.StatusNormalClosure, "register-failed")
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return 0
			}
			continue
		}
		machine.Status = runtimeController.machineStatus()
		machine.RuntimeStatus = runtimeController.runtimeStatus()
		if err := sendLiveSnapshot(session, machine, agentManager, runtimeName); err != nil {
			_ = conn.Close(cws.StatusNormalClosure, "snapshot-failed")
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return 0
			}
			continue
		}

		backoff = 0
		if err := runConnection(shutdownCtx, conn, session, agentManager, runtimeName, runtimeStreamsTurnEvents, runtimeController, heartbeatInterval); err != nil {
			_ = conn.Close(cws.StatusNormalClosure, "reconnect")
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return 0
			}
			continue
		}

		_ = conn.Close(cws.StatusNormalClosure, "done")
		return 0
	}
}

type runtimeFactories struct {
	newFake      func(cfg config.Config, now func() time.Time) agenttypes.Runtime
	newAppServer func(ctx context.Context, cfg config.Config) (agenttypes.Runtime, func() error, error)
}

type approvalContext struct {
	threadID string
	turnID   string
}

type runtimeController struct {
	mu                       sync.Mutex
	ctx                      context.Context
	cfg                      config.Config
	now                      func() time.Time
	factories                runtimeFactories
	registry                 *agentregistry.Registry
	runtimeName              string
	machineID                string
	runtime                  agenttypes.Runtime
	cleanup                  func() error
	runtimeStreamsTurnEvents bool
	stopped                  bool
}

type stoppedRuntime struct{}

func newRuntimeController(ctx context.Context, cfg config.Config, now func() time.Time, factories runtimeFactories, registry *agentregistry.Registry, runtimeName string, runtime agenttypes.Runtime, cleanup func() error) *runtimeController {
	return &runtimeController{
		ctx:         ctx,
		cfg:         cfg,
		now:         now,
		factories:   factories,
		registry:    registry,
		runtimeName: runtimeName,
		machineID:   cfg.MachineID,
		runtime:     runtime,
		cleanup:     cleanup,
		stopped:     false,
	}
}

func (c *runtimeController) bindSession(session *clientSession, mgr *manager.Manager, runtimeName string) bool {
	if c == nil {
		return false
	}

	c.mu.Lock()
	runtime := c.runtime
	if runtimeName != "" {
		c.runtimeName = runtimeName
	}
	c.mu.Unlock()

	streamsTurnEvents := bindRuntimeTurnEvents(runtime, session, mgr, c.runtimeName)
	bindRuntimeApprovalEvents(runtime, session)

	c.mu.Lock()
	c.runtimeStreamsTurnEvents = streamsTurnEvents
	c.mu.Unlock()
	return streamsTurnEvents
}

func (c *runtimeController) runtimeStreams() bool {
	if c == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.runtimeStreamsTurnEvents
}

func (c *runtimeController) machineStatus() domain.MachineStatus {
	return domain.MachineStatusOnline
}

func (c *runtimeController) runtimeStatus() domain.MachineRuntimeStatus {
	if c == nil {
		return domain.MachineRuntimeStatusRunning
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return domain.MachineRuntimeStatusStopped
	}
	return domain.MachineRuntimeStatusRunning
}

func (c *runtimeController) Stop() error {
	if c == nil {
		return fmt.Errorf("runtime controller is not configured")
	}

	c.mu.Lock()
	cleanup := c.cleanup
	c.runtime = stoppedRuntime{}
	c.cleanup = func() error { return nil }
	c.runtimeStreamsTurnEvents = false
	c.stopped = true
	if c.registry != nil {
		c.registry.Register(c.runtimeName, c.runtime)
	}
	c.mu.Unlock()

	if cleanup != nil {
		return cleanup()
	}
	return nil
}

func (c *runtimeController) Start(session *clientSession, mgr *manager.Manager, runtimeName string) error {
	if c == nil {
		return fmt.Errorf("runtime controller is not configured")
	}

	c.mu.Lock()
	if !c.stopped {
		c.mu.Unlock()
		if session != nil && mgr != nil {
			c.bindSession(session, mgr, runtimeName)
		}
		return nil
	}
	c.mu.Unlock()

	runtime, cleanup, err := buildRuntime(c.ctx, c.cfg, c.now, c.factories)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.runtime = runtime
	c.cleanup = cleanup
	c.stopped = false
	if c.registry != nil {
		c.registry.Register(c.runtimeName, runtime)
	}
	c.mu.Unlock()

	if session != nil && mgr != nil {
		c.bindSession(session, mgr, runtimeName)
	}
	return nil
}

func (c *runtimeController) Close() error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	cleanup := c.cleanup
	c.cleanup = func() error { return nil }
	c.mu.Unlock()

	if cleanup != nil {
		return cleanup()
	}
	return nil
}

type clientSession struct {
	delegate     *gateway.Session
	machineID    string
	send         gateway.Sender
	now          func() time.Time
	sendMu       sync.Mutex
	approvalMu   sync.Mutex
	approvalByID map[string]approvalContext
}

func newClientSession(machineID string, send gateway.Sender, now func() time.Time) *clientSession {
	if now == nil {
		now = time.Now
	}

	session := &clientSession{
		machineID:    machineID,
		now:          now,
		approvalByID: map[string]approvalContext{},
	}
	session.send = func(msg []byte) error {
		session.sendMu.Lock()
		defer session.sendMu.Unlock()
		return send(msg)
	}
	session.delegate = gateway.NewSession(machineID, session.send, now)
	return session
}

func (s *clientSession) Register() error {
	return s.delegate.Register()
}

func (s *clientSession) Heartbeat() error {
	return s.delegate.Heartbeat()
}

func (s *clientSession) MachineSnapshot(machine domain.Machine) error {
	return s.delegate.MachineSnapshot(machine)
}

func (s *clientSession) ThreadSnapshot(threads []domain.Thread) error {
	return s.delegate.ThreadSnapshot(threads)
}

func (s *clientSession) EnvironmentSnapshot(environment []domain.EnvironmentResource) error {
	return s.delegate.EnvironmentSnapshot(environment)
}

func (s *clientSession) CommandCompleted(requestID string, commandName string, result any) error {
	return s.delegate.CommandCompleted(requestID, commandName, result)
}

func (s *clientSession) CommandRejected(requestID string, commandName string, reason string, threadID string) error {
	return s.delegate.CommandRejected(requestID, commandName, reason, threadID)
}

func (s *clientSession) TurnDelta(requestID string, payload protocol.TurnDeltaPayload) error {
	return s.delegate.TurnDelta(requestID, payload)
}

func (s *clientSession) TurnCompleted(requestID string, payload protocol.TurnCompletedPayload) error {
	return s.delegate.TurnCompleted(requestID, payload)
}

func (s *clientSession) TurnStarted(requestID string, payload protocol.TurnStartedPayload) error {
	return s.sendEnvelope(protocol.CategoryEvent, "turn.started", requestID, payload)
}

func (s *clientSession) ApprovalRequired(payload protocol.ApprovalRequiredPayload) error {
	s.rememberApprovalContext(payload.RequestID, payload.ThreadID, payload.TurnID)
	return s.sendEnvelope(protocol.CategoryEvent, "approval.required", payload.RequestID, payload)
}

func (s *clientSession) ApprovalResolved(payload protocol.ApprovalResolvedPayload) error {
	if payload.ThreadID == "" || payload.TurnID == "" {
		context := s.approvalContext(payload.RequestID)
		if payload.ThreadID == "" {
			payload.ThreadID = context.threadID
		}
		if payload.TurnID == "" {
			payload.TurnID = context.turnID
		}
	}
	if err := s.sendEnvelope(protocol.CategoryEvent, "approval.resolved", payload.RequestID, payload); err != nil {
		return err
	}
	s.clearApprovalContext(payload.RequestID)
	return nil
}

func (s *clientSession) rememberApprovalContext(requestID string, threadID string, turnID string) {
	if strings.TrimSpace(requestID) == "" {
		return
	}

	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	s.approvalByID[requestID] = approvalContext{
		threadID: threadID,
		turnID:   turnID,
	}
}

func (s *clientSession) approvalContext(requestID string) approvalContext {
	if strings.TrimSpace(requestID) == "" {
		return approvalContext{}
	}

	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	return s.approvalByID[requestID]
}

func (s *clientSession) clearApprovalContext(requestID string) {
	if strings.TrimSpace(requestID) == "" {
		return
	}

	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	delete(s.approvalByID, requestID)
}

func (s *clientSession) sendEnvelope(category protocol.Category, name string, requestID string, payload any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	frame := protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  category,
		Name:      name,
		RequestID: requestID,
		MachineID: s.machineID,
		Timestamp: s.now().Format(time.RFC3339),
		Payload:   payloadJSON,
	}

	encoded, err := transport.Encode(frame)
	if err != nil {
		return err
	}

	return s.send(encoded)
}

func defaultRuntimeFactories() runtimeFactories {
	return runtimeFactories{
		newFake: newFakeRuntime,
		newAppServer: func(ctx context.Context, cfg config.Config) (agenttypes.Runtime, func() error, error) {
			runner, err := codex.NewStdioRunner(ctx, cfg.CodexBin)
			if err != nil {
				return nil, nil, err
			}
			client := codex.NewAppServerClient(runner)
			if err := client.Initialize(); err != nil {
				_ = runner.Close()
				return nil, nil, err
			}
			return client, runner.Close, nil
		},
	}
}

func buildRuntime(ctx context.Context, cfg config.Config, now func() time.Time, factories runtimeFactories) (agenttypes.Runtime, func() error, error) {
	switch cfg.RuntimeMode {
	case "", config.RuntimeModeAppServer:
		if factories.newAppServer == nil {
			return nil, nil, fmt.Errorf("app-server runtime factory is not configured")
		}
		return factories.newAppServer(ctx, cfg)
	case config.RuntimeModeFake:
		if factories.newFake == nil {
			return nil, nil, fmt.Errorf("fake runtime factory is not configured")
		}
		return factories.newFake(cfg, now), func() error { return nil }, nil
	default:
		return nil, nil, fmt.Errorf("unsupported runtime mode %q", cfg.RuntimeMode)
	}
}

func newFakeRuntime(cfg config.Config, now func() time.Time) agenttypes.Runtime {
	adapter := codex.NewFakeAdapter()
	adapter.SeedSnapshot(
		[]domain.Thread{
			{
				ThreadID:  "thread-01",
				MachineID: cfg.MachineID,
				Status:    domain.ThreadStatusIdle,
				Title:     "Gateway bootstrap thread",
			},
		},
		[]domain.EnvironmentResource{
			{
				ResourceID:      "skill-01",
				MachineID:       cfg.MachineID,
				Kind:            domain.EnvironmentKindSkill,
				DisplayName:     "Bootstrap Skill",
				Status:          domain.EnvironmentResourceStatusEnabled,
				RestartRequired: false,
				LastObservedAt:  now().UTC().Format(time.RFC3339),
			},
		},
	)
	return adapter
}

func (stoppedRuntime) ListThreads() ([]domain.Thread, error) {
	return nil, nil
}

func (stoppedRuntime) ListEnvironment() ([]domain.EnvironmentResource, error) {
	return nil, nil
}

func (stoppedRuntime) CreateThread(agenttypes.CreateThreadParams) (domain.Thread, error) {
	return domain.Thread{}, fmt.Errorf("runtime is stopped")
}

func (stoppedRuntime) ReadThread(string) (domain.Thread, error) {
	return domain.Thread{}, fmt.Errorf("runtime is stopped")
}

func (stoppedRuntime) ResumeThread(string) (domain.Thread, error) {
	return domain.Thread{}, fmt.Errorf("runtime is stopped")
}

func (stoppedRuntime) ArchiveThread(string) error {
	return fmt.Errorf("runtime is stopped")
}

func (stoppedRuntime) StartTurn(agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	return agenttypes.StartTurnResult{}, fmt.Errorf("runtime is stopped")
}

func (stoppedRuntime) SteerTurn(agenttypes.SteerTurnParams) (agenttypes.SteerTurnResult, error) {
	return agenttypes.SteerTurnResult{}, fmt.Errorf("runtime is stopped")
}

func (stoppedRuntime) InterruptTurn(agenttypes.InterruptTurnParams) (domain.Turn, error) {
	return domain.Turn{}, fmt.Errorf("runtime is stopped")
}

func bindRuntimeTurnEvents(runtime agenttypes.Runtime, session *clientSession, mgr *manager.Manager, runtimeName string) bool {
	source, ok := runtime.(agenttypes.RuntimeTurnEventSource)
	if !ok {
		return false
	}

	source.SetTurnEventHandler(func(event agenttypes.RuntimeTurnEvent) {
		_ = handleRuntimeTurnEvent(session, mgr, runtimeName, event)
	})
	return true
}

func handleRuntimeTurnEvent(session *clientSession, mgr *manager.Manager, runtimeName string, event agenttypes.RuntimeTurnEvent) error {
	if err := emitRuntimeTurnEvent(session, event); err != nil {
		return err
	}
	if !shouldRefreshThreadSnapshotForTurnEvent(event) || mgr == nil || runtimeName == "" {
		return nil
	}
	return refreshThreadSnapshot(session, mgr, runtimeName)
}

func shouldRefreshThreadSnapshotForTurnEvent(event agenttypes.RuntimeTurnEvent) bool {
	switch event.Type {
	case agenttypes.RuntimeTurnEventTypeStarted, agenttypes.RuntimeTurnEventTypeCompleted:
		return true
	default:
		return false
	}
}

func emitRuntimeTurnEvent(session *clientSession, event agenttypes.RuntimeTurnEvent) error {
	switch event.Type {
	case agenttypes.RuntimeTurnEventTypeStarted:
		return session.TurnStarted(event.RequestID, protocol.TurnStartedPayload{
			ThreadID: event.ThreadID,
			TurnID:   event.TurnID,
		})
	case agenttypes.RuntimeTurnEventTypeDelta:
		return session.TurnDelta(event.RequestID, protocol.TurnDeltaPayload{
			ThreadID: event.ThreadID,
			TurnID:   event.TurnID,
			Sequence: event.Sequence,
			Delta:    event.Delta,
		})
	case agenttypes.RuntimeTurnEventTypeCompleted:
		return session.TurnCompleted(event.RequestID, protocol.TurnCompletedPayload{
			Turn: event.Turn,
		})
	default:
		return nil
	}
}

func bindRuntimeApprovalEvents(runtime agenttypes.Runtime, session *clientSession) bool {
	bound := false

	source, ok := runtime.(agenttypes.RuntimeApprovalEventSource)
	if ok {
		source.SetApprovalHandler(func(event agenttypes.RuntimeApprovalRequest) {
			_ = emitRuntimeApprovalEvent(session, event)
		})
		bound = true
	}

	resolvedSource, ok := runtime.(interface {
		SetApprovalResolvedHandler(func(codex.ApprovalResolvedEvent))
	})
	if ok {
		resolvedSource.SetApprovalResolvedHandler(func(event codex.ApprovalResolvedEvent) {
			_ = session.ApprovalResolved(protocol.ApprovalResolvedPayload{
				RequestID: event.RequestID,
				ThreadID:  event.ThreadID,
				TurnID:    event.TurnID,
				Decision:  event.Decision,
			})
		})
		bound = true
	}

	return bound
}

func emitRuntimeApprovalEvent(session *clientSession, event agenttypes.RuntimeApprovalRequest) error {
	return session.ApprovalRequired(protocol.ApprovalRequiredPayload{
		RequestID: event.RequestID,
		ThreadID:  event.ThreadID,
		TurnID:    event.TurnID,
		ItemID:    event.ItemID,
		Kind:      event.Kind,
		Reason:    event.Reason,
		Command:   event.Command,
	})
}

func runConnection(ctx context.Context, conn *cws.Conn, session *clientSession, mgr *manager.Manager, runtimeName string, runtimeStreamsTurnEvents bool, runtimeController *runtimeController, heartbeatInterval time.Duration) error {
	loopCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		errCh <- runHeartbeatLoop(loopCtx, session, heartbeatInterval)
	}()

	go func() {
		errCh <- runCommandLoop(loopCtx, conn, session, mgr, runtimeName, runtimeStreamsTurnEvents, runtimeController)
	}()

	err := <-errCh
	cancel()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func sendLiveSnapshot(session *clientSession, machine domain.Machine, mgr *manager.Manager, runtimeName string) error {
	snap, err := mgr.Snapshot(runtimeName)
	if err != nil {
		return err
	}

	return sendInitialSnapshot(session, machine, snap)
}

func sendInitialSnapshot(session *clientSession, machine domain.Machine, snap snapshot.Snapshot) error {
	if err := session.MachineSnapshot(machine); err != nil {
		return err
	}
	if err := session.ThreadSnapshot(snap.Threads); err != nil {
		return err
	}
	return session.EnvironmentSnapshot(snap.Environment)
}

func runHeartbeatLoop(ctx context.Context, session *clientSession, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := session.Heartbeat(); err != nil {
				return err
			}
		}
	}
}

func runCommandLoop(ctx context.Context, conn *cws.Conn, session *clientSession, mgr *manager.Manager, runtimeName string, runtimeStreamsTurnEvents bool, runtimeController *runtimeController) error {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		var envelope protocol.Envelope
		if err := transport.Decode(data, &envelope); err != nil {
			continue
		}

		if envelope.Category != protocol.CategoryCommand {
			continue
		}

		if err := handleCommandEnvelope(session, mgr, runtimeName, runtimeStreamsTurnEvents, runtimeController, envelope); err != nil {
			return err
		}
	}
}

func handleCommandEnvelope(session *clientSession, mgr *manager.Manager, runtimeName string, runtimeStreamsTurnEvents bool, runtimeController *runtimeController, envelope protocol.Envelope) error {
	if runtimeController != nil {
		runtimeStreamsTurnEvents = runtimeController.runtimeStreams()
	}

	switch envelope.Name {
	case "thread.create":
		var payload protocol.ThreadCreateCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		thread, err := mgr.CreateThread(runtimeName, agenttypes.CreateThreadParams{
			Title: payload.Title,
		})
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadCreateCommandResult{
			Thread: thread,
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, runtimeName)
	case "thread.read":
		var payload protocol.ThreadReadCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		thread, err := mgr.ReadThread(runtimeName, payload.ThreadID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		return session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadReadCommandResult{
			Thread: thread,
		})
	case "thread.resume":
		var payload protocol.ThreadResumeCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		thread, err := mgr.ResumeThread(runtimeName, payload.ThreadID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadResumeCommandResult{
			Thread: thread,
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, runtimeName)
	case "thread.archive":
		var payload protocol.ThreadArchiveCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		if err := mgr.ArchiveThread(runtimeName, payload.ThreadID); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadArchiveCommandResult{
			ThreadID: payload.ThreadID,
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, runtimeName)
	case "turn.start":
		var payload protocol.TurnStartCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		result, err := mgr.StartTurn(runtimeName, agenttypes.StartTurnParams{
			ThreadID: payload.ThreadID,
			Input:    payload.Input,
		})
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.TurnStartCommandResult{
			TurnID:   result.TurnID,
			ThreadID: result.ThreadID,
		}); err != nil {
			return err
		}
		if runtimeStreamsTurnEvents {
			return nil
		}

		for _, delta := range result.Deltas {
			if err := session.TurnDelta(envelope.RequestID, protocol.TurnDeltaPayload{
				ThreadID: result.ThreadID,
				TurnID:   result.TurnID,
				Sequence: delta.Sequence,
				Delta:    delta.Delta,
			}); err != nil {
				return err
			}
		}

		if err := session.TurnCompleted(envelope.RequestID, protocol.TurnCompletedPayload{
			Turn: domain.Turn{
				TurnID:   result.TurnID,
				ThreadID: result.ThreadID,
				Status:   domain.TurnStatusCompleted,
			},
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, runtimeName)
	case "turn.steer":
		var payload protocol.TurnSteerCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		result, err := mgr.SteerTurn(runtimeName, agenttypes.SteerTurnParams{
			ThreadID: payload.ThreadID,
			TurnID:   payload.TurnID,
			Input:    payload.Input,
		})
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.TurnSteerCommandResult{
			TurnID:   result.TurnID,
			ThreadID: result.ThreadID,
		}); err != nil {
			return err
		}
		if runtimeStreamsTurnEvents {
			return nil
		}

		for _, delta := range result.Deltas {
			if err := session.TurnDelta(envelope.RequestID, protocol.TurnDeltaPayload{
				ThreadID: result.ThreadID,
				TurnID:   result.TurnID,
				Sequence: delta.Sequence,
				Delta:    delta.Delta,
			}); err != nil {
				return err
			}
		}

		if err := session.TurnCompleted(envelope.RequestID, protocol.TurnCompletedPayload{
			Turn: domain.Turn{
				TurnID:   result.TurnID,
				ThreadID: result.ThreadID,
				Status:   domain.TurnStatusCompleted,
			},
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, runtimeName)
	case "turn.interrupt":
		var payload protocol.TurnInterruptCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		turn, err := mgr.InterruptTurn(runtimeName, agenttypes.InterruptTurnParams{
			ThreadID: payload.ThreadID,
			TurnID:   payload.TurnID,
		})
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.TurnInterruptCommandResult{
			Turn: turn,
		}); err != nil {
			return err
		}

		if err := session.TurnCompleted(envelope.RequestID, protocol.TurnCompletedPayload{
			Turn: turn,
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, runtimeName)
	case "approval.respond":
		var payload protocol.ApprovalRespondCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		if err := mgr.RespondApproval(runtimeName, payload.RequestID, payload.Decision); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ApprovalRespondCommandResult{
			RequestID: payload.RequestID,
			Decision:  payload.Decision,
		}); err != nil {
			return err
		}

		return session.ApprovalResolved(protocol.ApprovalResolvedPayload{
			RequestID: payload.RequestID,
			ThreadID:  payload.ThreadID,
			TurnID:    payload.TurnID,
			Decision:  payload.Decision,
		})
	case "runtime.stop":
		var payload protocol.RuntimeStopCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if runtimeController == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime controller unavailable", "")
		}
		if err := runtimeController.Stop(); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.RuntimeStopCommandResult{}); err != nil {
			return err
		}
		return sendLiveSnapshot(session, domain.Machine{
			ID:            session.machineID,
			Name:          session.machineID,
			Status:        runtimeController.machineStatus(),
			RuntimeStatus: runtimeController.runtimeStatus(),
		}, mgr, runtimeName)
	case "runtime.start":
		var payload protocol.RuntimeStartCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if runtimeController == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime controller unavailable", "")
		}
		if err := runtimeController.Start(session, mgr, runtimeName); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.RuntimeStartCommandResult{}); err != nil {
			return err
		}
		return sendLiveSnapshot(session, domain.Machine{
			ID:            session.machineID,
			Name:          session.machineID,
			Status:        runtimeController.machineStatus(),
			RuntimeStatus: runtimeController.runtimeStatus(),
		}, mgr, runtimeName)
	case "environment.skill.enable", "environment.skill.disable":
		var payload protocol.EnvironmentSkillSetEnabledCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.SetSkillEnabled(runtimeName, payload.SkillID, payload.Enabled); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentSkillSetEnabledCommandResult{
			SkillID: payload.SkillID,
			Enabled: payload.Enabled,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, runtimeName)
	case "environment.plugin.uninstall":
		var payload protocol.EnvironmentPluginUninstallCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.UninstallPlugin(runtimeName, payload.PluginID); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentPluginUninstallCommandResult{
			PluginID: payload.PluginID,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, runtimeName)
	default:
		return session.CommandRejected(envelope.RequestID, envelope.Name, "unsupported command", "")
	}
}

func refreshThreadSnapshot(session *clientSession, mgr *manager.Manager, runtimeName string) error {
	threads, err := mgr.Threads(runtimeName)
	if err != nil {
		return err
	}

	return session.ThreadSnapshot(threads)
}

func refreshEnvironmentSnapshot(session *clientSession, mgr *manager.Manager, runtimeName string) error {
	environment, err := mgr.Environment(runtimeName)
	if err != nil {
		return err
	}

	return session.EnvironmentSnapshot(environment)
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func nextBackoff(current, max time.Duration) time.Duration {
	if current <= 0 {
		return 1 * time.Second
	}

	next := current * 2
	if next > max {
		return max
	}

	return next
}
