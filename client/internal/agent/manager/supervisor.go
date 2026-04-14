package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	agentregistry "code-agent-gateway/client/internal/agent/registry"
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/client/internal/agent/codex"
	"code-agent-gateway/common/domain"
)

const managedAgentMetadataName = "agent.json"

type Supervisor struct {
	mu                sync.RWMutex
	ctx               context.Context
	managedAgentsDir  string
	registry          *agentregistry.Registry
	factories         map[domain.AgentType]agenttypes.RuntimeFactory
	records           map[string]managedAgentRecord
	cleanups          map[string]func() error
	statusByAgentID   map[string]domain.AgentInstanceStatus
}

type managedAgentRecord struct {
	AgentID     string           `json:"agentId"`
	AgentType   domain.AgentType `json:"agentType"`
	DisplayName string           `json:"displayName"`
}

func NewSupervisor(ctx context.Context, managedAgentsDir string, registry *agentregistry.Registry, factories map[domain.AgentType]agenttypes.RuntimeFactory) (*Supervisor, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(managedAgentsDir) == "" {
		return nil, fmt.Errorf("managed agents directory is required")
	}
	if registry == nil {
		return nil, fmt.Errorf("agent runtime registry is not configured")
	}

	supervisor := &Supervisor{
		ctx:              ctx,
		managedAgentsDir: managedAgentsDir,
		registry:         registry,
		factories:        factories,
		records:          map[string]managedAgentRecord{},
		cleanups:         map[string]func() error{},
		statusByAgentID:  map[string]domain.AgentInstanceStatus{},
	}
	if err := supervisor.bootstrap(); err != nil {
		return nil, err
	}
	return supervisor, nil
}

func (s *Supervisor) bootstrap() error {
	if err := os.MkdirAll(s.managedAgentsDir, 0o755); err != nil {
		return err
	}
	if err := s.loadRecords(); err != nil {
		return err
	}
	if len(s.records) == 0 {
		record := managedAgentRecord{
			AgentID:     "agent-01",
			AgentType:   domain.AgentTypeCodex,
			DisplayName: "Codex",
		}
		if err := s.saveRecord(record); err != nil {
			return err
		}
		s.records[record.AgentID] = record
	}
	return s.StartAll()
}

func (s *Supervisor) loadRecords() error {
	entries, err := os.ReadDir(s.managedAgentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		recordPath := filepath.Join(s.managedAgentsDir, entry.Name(), managedAgentMetadataName)
		data, err := os.ReadFile(recordPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		var record managedAgentRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return err
		}
		if strings.TrimSpace(record.AgentID) == "" {
			record.AgentID = entry.Name()
		}
		if record.DisplayName == "" {
			record.DisplayName = record.AgentID
		}
		s.records[record.AgentID] = record
	}
	return nil
}

func (s *Supervisor) StartAll() error {
	s.mu.Lock()
	records := sortedManagedAgentRecords(s.records)
	s.mu.Unlock()

	for _, record := range records {
		if err := s.startAgent(record); err != nil {
			return err
		}
	}
	return nil
}

func (s *Supervisor) StopAll() error {
	s.mu.Lock()
	agentIDs := make([]string, 0, len(s.cleanups))
	for agentID := range s.cleanups {
		agentIDs = append(agentIDs, agentID)
	}
	s.mu.Unlock()

	sort.Strings(agentIDs)
	for _, agentID := range agentIDs {
		if err := s.stopAgent(agentID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Supervisor) InstallAgent(agentType domain.AgentType, displayName string) (domain.AgentInstance, error) {
	if strings.TrimSpace(displayName) == "" {
		return domain.AgentInstance{}, fmt.Errorf("displayName is required")
	}

	s.mu.Lock()
	agentID := s.nextAgentIDLocked()
	record := managedAgentRecord{
		AgentID:     agentID,
		AgentType:   agentType,
		DisplayName: displayName,
	}
	if err := s.saveRecord(record); err != nil {
		s.mu.Unlock()
		return domain.AgentInstance{}, err
	}
	s.records[agentID] = record
	s.mu.Unlock()

	if err := s.startAgent(record); err != nil {
		return domain.AgentInstance{}, err
	}
	return s.agentInstance(record), nil
}

func (s *Supervisor) DeleteAgent(agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("agentID is required")
	}

	if err := s.stopAgent(agentID); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.records, agentID)
	delete(s.statusByAgentID, agentID)
	s.mu.Unlock()

	layout := codex.NewInstanceLayout(s.managedAgentsDir, agentID)
	return os.RemoveAll(layout.RootDir)
}

func (s *Supervisor) ReadConfig(agentID string) (domain.AgentConfigDocument, error) {
	record, ok := s.record(agentID)
	if !ok {
		return domain.AgentConfigDocument{}, fmt.Errorf("agent %q is not installed", agentID)
	}

	layout := codex.NewInstanceLayout(s.managedAgentsDir, record.AgentID)
	data, err := os.ReadFile(layout.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.AgentConfigDocument{
				AgentType: record.AgentType,
				Format:    domain.AgentConfigFormatTOML,
			}, nil
		}
		return domain.AgentConfigDocument{}, err
	}
	return domain.AgentConfigDocument{
		AgentType: record.AgentType,
		Format:    domain.AgentConfigFormatTOML,
		Content:   string(data),
	}, nil
}

func (s *Supervisor) WriteConfig(agentID string, document domain.AgentConfigDocument) (domain.AgentConfigDocument, error) {
	record, ok := s.record(agentID)
	if !ok {
		return domain.AgentConfigDocument{}, fmt.Errorf("agent %q is not installed", agentID)
	}

	document.AgentType = record.AgentType
	if document.Format == "" {
		document.Format = domain.AgentConfigFormatTOML
	}

	layout := codex.NewInstanceLayout(s.managedAgentsDir, record.AgentID)
	if _, err := layout.ApplyConfig(document); err != nil {
		return domain.AgentConfigDocument{}, err
	}
	return document, nil
}

func (s *Supervisor) ApplyConfigToType(agentType domain.AgentType, document domain.AgentConfigDocument) (agenttypes.ApplyConfigResult, error) {
	records := s.recordsByType(agentType)
	if len(records) == 0 {
		return agenttypes.ApplyConfigResult{}, fmt.Errorf("no managed agent installed for %q", agentType)
	}

	var result agenttypes.ApplyConfigResult
	for idx, record := range records {
		layout := codex.NewInstanceLayout(s.managedAgentsDir, record.AgentID)
		document.AgentType = record.AgentType
		if document.Format == "" {
			document.Format = domain.AgentConfigFormatTOML
		}

		applied, err := layout.ApplyConfig(document)
		if err != nil {
			return agenttypes.ApplyConfigResult{}, err
		}
		if idx == 0 {
			result = applied
		}
	}
	return result, nil
}

func (s *Supervisor) AgentInstances() []domain.AgentInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := sortedManagedAgentRecords(s.records)
	items := make([]domain.AgentInstance, 0, len(records))
	for _, record := range records {
		items = append(items, s.agentInstance(record))
	}
	return items
}

func (s *Supervisor) ResolveAgentID(agentID string) (string, error) {
	trimmed := strings.TrimSpace(agentID)
	if trimmed != "" {
		if _, ok := s.record(trimmed); !ok {
			return "", fmt.Errorf("agent %q is not installed", trimmed)
		}
		return trimmed, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.records) == 1 {
		for id := range s.records {
			return id, nil
		}
	}
	return "", fmt.Errorf("agentId is required")
}

func (s *Supervisor) startAgent(record managedAgentRecord) error {
	s.mu.RLock()
	_, alreadyRunning := s.cleanups[record.AgentID]
	s.mu.RUnlock()
	if alreadyRunning {
		return nil
	}

	factory, ok := s.factories[record.AgentType]
	if !ok {
		return fmt.Errorf("agent type %q is not supported", record.AgentType)
	}

	runtime, cleanup, err := factory.Start(s.ctx, agenttypes.ManagedAgentSpec{
		AgentID:     record.AgentID,
		AgentType:   record.AgentType,
		DisplayName: record.DisplayName,
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.statusByAgentID[record.AgentID] = domain.AgentInstanceStatusError
		return err
	}
	s.registry.Register(record.AgentID, runtime)
	s.cleanups[record.AgentID] = cleanup
	s.statusByAgentID[record.AgentID] = domain.AgentInstanceStatusRunning
	return nil
}

func (s *Supervisor) stopAgent(agentID string) error {
	s.mu.Lock()
	cleanup := s.cleanups[agentID]
	delete(s.cleanups, agentID)
	s.registry.Remove(agentID)
	s.statusByAgentID[agentID] = domain.AgentInstanceStatusStopped
	s.mu.Unlock()

	if cleanup != nil {
		return cleanup()
	}
	return nil
}

func (s *Supervisor) record(agentID string) (managedAgentRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[strings.TrimSpace(agentID)]
	return record, ok
}

func (s *Supervisor) firstRecordByType(agentType domain.AgentType) (managedAgentRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := sortedManagedAgentRecords(s.records)
	for _, record := range records {
		if record.AgentType == agentType {
			return record, true
		}
	}
	return managedAgentRecord{}, false
}

func (s *Supervisor) recordsByType(agentType domain.AgentType) []managedAgentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := sortedManagedAgentRecords(s.records)
	filtered := make([]managedAgentRecord, 0, len(records))
	for _, record := range records {
		if record.AgentType == agentType {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func (s *Supervisor) saveRecord(record managedAgentRecord) error {
	layout := codex.NewInstanceLayout(s.managedAgentsDir, record.AgentID)
	if err := os.MkdirAll(layout.RootDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(layout.RootDir, managedAgentMetadataName), data, 0o644)
}

func (s *Supervisor) nextAgentIDLocked() string {
	for idx := 1; ; idx++ {
		candidate := fmt.Sprintf("agent-%02d", idx)
		if _, exists := s.records[candidate]; !exists {
			return candidate
		}
	}
}

func (s *Supervisor) agentInstance(record managedAgentRecord) domain.AgentInstance {
	status := s.statusByAgentID[record.AgentID]
	if status == "" {
		status = domain.AgentInstanceStatusStopped
	}
	return domain.AgentInstance{
		AgentID:     record.AgentID,
		AgentType:   record.AgentType,
		DisplayName: record.DisplayName,
		Status:      status,
	}
}

func sortedManagedAgentRecords(records map[string]managedAgentRecord) []managedAgentRecord {
	items := make([]managedAgentRecord, 0, len(records))
	for _, record := range records {
		items = append(items, record)
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].AgentID < items[j].AgentID
	})
	return items
}
