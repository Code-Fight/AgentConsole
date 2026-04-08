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
	registry          *registry.Store
	runtimeIndex      *runtimeindex.Store
	router            *routing.Router
	snapshotByMachine map[string]machineSnapshotState
	pendingCommands   map[string]chan protocol.CommandCompletedPayload
	nextRequestID     atomic.Uint64
}

type machineSnapshotState struct {
	threads     []domain.Thread
	environment []domain.EnvironmentResource
}

type clientConn struct {
	conn    *cws.Conn
	writeMu sync.Mutex
}

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
		pendingCommands:   map[string]chan protocol.CommandCompletedPayload{},
	}
}

func (h *ClientHub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
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
			if registeredMachineID == "" {
				return
			}

			markOffline := false
			h.mu.Lock()
			if existing := h.clients[registeredMachineID]; existing != nil && existing.conn == conn {
				delete(h.clients, registeredMachineID)
				markOffline = true
			}
			h.mu.Unlock()

			if markOffline && h.registry != nil {
				h.registry.MarkOffline(registeredMachineID)
			}
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
				h.handleEventEnvelope(envelope)
			case protocol.CategorySnapshot:
				h.handleSnapshotEnvelope(envelope)
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

		previousMachineID := ""
		h.mu.Lock()
		if *registeredMachineID != "" && *registeredMachineID != envelope.MachineID {
			if existing := h.clients[*registeredMachineID]; existing != nil && existing.conn == conn {
				delete(h.clients, *registeredMachineID)
				previousMachineID = *registeredMachineID
			}
		}
		h.clients[envelope.MachineID] = &clientConn{conn: conn}
		*registeredMachineID = envelope.MachineID
		h.mu.Unlock()

		if previousMachineID != "" && h.registry != nil {
			h.registry.MarkOffline(previousMachineID)
		}

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

func (h *ClientHub) handleEventEnvelope(envelope protocol.Envelope) {
	if envelope.Name != "command.completed" || envelope.RequestID == "" {
		return
	}

	var payload protocol.CommandCompletedPayload
	if err := transport.Decode(envelope.Payload, &payload); err != nil {
		return
	}

	h.mu.Lock()
	waiter := h.pendingCommands[envelope.RequestID]
	delete(h.pendingCommands, envelope.RequestID)
	h.mu.Unlock()

	if waiter == nil {
		return
	}

	select {
	case waiter <- payload:
	default:
	}
}

func (h *ClientHub) SendCommand(ctx context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
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

	responseCh := make(chan protocol.CommandCompletedPayload, 1)

	h.mu.Lock()
	h.pendingCommands[requestID] = responseCh
	h.mu.Unlock()

	client.writeMu.Lock()
	err = client.conn.Write(ctx, cws.MessageText, encoded)
	client.writeMu.Unlock()
	if err != nil {
		h.mu.Lock()
		delete(h.pendingCommands, requestID)
		h.mu.Unlock()
		return protocol.CommandCompletedPayload{}, err
	}

	select {
	case <-ctx.Done():
		h.mu.Lock()
		delete(h.pendingCommands, requestID)
		h.mu.Unlock()
		return protocol.CommandCompletedPayload{}, ctx.Err()
	case response := <-responseCh:
		return response, nil
	}
}

func (h *ClientHub) handleSnapshotEnvelope(envelope protocol.Envelope) {
	if envelope.MachineID == "" {
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
