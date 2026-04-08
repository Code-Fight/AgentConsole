package config

import "testing"

func TestReadDefaults(t *testing.T) {
	t.Setenv("MACHINE_ID", "")
	t.Setenv("GATEWAY_URL", "")

	cfg := Read()

	if cfg.MachineID != "machine-01" {
		t.Fatalf("expected default machine id, got %q", cfg.MachineID)
	}
	if cfg.GatewayURL != "ws://localhost:8080/ws/client" {
		t.Fatalf("expected default gateway url, got %q", cfg.GatewayURL)
	}
}

func TestReadFromEnv(t *testing.T) {
	t.Setenv("MACHINE_ID", "machine-99")
	t.Setenv("GATEWAY_URL", "ws://example.test/ws/client")

	cfg := Read()

	if cfg.MachineID != "machine-99" {
		t.Fatalf("expected env machine id, got %q", cfg.MachineID)
	}
	if cfg.GatewayURL != "ws://example.test/ws/client" {
		t.Fatalf("expected env gateway url, got %q", cfg.GatewayURL)
	}
}
