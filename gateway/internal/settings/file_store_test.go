package settings

import (
	"os"
	"path/filepath"
	"testing"

	"code-agent-gateway/common/domain"
)

func TestMemoryStorePersistsGlobalAndMachineDocuments(t *testing.T) {
	agents := []domain.AgentDescriptor{{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"}}
	store := NewMemoryStore(agents)

	if got := store.ListAgentTypes(); len(got) != 1 || got[0].AgentType != domain.AgentTypeCodex {
		t.Fatalf("unexpected agent list: %+v", got)
	}

	global := domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.4\"",
	}
	if err := store.PutGlobal(domain.AgentTypeCodex, global); err != nil {
		t.Fatal(err)
	}

	machine := domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.2\"",
	}
	if err := store.PutMachine("machine-01", domain.AgentTypeCodex, machine); err != nil {
		t.Fatal(err)
	}

	gotGlobal, ok, err := store.GetGlobal(domain.AgentTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || gotGlobal.Content != global.Content || gotGlobal.Version != 1 {
		t.Fatalf("unexpected global document: %+v ok=%v", gotGlobal, ok)
	}

	if err := store.PutGlobal(domain.AgentTypeCodex, domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.2\"",
		UpdatedBy: "tester",
	}); err != nil {
		t.Fatal(err)
	}
	gotGlobal, ok, err = store.GetGlobal(domain.AgentTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || gotGlobal.Version != 2 || gotGlobal.UpdatedBy != "tester" {
		t.Fatalf("unexpected updated global document: %+v ok=%v", gotGlobal, ok)
	}

	gotMachine, ok, err := store.GetMachine("machine-01", domain.AgentTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || gotMachine.Content != machine.Content || gotMachine.Version != 1 {
		t.Fatalf("unexpected machine document: %+v ok=%v", gotMachine, ok)
	}

	if err := store.DeleteMachine("machine-01", domain.AgentTypeCodex); err != nil {
		t.Fatal(err)
	}
	_, ok, err = store.GetMachine("machine-01", domain.AgentTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected machine override to be deleted")
	}

	if err := store.DeleteMachine("machine-01", domain.AgentTypeCodex); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryStoreRejectsMissingIdentifiers(t *testing.T) {
	store := NewMemoryStore(nil)

	if err := store.PutGlobal("", domain.AgentConfigDocument{}); err == nil {
		t.Fatal("expected missing agentType error for global document")
	}
	if err := store.PutMachine("", domain.AgentTypeCodex, domain.AgentConfigDocument{}); err == nil {
		t.Fatal("expected missing machineID error")
	}
	if err := store.PutMachine("machine-01", "", domain.AgentConfigDocument{}); err == nil {
		t.Fatal("expected missing agentType error")
	}
	if err := store.DeleteMachine("", domain.AgentTypeCodex); err == nil {
		t.Fatal("expected missing machineID error for delete")
	}
	if err := store.DeleteMachine("machine-01", ""); err == nil {
		t.Fatal("expected missing agentType error for delete")
	}
}

func TestMemoryStorePersistsConsolePreferences(t *testing.T) {
	store := NewMemoryStore(nil)

	_, ok, err := store.GetConsolePreferences()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected console preferences to be empty")
	}

	preferences := domain.ConsolePreferences{
		ConsoleURL:   "http://localhost:3100",
		APIKey:       "test-key",
		Profile:      "dev",
		SafetyPolicy: "strict",
		LastThreadID: "thread-01",
	}
	if err := store.PutConsolePreferences(preferences); err != nil {
		t.Fatal(err)
	}

	got, ok, err := store.GetConsolePreferences()
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.ConsoleURL != preferences.ConsoleURL || got.LastThreadID != preferences.LastThreadID {
		t.Fatalf("unexpected console preferences: %+v ok=%v", got, ok)
	}
}

func TestFileStoreReloadsPersistedDocuments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	store, err := NewFileStore(path, []domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.PutGlobal(domain.AgentTypeCodex, domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "approval_policy = \"never\"",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutMachine("machine-01", domain.AgentTypeCodex, domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.2\"",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutConsolePreferences(domain.ConsolePreferences{
		ConsoleURL:   "http://localhost:3100",
		APIKey:       "test-key",
		Profile:      "dev",
		SafetyPolicy: "strict",
		LastThreadID: "thread-01",
	}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewFileStore(path, []domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	if err != nil {
		t.Fatal(err)
	}

	global, ok, err := reloaded.GetGlobal(domain.AgentTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || global.Content != "approval_policy = \"never\"" {
		t.Fatalf("unexpected reloaded global: %+v ok=%v", global, ok)
	}

	machine, ok, err := reloaded.GetMachine("machine-01", domain.AgentTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || machine.Content != "model = \"gpt-5.2\"" {
		t.Fatalf("unexpected reloaded machine override: %+v ok=%v", machine, ok)
	}

	if err := reloaded.DeleteMachine("machine-01", domain.AgentTypeCodex); err != nil {
		t.Fatal(err)
	}

	reloaded, err = NewFileStore(path, []domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, ok, err = reloaded.GetMachine("machine-01", domain.AgentTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected deleted machine override to stay deleted after reload")
	}

	consolePrefs, ok, err := reloaded.GetConsolePreferences()
	if err != nil {
		t.Fatal(err)
	}
	if !ok || consolePrefs.ConsoleURL != "http://localhost:3100" || consolePrefs.LastThreadID != "thread-01" {
		t.Fatalf("unexpected console preferences after reload: %+v ok=%v", consolePrefs, ok)
	}
}

func TestFileStoreRejectsInvalidConfiguration(t *testing.T) {
	dir := t.TempDir()

	if _, err := NewFileStore("", nil); err == nil {
		t.Fatal("expected missing path error")
	}

	badJSONPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badJSONPath, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileStore(badJSONPath, nil); err == nil {
		t.Fatal("expected invalid persisted json error")
	}
}

func TestFileStorePropagatesPersistErrors(t *testing.T) {
	dir := t.TempDir()
	parentAsFile := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(parentAsFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewFileStore(filepath.Join(dir, "settings.json"), []domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	if err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(parentAsFile, "settings.json")

	if err := store.PutGlobal(domain.AgentTypeCodex, domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.4\"",
	}); err == nil {
		t.Fatal("expected persist error")
	}
}
