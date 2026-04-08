package config

import "testing"

func TestReadDefaults(t *testing.T) {
	t.Setenv("MACHINE_ID", "")
	t.Setenv("GATEWAY_URL", "")
	t.Setenv("CODEX_RUNTIME_MODE", "")
	t.Setenv("CODEX_BIN", "")

	cfg := Read()

	if cfg.MachineID != "machine-01" {
		t.Fatalf("expected default machine id, got %q", cfg.MachineID)
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

func TestReadFromEnv(t *testing.T) {
	t.Setenv("MACHINE_ID", "machine-99")
	t.Setenv("GATEWAY_URL", "ws://example.test/ws/client")
	t.Setenv("CODEX_RUNTIME_MODE", RuntimeModeFake)
	t.Setenv("CODEX_BIN", "/opt/homebrew/bin/codex")

	cfg := Read()

	if cfg.MachineID != "machine-99" {
		t.Fatalf("expected env machine id, got %q", cfg.MachineID)
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
