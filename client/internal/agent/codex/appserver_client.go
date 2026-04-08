package codex

import (
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type Runner interface {
	Call(method string, payload any, out any) error
}

type ThreadRecord struct {
	ID        string              `json:"id"`
	ThreadID  string              `json:"threadId"`
	MachineID string              `json:"machineId"`
	Status    domain.ThreadStatus `json:"status"`
	Title     string              `json:"title"`
}

type EnvironmentRecord struct {
	ResourceID      string                           `json:"resourceId"`
	MachineID       string                           `json:"machineId"`
	Kind            domain.EnvironmentKind           `json:"kind"`
	DisplayName     string                           `json:"displayName"`
	Status          domain.EnvironmentResourceStatus `json:"status"`
	RestartRequired bool                             `json:"restartRequired"`
	LastObservedAt  string                           `json:"lastObservedAt"`
}

type AppServerClient struct {
	runner Runner
}

var _ agenttypes.Runtime = (*AppServerClient)(nil)

func NewAppServerClient(runner Runner) *AppServerClient {
	return &AppServerClient{runner: runner}
}

func (c *AppServerClient) ListThreads() ([]domain.Thread, error) {
	var records []ThreadRecord
	if err := c.runner.Call("thread/list", map[string]any{}, &records); err != nil {
		return nil, err
	}

	threads := make([]domain.Thread, 0, len(records))
	for _, record := range records {
		threads = append(threads, record.toDomain())
	}
	return threads, nil
}

func (r ThreadRecord) toDomain() domain.Thread {
	threadID := r.ThreadID
	if threadID == "" {
		threadID = r.ID
	}
	status := r.Status
	if status == "" {
		status = domain.ThreadStatusNotLoaded
	}
	return domain.Thread{
		ThreadID:  threadID,
		MachineID: r.MachineID,
		Status:    status,
		Title:     r.Title,
	}
}

func (r EnvironmentRecord) toDomain() domain.EnvironmentResource {
	return domain.EnvironmentResource{
		ResourceID:      r.ResourceID,
		MachineID:       r.MachineID,
		Kind:            r.Kind,
		DisplayName:     r.DisplayName,
		Status:          r.Status,
		RestartRequired: r.RestartRequired,
		LastObservedAt:  r.LastObservedAt,
	}
}
