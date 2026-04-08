package snapshot

import (
	"code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type Snapshot struct {
	Threads     []domain.Thread              `json:"threads"`
	Environment []domain.EnvironmentResource `json:"environment"`
}

func Build(runtime types.Runtime) (Snapshot, error) {
	threads, err := runtime.ListThreads()
	if err != nil {
		return Snapshot{}, err
	}

	environment, err := runtime.ListEnvironment()
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		Threads:     threads,
		Environment: environment,
	}, nil
}
