package registry

import (
	"sync"

	"code-agent-gateway/common/domain"
)

type Store struct {
	mu       sync.RWMutex
	machines map[string]domain.Machine
}

func NewStore() *Store {
	return &Store{machines: map[string]domain.Machine{}}
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

func (s *Store) Upsert(machine domain.Machine) {
	if machine.ID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.machines[machineID] = machine
}
