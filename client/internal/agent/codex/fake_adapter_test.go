package codex

import "testing"

func TestFakeAdapterSnapshot(t *testing.T) {
	adapter := NewFakeAdapter()

	threads, err := adapter.ListThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 0 {
		t.Fatalf("expected empty threads, got %d", len(threads))
	}

	environment, err := adapter.ListEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if len(environment) != 0 {
		t.Fatalf("expected empty environment, got %d", len(environment))
	}
}
