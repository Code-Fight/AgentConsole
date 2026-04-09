package config

import "testing"

func TestReadConfigFallsBackAndRejectsInvalidPort(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("SETTINGS_FILE", "")
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
