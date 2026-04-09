package registry

import (
	"testing"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
)

func TestStoreUpsertAndMarkOffline(t *testing.T) {
	store := NewStore()

	store.Upsert(domain.Machine{
		ID:     "machine-01",
		Name:   "Dev Mac",
		Status: domain.MachineStatusOnline,
	})

	items := store.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 machine after upsert, got %d", len(items))
	}
	if items[0].Status != domain.MachineStatusOnline {
		t.Fatalf("expected online status after upsert, got %q", items[0].Status)
	}

	store.MarkOffline("machine-01")

	items = store.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 machine after mark offline, got %d", len(items))
	}
	if items[0].Status != domain.MachineStatusOffline {
		t.Fatalf("expected offline status after mark offline, got %q", items[0].Status)
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
