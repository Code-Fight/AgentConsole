package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDefaults(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("MACHINE_ID", "")
	t.Setenv("MACHINE_NAME", "")
	t.Setenv("GATEWAY_URL", "")
	t.Setenv("CODEX_RUNTIME_MODE", "")
	t.Setenv("CODEX_BIN", "")

	cfg, err := Read()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.MachineID == "" {
		t.Fatal("expected generated machine id")
	}
	if cfg.MachineID == "machine-01" {
		t.Fatalf("expected generated machine id instead of legacy default, got %q", cfg.MachineID)
	}
	if !strings.Contains(cfg.MachineID, "_") {
		t.Fatalf("expected generated machine id to include hostname prefix, got %q", cfg.MachineID)
	}
	if cfg.MachineName == "" {
		t.Fatal("expected default machine name")
	}
	if cfg.MachineName != hostNameOrFallback(t) {
		t.Fatalf("expected hostname-based machine name, got %q", cfg.MachineName)
	}
	wantIdentityPath := filepath.Join(homeDir, ".code-agent-gateway", "machine.json")
	if _, err := os.Stat(wantIdentityPath); err != nil {
		t.Fatalf("expected persisted identity file at %q: %v", wantIdentityPath, err)
	}
	if !strings.Contains(cfg.ManagedAgentsDir, cfg.MachineID) {
		t.Fatalf("expected managed agents dir %q to include generated machine id %q", cfg.ManagedAgentsDir, cfg.MachineID)
	}
	if cfg.GatewayURL != "ws://localhost:8080/ws/client" {
		t.Fatalf("expected default gateway url, got %q", cfg.GatewayURL)
	}
	if cfg.RuntimeMode != RuntimeModeAppServer {
		t.Fatalf("expected default runtime mode %q, got %q", RuntimeModeAppServer, cfg.RuntimeMode)
	}
	if cfg.CodexBin != "codex" {
		t.Fatalf("expected default codex bin %q, got %q", "codex", cfg.CodexBin)
	}
}

func TestReadReusesPersistedMachineID(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("MACHINE_NAME", "")
	t.Setenv("GATEWAY_URL", "")
	t.Setenv("CODEX_RUNTIME_MODE", "")
	t.Setenv("CODEX_BIN", "")

	first, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Read()
	if err != nil {
		t.Fatal(err)
	}

	if first.MachineID != second.MachineID {
		t.Fatalf("expected persisted machine id to be stable, got %q then %q", first.MachineID, second.MachineID)
	}
}

func TestReadUsesMachineNameEnvOverride(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MACHINE_NAME", "Console Friendly")
	t.Setenv("MACHINE_ID", "legacy-machine-id")
	t.Setenv("GATEWAY_URL", "ws://example.test/ws/client")
	t.Setenv("CODEX_RUNTIME_MODE", RuntimeModeFake)
	t.Setenv("CODEX_BIN", "/opt/homebrew/bin/codex")

	cfg, err := Read()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.MachineID == "legacy-machine-id" {
		t.Fatalf("expected MACHINE_ID env to be ignored, got %q", cfg.MachineID)
	}
	if cfg.MachineName != "Console Friendly" {
		t.Fatalf("expected machine name override, got %q", cfg.MachineName)
	}
	if cfg.GatewayURL != "ws://example.test/ws/client" {
		t.Fatalf("expected env gateway url, got %q", cfg.GatewayURL)
	}
	if cfg.RuntimeMode != RuntimeModeFake {
		t.Fatalf("expected env runtime mode %q, got %q", RuntimeModeFake, cfg.RuntimeMode)
	}
	if cfg.CodexBin != "/opt/homebrew/bin/codex" {
		t.Fatalf("expected env codex bin, got %q", cfg.CodexBin)
	}
}

func hostNameOrFallback(t *testing.T) string {
	t.Helper()

	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "machine"
	}
	return name
}
