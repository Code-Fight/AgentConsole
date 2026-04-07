package registry

import "code-agent-gateway/common/domain"

type Store struct {
	machines map[string]domain.Machine
}

func NewStore() *Store {
	return &Store{machines: map[string]domain.Machine{}}
}

func (s *Store) List() []domain.Machine {
	items := make([]domain.Machine, 0, len(s.machines))
	for _, item := range s.machines {
		items = append(items, item)
	}
	return items
}
