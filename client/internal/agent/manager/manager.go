package manager

import (
	"code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/client/internal/snapshot"
)

type Manager struct {
	runtime types.Runtime
}

func New(runtime types.Runtime) *Manager {
	return &Manager{runtime: runtime}
}

func (m *Manager) Snapshot() (snapshot.Snapshot, error) {
	return snapshot.Build(m.runtime)
}
