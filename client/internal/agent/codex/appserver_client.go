package codex

import (
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

type Runner interface {
	Call(method string, payload any, out any) error
}

type notificationRunner interface {
	SetNotificationHandler(func(jsonRPCNotification))
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
	runner           Runner
	now              func() time.Time
	turnEventMu      sync.RWMutex
	turnEventHandler func(agenttypes.RuntimeTurnEvent)
	deltaMu          sync.Mutex
	deltaSequence    map[string]int
}

var _ agenttypes.Runtime = (*AppServerClient)(nil)
var _ agenttypes.RuntimeTurnEventSource = (*AppServerClient)(nil)

func NewAppServerClient(runner Runner) *AppServerClient {
	client := &AppServerClient{
		runner:        runner,
		now:           time.Now,
		deltaSequence: make(map[string]int),
	}
	if notifier, ok := runner.(notificationRunner); ok {
		notifier.SetNotificationHandler(client.handleNotification)
	}
	return client
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

func (c *AppServerClient) SetTurnEventHandler(handler func(agenttypes.RuntimeTurnEvent)) {
	c.turnEventMu.Lock()
	c.turnEventHandler = handler
	c.turnEventMu.Unlock()
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

func (c *AppServerClient) handleNotification(notification jsonRPCNotification) {
	switch notification.Method {
	case "turn/started":
		threadID, turnID, requestID := extractTurnNotificationIDs(notification.Params)
		if threadID == "" || turnID == "" {
			return
		}
		c.deltaMu.Lock()
		c.deltaSequence[turnID] = 0
		c.deltaMu.Unlock()
		c.emitTurnEvent(agenttypes.RuntimeTurnEvent{
			Type:      agenttypes.RuntimeTurnEventTypeStarted,
			RequestID: requestID,
			ThreadID:  threadID,
			TurnID:    turnID,
		})
	case "item/agentMessage/delta":
		threadID, turnID, requestID := extractTurnNotificationIDs(notification.Params)
		delta := extractNotificationText(notification.Params,
			[]string{"delta"},
			[]string{"text"},
			[]string{"item", "delta"},
			[]string{"item", "text"},
		)
		if threadID == "" || turnID == "" || delta == "" {
			return
		}
		c.emitTurnEvent(agenttypes.RuntimeTurnEvent{
			Type:      agenttypes.RuntimeTurnEventTypeDelta,
			RequestID: requestID,
			ThreadID:  threadID,
			TurnID:    turnID,
			Sequence:  c.nextDeltaSequence(turnID),
			Delta:     delta,
		})
	case "turn/completed":
		threadID, turnID, requestID := extractTurnNotificationIDs(notification.Params)
		if threadID == "" || turnID == "" {
			return
		}
		status := domain.TurnStatus(extractNotificationString(notification.Params,
			[]string{"status"},
			[]string{"turn", "status"},
		))
		if status == "" {
			status = domain.TurnStatusCompleted
		}
		c.deltaMu.Lock()
		delete(c.deltaSequence, turnID)
		c.deltaMu.Unlock()
		c.emitTurnEvent(agenttypes.RuntimeTurnEvent{
			Type:      agenttypes.RuntimeTurnEventTypeCompleted,
			RequestID: requestID,
			Turn: domain.Turn{
				TurnID:   turnID,
				ThreadID: threadID,
				Status:   status,
			},
		})
	}
}

func (c *AppServerClient) emitTurnEvent(event agenttypes.RuntimeTurnEvent) {
	c.turnEventMu.RLock()
	handler := c.turnEventHandler
	c.turnEventMu.RUnlock()
	if handler != nil {
		handler(event)
	}
}

func (c *AppServerClient) nextDeltaSequence(turnID string) int {
	c.deltaMu.Lock()
	defer c.deltaMu.Unlock()
	c.deltaSequence[turnID]++
	return c.deltaSequence[turnID]
}

func extractTurnNotificationIDs(params json.RawMessage) (threadID string, turnID string, requestID string) {
	threadID = extractNotificationString(params,
		[]string{"threadId"},
		[]string{"turn", "threadId"},
		[]string{"item", "threadId"},
	)
	turnID = extractNotificationString(params,
		[]string{"turnId"},
		[]string{"turn", "turnId"},
		[]string{"turn", "id"},
		[]string{"item", "turnId"},
		[]string{"item", "id"},
	)
	requestID = extractNotificationString(params,
		[]string{"requestId"},
		[]string{"requestID"},
	)
	return threadID, turnID, requestID
}

func extractNotificationString(params json.RawMessage, paths ...[]string) string {
	if len(params) == 0 {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return ""
	}

	for _, path := range paths {
		if value, ok := nestedValue(payload, path...); ok {
			switch typed := value.(type) {
			case string:
				if trimmed := strings.TrimSpace(typed); trimmed != "" {
					return trimmed
				}
			case map[string]any:
				if text, ok := typed["text"].(string); ok && strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
				if delta, ok := typed["delta"].(string); ok && strings.TrimSpace(delta) != "" {
					return strings.TrimSpace(delta)
				}
			}
		}
	}

	return ""
}

func extractNotificationText(params json.RawMessage, paths ...[]string) string {
	if len(params) == 0 {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return ""
	}

	for _, path := range paths {
		if value, ok := nestedValue(payload, path...); ok {
			switch typed := value.(type) {
			case string:
				return typed
			case map[string]any:
				if text, ok := typed["text"].(string); ok {
					return text
				}
				if delta, ok := typed["delta"].(string); ok {
					return delta
				}
			}
		}
	}

	return ""
}

func nestedValue(payload map[string]any, path ...string) (any, bool) {
	current := any(payload)
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = next[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
