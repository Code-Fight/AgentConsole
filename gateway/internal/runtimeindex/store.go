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

func (s *Store) ReplaceSnapshot(threads []domain.Thread, environment []domain.EnvironmentResource) {
	threadsByMachine := groupThreadsByMachine(threads)
	environmentByMachine := groupEnvironmentByMachine(environment)
	machineIDs := collectMachineIDs(threadsByMachine, environmentByMachine)

	if len(machineIDs) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, machineID := range machineIDs {
		s.threadsByMachine[machineID] = cloneThreads(threadsByMachine[machineID])
		s.environmentByMachine[machineID] = cloneEnvironment(environmentByMachine[machineID])
	}
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

func groupThreadsByMachine(threads []domain.Thread) map[string][]domain.Thread {
	items := make(map[string][]domain.Thread)
	for _, item := range threads {
		items[item.MachineID] = append(items[item.MachineID], item)
	}
	return items
}

func groupEnvironmentByMachine(environment []domain.EnvironmentResource) map[string][]domain.EnvironmentResource {
	items := make(map[string][]domain.EnvironmentResource)
	for _, item := range environment {
		items[item.MachineID] = append(items[item.MachineID], item)
	}
	return items
}

func collectMachineIDs(
	threadsByMachine map[string][]domain.Thread,
	environmentByMachine map[string][]domain.EnvironmentResource,
) []string {
	items := make([]string, 0, len(threadsByMachine)+len(environmentByMachine))
	seen := map[string]bool{}

	for machineID := range threadsByMachine {
		if seen[machineID] {
			continue
		}
		seen[machineID] = true
		items = append(items, machineID)
	}

	for machineID := range environmentByMachine {
		if seen[machineID] {
			continue
		}
		seen[machineID] = true
		items = append(items, machineID)
	}

	return items
}

func cloneThreads(threads []domain.Thread) []domain.Thread {
	return append([]domain.Thread(nil), threads...)
}

func cloneEnvironment(environment []domain.EnvironmentResource) []domain.EnvironmentResource {
	return append([]domain.EnvironmentResource(nil), environment...)
}
