package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadConfigFallsBackAndRejectsInvalidPort(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("SETTINGS_FILE", "")
	t.Setenv("GATEWAY_CONFIG_FILE", "")
	t.Setenv("GATEWAY_API_KEY", "test-key")
	cfg, err := Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 8080 || cfg.SettingsFilePath != "data/settings.json" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	t.Setenv("PORT", "abc")
	if _, err := Read(); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestReadConfigLoadsTOMLAndEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.toml")
	if err := os.WriteFile(path, []byte("host = \"127.0.0.1\"\nport = 19090\nsettings_file = \"data/custom.json\"\napi_key = \"file-key\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GATEWAY_CONFIG_FILE", path)
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("PORT", "8088")
	t.Setenv("SETTINGS_FILE", "data/env.json")
	t.Setenv("GATEWAY_API_KEY", "env-key")

	cfg, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "0.0.0.0" ||
		cfg.Port != 8088 ||
		cfg.SettingsFilePath != "data/env.json" ||
		cfg.APIKey != "env-key" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestReadConfigResolvesRelativeSettingsFileFromConfigDirectory(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "gateway.toml")
	if err := os.WriteFile(path, []byte("settings_file = \"data/custom.json\"\napi_key = \"file-key\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("SETTINGS_FILE", "")
	t.Setenv("GATEWAY_CONFIG_FILE", path)
	t.Setenv("GATEWAY_API_KEY", "")

	cfg, err := Read()
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(configDir, "data/custom.json")
	if cfg.SettingsFilePath != expected {
		t.Fatalf("settings path = %q, expected %q", cfg.SettingsFilePath, expected)
	}
}

func TestReadConfigRequiresAPIKey(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("SETTINGS_FILE", "")
	t.Setenv("GATEWAY_CONFIG_FILE", "")
	t.Setenv("GATEWAY_API_KEY", "")

	_, err := Read()
	if err == nil {
		t.Fatal("expected missing api key error")
	}
	if !strings.Contains(err.Error(), "GATEWAY_API_KEY is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
