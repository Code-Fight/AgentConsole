package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"code-agent-gateway/common/domain"
)

type FileStore struct {
	*MemoryStore
	path string
}

func NewFileStore(path string, agents []domain.AgentDescriptor) (*FileStore, error) {
	if path == "" {
		return nil, fmt.Errorf("settings file path is required")
	}

	store := &FileStore{
		MemoryStore: NewMemoryStore(agents),
		path:        path,
	}
	if err := store.loadFromFile(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileStore) PutGlobal(agentType domain.AgentType, document domain.AgentConfigDocument) error {
	if err := s.MemoryStore.PutGlobal(agentType, document); err != nil {
		return err
	}
	return s.persist()
}

func (s *FileStore) PutMachine(machineID string, agentType domain.AgentType, document domain.AgentConfigDocument) error {
	if err := s.MemoryStore.PutMachine(machineID, agentType, document); err != nil {
		return err
	}
	return s.persist()
}

func (s *FileStore) DeleteMachine(machineID string, agentType domain.AgentType) error {
	if err := s.MemoryStore.DeleteMachine(machineID, agentType); err != nil {
		return err
	}
	return s.persist()
}

func (s *FileStore) loadFromFile() error {
	if _, err := os.Stat(s.path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var state persistedState
	if len(content) > 0 {
		if err := json.Unmarshal(content, &state); err != nil {
			return err
		}
	}

	s.MemoryStore.load(state)
	return nil
}

func (s *FileStore) persist() error {
	state := s.MemoryStore.snapshot()
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
