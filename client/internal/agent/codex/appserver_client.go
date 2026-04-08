package codex

import (
	"bytes"
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
	"encoding/json"
	"fmt"
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

type serverRequestRunner interface {
	SetServerRequestHandler(func(jsonRPCServerRequest))
	Respond(id json.RawMessage, result any, rpcErr *jsonRPCError) error
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
	approvalMu       sync.RWMutex
	approvalHandler  func(agenttypes.RuntimeApprovalRequest)
	pendingApprovals map[string]pendingApprovalRequest
	deltaMu          sync.Mutex
	deltaSequence    map[string]int
}

var _ agenttypes.Runtime = (*AppServerClient)(nil)
var _ agenttypes.RuntimeTurnEventSource = (*AppServerClient)(nil)
var _ agenttypes.RuntimeApprovalEventSource = (*AppServerClient)(nil)
var _ agenttypes.RuntimeApprovalResponder = (*AppServerClient)(nil)

func NewAppServerClient(runner Runner) *AppServerClient {
	client := &AppServerClient{
		runner:           runner,
		now:              time.Now,
		pendingApprovals: make(map[string]pendingApprovalRequest),
		deltaSequence:    make(map[string]int),
	}
	if notifier, ok := runner.(notificationRunner); ok {
		notifier.SetNotificationHandler(client.handleNotification)
	}
	if requester, ok := runner.(serverRequestRunner); ok {
		requester.SetServerRequestHandler(client.handleServerRequest)
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

func (c *AppServerClient) SetApprovalHandler(handler func(agenttypes.RuntimeApprovalRequest)) {
	c.approvalMu.Lock()
	c.approvalHandler = handler
	c.approvalMu.Unlock()
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
	case "serverRequest/resolved":
		requestID := extractNotificationString(notification.Params,
			[]string{"requestId"},
			[]string{"requestID"},
			[]string{"id"},
		)
		if requestID != "" {
			c.deletePendingApproval(requestID)
		}
	}
}

func (c *AppServerClient) handleServerRequest(request jsonRPCServerRequest) {
	requestID, err := normalizeServerRequestID(request.ID)
	if err != nil {
		return
	}

	approvalKind, supported := approvalKindFromMethod(request.Method)
	if !supported {
		c.respondToServerRequestError(request.ID, fmt.Sprintf("unsupported approval kind: %s", approvalKind))
		return
	}

	approval := ApprovalRequest{
		RequestID: requestID,
		ThreadID: extractNotificationString(request.Params,
			[]string{"threadId"},
			[]string{"item", "threadId"},
		),
		TurnID: extractNotificationString(request.Params,
			[]string{"turnId"},
			[]string{"item", "turnId"},
		),
		ItemID: extractNotificationString(request.Params,
			[]string{"itemId"},
			[]string{"item", "itemId"},
			[]string{"item", "id"},
		),
		Kind: approvalKind,
		Reason: extractNotificationString(request.Params,
			[]string{"reason"},
			[]string{"message"},
			[]string{"item", "reason"},
		),
		Command: extractNotificationString(request.Params,
			[]string{"command"},
			[]string{"item", "command"},
		),
	}

	c.approvalMu.Lock()
	c.pendingApprovals[requestID] = pendingApprovalRequest{
		id:      append(json.RawMessage(nil), request.ID...),
		request: approval,
	}
	handler := c.approvalHandler
	c.approvalMu.Unlock()

	if handler != nil {
		handler(agenttypes.RuntimeApprovalRequest{
			RequestID: approval.RequestID,
			ThreadID:  approval.ThreadID,
			TurnID:    approval.TurnID,
			ItemID:    approval.ItemID,
			Kind:      approval.Kind,
			Reason:    approval.Reason,
			Command:   approval.Command,
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

func (c *AppServerClient) deletePendingApproval(requestID string) {
	c.approvalMu.Lock()
	delete(c.pendingApprovals, requestID)
	c.approvalMu.Unlock()
}

func (c *AppServerClient) respondToServerRequestError(id json.RawMessage, message string) {
	responder, ok := c.runner.(serverRequestRunner)
	if !ok {
		return
	}
	_ = responder.Respond(id, nil, &jsonRPCError{
		Code:    -32000,
		Message: message,
	})
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

func normalizeServerRequestID(raw json.RawMessage) (string, error) {
	var stringID string
	if err := json.Unmarshal(raw, &stringID); err == nil {
		stringID = strings.TrimSpace(stringID)
		if stringID != "" {
			return stringID, nil
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var number json.Number
	if err := decoder.Decode(&number); err == nil {
		normalized := strings.TrimSpace(number.String())
		if normalized != "" {
			return normalized, nil
		}
	}

	return "", fmt.Errorf("unsupported server request id")
}
