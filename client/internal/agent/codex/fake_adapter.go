package codex

import (
	"fmt"
	"sync"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type FakeAdapter struct {
	mu          sync.RWMutex
	threads     []domain.Thread
	environment []domain.EnvironmentResource
	nextThread  int
	nextTurn    int
}

func NewFakeAdapter() *FakeAdapter {
	return &FakeAdapter{
		threads:     []domain.Thread{},
		environment: []domain.EnvironmentResource{},
		nextThread:  1,
		nextTurn:    1,
	}
}

func (a *FakeAdapter) SeedSnapshot(threads []domain.Thread, environment []domain.EnvironmentResource) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.threads = append([]domain.Thread(nil), threads...)
	a.environment = append([]domain.EnvironmentResource(nil), environment...)
	a.nextThread = len(a.threads) + 1
}

func (a *FakeAdapter) ListThreads() ([]domain.Thread, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return append([]domain.Thread(nil), a.threads...), nil
}

func (a *FakeAdapter) ListEnvironment() ([]domain.EnvironmentResource, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return append([]domain.EnvironmentResource(nil), a.environment...), nil
}

func (a *FakeAdapter) CreateThread(params agenttypes.CreateThreadParams) (domain.Thread, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	thread := domain.Thread{
		ThreadID: fmt.Sprintf("thread-%02d", a.nextThread),
		Status:   domain.ThreadStatusIdle,
		Title:    params.Title,
	}
	a.nextThread++
	a.threads = append(a.threads, thread)

	return thread, nil
}

func (a *FakeAdapter) StartTurn(params agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.hasThread(params.ThreadID) {
		return agenttypes.StartTurnResult{}, fmt.Errorf("thread %q not found", params.ThreadID)
	}

	result := agenttypes.StartTurnResult{
		TurnID:   fmt.Sprintf("turn-%02d", a.nextTurn),
		ThreadID: params.ThreadID,
		Deltas: []agenttypes.TurnDelta{
			{Sequence: 1, Delta: "assistant: thinking"},
			{Sequence: 2, Delta: "assistant: done"},
		},
	}
	a.nextTurn++

	return result, nil
}

func (a *FakeAdapter) hasThread(threadID string) bool {
	for _, thread := range a.threads {
		if thread.ThreadID == threadID {
			return true
		}
	}

	return false
}
