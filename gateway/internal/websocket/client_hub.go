package websocket

import (
	"context"
	"net/http"
	"sync"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/runtimeindex"
	cws "github.com/coder/websocket"
)

type ClientHub struct {
	mu                sync.RWMutex
	clients           map[string]*cws.Conn
	registry          *registry.Store
	runtimeIndex      *runtimeindex.Store
	snapshotByMachine map[string]machineSnapshotState
}

type machineSnapshotState struct {
	threads     []domain.Thread
	environment []domain.EnvironmentResource
}

func NewClientHub() *ClientHub {
	return NewClientHubWithStores(nil, nil)
}

func NewClientHubWithStores(reg *registry.Store, idx *runtimeindex.Store) *ClientHub {
	return &ClientHub{
		clients:           map[string]*cws.Conn{},
		registry:          reg,
		runtimeIndex:      idx,
		snapshotByMachine: map[string]machineSnapshotState{},
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
			if h.clients[registeredMachineID] == conn {
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
		if *registeredMachineID != "" && *registeredMachineID != envelope.MachineID && h.clients[*registeredMachineID] == conn {
			delete(h.clients, *registeredMachineID)
			previousMachineID = *registeredMachineID
		}
		h.clients[envelope.MachineID] = conn
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
		if h.runtimeIndex == nil {
			return
		}

		var payload protocol.ThreadSnapshotPayload
		if err := transport.Decode(envelope.Payload, &payload); err != nil {
			return
		}

		threads := normalizeThreads(payload.Threads, envelope.MachineID)
		h.replaceMachineSnapshot(envelope.MachineID, threads, nil, true, false)
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
