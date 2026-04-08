package runtimeindex

import (
	"testing"

	"code-agent-gateway/common/domain"
)

func TestStoreKeepsMachineSnapshotsIsolated(t *testing.T) {
	store := NewStore()

	store.ReplaceSnapshot(
		[]domain.Thread{
			{ThreadID: "thread-a-1", MachineID: "machine-a"},
		},
		[]domain.EnvironmentResource{
			{ResourceID: "skill-a-1", MachineID: "machine-a", Kind: domain.EnvironmentKindSkill},
		},
	)

	store.ReplaceSnapshot(
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
