package codex

import (
	"os"
	"path/filepath"
	"testing"

	"code-agent-gateway/common/domain"
)

func TestManagedInstanceLayoutKeepsAgentHomesIsolated(t *testing.T) {
	rootDir := t.TempDir()

	first, err := NewInstanceLayout(rootDir, "agent-01")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewInstanceLayout(rootDir, "agent-02")
	if err != nil {
		t.Fatal(err)
	}

	if first.AgentID != "agent-01" {
		t.Fatalf("unexpected first agent id: %+v", first)
	}
	if second.AgentID != "agent-02" {
		t.Fatalf("unexpected second agent id: %+v", second)
	}
	if first.HomeDir == second.HomeDir {
		t.Fatalf("expected isolated home dirs, got %q", first.HomeDir)
	}
	if first.CodexHomeDir == second.CodexHomeDir {
		t.Fatalf("expected isolated codex home dirs, got %q", first.CodexHomeDir)
	}
	if first.ConfigPath != filepath.Join(first.HomeDir, ".codex", "config.toml") {
		t.Fatalf("unexpected config path: %q", first.ConfigPath)
	}
}

func TestManagedInstanceLayoutAppliesConfigIntoAgentHome(t *testing.T) {
	layout, err := NewInstanceLayout(t.TempDir(), "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	result, err := layout.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.4\"\n",
	})
	if err != nil {
		t.Fatalf("ApplyConfig returned error: %v", err)
	}
	if result.FilePath != layout.ConfigPath {
		t.Fatalf("unexpected config path: %q", result.FilePath)
	}

	data, err := os.ReadFile(layout.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "model = \"gpt-5.4\"\n" {
		t.Fatalf("unexpected config contents: %q", string(data))
	}
}

func TestManagedInstanceLayoutRejectsUnsafeAgentID(t *testing.T) {
	if _, err := NewInstanceLayout(t.TempDir(), "../escape"); err == nil {
		t.Fatal("expected unsafe agentID to be rejected")
	}
}

func TestManagedInstanceLayoutSeedsSharedCodexAuthIntoIsolatedCodexHome(t *testing.T) {
	rootDir := t.TempDir()
	layout, err := NewInstanceLayout(rootDir, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	sharedHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sharedHome, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(sharedHome, ".codex", "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"token":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	originalHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", sharedHome); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if originalHome == "" {
			_ = os.Unsetenv("HOME")
			return
		}
		_ = os.Setenv("HOME", originalHome)
	})

	if err := os.MkdirAll(layout.CodexHomeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := seedSharedCodexFiles(layout); err != nil {
		t.Fatalf("seedSharedCodexFiles returned error: %v", err)
	}

	seededAuthPath := filepath.Join(layout.CodexHomeDir, "auth.json")
	data, err := os.ReadFile(seededAuthPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != `{"token":"secret"}` {
		t.Fatalf("unexpected auth contents: %q", string(data))
	}
}

func TestManagedInstanceLayoutSeedsSharedCodexConfigIntoIsolatedAgentHome(t *testing.T) {
	rootDir := t.TempDir()
	layout, err := NewInstanceLayout(rootDir, "agent-01")
	if err != nil {
		t.Fatal(err)
	}

	sharedHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sharedHome, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(sharedHome, ".codex", "config.toml")
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5.4\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	originalHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", sharedHome); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if originalHome == "" {
			_ = os.Unsetenv("HOME")
			return
		}
		_ = os.Setenv("HOME", originalHome)
	})

	if err := os.MkdirAll(filepath.Dir(layout.ConfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := seedSharedCodexFiles(layout); err != nil {
		t.Fatalf("seedSharedCodexFiles returned error: %v", err)
	}

	data, err := os.ReadFile(layout.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "model = \"gpt-5.4\"\n" {
		t.Fatalf("unexpected config contents: %q", string(data))
	}
}
