package codex

import (
	"os"
	"path/filepath"
	"testing"

	"code-agent-gateway/common/domain"
)

func TestApplyConfigWritesTomlToCodexConfigPath(t *testing.T) {
	client := NewAppServerClient(&fakeRunner{})
	homeDir := t.TempDir()
	client.homeDir = func() (string, error) {
		return homeDir, nil
	}

	result, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.4\"\napproval_policy = \"never\"\n",
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := filepath.Join(homeDir, ".codex", "config.toml")
	if result.FilePath != expectedPath {
		t.Fatalf("filePath = %q, want %q", result.FilePath, expectedPath)
	}

	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "model = \"gpt-5.4\"\napproval_policy = \"never\"\n" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestApplyConfigMirrorsTomlToConfiguredCodexHomePath(t *testing.T) {
	client := NewAppServerClient(&fakeRunner{})
	homeDir := t.TempDir()
	client.homeDir = func() (string, error) {
		return homeDir, nil
	}
	client.configMirrorPath = filepath.Join(homeDir, "codex-home", "config.toml")

	_, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.4\"\n",
	})
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(client.configMirrorPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "model = \"gpt-5.4\"\n" {
		t.Fatalf("unexpected mirrored content: %q", string(content))
	}
}

func TestApplyConfigRejectsInvalidToml(t *testing.T) {
	client := NewAppServerClient(&fakeRunner{})
	client.homeDir = func() (string, error) {
		return t.TempDir(), nil
	}

	_, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = [",
	})
	if err == nil {
		t.Fatal("expected invalid toml error")
	}
}

func TestApplyConfigRejectsUnsupportedDocuments(t *testing.T) {
	client := NewAppServerClient(&fakeRunner{})
	client.homeDir = func() (string, error) {
		return t.TempDir(), nil
	}

	if _, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: "claude_code",
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"x\"",
	}); err == nil {
		t.Fatal("expected unsupported agent error")
	}

	if _, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    "json",
		Content:   "{}",
	}); err == nil {
		t.Fatal("expected unsupported format error")
	}

	if _, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "   ",
	}); err == nil {
		t.Fatal("expected empty content error")
	}
}

func TestApplyConfigPropagatesHomeDirectoryErrors(t *testing.T) {
	client := NewAppServerClient(&fakeRunner{})
	client.homeDir = func() (string, error) {
		return "", os.ErrNotExist
	}

	if _, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.4\"",
	}); err == nil {
		t.Fatal("expected home dir error")
	}
}

func TestFakeAdapterApplyConfigWritesTomlToCodexConfigPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	adapter := NewFakeAdapter()
	result, err := adapter.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.2\"\n",
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := filepath.Join(homeDir, ".codex", "config.toml")
	if result.FilePath != expectedPath {
		t.Fatalf("filePath = %q, want %q", result.FilePath, expectedPath)
	}
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "model = \"gpt-5.2\"\n" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestResolveUserHomeDirUsesEnvironment(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	got, err := resolveUserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != homeDir {
		t.Fatalf("homeDir = %q, want %q", got, homeDir)
	}
}

func TestApplyConfigFallsBackToProcessHomeDirWhenResolverMissing(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	client := NewAppServerClient(&fakeRunner{})
	client.homeDir = nil

	result, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Content:   "model = \"gpt-5.4\"\n",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.FilePath != filepath.Join(homeDir, ".codex", "config.toml") {
		t.Fatalf("unexpected file path: %+v", result)
	}
}

func TestApplyConfigPropagatesFilesystemErrors(t *testing.T) {
	blockedHome := filepath.Join(t.TempDir(), "blocked-home")
	if err := os.WriteFile(blockedHome, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewAppServerClient(&fakeRunner{})
	client.homeDir = func() (string, error) {
		return blockedHome, nil
	}

	if _, err := client.ApplyConfig(domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Content:   "model = \"gpt-5.4\"\n",
	}); err == nil {
		t.Fatal("expected filesystem error")
	}
}
