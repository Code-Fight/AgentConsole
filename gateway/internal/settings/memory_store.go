package settings

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"code-agent-gateway/common/domain"
)

type MemoryStore struct {
	mu                 sync.RWMutex
	agents             []domain.AgentDescriptor
	globals            map[domain.AgentType]domain.AgentConfigDocument
	machines           map[string]map[domain.AgentType]domain.AgentConfigDocument
	consolePreferences *domain.ConsolePreferences
	timeNow            func() time.Time
	updatedBy          string
}

func NewMemoryStore(agents []domain.AgentDescriptor) *MemoryStore {
	return &MemoryStore{
		agents:    append([]domain.AgentDescriptor(nil), agents...),
		globals:   map[domain.AgentType]domain.AgentConfigDocument{},
		machines:  map[string]map[domain.AgentType]domain.AgentConfigDocument{},
		timeNow:   time.Now,
		updatedBy: "console",
	}
}

func (s *MemoryStore) ListAgentTypes() []domain.AgentDescriptor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.AgentDescriptor(nil), s.agents...)
}

func (s *MemoryStore) GetGlobal(agentType domain.AgentType) (domain.AgentConfigDocument, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document, ok := s.globals[agentType]
	return document, ok, nil
}

func (s *MemoryStore) PutGlobal(agentType domain.AgentType, document domain.AgentConfigDocument) error {
	if strings.TrimSpace(string(agentType)) == "" {
		return fmt.Errorf("agentType is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.globals[agentType]
	document = s.stampDocument(agentType, document, current.Version)
	s.globals[agentType] = document
	return nil
}

func (s *MemoryStore) GetMachine(machineID string, agentType domain.AgentType) (domain.AgentConfigDocument, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	documents := s.machines[machineID]
	document, ok := documents[agentType]
	return document, ok, nil
}

func (s *MemoryStore) PutMachine(machineID string, agentType domain.AgentType, document domain.AgentConfigDocument) error {
	if strings.TrimSpace(machineID) == "" {
		return fmt.Errorf("machineID is required")
	}
	if strings.TrimSpace(string(agentType)) == "" {
		return fmt.Errorf("agentType is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	documents := s.machines[machineID]
	if documents == nil {
		documents = map[domain.AgentType]domain.AgentConfigDocument{}
	}
	current := documents[agentType]
	document = s.stampDocument(agentType, document, current.Version)
	documents[agentType] = document
	s.machines[machineID] = documents
	return nil
}

func (s *MemoryStore) DeleteMachine(machineID string, agentType domain.AgentType) error {
	if strings.TrimSpace(machineID) == "" {
		return fmt.Errorf("machineID is required")
	}
	if strings.TrimSpace(string(agentType)) == "" {
		return fmt.Errorf("agentType is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	documents := s.machines[machineID]
	if documents == nil {
		return nil
	}
	delete(documents, agentType)
	if len(documents) == 0 {
		delete(s.machines, machineID)
		return nil
	}
	s.machines[machineID] = documents
	return nil
}

func (s *MemoryStore) GetConsolePreferences() (domain.ConsolePreferences, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.consolePreferences == nil {
		return domain.ConsolePreferences{}, false, nil
	}
	copyPreferences := *s.consolePreferences
	copyPreferences.ThreadTitles = cloneThreadTitles(copyPreferences.ThreadTitles)
	return copyPreferences, true, nil
}

func (s *MemoryStore) PutConsolePreferences(preferences domain.ConsolePreferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyPreferences := preferences
	copyPreferences.ThreadTitles = cloneThreadTitles(copyPreferences.ThreadTitles)
	s.consolePreferences = &copyPreferences
	return nil
}

func (s *MemoryStore) snapshot() persistedState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var consolePreferences *domain.ConsolePreferences
	if s.consolePreferences != nil {
		copyPreferences := *s.consolePreferences
		copyPreferences.ThreadTitles = cloneThreadTitles(copyPreferences.ThreadTitles)
		consolePreferences = &copyPreferences
	}
	return persistedState{
		Agents:             append([]domain.AgentDescriptor(nil), s.agents...),
		Globals:            cloneDocumentMap(s.globals),
		Machines:           cloneMachineDocumentMap(s.machines),
		ConsolePreferences: consolePreferences,
	}
}

func (s *MemoryStore) load(state persistedState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents = append([]domain.AgentDescriptor(nil), state.Agents...)
	s.globals = cloneDocumentMap(state.Globals)
	s.machines = cloneMachineDocumentMap(state.Machines)
	if state.ConsolePreferences != nil {
		copyPreferences := *state.ConsolePreferences
		copyPreferences.ThreadTitles = cloneThreadTitles(copyPreferences.ThreadTitles)
		s.consolePreferences = &copyPreferences
	} else {
		s.consolePreferences = nil
	}
}

func (s *MemoryStore) stampDocument(agentType domain.AgentType, document domain.AgentConfigDocument, currentVersion int64) domain.AgentConfigDocument {
	document.AgentType = agentType
	if document.Format == "" {
		document.Format = domain.AgentConfigFormatTOML
	}
	document.Version = currentVersion + 1
	document.UpdatedAt = s.timeNow().UTC().Format(time.RFC3339)
	if strings.TrimSpace(document.UpdatedBy) == "" {
		document.UpdatedBy = s.updatedBy
	}
	return document
}

type persistedState struct {
	Agents             []domain.AgentDescriptor                                   `json:"agents"`
	Globals            map[domain.AgentType]domain.AgentConfigDocument            `json:"globals"`
	Machines           map[string]map[domain.AgentType]domain.AgentConfigDocument `json:"machines"`
	ConsolePreferences *domain.ConsolePreferences                                 `json:"consolePreferences,omitempty"`
}

func cloneDocumentMap(source map[domain.AgentType]domain.AgentConfigDocument) map[domain.AgentType]domain.AgentConfigDocument {
	if len(source) == 0 {
		return map[domain.AgentType]domain.AgentConfigDocument{}
	}
	cloned := make(map[domain.AgentType]domain.AgentConfigDocument, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneMachineDocumentMap(source map[string]map[domain.AgentType]domain.AgentConfigDocument) map[string]map[domain.AgentType]domain.AgentConfigDocument {
	if len(source) == 0 {
		return map[string]map[domain.AgentType]domain.AgentConfigDocument{}
	}
	cloned := make(map[string]map[domain.AgentType]domain.AgentConfigDocument, len(source))
	for machineID, documents := range source {
		cloned[machineID] = cloneDocumentMap(documents)
	}
	return cloned
}

func cloneThreadTitles(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
