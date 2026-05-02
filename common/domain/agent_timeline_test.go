package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentTimelineEventValidateRequiresStableFields(t *testing.T) {
	event := AgentTimelineEvent{
		EventID:   "event-1",
		Sequence:  1,
		ThreadID:  "thread-1",
		TurnID:    "turn-1",
		EventType: AgentTimelineEventItemDelta,
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("expected valid event, got %v", err)
	}

	event.TurnID = ""
	err := event.Validate()
	if err == nil || !strings.Contains(err.Error(), "turnId") {
		t.Fatalf("expected turnId validation error, got %v", err)
	}

	event.EventType = AgentTimelineEventSystem
	if err := event.Validate(); err != nil {
		t.Fatalf("expected system event to allow empty turnId, got %v", err)
	}
}

func TestAgentTimelineEventWithDefaultsAndJSON(t *testing.T) {
	event := AgentTimelineEvent{
		EventID:   "event-1",
		Sequence:  3,
		ThreadID:  "thread-1",
		TurnID:    "turn-1",
		ItemID:    "message-1",
		EventType: AgentTimelineEventItemDelta,
		ItemType:  AgentTimelineItemMessage,
		Role:      AgentTimelineRoleAssistant,
		Phase:     AgentTimelinePhaseFinal,
		Status:    AgentTimelineStatusRunning,
		Content: &AgentTimelineContent{
			ContentType: AgentTimelineContentMarkdown,
			Delta:       "**报告**",
			AppendMode:  AgentTimelineAppendAppend,
		},
	}.WithDefaults()

	blob, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded AgentTimelineEvent
	if err := json.Unmarshal(blob, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.SchemaVersion != AgentTimelineSchemaVersion {
		t.Fatalf("expected schema version %q, got %q", AgentTimelineSchemaVersion, decoded.SchemaVersion)
	}
	if decoded.Content == nil || decoded.Content.Delta != "**报告**" {
		t.Fatalf("expected markdown delta to round-trip, got %#v", decoded.Content)
	}
}

func TestAgentTimelineToolApprovalAndRawRoundTrip(t *testing.T) {
	event := AgentTimelineEvent{
		SchemaVersion: AgentTimelineSchemaVersion,
		EventID:       "event-approval",
		Sequence:      7,
		ThreadID:      "thread-1",
		TurnID:        "turn-1",
		ItemID:        "command-1",
		EventType:     AgentTimelineEventApprovalRequested,
		ItemType:      AgentTimelineItemCommand,
		Phase:         AgentTimelinePhaseProgress,
		Status:        AgentTimelineStatusBlocked,
		Tool: &AgentTimelineTool{
			Kind:  AgentTimelineToolShell,
			Name:  "go test",
			Input: json.RawMessage(`{"command":"go test ./..."}`),
		},
		Approval: &AgentTimelineApproval{
			RequestID: "approval-1",
			Kind:      "command",
			Title:     "go test ./...",
			Questions: []AgentTimelineApprovalQuestion{
				{ID: "mode", Label: "选择模式", Options: []string{"accept", "decline"}},
			},
		},
		Raw: &AgentTimelineRaw{
			Provider: "codex",
			Method:   "serverRequest/start",
			Payload:  json.RawMessage(`{"id":"approval-1"}`),
		},
	}

	blob, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded AgentTimelineEvent
	if err := json.Unmarshal(blob, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Tool == nil || string(decoded.Tool.Input) != `{"command":"go test ./..."}` {
		t.Fatalf("expected tool input to round-trip, got %#v", decoded.Tool)
	}
	if decoded.Approval == nil || len(decoded.Approval.Questions) != 1 {
		t.Fatalf("expected approval question to round-trip, got %#v", decoded.Approval)
	}
	if decoded.Raw == nil || decoded.Raw.Provider != "codex" {
		t.Fatalf("expected raw provider to round-trip, got %#v", decoded.Raw)
	}
}
