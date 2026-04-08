package routing

import "sync"

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

func (r *Router) ResolveThread(threadID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	machineID, ok := r.threads[threadID]
	return machineID, ok
}
