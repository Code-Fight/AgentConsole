package types

import "code-agent-gateway/common/domain"

type Runtime interface {
	ListThreads() ([]domain.Thread, error)
	ListEnvironment() ([]domain.EnvironmentResource, error)
}
