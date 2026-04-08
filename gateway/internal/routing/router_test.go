package routing

import (
	"testing"

	"code-agent-gateway/common/domain"
)

func TestResolveMachineForThread(t *testing.T) {
	router := NewRouter()
	router.TrackThread("thread-1", "machine-01")

	machineID, ok := router.ResolveThread("thread-1")
	if !ok || machineID != "machine-01" {
		t.Fatalf("expected machine-01, got %q", machineID)
	}
}

func TestReplaceSnapshotRebuildsThreadOwnershipForMachine(t *testing.T) {
	router := NewRouter()
	router.TrackThread("thread-1", "machine-01")
	router.TrackThread("thread-2", "machine-02")

	router.ReplaceSnapshot("machine-01", []domain.Thread{
		{ThreadID: "thread-3", MachineID: "machine-01"},
	})

	if _, ok := router.ResolveThread("thread-1"); ok {
		t.Fatal("expected thread-1 route to be removed")
	}

	machineID, ok := router.ResolveThread("thread-3")
	if !ok || machineID != "machine-01" {
		t.Fatalf("expected thread-3 to resolve to machine-01, got %q", machineID)
	}

	machineID, ok = router.ResolveThread("thread-2")
	if !ok || machineID != "machine-02" {
		t.Fatalf("expected thread-2 to remain on machine-02, got %q", machineID)
	}
}
