package domain

import (
	"encoding/json"
	"errors"
	"strings"
)

type AgentTimelineEventType string
type AgentTimelineItemType string
type AgentTimelineRole string
type AgentTimelinePhase string
type AgentTimelineStatus string
type AgentTimelineContentType string
type AgentTimelineAppendMode string
type AgentTimelineToolKind string
type AgentTimelineApprovalDecision string

const AgentTimelineSchemaVersion = "agent-timeline.v1"

const (
	AgentTimelineEventTurnStarted       AgentTimelineEventType = "turn.started"
	AgentTimelineEventTurnCompleted     AgentTimelineEventType = "turn.completed"
	AgentTimelineEventTurnFailed        AgentTimelineEventType = "turn.failed"
	AgentTimelineEventItemStarted       AgentTimelineEventType = "item.started"
	AgentTimelineEventItemDelta         AgentTimelineEventType = "item.delta"
	AgentTimelineEventItemCompleted     AgentTimelineEventType = "item.completed"
	AgentTimelineEventItemFailed        AgentTimelineEventType = "item.failed"
	AgentTimelineEventApprovalRequested AgentTimelineEventType = "approval.requested"
	AgentTimelineEventApprovalResolved  AgentTimelineEventType = "approval.resolved"
	AgentTimelineEventSystem            AgentTimelineEventType = "system.event"
)

const (
	AgentTimelineItemMessage       AgentTimelineItemType = "message"
	AgentTimelineItemReasoning     AgentTimelineItemType = "reasoning"
	AgentTimelineItemPlan          AgentTimelineItemType = "plan"
	AgentTimelineItemTool          AgentTimelineItemType = "tool"
	AgentTimelineItemCommand       AgentTimelineItemType = "command"
	AgentTimelineItemFileChange    AgentTimelineItemType = "file_change"
	AgentTimelineItemWebSearch     AgentTimelineItemType = "web_search"
	AgentTimelineItemBrowserAction AgentTimelineItemType = "browser_action"
	AgentTimelineItemMCPTool       AgentTimelineItemType = "mcp_tool"
	AgentTimelineItemSubagent      AgentTimelineItemType = "subagent"
	AgentTimelineItemArtifact      AgentTimelineItemType = "artifact"
	AgentTimelineItemImage         AgentTimelineItemType = "image"
	AgentTimelineItemContext       AgentTimelineItemType = "context"
	AgentTimelineItemModeChange    AgentTimelineItemType = "mode_change"
	AgentTimelineItemUnknown       AgentTimelineItemType = "unknown"
)

const (
	AgentTimelineRoleUser      AgentTimelineRole = "user"
	AgentTimelineRoleAssistant AgentTimelineRole = "assistant"
	AgentTimelineRoleSystem    AgentTimelineRole = "system"
	AgentTimelineRoleTool      AgentTimelineRole = "tool"
)

const (
	AgentTimelinePhaseInput    AgentTimelinePhase = "input"
	AgentTimelinePhaseAnalysis AgentTimelinePhase = "analysis"
	AgentTimelinePhaseProgress AgentTimelinePhase = "progress"
	AgentTimelinePhaseFinal    AgentTimelinePhase = "final"
	AgentTimelinePhaseSystem   AgentTimelinePhase = "system"
)

const (
	AgentTimelineStatusPending   AgentTimelineStatus = "pending"
	AgentTimelineStatusRunning   AgentTimelineStatus = "running"
	AgentTimelineStatusBlocked   AgentTimelineStatus = "blocked"
	AgentTimelineStatusCompleted AgentTimelineStatus = "completed"
	AgentTimelineStatusFailed    AgentTimelineStatus = "failed"
	AgentTimelineStatusDeclined  AgentTimelineStatus = "declined"
	AgentTimelineStatusCancelled AgentTimelineStatus = "cancelled"
)

const (
	AgentTimelineContentMarkdown AgentTimelineContentType = "markdown"
	AgentTimelineContentText     AgentTimelineContentType = "text"
	AgentTimelineContentJSON     AgentTimelineContentType = "json"
	AgentTimelineContentTerminal AgentTimelineContentType = "terminal"
	AgentTimelineContentDiff     AgentTimelineContentType = "diff"
	AgentTimelineContentImage    AgentTimelineContentType = "image"
	AgentTimelineContentFile     AgentTimelineContentType = "file"
)

const (
	AgentTimelineAppendAppend   AgentTimelineAppendMode = "append"
	AgentTimelineAppendReplace  AgentTimelineAppendMode = "replace"
	AgentTimelineAppendSnapshot AgentTimelineAppendMode = "snapshot"
)

const (
	AgentTimelineToolShell           AgentTimelineToolKind = "shell"
	AgentTimelineToolWebSearch       AgentTimelineToolKind = "web_search"
	AgentTimelineToolBrowser         AgentTimelineToolKind = "browser"
	AgentTimelineToolMCP             AgentTimelineToolKind = "mcp"
	AgentTimelineToolFunction        AgentTimelineToolKind = "function"
	AgentTimelineToolFileEdit        AgentTimelineToolKind = "file_edit"
	AgentTimelineToolSubagent        AgentTimelineToolKind = "subagent"
	AgentTimelineToolImageGeneration AgentTimelineToolKind = "image_generation"
	AgentTimelineToolUnknown         AgentTimelineToolKind = "unknown"
)

const (
	AgentTimelineApprovalAccept  AgentTimelineApprovalDecision = "accept"
	AgentTimelineApprovalDecline AgentTimelineApprovalDecision = "decline"
	AgentTimelineApprovalCancel  AgentTimelineApprovalDecision = "cancel"
)

type AgentTimelineEvent struct {
	SchemaVersion string                 `json:"schemaVersion"`
	EventID       string                 `json:"eventId"`
	Sequence      int                    `json:"sequence"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	MachineID     string                 `json:"machineId,omitempty"`
	AgentID       string                 `json:"agentId,omitempty"`
	ThreadID      string                 `json:"threadId"`
	TurnID        string                 `json:"turnId,omitempty"`
	ItemID        string                 `json:"itemId,omitempty"`
	EventType     AgentTimelineEventType `json:"eventType"`
	ItemType      AgentTimelineItemType  `json:"itemType,omitempty"`
	Role          AgentTimelineRole      `json:"role,omitempty"`
	Phase         AgentTimelinePhase     `json:"phase,omitempty"`
	Status        AgentTimelineStatus    `json:"status,omitempty"`
	Content       *AgentTimelineContent  `json:"content,omitempty"`
	Tool          *AgentTimelineTool     `json:"tool,omitempty"`
	Approval      *AgentTimelineApproval `json:"approval,omitempty"`
	Error         *AgentTimelineError    `json:"error,omitempty"`
	Raw           *AgentTimelineRaw      `json:"raw,omitempty"`
}

type AgentTimelineContent struct {
	ContentType AgentTimelineContentType `json:"contentType"`
	Delta       string                   `json:"delta,omitempty"`
	Text        string                   `json:"text,omitempty"`
	Snapshot    json.RawMessage          `json:"snapshot,omitempty"`
	AppendMode  AgentTimelineAppendMode  `json:"appendMode,omitempty"`
}

type AgentTimelineTool struct {
	Kind   AgentTimelineToolKind `json:"kind"`
	Name   string                `json:"name,omitempty"`
	Input  json.RawMessage       `json:"input,omitempty"`
	Output json.RawMessage       `json:"output,omitempty"`
}

type AgentTimelineApproval struct {
	RequestID string                          `json:"requestId"`
	Kind      string                          `json:"kind"`
	Title     string                          `json:"title,omitempty"`
	Reason    string                          `json:"reason,omitempty"`
	Questions []AgentTimelineApprovalQuestion `json:"questions,omitempty"`
	Decision  AgentTimelineApprovalDecision   `json:"decision,omitempty"`
}

type AgentTimelineApprovalQuestion struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Options []string `json:"options,omitempty"`
}

type AgentTimelineError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type AgentTimelineRaw struct {
	Provider string          `json:"provider"`
	Method   string          `json:"method,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

func (e AgentTimelineEvent) Validate() error {
	if strings.TrimSpace(e.EventID) == "" {
		return errors.New("eventId is required")
	}
	if e.Sequence <= 0 {
		return errors.New("sequence must be positive")
	}
	if strings.TrimSpace(e.ThreadID) == "" {
		return errors.New("threadId is required")
	}
	if strings.TrimSpace(string(e.EventType)) == "" {
		return errors.New("eventType is required")
	}
	if e.SchemaVersion != "" && e.SchemaVersion != AgentTimelineSchemaVersion {
		return errors.New("unsupported schemaVersion")
	}
	if requiresTimelineTurnID(e.EventType) && strings.TrimSpace(e.TurnID) == "" {
		return errors.New("turnId is required")
	}
	return nil
}

func (e AgentTimelineEvent) WithDefaults() AgentTimelineEvent {
	if strings.TrimSpace(e.SchemaVersion) == "" {
		e.SchemaVersion = AgentTimelineSchemaVersion
	}
	return e
}

func requiresTimelineTurnID(eventType AgentTimelineEventType) bool {
	switch eventType {
	case AgentTimelineEventTurnStarted,
		AgentTimelineEventTurnCompleted,
		AgentTimelineEventTurnFailed,
		AgentTimelineEventItemStarted,
		AgentTimelineEventItemDelta,
		AgentTimelineEventItemCompleted,
		AgentTimelineEventItemFailed,
		AgentTimelineEventApprovalRequested,
		AgentTimelineEventApprovalResolved:
		return true
	default:
		return false
	}
}
