package codex

import (
	"encoding/json"
	"fmt"
	"strings"

	"code-agent-gateway/common/domain"
)

func codexItemTimelineEvent(params json.RawMessage, eventType domain.AgentTimelineEventType, status domain.AgentTimelineStatus, sequence int) (domain.AgentTimelineEvent, bool) {
	threadID, turnID, _ := extractTurnNotificationIDs(params)
	item, ok := extractNotificationItem(params)
	if !ok || threadID == "" || turnID == "" {
		return domain.AgentTimelineEvent{}, false
	}

	itemID := stringFromMap(item, "id")
	itemType := stringFromMap(item, "type")
	if itemID == "" {
		itemID = fmt.Sprintf("%s:%s:%d", turnID, itemType, sequence)
	}

	event := domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       fmt.Sprintf("%s:%d:%s:%s", turnID, sequence, eventType, itemID),
		Sequence:      sequence,
		ThreadID:      threadID,
		TurnID:        turnID,
		ItemID:        itemID,
		EventType:     eventType,
		ItemType:      codexTimelineItemType(itemType),
		Role:          codexTimelineRole(itemType),
		Phase:         codexTimelinePhase(itemType, stringFromMap(item, "phase")),
		Status:        firstTimelineStatus(status, codexTimelineStatus(stringFromMap(item, "status"))),
		Raw:           codexTimelineRaw("item/"+string(eventType), params),
	}

	if event.Status == "" && eventType == domain.AgentTimelineEventItemStarted {
		event.Status = domain.AgentTimelineStatusRunning
	}
	if event.Status == "" && eventType == domain.AgentTimelineEventItemCompleted {
		event.Status = domain.AgentTimelineStatusCompleted
	}
	event.Tool = codexTimelineTool(itemType, item)
	event.Content = codexTimelineContentSnapshot(itemType, item)
	if event.ItemType == "" {
		event.ItemType = domain.AgentTimelineItemUnknown
	}
	return event, true
}

func codexDeltaTimelineEvent(params json.RawMessage, method string, itemID string, itemType domain.AgentTimelineItemType, phase domain.AgentTimelinePhase, contentType domain.AgentTimelineContentType, delta string, sequence int) (domain.AgentTimelineEvent, bool) {
	threadID, turnID, _ := extractTurnNotificationIDs(params)
	turnID = strings.TrimSpace(turnID)
	if threadID == "" || turnID == "" || delta == "" {
		return domain.AgentTimelineEvent{}, false
	}
	if itemID == "" {
		itemID = extractNotificationString(params, []string{"itemId"}, []string{"item", "id"})
	}
	if itemID == "" {
		itemID = fmt.Sprintf("%s:%s", turnID, itemType)
	}
	if phase == "" {
		phase = domain.AgentTimelinePhaseProgress
	}
	if contentType == "" {
		contentType = domain.AgentTimelineContentText
	}
	return domain.AgentTimelineEvent{
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
		Content: &domain.AgentTimelineContent{
			ContentType: contentType,
			Delta:       delta,
			AppendMode:  domain.AgentTimelineAppendAppend,
		},
		Raw: codexTimelineRaw(method, params),
	}, true
}

func codexTurnTimelineEvent(params json.RawMessage, eventType domain.AgentTimelineEventType, status domain.AgentTimelineStatus, sequence int, errorMessage string) (domain.AgentTimelineEvent, bool) {
	threadID, turnID, _ := extractTurnNotificationIDs(params)
	if threadID == "" || turnID == "" {
		return domain.AgentTimelineEvent{}, false
	}
	event := domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       fmt.Sprintf("%s:%d:%s", turnID, sequence, eventType),
		Sequence:      sequence,
		ThreadID:      threadID,
		TurnID:        turnID,
		EventType:     eventType,
		Status:        status,
		Raw:           codexTimelineRaw(string(eventType), params),
	}
	if errorMessage != "" {
		event.Error = &domain.AgentTimelineError{Message: errorMessage}
	}
	return event, true
}

func codexApprovalTimelineEvent(threadID string, turnID string, requestID string, itemID string, kind string, reason string, command string, questions []approvalQuestion, decision string, sequence int, resolved bool) domain.AgentTimelineEvent {
	eventType := domain.AgentTimelineEventApprovalRequested
	status := domain.AgentTimelineStatusBlocked
	if resolved {
		eventType = domain.AgentTimelineEventApprovalResolved
		status = domain.AgentTimelineStatusCompleted
	}
	approvalQuestions := make([]domain.AgentTimelineApprovalQuestion, 0, len(questions))
	for _, question := range questions {
		label := firstNonEmptyString(question.Text, question.Header, question.Key)
		approvalQuestions = append(approvalQuestions, domain.AgentTimelineApprovalQuestion{
			ID:      question.Key,
			Label:   label,
			Options: append([]string(nil), question.Options...),
		})
	}
	title := firstNonEmptyString(command, reason, kind)
	event := domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       fmt.Sprintf("%s:%d:%s:%s", turnID, sequence, eventType, requestID),
		Sequence:      sequence,
		ThreadID:      threadID,
		TurnID:        turnID,
		ItemID:        itemID,
		EventType:     eventType,
		ItemType:      domain.AgentTimelineItemTool,
		Phase:         domain.AgentTimelinePhaseProgress,
		Status:        status,
		Approval: &domain.AgentTimelineApproval{
			RequestID: requestID,
			Kind:      kind,
			Title:     title,
			Reason:    reason,
			Questions: approvalQuestions,
		},
	}
	if decision != "" {
		event.Approval.Decision = domain.AgentTimelineApprovalDecision(decision)
	}
	if command != "" {
		event.ItemType = domain.AgentTimelineItemCommand
		event.Tool = &domain.AgentTimelineTool{
			Kind:  domain.AgentTimelineToolShell,
			Name:  "command",
			Input: mustTimelineJSON(map[string]any{"command": command}),
		}
	}
	return event
}

func extractNotificationItem(params json.RawMessage) (map[string]any, bool) {
	if len(params) == 0 {
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(params, &payload); err != nil {
		return nil, false
	}
	rawItem, ok := nestedValue(payload, "item")
	if !ok {
		return nil, false
	}
	item, ok := rawItem.(map[string]any)
	return item, ok
}

func codexTimelineItemType(itemType string) domain.AgentTimelineItemType {
	switch strings.TrimSpace(itemType) {
	case "userMessage", "hookPrompt", "agentMessage":
		return domain.AgentTimelineItemMessage
	case "plan":
		return domain.AgentTimelineItemPlan
	case "reasoning":
		return domain.AgentTimelineItemReasoning
	case "commandExecution":
		return domain.AgentTimelineItemCommand
	case "fileChange":
		return domain.AgentTimelineItemFileChange
	case "mcpToolCall":
		return domain.AgentTimelineItemMCPTool
	case "dynamicToolCall":
		return domain.AgentTimelineItemTool
	case "collabAgentToolCall":
		return domain.AgentTimelineItemSubagent
	case "webSearch":
		return domain.AgentTimelineItemWebSearch
	case "imageView", "imageGeneration":
		return domain.AgentTimelineItemImage
	case "enteredReviewMode", "exitedReviewMode":
		return domain.AgentTimelineItemModeChange
	case "contextCompaction":
		return domain.AgentTimelineItemContext
	default:
		return domain.AgentTimelineItemUnknown
	}
}

func codexTimelineRole(itemType string) domain.AgentTimelineRole {
	switch strings.TrimSpace(itemType) {
	case "userMessage":
		return domain.AgentTimelineRoleUser
	case "hookPrompt":
		return domain.AgentTimelineRoleSystem
	case "commandExecution", "fileChange", "mcpToolCall", "dynamicToolCall", "webSearch":
		return domain.AgentTimelineRoleTool
	default:
		return domain.AgentTimelineRoleAssistant
	}
}

func codexTimelinePhase(itemType string, phase string) domain.AgentTimelinePhase {
	switch strings.TrimSpace(itemType) {
	case "userMessage":
		return domain.AgentTimelinePhaseInput
	case "hookPrompt", "contextCompaction", "enteredReviewMode", "exitedReviewMode":
		return domain.AgentTimelinePhaseSystem
	case "agentMessage":
		switch strings.TrimSpace(phase) {
		case "commentary":
			return domain.AgentTimelinePhaseProgress
		default:
			return domain.AgentTimelinePhaseFinal
		}
	case "reasoning":
		return domain.AgentTimelinePhaseAnalysis
	default:
		return domain.AgentTimelinePhaseProgress
	}
}

func codexTimelineStatus(status string) domain.AgentTimelineStatus {
	switch strings.TrimSpace(status) {
	case "inProgress", "running":
		return domain.AgentTimelineStatusRunning
	case "completed", "success":
		return domain.AgentTimelineStatusCompleted
	case "failed", "error":
		return domain.AgentTimelineStatusFailed
	case "declined":
		return domain.AgentTimelineStatusDeclined
	default:
		return ""
	}
}

func codexTimelineTool(itemType string, item map[string]any) *domain.AgentTimelineTool {
	switch strings.TrimSpace(itemType) {
	case "commandExecution":
		return &domain.AgentTimelineTool{
			Kind:  domain.AgentTimelineToolShell,
			Name:  "command",
			Input: mustTimelineJSON(map[string]any{"command": stringFromMap(item, "command"), "cwd": stringFromMap(item, "cwd")}),
		}
	case "fileChange":
		return &domain.AgentTimelineTool{Kind: domain.AgentTimelineToolFileEdit, Name: "fileChange", Input: mustTimelineJSON(item["changes"])}
	case "mcpToolCall":
		return &domain.AgentTimelineTool{Kind: domain.AgentTimelineToolMCP, Name: stringFromMap(item, "tool"), Input: mustTimelineJSON(map[string]any{"server": stringFromMap(item, "server"), "arguments": item["arguments"]}), Output: mustTimelineJSON(firstNonNil(item["result"], item["error"]))}
	case "dynamicToolCall":
		return &domain.AgentTimelineTool{Kind: domain.AgentTimelineToolFunction, Name: stringFromMap(item, "tool"), Input: mustTimelineJSON(item["arguments"]), Output: mustTimelineJSON(item["contentItems"])}
	case "collabAgentToolCall":
		return &domain.AgentTimelineTool{Kind: domain.AgentTimelineToolSubagent, Name: stringFromMap(item, "tool"), Input: mustTimelineJSON(map[string]any{"prompt": item["prompt"], "receiverThreadIds": item["receiverThreadIds"]}), Output: mustTimelineJSON(item["agentsStates"])}
	case "webSearch":
		return &domain.AgentTimelineTool{Kind: domain.AgentTimelineToolWebSearch, Name: "webSearch", Input: mustTimelineJSON(map[string]any{"query": item["query"], "action": item["action"]})}
	case "imageGeneration":
		return &domain.AgentTimelineTool{Kind: domain.AgentTimelineToolImageGeneration, Name: "imageGeneration", Input: mustTimelineJSON(map[string]any{"revisedPrompt": item["revisedPrompt"]}), Output: mustTimelineJSON(firstNonNil(item["savedPath"], item["result"]))}
	default:
		return nil
	}
}

func codexTimelineContentSnapshot(itemType string, item map[string]any) *domain.AgentTimelineContent {
	switch strings.TrimSpace(itemType) {
	case "agentMessage", "plan":
		if text := stringFromMap(item, "text"); text != "" {
			return &domain.AgentTimelineContent{ContentType: domain.AgentTimelineContentMarkdown, Text: text, AppendMode: domain.AgentTimelineAppendSnapshot}
		}
	case "reasoning":
		if summary, ok := item["summary"]; ok {
			return &domain.AgentTimelineContent{ContentType: domain.AgentTimelineContentMarkdown, Snapshot: mustTimelineJSON(summary), AppendMode: domain.AgentTimelineAppendSnapshot}
		}
	case "commandExecution":
		if output := stringFromMap(item, "aggregatedOutput"); output != "" {
			return &domain.AgentTimelineContent{ContentType: domain.AgentTimelineContentTerminal, Text: output, AppendMode: domain.AgentTimelineAppendSnapshot}
		}
	case "imageView":
		if path := stringFromMap(item, "path"); path != "" {
			return &domain.AgentTimelineContent{ContentType: domain.AgentTimelineContentImage, Text: path, AppendMode: domain.AgentTimelineAppendSnapshot}
		}
	case "imageGeneration":
		if path := firstNonEmptyString(stringFromMap(item, "savedPath"), stringFromMap(item, "result")); path != "" {
			return &domain.AgentTimelineContent{ContentType: domain.AgentTimelineContentImage, Text: path, AppendMode: domain.AgentTimelineAppendSnapshot}
		}
	}
	return nil
}

func codexTimelineRaw(method string, payload json.RawMessage) *domain.AgentTimelineRaw {
	return &domain.AgentTimelineRaw{
		Provider: "codex",
		Method:   method,
		Payload:  append(json.RawMessage(nil), payload...),
	}
}

func firstTimelineStatus(values ...domain.AgentTimelineStatus) domain.AgentTimelineStatus {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func mustTimelineJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil || string(encoded) == "null" {
		return nil
	}
	return encoded
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
