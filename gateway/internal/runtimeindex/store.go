package runtimeindex

import (
	"strings"
	"sync"
	"time"

	"code-agent-gateway/common/domain"
)

type Store struct {
	mu                   sync.RWMutex
	threadsByMachine     map[string][]domain.Thread
	environmentByMachine map[string][]domain.EnvironmentResource
	threadRoutes         map[string]domain.ThreadRoute
}

func NewStore() *Store {
	return &Store{
		threadsByMachine:     map[string][]domain.Thread{},
		environmentByMachine: map[string][]domain.EnvironmentResource{},
		threadRoutes:         map[string]domain.ThreadRoute{},
	}
}

func (s *Store) ReplaceSnapshot(machineID string, threads []domain.Thread, environment []domain.EnvironmentResource) {
	s.mu.Lock()
	defer s.mu.Unlock()

	previousByThreadID := map[string]domain.Thread{}
	for _, item := range s.threadsByMachine[machineID] {
		if strings.TrimSpace(item.ThreadID) == "" {
			continue
		}
		previousByThreadID[item.ThreadID] = item
	}

	nextThreads := cloneThreads(threads)
	for idx := range nextThreads {
		threadID := strings.TrimSpace(nextThreads[idx].ThreadID)
		if threadID == "" {
			continue
		}
		if existing, ok := previousByThreadID[threadID]; ok {
			nextThreads[idx].LastActivityAt = latestThreadActivity(existing.LastActivityAt, nextThreads[idx].LastActivityAt)
		}
	}

	s.threadsByMachine[machineID] = nextThreads
	s.environmentByMachine[machineID] = cloneEnvironment(environment)
	for threadID, route := range s.threadRoutes {
		if route.MachineID == machineID {
			delete(s.threadRoutes, threadID)
		}
	}
	for _, thread := range threads {
		if thread.ThreadID == "" {
			continue
		}
		ownerMachineID := thread.MachineID
		if ownerMachineID == "" {
			ownerMachineID = machineID
		}
		s.threadRoutes[thread.ThreadID] = domain.ThreadRoute{
			MachineID: ownerMachineID,
			AgentID:   thread.AgentID,
		}
	}
}

func (s *Store) UpsertThread(machineID string, thread domain.Thread) {
	if s == nil || machineID == "" || thread.ThreadID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if thread.MachineID == "" {
		thread.MachineID = machineID
	}

	items := cloneThreads(s.threadsByMachine[machineID])
	replaced := false
	for idx := range items {
		if items[idx].ThreadID != thread.ThreadID {
			continue
		}
		thread.LastActivityAt = latestThreadActivity(items[idx].LastActivityAt, thread.LastActivityAt)
		items[idx] = thread
		replaced = true
		break
	}
	if !replaced {
		items = append(items, thread)
	}
	s.threadsByMachine[machineID] = items
	s.threadRoutes[thread.ThreadID] = domain.ThreadRoute{
		MachineID: thread.MachineID,
		AgentID:   thread.AgentID,
	}
}

func (s *Store) RemoveThread(machineID string, threadID string) {
	if s == nil || machineID == "" || threadID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.threadsByMachine[machineID]
	filtered := make([]domain.Thread, 0, len(items))
	for _, thread := range items {
		if thread.ThreadID == threadID {
			continue
		}
		filtered = append(filtered, thread)
	}
	s.threadsByMachine[machineID] = filtered
	delete(s.threadRoutes, threadID)
}

func (s *Store) ClearMachine(machineID string) {
	if machineID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.threadsByMachine, machineID)
	delete(s.environmentByMachine, machineID)
	for threadID, route := range s.threadRoutes {
		if route.MachineID == machineID {
			delete(s.threadRoutes, threadID)
		}
	}
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

func (s *Store) ThreadRoute(threadID string) (domain.ThreadRoute, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	route, ok := s.threadRoutes[threadID]
	return route, ok
}

func (s *Store) OverviewMetrics() domain.OverviewMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := domain.OverviewMetrics{}
	for _, threads := range s.threadsByMachine {
		for _, thread := range threads {
			if thread.Status == domain.ThreadStatusActive {
				metrics.ActiveThreads++
			}
		}
	}
	for _, items := range s.environmentByMachine {
		metrics.EnvironmentItems += len(items)
	}
	return metrics
}

func cloneThreads(threads []domain.Thread) []domain.Thread {
	return append([]domain.Thread(nil), threads...)
}

func cloneEnvironment(environment []domain.EnvironmentResource) []domain.EnvironmentResource {
	return append([]domain.EnvironmentResource(nil), environment...)
}

func parseThreadActivity(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

func normalizeThreadActivity(raw string) string {
	parsed, ok := parseThreadActivity(raw)
	if !ok {
		return ""
	}
	return parsed.Format(time.RFC3339)
}

func latestThreadActivity(current string, candidate string) string {
	current = normalizeThreadActivity(current)
	candidate = normalizeThreadActivity(candidate)
	if current == "" {
		return candidate
	}
	if candidate == "" {
		return current
	}

	currentTime, _ := parseThreadActivity(current)
	candidateTime, _ := parseThreadActivity(candidate)
	if candidateTime.After(currentTime) {
		return candidate
	}
	return current
}
