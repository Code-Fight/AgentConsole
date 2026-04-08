package runtimeindex

import (
	"testing"

	"code-agent-gateway/common/domain"
)

func TestStoreKeepsMachineSnapshotsIsolated(t *testing.T) {
	store := NewStore()

	store.ReplaceSnapshot(
		"machine-a",
		[]domain.Thread{
			{ThreadID: "thread-a-1", MachineID: "machine-a"},
		},
		[]domain.EnvironmentResource{
			{ResourceID: "skill-a-1", MachineID: "machine-a", Kind: domain.EnvironmentKindSkill},
		},
	)

	store.ReplaceSnapshot(
		"machine-b",
		[]domain.Thread{
			{ThreadID: "thread-b-1", MachineID: "machine-b"},
		},
		[]domain.EnvironmentResource{
			{ResourceID: "skill-b-1", MachineID: "machine-b", Kind: domain.EnvironmentKindSkill},
		},
	)

	threads := store.Threads()
	if len(threads) != 2 {
		t.Fatalf("expected 2 threads after two machine snapshots, got %d", len(threads))
	}

	threadIDs := map[string]bool{}
	for _, item := range threads {
		threadIDs[item.ThreadID] = true
	}
	if !threadIDs["thread-a-1"] || !threadIDs["thread-b-1"] {
		t.Fatalf("expected threads from both machines, got %+v", threads)
	}

	skills := store.Environment(domain.EnvironmentKindSkill)
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills after two machine snapshots, got %d", len(skills))
	}

	skillIDs := map[string]bool{}
	for _, item := range skills {
		skillIDs[item.ResourceID] = true
	}
	if !skillIDs["skill-a-1"] || !skillIDs["skill-b-1"] {
		t.Fatalf("expected skills from both machines, got %+v", skills)
	}

	store.ReplaceSnapshot(
		"machine-a",
		[]domain.Thread{
			{ThreadID: "thread-a-2", MachineID: "machine-a"},
		},
		[]domain.EnvironmentResource{
			{ResourceID: "skill-a-2", MachineID: "machine-a", Kind: domain.EnvironmentKindSkill},
		},
	)

	threads = store.Threads()
	if len(threads) != 2 {
		t.Fatalf("expected 2 threads after replacing machine-a snapshot, got %d", len(threads))
	}
	threadIDs = map[string]bool{}
	for _, item := range threads {
		threadIDs[item.ThreadID] = true
	}
	if !threadIDs["thread-a-2"] || !threadIDs["thread-b-1"] || threadIDs["thread-a-1"] {
		t.Fatalf("expected machine-a replacement without deleting machine-b, got %+v", threads)
	}

	skills = store.Environment(domain.EnvironmentKindSkill)
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills after replacing machine-a snapshot, got %d", len(skills))
	}
	skillIDs = map[string]bool{}
	for _, item := range skills {
		skillIDs[item.ResourceID] = true
	}
	if !skillIDs["skill-a-2"] || !skillIDs["skill-b-1"] || skillIDs["skill-a-1"] {
		t.Fatalf("expected machine-a replacement without deleting machine-b skills, got %+v", skills)
	}
}

func TestStoreReplaceSnapshotClearsMachineOnEmptySnapshot(t *testing.T) {
	store := NewStore()

	store.ReplaceSnapshot(
		"machine-a",
		[]domain.Thread{
			{ThreadID: "thread-a-1", MachineID: "machine-a"},
		},
		[]domain.EnvironmentResource{
			{ResourceID: "skill-a-1", MachineID: "machine-a", Kind: domain.EnvironmentKindSkill},
		},
	)

	if got := len(store.Threads()); got != 1 {
		t.Fatalf("expected 1 thread before clearing snapshot, got %d", got)
	}
	if got := len(store.Environment(domain.EnvironmentKindSkill)); got != 1 {
		t.Fatalf("expected 1 skill before clearing snapshot, got %d", got)
	}

	store.ReplaceSnapshot("machine-a", []domain.Thread{}, []domain.EnvironmentResource{})

	if got := len(store.Threads()); got != 0 {
		t.Fatalf("expected cleared threads after empty snapshot, got %d", got)
	}
	if got := len(store.Environment(domain.EnvironmentKindSkill)); got != 0 {
		t.Fatalf("expected cleared skills after empty snapshot, got %d", got)
	}
}

func TestStoreClearMachineRemovesOnlyTargetMachineData(t *testing.T) {
	store := NewStore()

	store.ReplaceSnapshot(
		"machine-a",
		[]domain.Thread{
			{ThreadID: "thread-a-1", MachineID: "machine-a"},
		},
		[]domain.EnvironmentResource{
			{ResourceID: "skill-a-1", MachineID: "machine-a", Kind: domain.EnvironmentKindSkill},
		},
	)
	store.ReplaceSnapshot(
		"machine-b",
		[]domain.Thread{
			{ThreadID: "thread-b-1", MachineID: "machine-b"},
		},
		[]domain.EnvironmentResource{
			{ResourceID: "skill-b-1", MachineID: "machine-b", Kind: domain.EnvironmentKindSkill},
		},
	)

	store.ClearMachine("machine-a")

	threads := store.Threads()
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread after clearing machine-a, got %d", len(threads))
	}
	if threads[0].ThreadID != "thread-b-1" {
		t.Fatalf("expected thread-b-1 to remain, got %+v", threads)
	}

	skills := store.Environment(domain.EnvironmentKindSkill)
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill after clearing machine-a, got %d", len(skills))
	}
	if skills[0].ResourceID != "skill-b-1" {
		t.Fatalf("expected skill-b-1 to remain, got %+v", skills)
	}
}
