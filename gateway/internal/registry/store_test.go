package registry

import (
	"testing"

	"code-agent-gateway/common/domain"
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
