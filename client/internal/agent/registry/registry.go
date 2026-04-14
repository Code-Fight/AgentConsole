package registry

import (
	"sort"
	"sync"

	"code-agent-gateway/client/internal/agent/types"
)

type Registry struct {
	mu       sync.RWMutex
	runtimes map[string]types.Runtime
}

func New() *Registry {
	return &Registry{
		runtimes: map[string]types.Runtime{},
	}
}

func (r *Registry) Register(name string, runtime types.Runtime) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runtimes[name] = runtime
}

func (r *Registry) Get(name string) (types.Runtime, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rt, ok := r.runtimes[name]
	return rt, ok
}

func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.runtimes, name)
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.runtimes))
	for name := range r.runtimes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
