package routing

import (
	"sync"

	"code-agent-gateway/common/domain"
)

type Router struct {
	mu      sync.RWMutex
	threads map[string]domain.ThreadRoute
}

func NewRouter() *Router {
	return &Router{threads: map[string]domain.ThreadRoute{}}
}

func (r *Router) TrackThread(threadID string, machineID string, agentID string) {
	if threadID == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.threads[threadID] = domain.ThreadRoute{
		MachineID: machineID,
		AgentID:   agentID,
	}
}

func (r *Router) ReplaceSnapshot(machineID string, threads []domain.Thread) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for threadID, route := range r.threads {
		if route.MachineID == machineID {
			delete(r.threads, threadID)
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
		r.threads[thread.ThreadID] = domain.ThreadRoute{
			MachineID: ownerMachineID,
			AgentID:   thread.AgentID,
		}
	}
}

func (r *Router) ClearMachine(machineID string) {
	if machineID == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for threadID, route := range r.threads {
		if route.MachineID == machineID {
			delete(r.threads, threadID)
		}
	}
}

func (r *Router) ResolveThread(threadID string) (domain.ThreadRoute, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	route, ok := r.threads[threadID]
	return route, ok
}
