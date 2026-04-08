package routing

import (
	"sync"

	"code-agent-gateway/common/domain"
)

type Router struct {
	mu      sync.RWMutex
	threads map[string]string
}

func NewRouter() *Router {
	return &Router{threads: map[string]string{}}
}

func (r *Router) TrackThread(threadID string, machineID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.threads[threadID] = machineID
}

func (r *Router) ReplaceSnapshot(machineID string, threads []domain.Thread) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for threadID, ownerMachineID := range r.threads {
		if ownerMachineID == machineID {
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
		r.threads[thread.ThreadID] = ownerMachineID
	}
}

func (r *Router) ClearMachine(machineID string) {
	if machineID == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for threadID, ownerMachineID := range r.threads {
		if ownerMachineID == machineID {
			delete(r.threads, threadID)
		}
	}
}

func (r *Router) ResolveThread(threadID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	machineID, ok := r.threads[threadID]
	return machineID, ok
}
