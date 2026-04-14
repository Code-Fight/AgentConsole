package codex

import (
	"os"
	"path/filepath"
	"testing"

	"code-agent-gateway/common/domain"
)

func TestManagedInstanceLayoutKeepsAgentHomesIsolated(t *testing.T) {
	rootDir := t.TempDir()

	first := NewInstanceLayout(rootDir, "agent-01")
	second := NewInstanceLayout(rootDir, "agent-02")

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
	layout := NewInstanceLayout(t.TempDir(), "agent-01")

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
