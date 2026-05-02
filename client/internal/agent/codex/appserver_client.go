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
	Turns     []turnRecord       `json:"turns"`
}

type threadStatusRecord struct {
	Type domain.ThreadStatus `json:"type"`
}

type threadListResponse struct {
	Data []threadRecord `json:"data"`
}

type threadStartResponse struct {
	Thread         threadRecord `json:"thread"`
	Model          string       `json:"model"`
	ApprovalPolicy any          `json:"approvalPolicy"`
	Sandbox        any          `json:"sandbox"`
}

type threadReadResponse struct {
	Thread threadRecord `json:"thread"`
}

type threadResumeResponse struct {
	Thread         threadRecord `json:"thread"`
	Model          string       `json:"model"`
	ApprovalPolicy any          `json:"approvalPolicy"`
	Sandbox        any          `json:"sandbox"`
}

type turnRecord struct {
	ID     string             `json:"id"`
	Status domain.TurnStatus  `json:"status"`
	Error  *turnErrorRecord   `json:"error,omitempty"`
	Items  []threadItemRecord `json:"items"`
}

type turnErrorRecord struct {
	Message string `json:"message"`
}

type threadItemRecord struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Text    string            `json:"text"`
	Content []userInputRecord `json:"content"`
}

type userInputRecord struct {
	Type string `json:"type"`
	Text string `json:"text"`
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

type skillsConfigWriteResponse struct {
	Data []skillMetadata `json:"data"`
}

type skillSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type appSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	NeedsAuth   bool   `json:"needsAuth"`
}

type pluginSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Installed     bool   `json:"installed"`
	Enabled       bool   `json:"enabled"`
	InstallPolicy string `json:"installPolicy"`
	AuthPolicy    string `json:"authPolicy"`
}

type pluginMarketplaceEntry struct {
	Name    string          `json:"name"`
	Path    string          `json:"path"`
	Plugins []pluginSummary `json:"plugins"`
}

type pluginListResponse struct {
	Marketplaces []pluginMarketplaceEntry `json:"marketplaces"`
}

type pluginDetail struct {
	MarketplaceName string         `json:"marketplaceName"`
	MarketplacePath string         `json:"marketplacePath"`
	Summary         pluginSummary  `json:"summary"`
	Description     string         `json:"description"`
	Skills          []skillSummary `json:"skills"`
	Apps            []appSummary   `json:"apps"`
	MCPServers      []string       `json:"mcpServers"`
}

type pluginReadResponse struct {
	Plugin pluginDetail `json:"plugin"`
}

type pluginInstallResponse struct {
	AuthPolicy string `json:"authPolicy"`
}

type mcpServerStatusRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
	Enabled     bool   `json:"enabled"`
	NeedsAuth   bool   `json:"needsAuth"`
	Error       string `json:"error"`
}

type mcpServerStatusListResponse struct {
	Data []mcpServerStatusRecord `json:"data"`
}

type configReadResponse struct {
	Config map[string]any `json:"config"`
}

type modelListResponse struct {
	Data []modelRecord `json:"data"`
}

type modelRecord struct {
	ID          string `json:"id"`
	Model       string `json:"model"`
	DisplayName string `json:"displayName"`
	Hidden      bool   `json:"hidden"`
	IsDefault   bool   `json:"isDefault"`
}

type configRequirementsReadResponse struct {
	Requirements *configRequirementsRecord `json:"requirements"`
}

type configRequirementsRecord struct {
	AllowedApprovalPolicies []any    `json:"allowedApprovalPolicies"`
	AllowedSandboxModes     []string `json:"allowedSandboxModes"`
}

type configWriteResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	FilePath string `json:"filePath"`
}

type AppServerClient struct {
	runner                  Runner
	now                     func() time.Time
	threadMu                sync.RWMutex
	threads                 map[string]domain.Thread
	threadRuntimeDesired    map[string]domain.ThreadRuntimePreferences
	threadRuntimeApplied    map[string]domain.ThreadRuntimePreferences
	turnEventMu             sync.RWMutex
	turnEventHandler        func(agenttypes.RuntimeTurnEvent)
	timelineEventHandler    func(domain.AgentTimelineEvent)
	approvalMu              sync.RWMutex
	approvalHandler         func(agenttypes.RuntimeApprovalRequest)
	approvalResolvedHandler func(ApprovalResolvedEvent)
	pendingApprovals        map[string]pendingApprovalRequest
	deltaMu                 sync.Mutex
	deltaSequence           map[string]int
	timelineSequence        map[string]int
	agentMessageText        map[string]agentMessageState
	turnErrors              map[string]string
	activeTurnsByThread     map[string]string
	restartMu               sync.RWMutex
	restartRequired         map[string]bool
	homeDir                 func() (string, error)
	configMirrorPath        string
}

type agentMessageState struct {
	turnID string
	text   string
	kind   agenttypes.RuntimeTurnDeltaKind
}

type ApprovalResolvedEvent struct {
	RequestID string
	ThreadID  string
	TurnID    string
	Decision  string
}

var _ agenttypes.Runtime = (*AppServerClient)(nil)
var _ agenttypes.RuntimeTurnEventSource = (*AppServerClient)(nil)
var _ agenttypes.RuntimeTimelineEventSource = (*AppServerClient)(nil)
var _ agenttypes.RuntimeApprovalEventSource = (*AppServerClient)(nil)
var _ agenttypes.RuntimeApprovalResponder = (*AppServerClient)(nil)
var _ agenttypes.RuntimeSkillConfigurator = (*AppServerClient)(nil)
var _ agenttypes.RuntimeSkillManager = (*AppServerClient)(nil)
var _ agenttypes.RuntimeMCPManager = (*AppServerClient)(nil)
var _ agenttypes.RuntimePluginManager = (*AppServerClient)(nil)
var _ agenttypes.RuntimeThreadRuntimeManager = (*AppServerClient)(nil)

func NewAppServerClient(runner Runner) *AppServerClient {
	client := &AppServerClient{
		runner:               runner,
		now:                  time.Now,
		threads:              make(map[string]domain.Thread),
		threadRuntimeDesired: make(map[string]domain.ThreadRuntimePreferences),
		threadRuntimeApplied: make(map[string]domain.ThreadRuntimePreferences),
		pendingApprovals:     make(map[string]pendingApprovalRequest),
		deltaSequence:        make(map[string]int),
		timelineSequence:     make(map[string]int),
		agentMessageText:     make(map[string]agentMessageState),
		turnErrors:           make(map[string]string),
		activeTurnsByThread:  make(map[string]string),
		restartRequired:      make(map[string]bool),
		homeDir:              resolveUserHomeDir,
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
		"capabilities": map[string]any{
			"experimentalApi": true,
		},
	}, &out)
}

func (c *AppServerClient) ListThreads() ([]domain.Thread, error) {
	var response threadListResponse
	if err := c.runner.Call("thread/list", map[string]any{}, &response); err != nil {
		return nil, err
	}

	threads := make([]domain.Thread, 0, len(response.Data))
	seen := make(map[string]struct{}, len(response.Data))
	for _, record := range response.Data {
		thread := record.toDomain()
		thread.Status = c.runtimeThreadStatus(thread)
		if current, ok := c.threadByID(thread.ThreadID); ok {
			thread = mergeRememberedThread(current, thread)
		}
		threads = append(threads, thread)
		seen[thread.ThreadID] = struct{}{}
		c.rememberThread(thread)
	}

	for _, thread := range c.cachedThreads() {
		if _, ok := seen[thread.ThreadID]; ok {
			continue
		}
		threads = append(threads, thread)
	}
	return threads, nil
}

func (c *AppServerClient) runtimeThreadStatus(thread domain.Thread) domain.ThreadStatus {
	threadID := strings.TrimSpace(thread.ThreadID)
	if threadID != "" {
		c.deltaMu.Lock()
		_, active := c.activeTurnsByThread[threadID]
		c.deltaMu.Unlock()
		if active {
			return domain.ThreadStatusActive
		}
	}

	switch thread.Status {
	case domain.ThreadStatusError:
		return domain.ThreadStatusError
	default:
		return domain.ThreadStatusIdle
	}
}

func (c *AppServerClient) CreateThread(params agenttypes.CreateThreadParams) (domain.Thread, error) {
	var response threadStartResponse
	if err := c.runner.Call("thread/start", map[string]any{
		"title":                  params.Title,
		"experimentalRawEvents":  true,
		"persistExtendedHistory": false,
	}, &response); err != nil {
		return domain.Thread{}, err
	}

	thread := response.Thread.toDomain()
	if strings.TrimSpace(thread.Title) == "" {
		thread.Title = strings.TrimSpace(params.Title)
	}
	c.rememberThread(thread)
	c.seedThreadRuntimeState(thread.ThreadID, domain.ThreadRuntimePreferences{
		Model:          strings.TrimSpace(response.Model),
		ApprovalPolicy: normalizeApprovalPolicy(response.ApprovalPolicy),
		SandboxMode:    normalizeSandboxMode(response.Sandbox),
	})
	return thread, nil
}

func (c *AppServerClient) StartTurn(params agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	payload := map[string]any{
		"threadId": params.ThreadID,
		"input": []map[string]any{
			{
				"type":          "text",
				"text":          params.Input,
				"text_elements": []any{},
			},
		},
	}
	override, applied := c.turnStartRuntimeOverride(params.ThreadID)
	for key, value := range override {
		payload[key] = value
	}

	var out turnStartResponse
	if err := c.runner.Call("turn/start", payload, &out); err != nil {
		return agenttypes.StartTurnResult{}, err
	}
	if applied != nil {
		c.setThreadRuntimeApplied(params.ThreadID, *applied)
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

func (c *AppServerClient) SetTimelineEventHandler(handler func(domain.AgentTimelineEvent)) {
	c.turnEventMu.Lock()
	c.timelineEventHandler = handler
	c.turnEventMu.Unlock()
}

func (c *AppServerClient) SetApprovalHandler(handler func(agenttypes.RuntimeApprovalRequest)) {
	c.approvalMu.Lock()
	c.approvalHandler = handler
	c.approvalMu.Unlock()
}

func (c *AppServerClient) SetApprovalResolvedHandler(handler func(ApprovalResolvedEvent)) {
	c.approvalMu.Lock()
	c.approvalResolvedHandler = handler
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
		Messages:  flattenThreadMessages(r.Turns),
	}
}

func (c *AppServerClient) rememberThread(thread domain.Thread) {
	if strings.TrimSpace(thread.ThreadID) == "" {
		return
	}
	c.threadMu.Lock()
	if current, ok := c.threads[thread.ThreadID]; ok {
		thread = mergeRememberedThread(current, thread)
	}
	c.threads[thread.ThreadID] = thread
	c.threadMu.Unlock()
}

func (c *AppServerClient) forgetThread(threadID string) {
	if strings.TrimSpace(threadID) == "" {
		return
	}
	c.threadMu.Lock()
	delete(c.threads, threadID)
	delete(c.threadRuntimeDesired, threadID)
	delete(c.threadRuntimeApplied, threadID)
	c.threadMu.Unlock()
}

func (c *AppServerClient) cachedThreads() []domain.Thread {
	c.threadMu.RLock()
	defer c.threadMu.RUnlock()

	items := make([]domain.Thread, 0, len(c.threads))
	for _, thread := range c.threads {
		items = append(items, thread)
	}
	return items
}

func (c *AppServerClient) threadByID(threadID string) (domain.Thread, bool) {
	if strings.TrimSpace(threadID) == "" {
		return domain.Thread{}, false
	}
	c.threadMu.RLock()
	defer c.threadMu.RUnlock()
	thread, ok := c.threads[threadID]
	return thread, ok
}

func mergeRememberedThread(current domain.Thread, next domain.Thread) domain.Thread {
	if strings.TrimSpace(next.MachineID) == "" {
		next.MachineID = current.MachineID
	}
	if strings.TrimSpace(next.Title) == "" {
		next.Title = current.Title
	}
	if next.Status == "" {
		next.Status = current.Status
	}
	return next
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
		delete(c.turnErrors, turnID)
		c.activeTurnsByThread[threadID] = turnID
		c.deltaMu.Unlock()
		c.emitTurnEvent(agenttypes.RuntimeTurnEvent{
			Type:      agenttypes.RuntimeTurnEventTypeStarted,
			RequestID: requestID,
			ThreadID:  threadID,
			TurnID:    turnID,
		})
		if event, ok := codexTurnTimelineEvent(notification.Params, domain.AgentTimelineEventTurnStarted, domain.AgentTimelineStatusRunning, c.nextTimelineSequence(turnID), ""); ok {
			c.emitTimelineEvent(event)
		}
	case "item/agentMessage/delta":
		threadID, turnID, requestID := extractTurnNotificationIDs(notification.Params)
		turnID = c.resolveTurnID(threadID, turnID)
		itemID := extractNotificationString(notification.Params,
			[]string{"itemId"},
			[]string{"item", "id"},
		)
		delta := extractNotificationText(notification.Params,
			[]string{"delta"},
			[]string{"text"},
			[]string{"item", "delta"},
			[]string{"item", "text"},
		)
		if threadID == "" || turnID == "" || delta == "" {
			return
		}
		phase := extractNotificationString(notification.Params,
			[]string{"phase"},
			[]string{"item", "phase"},
		)
		var directDeltaKind agenttypes.RuntimeTurnDeltaKind
		if phase != "" {
			directDeltaKind = deltaKindForAgentMessagePhase(phase)
		}
		deltaKind := firstNonEmptyDeltaKind(directDeltaKind, c.agentMessageDeltaKind(itemID))
		if itemID != "" {
			c.appendAgentMessageDelta(itemID, turnID, delta, deltaKind)
		}
		sequence := c.nextTimelineSequence(turnID)
		if itemID == "" {
			itemID = fmt.Sprintf("%s:message", turnID)
		}
		c.emitTimelineEvent(domain.AgentTimelineEvent{
			SchemaVersion: domain.AgentTimelineSchemaVersion,
			EventID:       fmt.Sprintf("%s:%d:%s:%s", turnID, sequence, domain.AgentTimelineEventItemDelta, itemID),
			Sequence:      sequence,
			ThreadID:      threadID,
			TurnID:        turnID,
			ItemID:        itemID,
			EventType:     domain.AgentTimelineEventItemDelta,
			ItemType:      domain.AgentTimelineItemMessage,
			Role:          domain.AgentTimelineRoleAssistant,
			Phase:         timelinePhaseForDeltaKind(deltaKind),
			Status:        domain.AgentTimelineStatusRunning,
			Content: &domain.AgentTimelineContent{
				ContentType: domain.AgentTimelineContentMarkdown,
				Delta:       delta,
				AppendMode:  domain.AgentTimelineAppendAppend,
			},
			Raw: codexTimelineRaw(notification.Method, notification.Params),
		})
		c.emitTurnEvent(agenttypes.RuntimeTurnEvent{
			Type:      agenttypes.RuntimeTurnEventTypeDelta,
			RequestID: requestID,
			ThreadID:  threadID,
			TurnID:    turnID,
			Sequence:  c.nextDeltaSequence(turnID),
			Delta:     delta,
			DeltaKind: deltaKind,
		})
	case "item/plan/delta":
		c.emitGenericTimelineDelta(notification, domain.AgentTimelineItemPlan, domain.AgentTimelinePhaseProgress, domain.AgentTimelineContentMarkdown)
	case "item/reasoning/summaryTextDelta":
		c.emitGenericTimelineDelta(notification, domain.AgentTimelineItemReasoning, domain.AgentTimelinePhaseAnalysis, domain.AgentTimelineContentMarkdown)
	case "item/reasoning/textDelta":
		c.emitRawTimelineEvent(notification, domain.AgentTimelineItemReasoning, domain.AgentTimelinePhaseAnalysis)
	case "command/exec/outputDelta", "item/commandExecution/outputDelta":
		c.emitGenericTimelineDelta(notification, domain.AgentTimelineItemCommand, domain.AgentTimelinePhaseProgress, domain.AgentTimelineContentTerminal)
	case "item/fileChange/outputDelta":
		c.emitGenericTimelineDelta(notification, domain.AgentTimelineItemFileChange, domain.AgentTimelinePhaseProgress, domain.AgentTimelineContentDiff)
	case "item/mcpToolCall/progress":
		c.emitGenericTimelineDelta(notification, domain.AgentTimelineItemMCPTool, domain.AgentTimelinePhaseProgress, domain.AgentTimelineContentText)
	case "item/started":
		if event, ok := codexItemTimelineEvent(notification.Params, domain.AgentTimelineEventItemStarted, domain.AgentTimelineStatusRunning, c.nextTimelineSequence(c.resolveTurnIDFromParams(notification.Params))); ok {
			c.emitTimelineEvent(event)
		}
		if c.rememberAgentMessageItem(notification.Params) {
			return
		}
		c.emitItemProgressDelta(notification.Params, true)
	case "item/completed":
		threadID, turnID, requestID := extractTurnNotificationIDs(notification.Params)
		turnID = c.resolveTurnID(threadID, turnID)
		if event, ok := codexItemTimelineEvent(notification.Params, domain.AgentTimelineEventItemCompleted, domain.AgentTimelineStatusCompleted, c.nextTimelineSequence(turnID)); ok {
			c.emitTimelineEvent(event)
		}
		itemID, completedText, completedKind, ok := extractCompletedAgentMessage(notification.Params)
		if !ok {
			c.emitItemProgressDelta(notification.Params, false)
			return
		}
		if threadID == "" || turnID == "" {
			return
		}

		if missingText, deltaKind := c.takeCompletedAgentMessageDelta(itemID, completedText, completedKind); missingText != "" {
			c.emitTurnEvent(agenttypes.RuntimeTurnEvent{
				Type:      agenttypes.RuntimeTurnEventTypeDelta,
				RequestID: requestID,
				ThreadID:  threadID,
				TurnID:    turnID,
				Sequence:  c.nextDeltaSequence(turnID),
				Delta:     missingText,
				DeltaKind: deltaKind,
			})
		}
	case "error":
		threadID, turnID, _ := extractTurnNotificationIDs(notification.Params)
		turnID = c.resolveTurnID(threadID, turnID)
		errorMessage := extractNotificationString(notification.Params,
			[]string{"error", "message"},
			[]string{"message"},
		)
		if threadID == "" || turnID == "" || errorMessage == "" {
			return
		}
		if extractNotificationBool(notification.Params, []string{"willRetry"}) {
			return
		}
		c.rememberTurnError(turnID, errorMessage)
		sequence := c.nextTimelineSequence(turnID)
		c.emitTimelineEvent(domain.AgentTimelineEvent{
			SchemaVersion: domain.AgentTimelineSchemaVersion,
			EventID:       fmt.Sprintf("%s:%d:error", turnID, sequence),
			Sequence:      sequence,
			ThreadID:      threadID,
			TurnID:        turnID,
			EventType:     domain.AgentTimelineEventItemFailed,
			ItemType:      domain.AgentTimelineItemUnknown,
			Phase:         domain.AgentTimelinePhaseSystem,
			Status:        domain.AgentTimelineStatusFailed,
			Error:         &domain.AgentTimelineError{Message: errorMessage},
			Raw:           codexTimelineRaw(notification.Method, notification.Params),
		})
	case "turn/completed":
		threadID, turnID, requestID := extractTurnNotificationIDs(notification.Params)
		turnID = c.resolveTurnID(threadID, turnID)
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
		eventType := agenttypes.RuntimeTurnEventTypeCompleted
		if status == domain.TurnStatusFailed {
			eventType = agenttypes.RuntimeTurnEventTypeFailed
		}
		errorMessage := extractNotificationString(notification.Params,
			[]string{"error", "message"},
			[]string{"turn", "error", "message"},
		)
		storedError := ""
		c.deltaMu.Lock()
		delete(c.deltaSequence, turnID)
		if c.activeTurnsByThread[threadID] == turnID {
			delete(c.activeTurnsByThread, threadID)
		}
		for itemID, state := range c.agentMessageText {
			if state.turnID == turnID {
				delete(c.agentMessageText, itemID)
			}
		}
		storedError = c.turnErrors[turnID]
		delete(c.turnErrors, turnID)
		c.deltaMu.Unlock()
		if errorMessage == "" {
			errorMessage = storedError
		}
		timelineEventType := domain.AgentTimelineEventTurnCompleted
		timelineStatus := domain.AgentTimelineStatusCompleted
		if eventType == agenttypes.RuntimeTurnEventTypeFailed {
			timelineEventType = domain.AgentTimelineEventTurnFailed
			timelineStatus = domain.AgentTimelineStatusFailed
		}
		if event, ok := codexTurnTimelineEvent(notification.Params, timelineEventType, timelineStatus, c.nextTimelineSequence(turnID), errorMessage); ok {
			c.emitTimelineEvent(event)
		}
		c.deltaMu.Lock()
		delete(c.timelineSequence, turnID)
		c.deltaMu.Unlock()
		c.emitTurnEvent(agenttypes.RuntimeTurnEvent{
			Type:         eventType,
			RequestID:    requestID,
			ErrorMessage: errorMessage,
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
		if requestID == "" {
			return
		}

		pending, ok := c.takePendingApproval(requestID)
		if !ok {
			return
		}
		decision := extractNotificationString(notification.Params,
			[]string{"decision"},
			[]string{"result", "decision"},
			[]string{"response", "decision"},
			[]string{"resolution"},
		)
		if pending.request.ThreadID != "" && pending.request.TurnID != "" {
			c.emitTimelineEvent(codexApprovalTimelineEvent(
				pending.request.ThreadID,
				pending.request.TurnID,
				requestID,
				pending.request.ItemID,
				pending.request.Kind,
				pending.request.Reason,
				pending.request.Command,
				pending.request.UserInputQuestions,
				decision,
				c.nextTimelineSequence(pending.request.TurnID),
				true,
			))
		}
		c.emitApprovalResolved(ApprovalResolvedEvent{
			RequestID: requestID,
			ThreadID:  pending.request.ThreadID,
			TurnID:    pending.request.TurnID,
			Decision:  decision,
		})
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

	userInputQuestions := extractToolUserInputQuestions(request.Params)
	reason := extractNotificationString(request.Params,
		[]string{"reason"},
		[]string{"message"},
		[]string{"item", "reason"},
	)
	if reason == "" && approvalKind == "tool_user_input" && len(userInputQuestions) > 0 {
		reason = userInputQuestions[0].Text
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
		Kind:   approvalKind,
		Reason: reason,
		Command: extractNotificationString(request.Params,
			[]string{"command"},
			[]string{"item", "command"},
		),
		Session: extractNotificationString(request.Params,
			[]string{"session"},
			[]string{"item", "session"},
		),
		Permissions: extractNotificationMap(request.Params,
			[]string{"permissions"},
			[]string{"item", "permissions"},
		),
		UserInputQuestions: userInputQuestions,
	}

	c.approvalMu.Lock()
	c.pendingApprovals[requestID] = pendingApprovalRequest{
		id:      append(json.RawMessage(nil), request.ID...),
		request: approval,
	}
	handler := c.approvalHandler
	c.approvalMu.Unlock()

	if approval.ThreadID != "" && approval.TurnID != "" {
		c.emitTimelineEvent(codexApprovalTimelineEvent(
			approval.ThreadID,
			approval.TurnID,
			approval.RequestID,
			approval.ItemID,
			approval.Kind,
			approval.Reason,
			approval.Command,
			approval.UserInputQuestions,
			"",
			c.nextTimelineSequence(approval.TurnID),
			false,
		))
	}

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

func (c *AppServerClient) emitTimelineEvent(event domain.AgentTimelineEvent) {
	event = event.WithDefaults()
	if event.EventID == "" {
		event.EventID = fmt.Sprintf("%s:%d:%s:%s", event.TurnID, event.Sequence, event.EventType, event.ItemID)
	}
	c.turnEventMu.RLock()
	handler := c.timelineEventHandler
	c.turnEventMu.RUnlock()
	if handler != nil {
		handler(event)
	}
}

func (c *AppServerClient) emitItemProgressDelta(params json.RawMessage, started bool) {
	threadID, turnID, requestID := extractTurnNotificationIDs(params)
	turnID = c.resolveTurnID(threadID, turnID)
	delta := extractItemProgressDelta(params, started)
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
		DeltaKind: agenttypes.RuntimeTurnDeltaKindProgress,
	})
}

func (c *AppServerClient) emitGenericTimelineDelta(notification jsonRPCNotification, itemType domain.AgentTimelineItemType, phase domain.AgentTimelinePhase, contentType domain.AgentTimelineContentType) {
	threadID, turnID, _ := extractTurnNotificationIDs(notification.Params)
	turnID = c.resolveTurnID(threadID, turnID)
	itemID := extractNotificationString(notification.Params,
		[]string{"itemId"},
		[]string{"item", "id"},
	)
	delta := extractNotificationText(notification.Params,
		[]string{"delta"},
		[]string{"text"},
		[]string{"output"},
		[]string{"item", "delta"},
		[]string{"item", "text"},
	)
	if threadID == "" || turnID == "" || delta == "" {
		return
	}
	sequence := c.nextTimelineSequence(turnID)
	if itemID == "" {
		itemID = fmt.Sprintf("%s:%s", turnID, itemType)
	}
	role := domain.AgentTimelineRoleAssistant
	if itemType == domain.AgentTimelineItemCommand || itemType == domain.AgentTimelineItemMCPTool || itemType == domain.AgentTimelineItemFileChange {
		role = domain.AgentTimelineRoleTool
	}
	c.emitTimelineEvent(domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       fmt.Sprintf("%s:%d:%s:%s", turnID, sequence, domain.AgentTimelineEventItemDelta, itemID),
		Sequence:      sequence,
		ThreadID:      threadID,
		TurnID:        turnID,
		ItemID:        itemID,
		EventType:     domain.AgentTimelineEventItemDelta,
		ItemType:      itemType,
		Role:          role,
		Phase:         phase,
		Status:        domain.AgentTimelineStatusRunning,
		Content: &domain.AgentTimelineContent{
			ContentType: contentType,
			Delta:       delta,
			AppendMode:  domain.AgentTimelineAppendAppend,
		},
		Raw: codexTimelineRaw(notification.Method, notification.Params),
	})
}

func (c *AppServerClient) emitRawTimelineEvent(notification jsonRPCNotification, itemType domain.AgentTimelineItemType, phase domain.AgentTimelinePhase) {
	threadID, turnID, _ := extractTurnNotificationIDs(notification.Params)
	turnID = c.resolveTurnID(threadID, turnID)
	if threadID == "" || turnID == "" {
		return
	}
	itemID := extractNotificationString(notification.Params,
		[]string{"itemId"},
		[]string{"item", "id"},
	)
	sequence := c.nextTimelineSequence(turnID)
	c.emitTimelineEvent(domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       fmt.Sprintf("%s:%d:%s:%s", turnID, sequence, domain.AgentTimelineEventItemDelta, itemID),
		Sequence:      sequence,
		ThreadID:      threadID,
		TurnID:        turnID,
		ItemID:        itemID,
		EventType:     domain.AgentTimelineEventItemDelta,
		ItemType:      itemType,
		Role:          domain.AgentTimelineRoleAssistant,
		Phase:         phase,
		Status:        domain.AgentTimelineStatusRunning,
		Raw:           codexTimelineRaw(notification.Method, notification.Params),
	})
}

func (c *AppServerClient) resolveTurnIDFromParams(params json.RawMessage) string {
	threadID, turnID, _ := extractTurnNotificationIDs(params)
	return c.resolveTurnID(threadID, turnID)
}

func (c *AppServerClient) emitApprovalResolved(event ApprovalResolvedEvent) {
	c.approvalMu.RLock()
	handler := c.approvalResolvedHandler
	c.approvalMu.RUnlock()
	if handler != nil {
		handler(event)
	}
}

func (c *AppServerClient) deletePendingApproval(requestID string) {
	c.approvalMu.Lock()
	delete(c.pendingApprovals, requestID)
	c.approvalMu.Unlock()
}

func (c *AppServerClient) takePendingApproval(requestID string) (pendingApprovalRequest, bool) {
	c.approvalMu.Lock()
	defer c.approvalMu.Unlock()

	pending, ok := c.pendingApprovals[requestID]
	if !ok {
		return pendingApprovalRequest{}, false
	}
	delete(c.pendingApprovals, requestID)
	return pending, true
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

func (c *AppServerClient) nextTimelineSequence(turnID string) int {
	c.deltaMu.Lock()
	defer c.deltaMu.Unlock()
	c.timelineSequence[turnID]++
	return c.timelineSequence[turnID]
}

func (c *AppServerClient) resolveTurnID(threadID string, turnID string) string {
	turnID = strings.TrimSpace(turnID)
	if turnID != "" {
		return turnID
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return ""
	}

	c.deltaMu.Lock()
	defer c.deltaMu.Unlock()
	return c.activeTurnsByThread[threadID]
}

func (c *AppServerClient) rememberAgentMessageItem(params json.RawMessage) bool {
	itemID, kind, ok := extractAgentMessageItemKind(params)
	if !ok || itemID == "" {
		return false
	}

	c.deltaMu.Lock()
	state := c.agentMessageText[itemID]
	state.kind = kind
	c.agentMessageText[itemID] = state
	c.deltaMu.Unlock()
	return true
}

func (c *AppServerClient) agentMessageDeltaKind(itemID string) agenttypes.RuntimeTurnDeltaKind {
	if itemID == "" {
		return agenttypes.RuntimeTurnDeltaKindContent
	}
	c.deltaMu.Lock()
	defer c.deltaMu.Unlock()
	return normalizeDeltaKind(c.agentMessageText[itemID].kind)
}

func (c *AppServerClient) appendAgentMessageDelta(itemID string, turnID string, delta string, kind agenttypes.RuntimeTurnDeltaKind) {
	c.deltaMu.Lock()
	state := c.agentMessageText[itemID]
	state.turnID = turnID
	state.text += delta
	state.kind = normalizeDeltaKind(kind)
	c.agentMessageText[itemID] = state
	c.deltaMu.Unlock()
}

func (c *AppServerClient) takeCompletedAgentMessageDelta(itemID string, completedText string, fallbackKind agenttypes.RuntimeTurnDeltaKind) (string, agenttypes.RuntimeTurnDeltaKind) {
	c.deltaMu.Lock()
	state := c.agentMessageText[itemID]
	delete(c.agentMessageText, itemID)
	c.deltaMu.Unlock()

	return trimSharedPrefix(state.text, completedText), normalizeDeltaKind(firstNonEmptyDeltaKind(state.kind, fallbackKind))
}

func (c *AppServerClient) rememberTurnError(turnID string, errorMessage string) {
	c.deltaMu.Lock()
	c.turnErrors[turnID] = errorMessage
	c.deltaMu.Unlock()
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

func extractNotificationMap(params json.RawMessage, paths ...[]string) map[string]any {
	if len(params) == 0 {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return nil
	}

	for _, path := range paths {
		if value, ok := nestedValue(payload, path...); ok {
			typed, ok := value.(map[string]any)
			if !ok || len(typed) == 0 {
				continue
			}

			cloned := make(map[string]any, len(typed))
			for key, item := range typed {
				cloned[key] = item
			}
			return cloned
		}
	}

	return nil
}

func extractNotificationBool(params json.RawMessage, paths ...[]string) bool {
	if len(params) == 0 {
		return false
	}

	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return false
	}

	for _, path := range paths {
		if value, ok := nestedValue(payload, path...); ok {
			typed, ok := value.(bool)
			if ok {
				return typed
			}
		}
	}

	return false
}

func extractAgentMessageItemKind(params json.RawMessage) (itemID string, kind agenttypes.RuntimeTurnDeltaKind, ok bool) {
	if len(params) == 0 {
		return "", "", false
	}

	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return "", "", false
	}

	itemType, _ := nestedValue(payload, "item", "type")
	typeValue, typeOK := itemType.(string)
	if !typeOK || strings.TrimSpace(typeValue) != "agentMessage" {
		return "", "", false
	}

	itemIDValue, _ := nestedValue(payload, "item", "id")
	itemID, _ = itemIDValue.(string)
	if strings.TrimSpace(itemID) == "" {
		return "", "", false
	}

	phaseValue, _ := nestedValue(payload, "item", "phase")
	phase, _ := phaseValue.(string)
	return strings.TrimSpace(itemID), deltaKindForAgentMessagePhase(phase), true
}

func extractCompletedAgentMessage(params json.RawMessage) (itemID string, text string, kind agenttypes.RuntimeTurnDeltaKind, ok bool) {
	if len(params) == 0 {
		return "", "", "", false
	}

	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return "", "", "", false
	}

	itemType, _ := nestedValue(payload, "item", "type")
	typeValue, typeOK := itemType.(string)
	if !typeOK || strings.TrimSpace(typeValue) != "agentMessage" {
		return "", "", "", false
	}

	itemIDValue, _ := nestedValue(payload, "item", "id")
	itemID, _ = itemIDValue.(string)
	if strings.TrimSpace(itemID) == "" {
		return "", "", "", false
	}

	textValue, _ := nestedValue(payload, "item", "text")
	text, _ = textValue.(string)
	phaseValue, _ := nestedValue(payload, "item", "phase")
	phase, _ := phaseValue.(string)
	return strings.TrimSpace(itemID), text, deltaKindForAgentMessagePhase(phase), true
}

func deltaKindForAgentMessagePhase(phase string) agenttypes.RuntimeTurnDeltaKind {
	switch strings.TrimSpace(phase) {
	case "commentary":
		return agenttypes.RuntimeTurnDeltaKindProgress
	case "final", "final_answer":
		return agenttypes.RuntimeTurnDeltaKindContent
	default:
		return agenttypes.RuntimeTurnDeltaKindContent
	}
}

func timelinePhaseForDeltaKind(kind agenttypes.RuntimeTurnDeltaKind) domain.AgentTimelinePhase {
	if kind == agenttypes.RuntimeTurnDeltaKindProgress {
		return domain.AgentTimelinePhaseProgress
	}
	return domain.AgentTimelinePhaseFinal
}

func normalizeDeltaKind(kind agenttypes.RuntimeTurnDeltaKind) agenttypes.RuntimeTurnDeltaKind {
	if kind == agenttypes.RuntimeTurnDeltaKindProgress {
		return agenttypes.RuntimeTurnDeltaKindProgress
	}
	return agenttypes.RuntimeTurnDeltaKindContent
}

func firstNonEmptyDeltaKind(values ...agenttypes.RuntimeTurnDeltaKind) agenttypes.RuntimeTurnDeltaKind {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return agenttypes.RuntimeTurnDeltaKindContent
}

func extractItemProgressDelta(params json.RawMessage, started bool) string {
	if len(params) == 0 {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return ""
	}

	rawItem, ok := nestedValue(payload, "item")
	if !ok {
		return ""
	}
	item, ok := rawItem.(map[string]any)
	if !ok {
		return ""
	}

	itemType := stringFromMap(item, "type")
	switch itemType {
	case "webSearch":
		if started {
			return "\n\n正在搜索..."
		}
		action, _ := item["action"].(map[string]any)
		actionType := stringFromMap(action, "type")
		if actionType == "openPage" || actionType == "open_page" {
			if url := stringFromMap(action, "url"); url != "" {
				return "\n\n已打开页面：" + truncateProgressValue(url)
			}
			return "\n\n已打开页面。"
		}
		query := stringFromMap(action, "query")
		if query == "" {
			query = stringFromMap(item, "query")
		}
		if query != "" {
			return "\n\n已完成搜索：" + truncateProgressValue(query)
		}
		return "\n\n搜索完成。"
	case "commandExecution":
		if started {
			command := stringFromMap(item, "command")
			if command != "" {
				return "\n\n正在执行命令：`" + escapeInlineCode(truncateProgressValue(command)) + "`"
			}
			return "\n\n正在执行命令..."
		}
		status := stringFromMap(item, "status")
		if status == "failed" {
			return "\n\n命令执行失败。"
		}
		return "\n\n命令执行完成。"
	default:
		return ""
	}
}

func stringFromMap(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func truncateProgressValue(value string) string {
	const maxRunes = 180
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func escapeInlineCode(value string) string {
	return strings.ReplaceAll(value, "`", "\\`")
}

func trimSharedPrefix(existing string, completed string) string {
	existingRunes := []rune(existing)
	completedRunes := []rune(completed)
	prefixLen := 0
	maxLen := len(existingRunes)
	if len(completedRunes) < maxLen {
		maxLen = len(completedRunes)
	}
	for prefixLen < maxLen && existingRunes[prefixLen] == completedRunes[prefixLen] {
		prefixLen++
	}
	return string(completedRunes[prefixLen:])
}

func flattenThreadMessages(turns []turnRecord) []domain.ThreadMessage {
	if len(turns) == 0 {
		return nil
	}

	messages := make([]domain.ThreadMessage, 0, len(turns)*2)
	for _, turn := range turns {
		turnID := strings.TrimSpace(turn.ID)
		agentMessageID := ""
		agentTexts := make([]string, 0)
		for _, item := range turn.Items {
			text := flattenThreadItemText(item)
			if text == "" {
				continue
			}
			switch strings.TrimSpace(item.Type) {
			case "userMessage":
				messages = append(messages, domain.ThreadMessage{
					ID:     firstNonEmptyString(strings.TrimSpace(item.ID), "user:"+turnID),
					TurnID: turnID,
					Kind:   domain.ThreadMessageKindUser,
					Text:   text,
				})
			case "agentMessage":
				if agentMessageID == "" {
					agentMessageID = firstNonEmptyString(strings.TrimSpace(item.ID), "agent:"+turnID)
				}
				agentTexts = append(agentTexts, text)
			}
		}

		if len(agentTexts) > 0 {
			messages = append(messages, domain.ThreadMessage{
				ID:     agentMessageID,
				TurnID: turnID,
				Kind:   domain.ThreadMessageKindAgent,
				Text:   strings.Join(agentTexts, "\n\n"),
			})
		}

		if turn.Status == domain.TurnStatusFailed && turn.Error != nil && strings.TrimSpace(turn.Error.Message) != "" {
			messages = append(messages, domain.ThreadMessage{
				ID:     firstNonEmptyString("completed:" + turnID),
				TurnID: turnID,
				Kind:   domain.ThreadMessageKindSystem,
				Text:   fmt.Sprintf("Turn %s failed: %s", turnID, strings.TrimSpace(turn.Error.Message)),
			})
		}
	}

	return messages
}

func flattenThreadItemText(item threadItemRecord) string {
	switch strings.TrimSpace(item.Type) {
	case "agentMessage":
		return strings.TrimSpace(item.Text)
	case "userMessage":
		parts := make([]string, 0, len(item.Content))
		for _, input := range item.Content {
			if strings.TrimSpace(input.Type) != "text" {
				continue
			}
			if text := strings.TrimSpace(input.Text); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func extractToolUserInputQuestions(params json.RawMessage) []approvalQuestion {
	if len(params) == 0 {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return nil
	}

	rawQuestions, ok := nestedValue(payload, "questions")
	if !ok {
		rawQuestions, ok = nestedValue(payload, "item", "questions")
	}
	if !ok {
		return nil
	}

	questions, ok := rawQuestions.([]any)
	if !ok {
		return nil
	}

	result := make([]approvalQuestion, 0, len(questions))
	for index, rawQuestion := range questions {
		question := approvalQuestion{Key: fmt.Sprintf("%d", index)}
		switch typed := rawQuestion.(type) {
		case string:
			question.Text = strings.TrimSpace(typed)
		case map[string]any:
			question.Key = approvalQuestionKey(typed, index)
			question.Header = extractApprovalQuestionHeader(typed)
			question.Text = extractApprovalQuestionText(typed)
			question.Options = extractApprovalQuestionOptions(typed)
		default:
			continue
		}
		result = append(result, question)
	}

	return result
}

func approvalQuestionKey(question map[string]any, index int) string {
	for _, key := range []string{"name", "id"} {
		value, ok := question[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fmt.Sprintf("%d", index)
}

func extractApprovalQuestionHeader(question map[string]any) string {
	for _, key := range []string{"header", "title"} {
		value, ok := question[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func extractApprovalQuestionText(question map[string]any) string {
	for _, key := range []string{"question", "text", "prompt", "label"} {
		value, ok := question[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func extractApprovalQuestionOptions(question map[string]any) []string {
	rawOptions, ok := question["options"]
	if !ok {
		return nil
	}

	options, ok := rawOptions.([]any)
	if !ok {
		return nil
	}

	values := make([]string, 0, len(options))
	for _, rawOption := range options {
		switch typed := rawOption.(type) {
		case string:
			if value := strings.TrimSpace(typed); value != "" {
				values = append(values, value)
			}
		case map[string]any:
			for _, key := range []string{"value", "id", "name", "label", "text"} {
				value, ok := typed[key].(string)
				if ok && strings.TrimSpace(value) != "" {
					values = append(values, strings.TrimSpace(value))
					break
				}
			}
		}
	}

	return values
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
