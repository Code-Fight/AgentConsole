package registry

import "code-agent-gateway/client/internal/agent/types"

type Registry struct {
	runtimes map[string]types.Runtime
}

func New() *Registry {
	return &Registry{
		runtimes: map[string]types.Runtime{},
	}
}

func (r *Registry) Register(name string, runtime types.Runtime) {
	r.runtimes[name] = runtime
}

func (r *Registry) Get(name string) (types.Runtime, bool) {
	rt, ok := r.runtimes[name]
	return rt, ok
}
