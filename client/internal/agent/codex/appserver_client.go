package codex

import (
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
	"time"
)

type Runner interface {
	Call(method string, payload any, out any) error
}

type initializeResponse struct {
	UserAgent string `json:"userAgent"`
}

type threadRecord struct {
	ID        string             `json:"id"`
	MachineID string             `json:"machineId"`
	Status    threadStatusRecord `json:"status"`
	Name      string             `json:"name"`
	Preview   string             `json:"preview"`
}

type threadStatusRecord struct {
	Type domain.ThreadStatus `json:"type"`
}

type threadListResponse struct {
	Data []threadRecord `json:"data"`
}

type threadStartResponse struct {
	Thread threadRecord `json:"thread"`
}

type threadReadResponse struct {
	Thread threadRecord `json:"thread"`
}

type threadResumeResponse struct {
	Thread threadRecord `json:"thread"`
}

type turnRecord struct {
	ID     string            `json:"id"`
	Status domain.TurnStatus `json:"status"`
}

type turnStartResponse struct {
	Turn turnRecord `json:"turn"`
}

type skillMetadata struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

type skillsListEntry struct {
	Cwd    string          `json:"cwd"`
	Skills []skillMetadata `json:"skills"`
}

type skillsListResponse struct {
	Data []skillsListEntry `json:"data"`
}

type pluginSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Enabled   bool   `json:"enabled"`
}

type pluginMarketplaceEntry struct {
	Name    string          `json:"name"`
	Plugins []pluginSummary `json:"plugins"`
}

type pluginListResponse struct {
	Marketplaces []pluginMarketplaceEntry `json:"marketplaces"`
}

type AppServerClient struct {
	runner Runner
	now    func() time.Time
}

var _ agenttypes.Runtime = (*AppServerClient)(nil)

func NewAppServerClient(runner Runner) *AppServerClient {
	return &AppServerClient{
		runner: runner,
		now:    time.Now,
	}
}

func (c *AppServerClient) Initialize() error {
	var out initializeResponse
	return c.runner.Call("initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "code-agent-gateway",
			"version": "dev",
		},
		"capabilities": nil,
	}, &out)
}

func (c *AppServerClient) ListThreads() ([]domain.Thread, error) {
	var response threadListResponse
	if err := c.runner.Call("thread/list", map[string]any{}, &response); err != nil {
		return nil, err
	}

	threads := make([]domain.Thread, 0, len(response.Data))
	for _, record := range response.Data {
		threads = append(threads, record.toDomain())
	}
	return threads, nil
}

func (c *AppServerClient) CreateThread(params agenttypes.CreateThreadParams) (domain.Thread, error) {
	var response threadStartResponse
	if err := c.runner.Call("thread/start", map[string]any{
		"title":                  params.Title,
		"experimentalRawEvents":  false,
		"persistExtendedHistory": false,
	}, &response); err != nil {
		return domain.Thread{}, err
	}

	return response.Thread.toDomain(), nil
}

func (c *AppServerClient) StartTurn(params agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	var out turnStartResponse
	if err := c.runner.Call("turn/start", map[string]any{
		"threadId": params.ThreadID,
		"input": []map[string]any{
			{
				"type":          "text",
				"text":          params.Input,
				"text_elements": []any{},
			},
		},
	}, &out); err != nil {
		return agenttypes.StartTurnResult{}, err
	}

	return agenttypes.StartTurnResult{
		TurnID:   out.Turn.ID,
		ThreadID: params.ThreadID,
	}, nil
}

func (r threadRecord) toDomain() domain.Thread {
	status := r.Status.Type
	if status == "" {
		status = domain.ThreadStatusNotLoaded
	}
	title := r.Name
	if title == "" {
		title = r.Preview
	}
	return domain.Thread{
		ThreadID:  r.ID,
		MachineID: r.MachineID,
		Status:    status,
		Title:     title,
	}
}
