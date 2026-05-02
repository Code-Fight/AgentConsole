package registry

import (
	"testing"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
)

func TestStoreUpsertAndMarkOffline(t *testing.T) {
	store := NewStore()

	store.Upsert(domain.Machine{
		ID:            "machine-01",
		Name:          "Dev Mac",
		Status:        domain.MachineStatusOnline,
		RuntimeStatus: domain.MachineRuntimeStatusRunning,
	})

	items := store.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 machine after upsert, got %d", len(items))
	}
	if items[0].Status != domain.MachineStatusOnline {
		t.Fatalf("expected online status after upsert, got %q", items[0].Status)
	}
	if items[0].RuntimeStatus != domain.MachineRuntimeStatusRunning {
		t.Fatalf("expected running runtime status after upsert, got %q", items[0].RuntimeStatus)
	}

	store.Upsert(domain.Machine{
		ID:            "machine-01",
		Name:          "Dev Mac",
		Status:        domain.MachineStatusOnline,
		RuntimeStatus: domain.MachineRuntimeStatusUnknown,
	})

	items = store.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 machine after second upsert, got %d", len(items))
	}
	if items[0].RuntimeStatus != domain.MachineRuntimeStatusRunning {
		t.Fatalf("expected runtime status to preserve the prior value, got %q", items[0].RuntimeStatus)
	}

	store.MarkOffline("machine-01")

	items = store.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 machine after mark offline, got %d", len(items))
	}
	if items[0].Status != domain.MachineStatusOffline {
		t.Fatalf("expected offline status after mark offline, got %q", items[0].Status)
	}
	if items[0].RuntimeStatus != domain.MachineRuntimeStatusUnknown {
		t.Fatalf("expected runtime status to become unknown after mark offline, got %q", items[0].RuntimeStatus)
	}
}

func TestStoreGetAndPendingApprovals(t *testing.T) {
	store := NewStore()
	store.Upsert(domain.Machine{
		ID:     "machine-01",
		Name:   "Dev Mac",
		Status: domain.MachineStatusOnline,
	})

	machine, ok := store.Get("machine-01")
	if !ok {
		t.Fatal("expected machine lookup to succeed")
	}
	if machine.Name != "Dev Mac" {
		t.Fatalf("unexpected machine: %+v", machine)
	}

	store.UpsertPendingApproval("machine-01", protocol.ApprovalRequiredPayload{
		RequestID: "approval-2",
		ThreadID:  "thread-01",
		Kind:      "command",
		Command:   "go test ./...",
	})
	store.UpsertPendingApproval("machine-01", protocol.ApprovalRequiredPayload{
		RequestID: "approval-1",
		ThreadID:  "thread-01",
		Kind:      "permissions",
		Reason:    "Need filesystem access",
	})
	store.UpsertPendingApproval("machine-01", protocol.ApprovalRequiredPayload{
		RequestID: "approval-3",
		ThreadID:  "thread-02",
		Kind:      "command",
	})

	approvals := store.PendingApprovalsForThread("thread-01")
	if len(approvals) != 2 {
		t.Fatalf("expected 2 approvals, got %+v", approvals)
	}
	if approvals[0].RequestID != "approval-1" || approvals[1].RequestID != "approval-2" {
		t.Fatalf("expected approvals to be sorted by request id, got %+v", approvals)
	}

	store.RemovePendingApproval("approval-1")
	approvals = store.PendingApprovalsForThread("thread-01")
	if len(approvals) != 1 || approvals[0].RequestID != "approval-2" {
		t.Fatalf("unexpected approvals after removal: %+v", approvals)
	}

	stored, ok := store.PendingApproval("approval-2")
	if !ok {
		t.Fatal("expected pending approval lookup to succeed")
	}
	if stored.ThreadID != "thread-01" || stored.Command != "go test ./..." {
		t.Fatalf("unexpected stored approval payload: %+v", stored)
	}
}

func TestStoreKeepsTimelineEventsPerThreadWithLimit(t *testing.T) {
	store := NewStore()
	store.timelineLimit = 2

	store.AppendTimelineEvent(domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       "event-1",
		Sequence:      1,
		ThreadID:      "thread-01",
		TurnID:        "turn-01",
		EventType:     domain.AgentTimelineEventTurnStarted,
	})
	store.AppendTimelineEvent(domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       "event-other",
		Sequence:      1,
		ThreadID:      "thread-02",
		TurnID:        "turn-02",
		EventType:     domain.AgentTimelineEventTurnStarted,
	})
	store.AppendTimelineEvent(domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       "event-2",
		Sequence:      2,
		ThreadID:      "thread-01",
		TurnID:        "turn-01",
		EventType:     domain.AgentTimelineEventItemDelta,
	})
	store.AppendTimelineEvent(domain.AgentTimelineEvent{
		SchemaVersion: domain.AgentTimelineSchemaVersion,
		EventID:       "event-3",
		Sequence:      3,
		ThreadID:      "thread-01",
		TurnID:        "turn-01",
		EventType:     domain.AgentTimelineEventTurnCompleted,
	})

	events := store.TimelineEventsForThread("thread-01")
	if len(events) != 2 {
		t.Fatalf("expected latest 2 events, got %+v", events)
	}
	if events[0].EventID != "event-2" || events[1].EventID != "event-3" {
		t.Fatalf("unexpected timeline events: %+v", events)
	}

	events[0].EventID = "mutated"
	events = store.TimelineEventsForThread("thread-01")
	if events[0].EventID != "event-2" {
		t.Fatalf("timeline events should be copied on read, got %+v", events)
	}
}
