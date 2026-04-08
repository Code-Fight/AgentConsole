package runtimeindex

import (
	"sync"

	"code-agent-gateway/common/domain"
)

type Store struct {
	mu                   sync.RWMutex
	threadsByMachine     map[string][]domain.Thread
	environmentByMachine map[string][]domain.EnvironmentResource
}

func NewStore() *Store {
	return &Store{
		threadsByMachine:     map[string][]domain.Thread{},
		environmentByMachine: map[string][]domain.EnvironmentResource{},
	}
}

func (s *Store) ReplaceSnapshot(machineID string, threads []domain.Thread, environment []domain.EnvironmentResource) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.threadsByMachine[machineID] = cloneThreads(threads)
	s.environmentByMachine[machineID] = cloneEnvironment(environment)
}

func (s *Store) ClearMachine(machineID string) {
	if machineID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.threadsByMachine, machineID)
	delete(s.environmentByMachine, machineID)
}

func (s *Store) MarkMachineUnknown(machineID string) {
	if machineID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	threads := s.threadsByMachine[machineID]
	if len(threads) == 0 {
		return
	}

	updated := cloneThreads(threads)
	for idx := range updated {
		updated[idx].Status = domain.ThreadStatusUnknown
	}
	s.threadsByMachine[machineID] = updated
}

func (s *Store) Threads() []domain.Thread {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.threadsByMachine) == 0 {
		return []domain.Thread{}
	}

	items := make([]domain.Thread, 0)
	for _, machineThreads := range s.threadsByMachine {
		items = append(items, machineThreads...)
	}
	return items
}

func (s *Store) Environment(kind domain.EnvironmentKind) []domain.EnvironmentResource {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.EnvironmentResource, 0)
	for _, machineEnvironment := range s.environmentByMachine {
		for _, item := range machineEnvironment {
			if item.Kind == kind {
				items = append(items, item)
			}
		}
	}
	return items
}

func cloneThreads(threads []domain.Thread) []domain.Thread {
	return append([]domain.Thread(nil), threads...)
}

func cloneEnvironment(environment []domain.EnvironmentResource) []domain.EnvironmentResource {
	return append([]domain.EnvironmentResource(nil), environment...)
}
