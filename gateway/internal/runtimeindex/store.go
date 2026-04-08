package runtimeindex

import "code-agent-gateway/common/domain"

type Store struct {
	threads     []domain.Thread
	environment []domain.EnvironmentResource
}

func NewStore() *Store { return &Store{} }

func (s *Store) ReplaceSnapshot(threads []domain.Thread, environment []domain.EnvironmentResource) {
	s.threads = append([]domain.Thread(nil), threads...)
	s.environment = append([]domain.EnvironmentResource(nil), environment...)
}

func (s *Store) Threads() []domain.Thread {
	if s.threads == nil {
		return []domain.Thread{}
	}
	return append([]domain.Thread(nil), s.threads...)
}

func (s *Store) Environment(kind domain.EnvironmentKind) []domain.EnvironmentResource {
	items := make([]domain.EnvironmentResource, 0)
	for _, item := range s.environment {
		if item.Kind == kind {
			items = append(items, item)
		}
	}
	return items
}
