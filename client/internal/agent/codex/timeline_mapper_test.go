package codex

import (
	"encoding/json"
	"testing"

	"code-agent-gateway/common/domain"
)

func TestCodexItemTimelineEventMapsKnownItemTypes(t *testing.T) {
	tests := []struct {
		name     string
		item     map[string]any
		itemType domain.AgentTimelineItemType
		role     domain.AgentTimelineRole
		phase    domain.AgentTimelinePhase
		toolKind domain.AgentTimelineToolKind
	}{
		{
			name:     "user message",
			item:     map[string]any{"id": "user-1", "type": "userMessage", "text": "hi"},
			itemType: domain.AgentTimelineItemMessage,
			role:     domain.AgentTimelineRoleUser,
			phase:    domain.AgentTimelinePhaseInput,
		},
		{
			name:     "hook prompt",
			item:     map[string]any{"id": "hook-1", "type": "hookPrompt", "text": "system"},
			itemType: domain.AgentTimelineItemMessage,
			role:     domain.AgentTimelineRoleSystem,
			phase:    domain.AgentTimelinePhaseSystem,
		},
		{
			name:     "agent message",
			item:     map[string]any{"id": "msg-1", "type": "agentMessage", "phase": "commentary", "text": "working"},
			itemType: domain.AgentTimelineItemMessage,
			role:     domain.AgentTimelineRoleAssistant,
			phase:    domain.AgentTimelinePhaseProgress,
		},
		{
			name:     "plan",
			item:     map[string]any{"id": "plan-1", "type": "plan", "text": "1. test"},
			itemType: domain.AgentTimelineItemPlan,
			role:     domain.AgentTimelineRoleAssistant,
			phase:    domain.AgentTimelinePhaseProgress,
		},
		{
			name:     "reasoning",
			item:     map[string]any{"id": "reasoning-1", "type": "reasoning", "summary": []any{"think"}},
			itemType: domain.AgentTimelineItemReasoning,
			role:     domain.AgentTimelineRoleAssistant,
			phase:    domain.AgentTimelinePhaseAnalysis,
		},
		{
			name:     "command",
			item:     map[string]any{"id": "cmd-1", "type": "commandExecution", "command": "go test ./..."},
			itemType: domain.AgentTimelineItemCommand,
			role:     domain.AgentTimelineRoleTool,
			phase:    domain.AgentTimelinePhaseProgress,
			toolKind: domain.AgentTimelineToolShell,
		},
		{
			name:     "file change",
			item:     map[string]any{"id": "file-1", "type": "fileChange", "changes": []any{map[string]any{"path": "main.go"}}},
			itemType: domain.AgentTimelineItemFileChange,
			role:     domain.AgentTimelineRoleTool,
			phase:    domain.AgentTimelinePhaseProgress,
			toolKind: domain.AgentTimelineToolFileEdit,
		},
		{
			name:     "mcp",
			item:     map[string]any{"id": "mcp-1", "type": "mcpToolCall", "server": "github", "tool": "search"},
			itemType: domain.AgentTimelineItemMCPTool,
			role:     domain.AgentTimelineRoleTool,
			phase:    domain.AgentTimelinePhaseProgress,
			toolKind: domain.AgentTimelineToolMCP,
		},
		{
			name:     "dynamic tool",
			item:     map[string]any{"id": "tool-1", "type": "dynamicToolCall", "tool": "lookup"},
			itemType: domain.AgentTimelineItemTool,
			role:     domain.AgentTimelineRoleTool,
			phase:    domain.AgentTimelinePhaseProgress,
			toolKind: domain.AgentTimelineToolFunction,
		},
		{
			name:     "subagent",
			item:     map[string]any{"id": "sub-1", "type": "collabAgentToolCall", "tool": "worker"},
			itemType: domain.AgentTimelineItemSubagent,
			role:     domain.AgentTimelineRoleAssistant,
			phase:    domain.AgentTimelinePhaseProgress,
			toolKind: domain.AgentTimelineToolSubagent,
		},
		{
			name:     "web search",
			item:     map[string]any{"id": "web-1", "type": "webSearch", "query": "agent"},
			itemType: domain.AgentTimelineItemWebSearch,
			role:     domain.AgentTimelineRoleTool,
			phase:    domain.AgentTimelinePhaseProgress,
			toolKind: domain.AgentTimelineToolWebSearch,
		},
		{
			name:     "image generation",
			item:     map[string]any{"id": "img-1", "type": "imageGeneration", "savedPath": "/tmp/out.png"},
			itemType: domain.AgentTimelineItemImage,
			role:     domain.AgentTimelineRoleAssistant,
			phase:    domain.AgentTimelinePhaseProgress,
			toolKind: domain.AgentTimelineToolImageGeneration,
		},
		{
			name:     "review mode",
			item:     map[string]any{"id": "mode-1", "type": "enteredReviewMode"},
			itemType: domain.AgentTimelineItemModeChange,
			role:     domain.AgentTimelineRoleAssistant,
			phase:    domain.AgentTimelinePhaseSystem,
		},
		{
			name:     "context compaction",
			item:     map[string]any{"id": "ctx-1", "type": "contextCompaction"},
			itemType: domain.AgentTimelineItemContext,
			role:     domain.AgentTimelineRoleAssistant,
			phase:    domain.AgentTimelinePhaseSystem,
		},
		{
			name:     "unknown",
			item:     map[string]any{"id": "unknown-1", "type": "vendorSpecific"},
			itemType: domain.AgentTimelineItemUnknown,
			role:     domain.AgentTimelineRoleAssistant,
			phase:    domain.AgentTimelinePhaseProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := mustJSON(t, map[string]any{
				"threadId": "thread-1",
				"turnId":   "turn-1",
				"item":     tt.item,
			})
			event, ok := codexItemTimelineEvent(params, domain.AgentTimelineEventItemCompleted, domain.AgentTimelineStatusCompleted, 1)
			if !ok {
				t.Fatal("expected item event")
			}
			if event.ItemType != tt.itemType || event.Role != tt.role || event.Phase != tt.phase {
				t.Fatalf("unexpected mapping: %+v", event)
			}
			if tt.toolKind != "" {
				if event.Tool == nil || event.Tool.Kind != tt.toolKind {
					t.Fatalf("expected tool kind %q, got %+v", tt.toolKind, event.Tool)
				}
			}
			if event.Raw == nil || event.Raw.Provider != "codex" {
				t.Fatalf("expected raw codex metadata, got %+v", event.Raw)
			}
		})
	}
}

func TestAppServerClientTimelineEventsStayScopedToEachTurn(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)

	events := make([]domain.AgentTimelineEvent, 0, 4)
	client.SetTimelineEventHandler(func(event domain.AgentTimelineEvent) {
		events = append(events, event)
	})

	runner.emitNotification(t, "turn/started", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-hi",
	})
	runner.emitNotification(t, "item/agentMessage/delta", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-hi",
		"itemId":   "msg-hi",
		"delta":    "hi",
	})
	runner.emitNotification(t, "turn/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-hi",
	})
	runner.emitNotification(t, "turn/started", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-research",
	})
	runner.emitNotification(t, "item/reasoning/summaryTextDelta", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-research",
		"itemId":   "reasoning-research",
		"delta":    "确认范围",
	})
	runner.emitNotification(t, "item/agentMessage/delta", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-research",
		"itemId":   "msg-research",
		"delta":    "总结报告",
	})

	if len(events) != 6 {
		t.Fatalf("unexpected timeline event count: %d", len(events))
	}
	if events[1].TurnID != "turn-hi" || events[1].Content == nil || events[1].Content.Delta != "hi" {
		t.Fatalf("unexpected first turn delta: %+v", events[1])
	}
	if events[4].TurnID != "turn-research" || events[4].ItemType != domain.AgentTimelineItemReasoning || events[4].Phase != domain.AgentTimelinePhaseAnalysis {
		t.Fatalf("unexpected reasoning delta: %+v", events[4])
	}
	if events[5].TurnID != "turn-research" || events[5].ItemType != domain.AgentTimelineItemMessage || events[5].Phase != domain.AgentTimelinePhaseFinal {
		t.Fatalf("unexpected final delta: %+v", events[5])
	}
	if events[1].Sequence != 2 || events[5].Sequence != 3 {
		t.Fatalf("expected per-turn timeline sequences, got first=%d second=%d", events[1].Sequence, events[5].Sequence)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	blob, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return blob
}
