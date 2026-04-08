package routing

import "testing"

func TestResolveMachineForThread(t *testing.T) {
	router := NewRouter()
	router.TrackThread("thread-1", "machine-01")

	machineID, ok := router.ResolveThread("thread-1")
	if !ok || machineID != "machine-01" {
		t.Fatalf("expected machine-01, got %q", machineID)
	}
}
