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
	turns       map[string]domain.Turn
	environment []domain.EnvironmentResource
	nextThread  int
	nextTurn    int
}

func NewFakeAdapter() *FakeAdapter {
	return &FakeAdapter{
		threads:     []domain.Thread{},
		turns:       map[string]domain.Turn{},
		environment: []domain.EnvironmentResource{},
		nextThread:  1,
		nextTurn:    1,
	}
}

func (a *FakeAdapter) SeedSnapshot(threads []domain.Thread, environment []domain.EnvironmentResource) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.threads = append([]domain.Thread(nil), threads...)
	a.turns = map[string]domain.Turn{}
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

func (a *FakeAdapter) ReadThread(threadID string) (domain.Thread, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	index := a.findThreadIndex(threadID)
	if index < 0 {
		return domain.Thread{}, fmt.Errorf("thread %q not found", threadID)
	}

	return a.threads[index], nil
}

func (a *FakeAdapter) ResumeThread(threadID string) (domain.Thread, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	index := a.findThreadIndex(threadID)
	if index < 0 {
		return domain.Thread{}, fmt.Errorf("thread %q not found", threadID)
	}

	a.threads[index].Status = domain.ThreadStatusIdle
	return a.threads[index], nil
}

func (a *FakeAdapter) ArchiveThread(threadID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	index := a.findThreadIndex(threadID)
	if index < 0 {
		return fmt.Errorf("thread %q not found", threadID)
	}

	a.threads[index].Status = domain.ThreadStatusNotLoaded
	return nil
}

func (a *FakeAdapter) StartTurn(params agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.findThreadIndex(params.ThreadID) < 0 {
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
	a.turns[result.TurnID] = domain.Turn{
		TurnID:   result.TurnID,
		ThreadID: params.ThreadID,
		Status:   domain.TurnStatusCompleted,
	}

	return result, nil
}

func (a *FakeAdapter) SteerTurn(params agenttypes.SteerTurnParams) (agenttypes.SteerTurnResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	turn, ok := a.turns[params.TurnID]
	if !ok || turn.ThreadID != params.ThreadID {
		return agenttypes.SteerTurnResult{}, fmt.Errorf("turn %q not found", params.TurnID)
	}

	return agenttypes.SteerTurnResult{
		TurnID:   params.TurnID,
		ThreadID: params.ThreadID,
		Deltas: []agenttypes.TurnDelta{
			{Sequence: 1, Delta: "assistant: steer accepted"},
			{Sequence: 2, Delta: "assistant: updated"},
		},
	}, nil
}

func (a *FakeAdapter) InterruptTurn(params agenttypes.InterruptTurnParams) (domain.Turn, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	turn, ok := a.turns[params.TurnID]
	if !ok || turn.ThreadID != params.ThreadID {
		return domain.Turn{}, fmt.Errorf("turn %q not found", params.TurnID)
	}

	turn.Status = domain.TurnStatusInterrupted
	a.turns[params.TurnID] = turn
	return turn, nil
}

func (a *FakeAdapter) findThreadIndex(threadID string) int {
	for idx, thread := range a.threads {
		if thread.ThreadID == threadID {
			return idx
		}
	}

	return -1
}
