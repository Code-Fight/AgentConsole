package codex

import "code-agent-gateway/common/domain"

type FakeAdapter struct{}

func NewFakeAdapter() *FakeAdapter {
	return &FakeAdapter{}
}

func (a *FakeAdapter) ListThreads() ([]domain.Thread, error) {
	return []domain.Thread{}, nil
}

func (a *FakeAdapter) ListEnvironment() ([]domain.EnvironmentResource, error) {
	return []domain.EnvironmentResource{}, nil
}
