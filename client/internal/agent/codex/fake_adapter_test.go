package codex

import (
	"testing"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

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

func TestFakeAdapterCreateThreadAndStartTurn(t *testing.T) {
	adapter := NewFakeAdapter()

	thread, err := adapter.CreateThread(agenttypes.CreateThreadParams{Title: "Investigate flaky test"})
	if err != nil {
		t.Fatal(err)
	}

	if thread.ThreadID != "thread-01" || thread.Status != domain.ThreadStatusIdle {
		t.Fatalf("unexpected thread: %+v", thread)
	}

	threads, err := adapter.ListThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || threads[0].ThreadID != "thread-01" {
		t.Fatalf("unexpected threads: %+v", threads)
	}

	result, err := adapter.StartTurn(agenttypes.StartTurnParams{
		ThreadID: thread.ThreadID,
		Input:    "run tests",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.TurnID != "turn-01" || result.ThreadID != "thread-01" {
		t.Fatalf("unexpected turn result: %+v", result)
	}

	if len(result.Deltas) != 2 {
		t.Fatalf("expected 2 deltas, got %d", len(result.Deltas))
	}

	expected := []agenttypes.TurnDelta{
		{Sequence: 1, Delta: "assistant: thinking"},
		{Sequence: 2, Delta: "assistant: done"},
	}
	for idx, delta := range expected {
		if result.Deltas[idx] != delta {
			t.Fatalf("delta %d = %+v, want %+v", idx, result.Deltas[idx], delta)
		}
	}
}

func TestFakeAdapterReadArchiveAndResumeThread(t *testing.T) {
	adapter := NewFakeAdapter()

	thread, err := adapter.CreateThread(agenttypes.CreateThreadParams{Title: "Investigate flaky test"})
	if err != nil {
		t.Fatal(err)
	}

	readThread, err := adapter.ReadThread(thread.ThreadID)
	if err != nil {
		t.Fatal(err)
	}
	if readThread != thread {
		t.Fatalf("unexpected read thread: %+v", readThread)
	}

	if err := adapter.ArchiveThread(thread.ThreadID); err != nil {
		t.Fatal(err)
	}

	archivedThread, err := adapter.ReadThread(thread.ThreadID)
	if err != nil {
		t.Fatal(err)
	}
	if archivedThread.Status != domain.ThreadStatusNotLoaded {
		t.Fatalf("unexpected archived thread: %+v", archivedThread)
	}

	resumedThread, err := adapter.ResumeThread(thread.ThreadID)
	if err != nil {
		t.Fatal(err)
	}
	if resumedThread.Status != domain.ThreadStatusIdle {
		t.Fatalf("unexpected resumed thread: %+v", resumedThread)
	}
}

func TestFakeAdapterSteerAndInterruptTurn(t *testing.T) {
	adapter := NewFakeAdapter()

	thread, err := adapter.CreateThread(agenttypes.CreateThreadParams{Title: "Investigate flaky test"})
	if err != nil {
		t.Fatal(err)
	}

	startResult, err := adapter.StartTurn(agenttypes.StartTurnParams{
		ThreadID: thread.ThreadID,
		Input:    "run tests",
	})
	if err != nil {
		t.Fatal(err)
	}

	steerResult, err := adapter.SteerTurn(agenttypes.SteerTurnParams{
		ThreadID: thread.ThreadID,
		TurnID:   startResult.TurnID,
		Input:    "try a smaller patch",
	})
	if err != nil {
		t.Fatal(err)
	}
	if steerResult.ThreadID != thread.ThreadID || steerResult.TurnID != startResult.TurnID {
		t.Fatalf("unexpected steer result: %+v", steerResult)
	}
	if len(steerResult.Deltas) == 0 {
		t.Fatalf("expected steer deltas, got %+v", steerResult)
	}

	turn, err := adapter.InterruptTurn(agenttypes.InterruptTurnParams{
		ThreadID: thread.ThreadID,
		TurnID:   startResult.TurnID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if turn.ThreadID != thread.ThreadID || turn.TurnID != startResult.TurnID || turn.Status != domain.TurnStatusInterrupted {
		t.Fatalf("unexpected interrupted turn: %+v", turn)
	}
}
