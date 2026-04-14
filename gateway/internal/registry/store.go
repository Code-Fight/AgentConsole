package registry

import (
	"sort"
	"sync"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
)

type Store struct {
	mu               sync.RWMutex
	machines         map[string]domain.Machine
	pendingApprovals map[string]storedApprovalRequest
}

type storedApprovalRequest struct {
	machineID string
	payload   protocol.ApprovalRequiredPayload
}

func NewStore() *Store {
	return &Store{
		machines:         map[string]domain.Machine{},
		pendingApprovals: map[string]storedApprovalRequest{},
	}
}

func (s *Store) List() []domain.Machine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.Machine, 0, len(s.machines))
	for _, item := range s.machines {
		items = append(items, item)
	}
	return items
}

func (s *Store) Get(machineID string) (domain.Machine, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	machine, ok := s.machines[machineID]
	return machine, ok
}

func (s *Store) Upsert(machine domain.Machine) {
	if machine.ID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.machines[machine.ID]; ok {
		if machine.Name == "" {
			machine.Name = existing.Name
		}
		if machine.RuntimeStatus == "" || machine.RuntimeStatus == domain.MachineRuntimeStatusUnknown {
			machine.RuntimeStatus = existing.RuntimeStatus
		}
		if len(machine.Agents) == 0 {
			machine.Agents = existing.Agents
		}
	}
	if machine.RuntimeStatus == "" {
		machine.RuntimeStatus = domain.MachineRuntimeStatusUnknown
	}
	s.machines[machine.ID] = machine
}

func (s *Store) MarkOffline(machineID string) {
	if machineID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	machine, ok := s.machines[machineID]
	if !ok {
		return
	}

	machine.Status = domain.MachineStatusOffline
	machine.RuntimeStatus = domain.MachineRuntimeStatusUnknown
	for idx := range machine.Agents {
		machine.Agents[idx].Status = domain.AgentInstanceStatusStopped
	}
	s.machines[machineID] = machine
}

func (s *Store) UpsertPendingApproval(machineID string, payload protocol.ApprovalRequiredPayload) {
	if payload.RequestID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pendingApprovals[payload.RequestID] = storedApprovalRequest{
		machineID: machineID,
		payload:   payload,
	}
}

func (s *Store) RemovePendingApproval(requestID string) {
	if requestID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pendingApprovals, requestID)
}

func (s *Store) PendingApproval(requestID string) (protocol.ApprovalRequiredPayload, bool) {
	if requestID == "" {
		return protocol.ApprovalRequiredPayload{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	approval, ok := s.pendingApprovals[requestID]
	if !ok {
		return protocol.ApprovalRequiredPayload{}, false
	}
	return approval.payload, true
}

func (s *Store) PendingApprovalsForThread(threadID string) []protocol.ApprovalRequiredPayload {
	if threadID == "" {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]protocol.ApprovalRequiredPayload, 0)
	for _, approval := range s.pendingApprovals {
		if approval.payload.ThreadID == threadID {
			items = append(items, approval.payload)
		}
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].RequestID < items[j].RequestID
	})
	return items
}
