package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/routing"
	"code-agent-gateway/gateway/internal/runtimeindex"
	cws "github.com/coder/websocket"
)

type ClientHub struct {
	mu                sync.RWMutex
	clients           map[string]*clientConn
	consoleHub        *ConsoleHub
	registry          *registry.Store
	runtimeIndex      *runtimeindex.Store
	router            *routing.Router
	snapshotByMachine map[string]machineSnapshotState
	approvalRequests  map[string]string
	pendingCommands   map[string]pendingCommandWaiter
	commandTimeout    time.Duration
	nextRequestID     atomic.Uint64
}

type machineSnapshotState struct {
	threads     []domain.Thread
	environment []domain.EnvironmentResource
}

type pendingCommandResult struct {
	response protocol.CommandCompletedPayload
	err      error
}

type pendingCommandWaiter struct {
	machineID string
	ch        chan pendingCommandResult
}

type clientConn struct {
	conn    *cws.Conn
	writeMu sync.Mutex
}

type CommandRejectedError struct {
	CommandName string
	Reason      string
}

type MachineDisconnectedError struct {
	MachineID string
}

func (e *CommandRejectedError) Error() string {
	if e == nil {
		return "command rejected"
	}
	if e.Reason == "" {
		return fmt.Sprintf("command %q rejected", e.CommandName)
	}
	return fmt.Sprintf("command %q rejected: %s", e.CommandName, e.Reason)
}

func (e *MachineDisconnectedError) Error() string {
	if e == nil {
		return "machine disconnected"
	}
	return fmt.Sprintf("machine %q disconnected", e.MachineID)
}

const defaultCommandTimeout = 30 * time.Second

func NewClientHub() *ClientHub {
	return NewClientHubWithStores(nil, nil)
}

func NewClientHubWithStores(reg *registry.Store, idx *runtimeindex.Store, routers ...*routing.Router) *ClientHub {
	var routerStore *routing.Router
	if len(routers) > 0 {
		routerStore = routers[0]
	}

	return &ClientHub{
		clients:           map[string]*clientConn{},
		registry:          reg,
		runtimeIndex:      idx,
		router:            routerStore,
		snapshotByMachine: map[string]machineSnapshotState{},
		approvalRequests:  map[string]string{},
		pendingCommands:   map[string]pendingCommandWaiter{},
		commandTimeout:    defaultCommandTimeout,
	}
}

func (h *ClientHub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}

func (h *ClientHub) SetConsoleHub(consoleHub *ConsoleHub) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.consoleHub = consoleHub
}

func (h *ClientHub) ResolveApprovalMachine(requestID string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	machineID, ok := h.approvalRequests[requestID]
	return machineID, ok
}

func (h *ClientHub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/client", func(w http.ResponseWriter, r *http.Request) {
		conn, err := cws.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(cws.StatusNormalClosure, "done")

		registeredMachineID := ""
		defer func() {
			h.disconnectClient(conn, registeredMachineID)
		}()

		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}

			var envelope protocol.Envelope
			if err := transport.Decode(data, &envelope); err != nil {
				continue
			}

			switch envelope.Category {
			case protocol.CategorySystem:
				h.handleSystemEnvelope(conn, envelope, &registeredMachineID)
			case protocol.CategoryEvent:
				h.handleEventEnvelope(conn, envelope)
			case protocol.CategorySnapshot:
				h.handleSnapshotEnvelope(conn, envelope)
			}
		}
	})

	return mux
}

func (h *ClientHub) handleSystemEnvelope(conn *cws.Conn, envelope protocol.Envelope, registeredMachineID *string) {
	switch envelope.Name {
	case "client.register":
		if envelope.MachineID == "" {
			return
		}

		var waiters []pendingCommandWaiter
		h.mu.Lock()
		if *registeredMachineID != "" && *registeredMachineID != envelope.MachineID {
			if existing := h.clients[*registeredMachineID]; existing != nil && existing.conn == conn {
				delete(h.clients, *registeredMachineID)
				waiters = h.cleanupMachineLocked(*registeredMachineID, true)
			}
		}
		h.clients[envelope.MachineID] = &clientConn{conn: conn}
		*registeredMachineID = envelope.MachineID
		h.mu.Unlock()
		h.failPendingWaiters(waiters)

		if h.registry != nil {
			h.registry.Upsert(domain.Machine{
				ID:     envelope.MachineID,
				Name:   envelope.MachineID,
				Status: domain.MachineStatusOnline,
			})
		}
	case "client.heartbeat":
		return
	}
}

func (h *ClientHub) handleEventEnvelope(conn *cws.Conn, envelope protocol.Envelope) {
	if !h.isCurrentOwner(conn, envelope.MachineID) {
		return
	}

	h.mu.RLock()
	consoleHub := h.consoleHub
	h.mu.RUnlock()
	if consoleHub != nil {
		_ = consoleHub.Broadcast(envelope)
	}

	if envelope.RequestID == "" {
		return
	}

	switch envelope.Name {
	case "command.completed":
		var payload protocol.CommandCompletedPayload
		if err := transport.Decode(envelope.Payload, &payload); err != nil {
			return
		}
		h.resolvePendingCommand(envelope.RequestID, pendingCommandResult{
			response: payload,
		})
	case "command.rejected":
		var payload protocol.CommandRejectedPayload
		if err := transport.Decode(envelope.Payload, &payload); err != nil {
			return
		}
		h.resolvePendingCommand(envelope.RequestID, pendingCommandResult{
			err: &CommandRejectedError{
				CommandName: payload.CommandName,
				Reason:      payload.Reason,
			},
		})
	case "approval.required":
		requestID := envelope.RequestID
		var payload protocol.ApprovalRequiredPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil && payload.RequestID != "" {
			requestID = payload.RequestID
		}
		if requestID == "" || envelope.MachineID == "" {
			return
		}
		payload.RequestID = requestID
		h.mu.Lock()
		h.approvalRequests[requestID] = envelope.MachineID
		h.mu.Unlock()
		if h.registry != nil {
			h.registry.UpsertPendingApproval(envelope.MachineID, payload)
		}
	case "approval.resolved":
		requestID := envelope.RequestID
		var payload protocol.ApprovalResolvedPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil && payload.RequestID != "" {
			requestID = payload.RequestID
		}
		if requestID == "" {
			return
		}
		h.mu.Lock()
		delete(h.approvalRequests, requestID)
		h.mu.Unlock()
		if h.registry != nil {
			h.registry.RemovePendingApproval(requestID)
		}
	}
}

func (h *ClientHub) resolvePendingCommand(requestID string, result pendingCommandResult) {
	h.mu.Lock()
	waiter := h.pendingCommands[requestID]
	delete(h.pendingCommands, requestID)
	h.mu.Unlock()

	if waiter.ch == nil {
		return
	}

	select {
	case waiter.ch <- result:
	default:
	}
}

func (h *ClientHub) SendCommand(ctx context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
	commandCtx, cancel := h.commandContext(ctx)
	defer cancel()

	requestID := fmt.Sprintf("req-%d", h.nextRequestID.Add(1))
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return protocol.CommandCompletedPayload{}, err
	}

	h.mu.RLock()
	client := h.clients[machineID]
	h.mu.RUnlock()
	if client == nil {
		return protocol.CommandCompletedPayload{}, fmt.Errorf("machine %q is not connected", machineID)
	}

	envelope := protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryCommand,
		Name:      name,
		RequestID: requestID,
		MachineID: machineID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payloadJSON,
	}

	encoded, err := transport.Encode(envelope)
	if err != nil {
		return protocol.CommandCompletedPayload{}, err
	}

	responseCh := make(chan pendingCommandResult, 1)

	h.mu.Lock()
	h.pendingCommands[requestID] = pendingCommandWaiter{
		machineID: machineID,
		ch:        responseCh,
	}
	h.mu.Unlock()

	client.writeMu.Lock()
	err = client.conn.Write(commandCtx, cws.MessageText, encoded)
	client.writeMu.Unlock()
	if err != nil {
		h.mu.Lock()
		delete(h.pendingCommands, requestID)
		h.mu.Unlock()
		return protocol.CommandCompletedPayload{}, err
	}

	select {
	case <-commandCtx.Done():
		h.mu.Lock()
		delete(h.pendingCommands, requestID)
		h.mu.Unlock()
		return protocol.CommandCompletedPayload{}, commandCtx.Err()
	case response := <-responseCh:
		if response.err != nil {
			return protocol.CommandCompletedPayload{}, response.err
		}
		return response.response, nil
	}
}

func (h *ClientHub) commandContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), h.commandTimeout)
	}

	if _, ok := ctx.Deadline(); ok || h.commandTimeout <= 0 {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, h.commandTimeout)
}

func (h *ClientHub) disconnectClient(conn *cws.Conn, machineID string) {
	if machineID == "" {
		return
	}

	var waiters []pendingCommandWaiter
	h.mu.Lock()
	if existing := h.clients[machineID]; existing != nil && existing.conn == conn {
		delete(h.clients, machineID)
		waiters = h.cleanupMachineLocked(machineID, true)
	}
	h.mu.Unlock()

	h.failPendingWaiters(waiters)
}

func (h *ClientHub) cleanupMachineLocked(machineID string, markOffline bool) []pendingCommandWaiter {
	if machineID == "" {
		return nil
	}

	state, hasSnapshot := h.snapshotByMachine[machineID]
	if hasSnapshot {
		state.threads = markThreadsUnknown(state.threads)
		h.snapshotByMachine[machineID] = state
	}
	waiters := make([]pendingCommandWaiter, 0)
	for requestID, waiter := range h.pendingCommands {
		if waiter.machineID != machineID {
			continue
		}
		waiters = append(waiters, waiter)
		delete(h.pendingCommands, requestID)
	}

	if markOffline && h.registry != nil {
		h.registry.MarkOffline(machineID)
	}
	if h.runtimeIndex != nil {
		if hasSnapshot {
			h.runtimeIndex.ReplaceSnapshot(machineID, state.threads, state.environment)
		} else {
			h.runtimeIndex.MarkMachineUnknown(machineID)
		}
	}
	if h.router != nil {
		h.router.ClearMachine(machineID)
	}

	return waiters
}

func (h *ClientHub) failPendingWaiters(waiters []pendingCommandWaiter) {
	for _, waiter := range waiters {
		if waiter.ch == nil {
			continue
		}

		select {
		case waiter.ch <- pendingCommandResult{
			err: &MachineDisconnectedError{MachineID: waiter.machineID},
		}:
		default:
		}
	}
}

func (h *ClientHub) handleSnapshotEnvelope(conn *cws.Conn, envelope protocol.Envelope) {
	if !h.isCurrentOwner(conn, envelope.MachineID) {
		return
	}

	switch envelope.Name {
	case "machine.snapshot":
		if h.registry == nil {
			return
		}

		var payload protocol.MachineSnapshotPayload
		if err := transport.Decode(envelope.Payload, &payload); err != nil {
			return
		}

		machine := payload.Machine
		if machine.ID == "" {
			machine.ID = envelope.MachineID
		}
		if machine.Name == "" {
			machine.Name = machine.ID
		}
		if machine.Status == "" {
			machine.Status = domain.MachineStatusOnline
		}
		h.registry.Upsert(machine)
	case "thread.snapshot":
		var payload protocol.ThreadSnapshotPayload
		if err := transport.Decode(envelope.Payload, &payload); err != nil {
			return
		}

		threads := normalizeThreads(payload.Threads, envelope.MachineID)
		if h.runtimeIndex != nil {
			h.replaceMachineSnapshot(envelope.MachineID, threads, nil, true, false)
		}
		if h.router != nil {
			h.router.ReplaceSnapshot(envelope.MachineID, threads)
		}
	case "environment.snapshot":
		if h.runtimeIndex == nil {
			return
		}

		var payload protocol.EnvironmentSnapshotPayload
		if err := transport.Decode(envelope.Payload, &payload); err != nil {
			return
		}

		environment := normalizeEnvironment(payload.Environment, envelope.MachineID)
		h.replaceMachineSnapshot(envelope.MachineID, nil, environment, false, true)
	}
}

func (h *ClientHub) replaceMachineSnapshot(machineID string, threads []domain.Thread, environment []domain.EnvironmentResource, replaceThreads bool, replaceEnvironment bool) {
	h.mu.Lock()
	state := h.snapshotByMachine[machineID]
	if replaceThreads {
		state.threads = append([]domain.Thread(nil), threads...)
	}
	if replaceEnvironment {
		state.environment = append([]domain.EnvironmentResource(nil), environment...)
	}
	h.snapshotByMachine[machineID] = state
	threadsToStore := append([]domain.Thread(nil), state.threads...)
	environmentToStore := append([]domain.EnvironmentResource(nil), state.environment...)
	h.mu.Unlock()

	h.runtimeIndex.ReplaceSnapshot(machineID, threadsToStore, environmentToStore)
}

func normalizeThreads(items []domain.Thread, machineID string) []domain.Thread {
	normalized := make([]domain.Thread, 0, len(items))
	for _, item := range items {
		thread := item
		if thread.MachineID == "" {
			thread.MachineID = machineID
		}
		normalized = append(normalized, thread)
	}
	return normalized
}

func markThreadsUnknown(items []domain.Thread) []domain.Thread {
	if len(items) == 0 {
		return nil
	}

	updated := append([]domain.Thread(nil), items...)
	for idx := range updated {
		updated[idx].Status = domain.ThreadStatusUnknown
	}
	return updated
}

func normalizeEnvironment(items []domain.EnvironmentResource, machineID string) []domain.EnvironmentResource {
	normalized := make([]domain.EnvironmentResource, 0, len(items))
	for _, item := range items {
		resource := item
		if resource.MachineID == "" {
			resource.MachineID = machineID
		}
		normalized = append(normalized, resource)
	}
	return normalized
}

func (h *ClientHub) isCurrentOwner(conn *cws.Conn, machineID string) bool {
	if conn == nil || machineID == "" {
		return false
	}

	h.mu.RLock()
	current := h.clients[machineID]
	h.mu.RUnlock()
	return current != nil && current.conn == conn
}
