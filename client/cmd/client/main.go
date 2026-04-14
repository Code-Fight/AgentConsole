package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
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

	runtimeRegistry := agentregistry.New()
	supervisor, err := manager.NewSupervisor(
		shutdownCtx,
		cfg.ManagedAgentsDir,
		runtimeRegistry,
		map[domain.AgentType]agenttypes.RuntimeFactory{
			domain.AgentTypeCodex: managedRuntimeFactory{
				cfg:       cfg,
				now:       now,
				factories: factories,
			},
		},
	)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "runtime bootstrap failed: %v\n", err)
		return 1
	}
	agentManager := manager.New(runtimeRegistry)
	defer func() {
		_ = supervisor.StopAll()
	}()

	machine := buildMachineSnapshot(cfg.MachineID, supervisor)

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
		bindAllManagedRuntimeEvents(runtimeRegistry, session, agentManager)
		if err := session.Register(); err != nil {
			_ = conn.Close(cws.StatusNormalClosure, "register-failed")
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return 0
			}
			continue
		}
		machine = buildMachineSnapshot(cfg.MachineID, supervisor)
		if err := sendLiveSnapshot(session, machine, agentManager, runtimeRegistry); err != nil {
			_ = conn.Close(cws.StatusNormalClosure, "snapshot-failed")
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return 0
			}
			continue
		}

		backoff = 0
		if err := runConnection(shutdownCtx, conn, session, agentManager, runtimeRegistry, supervisor, heartbeatInterval); err != nil {
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
	newFake      func(cfg config.Config, spec agenttypes.ManagedAgentSpec, now func() time.Time) agenttypes.Runtime
	newAppServer func(ctx context.Context, cfg config.Config, spec agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error)
}

type managedRuntimeFactory struct {
	cfg       config.Config
	now       func() time.Time
	factories runtimeFactories
}

func (f managedRuntimeFactory) Start(ctx context.Context, spec agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
	return buildManagedRuntime(ctx, f.cfg, spec, f.now, f.factories)
}

type approvalContext struct {
	rawRequestID    string
	publicRequestID string
	agentID         string
	threadID        string
	turnID          string
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

	streamsTurnEvents := bindRuntimeTurnEvents(runtime, session, mgr, c.registry, c.runtimeName)
	bindRuntimeApprovalEvents(runtime, session, c.runtimeName)

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

func (s *clientSession) TurnFailed(requestID string, payload protocol.TurnCompletedPayload) error {
	return s.delegate.TurnFailed(requestID, payload)
}

func (s *clientSession) TurnStarted(requestID string, payload protocol.TurnStartedPayload) error {
	return s.sendEnvelope(protocol.CategoryEvent, "turn.started", requestID, payload)
}

func (s *clientSession) ApprovalRequired(agentID string, payload protocol.ApprovalRequiredPayload) error {
	publicRequestID := s.publicApprovalRequestID(payload.RequestID)
	s.rememberApprovalContext(payload.RequestID, publicRequestID, agentID, payload.ThreadID, payload.TurnID)
	payload.RequestID = publicRequestID
	return s.sendEnvelope(protocol.CategoryEvent, "approval.required", publicRequestID, payload)
}

func (s *clientSession) ApprovalResolved(payload protocol.ApprovalResolvedPayload) error {
	payload.RequestID = s.publicApprovalRequestID(payload.RequestID)
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

func (s *clientSession) rememberApprovalContext(rawRequestID string, publicRequestID string, agentID string, threadID string, turnID string) {
	if strings.TrimSpace(publicRequestID) == "" {
		return
	}

	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	s.approvalByID[publicRequestID] = approvalContext{
		rawRequestID:    strings.TrimSpace(rawRequestID),
		publicRequestID: publicRequestID,
		agentID:         strings.TrimSpace(agentID),
		threadID:        threadID,
		turnID:          turnID,
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

func (s *clientSession) publicApprovalRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ""
	}
	if _, ok := decodePublicApprovalRequestID(s.machineID, requestID); ok {
		return requestID
	}
	return encodePublicApprovalRequestID(s.machineID, requestID)
}

func (s *clientSession) rawApprovalRequestID(requestID string) (string, bool) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return "", false
	}
	if rawRequestID, ok := decodePublicApprovalRequestID(s.machineID, requestID); ok {
		return rawRequestID, true
	}

	s.approvalMu.Lock()
	context, ok := s.approvalByID[requestID]
	s.approvalMu.Unlock()
	if ok && strings.TrimSpace(context.rawRequestID) != "" {
		return context.rawRequestID, true
	}

	if strings.HasPrefix(requestID, "apr.") {
		return "", false
	}
	return requestID, true
}

func encodePublicApprovalRequestID(machineID string, rawRequestID string) string {
	if strings.TrimSpace(machineID) == "" || strings.TrimSpace(rawRequestID) == "" {
		return ""
	}
	return "apr." +
		base64.RawURLEncoding.EncodeToString([]byte(machineID)) +
		"." +
		base64.RawURLEncoding.EncodeToString([]byte(rawRequestID))
}

func decodePublicApprovalRequestID(machineID string, publicRequestID string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(publicRequestID), ".")
	if len(parts) != 3 || parts[0] != "apr" {
		return "", false
	}

	decodedMachineID, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || string(decodedMachineID) != machineID {
		return "", false
	}

	decodedRequestID, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || strings.TrimSpace(string(decodedRequestID)) == "" {
		return "", false
	}

	return string(decodedRequestID), true
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
		newAppServer: func(ctx context.Context, cfg config.Config, spec agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			layout := codex.NewInstanceLayout(cfg.ManagedAgentsDir, spec.AgentID)
			return codex.NewIsolatedAppServerClient(ctx, cfg.CodexBin, layout)
		},
	}
}

func buildRuntime(ctx context.Context, cfg config.Config, now func() time.Time, factories runtimeFactories) (agenttypes.Runtime, func() error, error) {
	return buildManagedRuntime(ctx, cfg, agenttypes.ManagedAgentSpec{
		AgentID:     "agent-01",
		AgentType:   domain.AgentTypeCodex,
		DisplayName: "Codex",
	}, now, factories)
}

func buildManagedRuntime(ctx context.Context, cfg config.Config, spec agenttypes.ManagedAgentSpec, now func() time.Time, factories runtimeFactories) (agenttypes.Runtime, func() error, error) {
	switch cfg.RuntimeMode {
	case "", config.RuntimeModeAppServer:
		if factories.newAppServer == nil {
			return nil, nil, fmt.Errorf("app-server runtime factory is not configured")
		}
		return factories.newAppServer(ctx, cfg, spec)
	case config.RuntimeModeFake:
		if factories.newFake == nil {
			return nil, nil, fmt.Errorf("fake runtime factory is not configured")
		}
		return factories.newFake(cfg, spec, now), func() error { return nil }, nil
	default:
		return nil, nil, fmt.Errorf("unsupported runtime mode %q", cfg.RuntimeMode)
	}
}

func newFakeRuntime(cfg config.Config, spec agenttypes.ManagedAgentSpec, now func() time.Time) agenttypes.Runtime {
	adapter := codex.NewFakeAdapter()
	adapter.SeedSnapshot(
		[]domain.Thread{
			{
				ThreadID:  "thread-01",
				MachineID: cfg.MachineID,
				AgentID:   spec.AgentID,
				Status:    domain.ThreadStatusIdle,
				Title:     firstNonEmpty(strings.TrimSpace(spec.DisplayName), "Gateway bootstrap thread"),
			},
		},
		[]domain.EnvironmentResource{
			{
				ResourceID:      "skill-01",
				MachineID:       cfg.MachineID,
				AgentID:         spec.AgentID,
				Kind:            domain.EnvironmentKindSkill,
				DisplayName:     firstNonEmpty(strings.TrimSpace(spec.DisplayName), "Bootstrap Skill"),
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

func bindRuntimeTurnEvents(runtime agenttypes.Runtime, session *clientSession, mgr *manager.Manager, registry *agentregistry.Registry, runtimeName string) bool {
	source, ok := runtime.(agenttypes.RuntimeTurnEventSource)
	if !ok {
		return false
	}

	source.SetTurnEventHandler(func(event agenttypes.RuntimeTurnEvent) {
		_ = handleRuntimeTurnEvent(session, mgr, registry, runtimeName, event)
	})
	return true
}

func handleRuntimeTurnEvent(session *clientSession, mgr *manager.Manager, registry *agentregistry.Registry, runtimeName string, event agenttypes.RuntimeTurnEvent) error {
	if err := emitRuntimeTurnEvent(session, event); err != nil {
		return err
	}
	if !shouldRefreshThreadSnapshotForTurnEvent(event) || mgr == nil || runtimeName == "" {
		return nil
	}
	return refreshThreadSnapshot(session, mgr, registry)
}

func shouldRefreshThreadSnapshotForTurnEvent(event agenttypes.RuntimeTurnEvent) bool {
	switch event.Type {
	case agenttypes.RuntimeTurnEventTypeStarted, agenttypes.RuntimeTurnEventTypeCompleted, agenttypes.RuntimeTurnEventTypeFailed:
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
	case agenttypes.RuntimeTurnEventTypeFailed:
		return session.TurnFailed(event.RequestID, protocol.TurnCompletedPayload{
			Turn: event.Turn,
		})
	default:
		return nil
	}
}

func bindRuntimeApprovalEvents(runtime agenttypes.Runtime, session *clientSession, agentID string) bool {
	bound := false

	source, ok := runtime.(agenttypes.RuntimeApprovalEventSource)
	if ok {
		source.SetApprovalHandler(func(event agenttypes.RuntimeApprovalRequest) {
			_ = emitRuntimeApprovalEvent(session, agentID, event)
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

func emitRuntimeApprovalEvent(session *clientSession, agentID string, event agenttypes.RuntimeApprovalRequest) error {
	return session.ApprovalRequired(agentID, protocol.ApprovalRequiredPayload{
		RequestID: event.RequestID,
		ThreadID:  event.ThreadID,
		TurnID:    event.TurnID,
		ItemID:    event.ItemID,
		Kind:      event.Kind,
		Reason:    event.Reason,
		Command:   event.Command,
	})
}

func runConnection(ctx context.Context, conn *cws.Conn, session *clientSession, mgr *manager.Manager, registry *agentregistry.Registry, supervisor *manager.Supervisor, heartbeatInterval time.Duration) error {
	loopCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		errCh <- runHeartbeatLoop(loopCtx, session, heartbeatInterval)
	}()

	go func() {
		errCh <- runCommandLoop(loopCtx, conn, session, mgr, registry, supervisor)
	}()

	err := <-errCh
	cancel()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func sendLiveSnapshot(session *clientSession, machine domain.Machine, mgr *manager.Manager, registry *agentregistry.Registry) error {
	snap, err := collectManagedSnapshot(session.machineID, mgr, registry)
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

func runCommandLoop(ctx context.Context, conn *cws.Conn, session *clientSession, mgr *manager.Manager, registry *agentregistry.Registry, supervisor *manager.Supervisor) error {
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

		if err := handleCommandEnvelope(session, mgr, registry, supervisor, envelope); err != nil {
			return err
		}
	}
}

func handleCommandEnvelope(session *clientSession, mgr *manager.Manager, registry *agentregistry.Registry, supervisor *manager.Supervisor, envelope protocol.Envelope) error {
	resolveAgentID := func(agentID string) (string, error) {
		if supervisor == nil {
			if strings.TrimSpace(agentID) != "" {
				return strings.TrimSpace(agentID), nil
			}
			if registry != nil {
				names := registry.Names()
				if len(names) == 1 {
					return names[0], nil
				}
			}
			return "", fmt.Errorf("agentId is required")
		}
		return supervisor.ResolveAgentID(agentID)
	}

	switch envelope.Name {
	case "thread.create":
		var payload protocol.ThreadCreateCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		thread, err := mgr.CreateThread(agentID, agenttypes.CreateThreadParams{
			Title: payload.Title,
		})
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if thread.MachineID == "" {
			thread.MachineID = session.machineID
		}
		thread.AgentID = agentID

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadCreateCommandResult{
			Thread: thread,
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, registry)
	case "thread.read":
		var payload protocol.ThreadReadCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		thread, err := mgr.ReadThread(agentID, payload.ThreadID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}
		if thread.MachineID == "" {
			thread.MachineID = session.machineID
		}
		thread.AgentID = agentID

		return session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadReadCommandResult{
			Thread: thread,
		})
	case "thread.resume":
		var payload protocol.ThreadResumeCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		thread, err := mgr.ResumeThread(agentID, payload.ThreadID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}
		if thread.MachineID == "" {
			thread.MachineID = session.machineID
		}
		thread.AgentID = agentID

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadResumeCommandResult{
			Thread: thread,
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, registry)
	case "thread.archive":
		var payload protocol.ThreadArchiveCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		if err := mgr.ArchiveThread(agentID, payload.ThreadID); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadArchiveCommandResult{
			ThreadID: payload.ThreadID,
		}); err != nil {
			return err
		}

		return refreshThreadSnapshot(session, mgr, registry)
	case "turn.start":
		var payload protocol.TurnStartCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		result, err := mgr.StartTurn(agentID, agenttypes.StartTurnParams{
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
		if runtimeStreamsTurnEventsForAgent(registry, agentID) {
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

		return refreshThreadSnapshot(session, mgr, registry)
	case "turn.steer":
		var payload protocol.TurnSteerCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		result, err := mgr.SteerTurn(agentID, agenttypes.SteerTurnParams{
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
		if runtimeStreamsTurnEventsForAgent(registry, agentID) {
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

		return refreshThreadSnapshot(session, mgr, registry)
	case "turn.interrupt":
		var payload protocol.TurnInterruptCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		turn, err := mgr.InterruptTurn(agentID, agenttypes.InterruptTurnParams{
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

		return refreshThreadSnapshot(session, mgr, registry)
	case "approval.respond":
		var payload protocol.ApprovalRespondCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		rawRequestID, ok := session.rawApprovalRequestID(payload.RequestID)
		if !ok {
			return session.CommandRejected(envelope.RequestID, envelope.Name, fmt.Sprintf("approval request %q not found", payload.RequestID), "")
		}
		approvalCtx := session.approvalContext(payload.RequestID)
		agentID, err := resolveAgentID(approvalCtx.agentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		if err := mgr.RespondApproval(agentID, rawRequestID, payload.Decision, payload.Answers); err != nil {
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
		if supervisor == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime supervisor unavailable", "")
		}
		if err := supervisor.StopAll(); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.RuntimeStopCommandResult{}); err != nil {
			return err
		}
		return sendLiveSnapshot(session, buildMachineSnapshot(session.machineID, supervisor), mgr, registry)
	case "runtime.start":
		var payload protocol.RuntimeStartCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if supervisor == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime supervisor unavailable", "")
		}
		if err := supervisor.StartAll(); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		bindAllManagedRuntimeEvents(registry, session, mgr)
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.RuntimeStartCommandResult{}); err != nil {
			return err
		}
		return sendLiveSnapshot(session, buildMachineSnapshot(session.machineID, supervisor), mgr, registry)
	case "machine.agent.install":
		var payload protocol.MachineAgentInstallCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if supervisor == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime supervisor unavailable", "")
		}
		agentType := domain.AgentType(strings.TrimSpace(payload.AgentType))
		agent, err := supervisor.InstallAgent(agentType, payload.DisplayName)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		bindManagedRuntimeEvents(agent.AgentID, registry, session, mgr)
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.MachineAgentInstallCommandResult{
			Agent: agent,
		}); err != nil {
			return err
		}
		return sendLiveSnapshot(session, buildMachineSnapshot(session.machineID, supervisor), mgr, registry)
	case "machine.agent.delete":
		var payload protocol.MachineAgentDeleteCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if supervisor == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime supervisor unavailable", "")
		}
		if err := supervisor.DeleteAgent(payload.AgentID); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.MachineAgentDeleteCommandResult{
			AgentID: payload.AgentID,
		}); err != nil {
			return err
		}
		return sendLiveSnapshot(session, buildMachineSnapshot(session.machineID, supervisor), mgr, registry)
	case "machine.agent.config.read":
		var payload protocol.MachineAgentConfigReadCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if supervisor == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime supervisor unavailable", "")
		}
		document, err := supervisor.ReadConfig(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		return session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.MachineAgentConfigReadCommandResult{
			Document: document,
		})
	case "machine.agent.config.write":
		var payload protocol.MachineAgentConfigWriteCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if supervisor == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime supervisor unavailable", "")
		}
		document, err := supervisor.WriteConfig(payload.AgentID, payload.Document)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		return session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.MachineAgentConfigWriteCommandResult{
			Document: document,
		})
	case "agent.config.apply":
		var payload protocol.AgentConfigApplyCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if supervisor == nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime supervisor unavailable", "")
		}
		agentType := payload.Document.AgentType
		if strings.TrimSpace(payload.AgentType) != "" {
			agentType = domain.AgentType(payload.AgentType)
		}
		result, err := supervisor.ApplyConfigToType(agentType, payload.Document)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.AgentConfigApplyCommandResult{
			AgentType: string(result.AgentType),
			FilePath:  result.FilePath,
			Source:    payload.Source,
		}); err != nil {
			return err
		}
		return nil
	case "environment.skill.enable", "environment.skill.disable":
		var payload protocol.EnvironmentSkillSetEnabledCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.SetSkillEnabled(agentID, payload.SkillID, payload.Enabled); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentSkillSetEnabledCommandResult{
			SkillID: payload.SkillID,
			Enabled: payload.Enabled,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	case "environment.skill.create":
		var payload protocol.EnvironmentSkillCreateCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		skillID, err := mgr.CreateSkill(agentID, agenttypes.CreateSkillParams{
			Name:        payload.Name,
			Description: payload.Description,
		})
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentSkillCreateCommandResult{
			SkillID: skillID,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	case "environment.skill.delete":
		var payload protocol.EnvironmentSkillDeleteCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.DeleteSkill(agentID, payload.SkillID); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentSkillDeleteCommandResult{
			SkillID: payload.SkillID,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	case "environment.mcp.upsert":
		var payload protocol.EnvironmentMCPUpsertCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.UpsertMCPServer(agentID, payload.ServerID, payload.Config); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentMCPUpsertCommandResult{
			ServerID: payload.ServerID,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	case "environment.mcp.enable", "environment.mcp.disable":
		var payload protocol.EnvironmentMCPSetEnabledCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.SetMCPServerEnabled(agentID, payload.ServerID, payload.Enabled); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentMCPSetEnabledCommandResult{
			ServerID: payload.ServerID,
			Enabled:  payload.Enabled,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	case "environment.mcp.remove":
		var payload protocol.EnvironmentMCPRemoveCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.RemoveMCPServer(agentID, payload.ServerID); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentMCPRemoveCommandResult{
			ServerID: payload.ServerID,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	case "environment.plugin.install":
		var payload protocol.EnvironmentPluginInstallCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.InstallPlugin(agentID, agenttypes.InstallPluginParams{
			PluginID:        payload.PluginID,
			MarketplacePath: payload.MarketplacePath,
			PluginName:      payload.PluginName,
		}); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentPluginInstallCommandResult{
			PluginID: payload.PluginID,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	case "environment.plugin.enable", "environment.plugin.disable":
		var payload protocol.EnvironmentPluginSetEnabledCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.SetPluginEnabled(agentID, payload.PluginID, payload.Enabled); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentPluginSetEnabledCommandResult{
			PluginID: payload.PluginID,
			Enabled:  payload.Enabled,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	case "environment.plugin.uninstall":
		var payload protocol.EnvironmentPluginUninstallCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		agentID, err := resolveAgentID(payload.AgentID)
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := mgr.UninstallPlugin(agentID, payload.PluginID); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}
		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.EnvironmentPluginUninstallCommandResult{
			PluginID: payload.PluginID,
		}); err != nil {
			return err
		}
		return refreshEnvironmentSnapshot(session, mgr, registry)
	default:
		return session.CommandRejected(envelope.RequestID, envelope.Name, "unsupported command", "")
	}
}

func refreshThreadSnapshot(session *clientSession, mgr *manager.Manager, registry *agentregistry.Registry) error {
	snap, err := collectManagedSnapshot(session.machineID, mgr, registry)
	if err != nil {
		return err
	}

	return session.ThreadSnapshot(snap.Threads)
}

func refreshEnvironmentSnapshot(session *clientSession, mgr *manager.Manager, registry *agentregistry.Registry) error {
	snap, err := collectManagedSnapshot(session.machineID, mgr, registry)
	if err != nil {
		return err
	}

	return session.EnvironmentSnapshot(snap.Environment)
}

func buildMachineSnapshot(machineID string, supervisor *manager.Supervisor) domain.Machine {
	machine := domain.Machine{
		ID:     machineID,
		Name:   machineID,
		Status: domain.MachineStatusOnline,
	}
	if supervisor == nil {
		machine.RuntimeStatus = domain.MachineRuntimeStatusUnknown
		return machine
	}

	machine.Agents = supervisor.AgentInstances()
	machine.RuntimeStatus = domain.MachineRuntimeStatusStopped
	for _, agent := range machine.Agents {
		if agent.Status == domain.AgentInstanceStatusRunning {
			machine.RuntimeStatus = domain.MachineRuntimeStatusRunning
			break
		}
	}
	return machine
}

func bindAllManagedRuntimeEvents(registry *agentregistry.Registry, session *clientSession, mgr *manager.Manager) bool {
	if registry == nil {
		return false
	}

	bound := false
	for _, agentID := range registry.Names() {
		if bindManagedRuntimeEvents(agentID, registry, session, mgr) {
			bound = true
		}
	}
	return bound
}

func bindManagedRuntimeEvents(agentID string, registry *agentregistry.Registry, session *clientSession, mgr *manager.Manager) bool {
	if registry == nil || strings.TrimSpace(agentID) == "" {
		return false
	}
	runtime, ok := registry.Get(agentID)
	if !ok {
		return false
	}

	streams := bindRuntimeTurnEvents(runtime, session, mgr, registry, agentID)
	bindRuntimeApprovalEvents(runtime, session, agentID)
	return streams
}

func runtimeStreamsTurnEventsForAgent(registry *agentregistry.Registry, agentID string) bool {
	if registry == nil || strings.TrimSpace(agentID) == "" {
		return false
	}
	runtime, ok := registry.Get(agentID)
	if !ok {
		return false
	}
	_, ok = runtime.(agenttypes.RuntimeTurnEventSource)
	return ok
}

func collectManagedSnapshot(machineID string, mgr *manager.Manager, registry *agentregistry.Registry) (snapshot.Snapshot, error) {
	if mgr == nil || registry == nil {
		return snapshot.Snapshot{}, nil
	}

	threads := make([]domain.Thread, 0)
	environment := make([]domain.EnvironmentResource, 0)
	for _, agentID := range registry.Names() {
		agentThreads, err := mgr.Threads(agentID)
		if err != nil {
			return snapshot.Snapshot{}, err
		}
		for _, thread := range agentThreads {
			if thread.MachineID == "" {
				thread.MachineID = machineID
			}
			if thread.AgentID == "" {
				thread.AgentID = agentID
			}
			threads = append(threads, thread)
		}

		agentEnvironment, err := mgr.Environment(agentID)
		if err != nil {
			return snapshot.Snapshot{}, err
		}
		for _, resource := range agentEnvironment {
			if resource.MachineID == "" {
				resource.MachineID = machineID
			}
			if resource.AgentID == "" {
				resource.AgentID = agentID
			}
			environment = append(environment, resource)
		}
	}

	sort.Slice(threads, func(i int, j int) bool {
		if threads[i].ThreadID == threads[j].ThreadID {
			return threads[i].AgentID < threads[j].AgentID
		}
		return threads[i].ThreadID < threads[j].ThreadID
	})
	sort.Slice(environment, func(i int, j int) bool {
		if environment[i].Kind == environment[j].Kind {
			if environment[i].ResourceID == environment[j].ResourceID {
				return environment[i].AgentID < environment[j].AgentID
			}
			return environment[i].ResourceID < environment[j].ResourceID
		}
		return environment[i].Kind < environment[j].Kind
	})

	return snapshot.Snapshot{
		Threads:     threads,
		Environment: environment,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
