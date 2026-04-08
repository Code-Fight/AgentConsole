package codex

import "code-agent-gateway/common/domain"

type FakeAdapter struct {
	threads     []domain.Thread
	environment []domain.EnvironmentResource
}

func NewFakeAdapter() *FakeAdapter {
	return &FakeAdapter{
		threads:     []domain.Thread{},
		environment: []domain.EnvironmentResource{},
	}
}

func (a *FakeAdapter) SeedSnapshot(threads []domain.Thread, environment []domain.EnvironmentResource) {
	a.threads = append([]domain.Thread(nil), threads...)
	a.environment = append([]domain.EnvironmentResource(nil), environment...)
}

func (a *FakeAdapter) ListThreads() ([]domain.Thread, error) {
	return append([]domain.Thread(nil), a.threads...), nil
}

func (a *FakeAdapter) ListEnvironment() ([]domain.EnvironmentResource, error) {
	return append([]domain.EnvironmentResource(nil), a.environment...), nil
}
