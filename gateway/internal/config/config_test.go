package config

import (
	"os"
	"path/filepath"
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
	if cfg.Host != "0.0.0.0" || cfg.Port != 8080 || cfg.SettingsFilePath != "data/settings.json" {
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
	t.Setenv("GATEWAY_API_KEY", "env-key")

	cfg, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "env-key" || cfg.SettingsFilePath != "data/custom.json" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
