package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Host             string
	Port             int
	SettingsFilePath string
	APIKey           string
	ConfigFilePath   string
}

type fileConfig struct {
	Host             string `toml:"host"`
	Port             int    `toml:"port"`
	SettingsFilePath string `toml:"settings_file"`
	APIKey           string `toml:"api_key"`
}

func Read() (Config, error) {
	cfg := Config{
		Host:             "0.0.0.0",
		Port:             8080,
		SettingsFilePath: "data/settings.json",
	}

	cfg.ConfigFilePath = strings.TrimSpace(os.Getenv("GATEWAY_CONFIG_FILE"))
	if cfg.ConfigFilePath != "" {
		loaded, err := readFromTOML(cfg.ConfigFilePath)
		if err != nil {
			return Config{}, err
		}
		cfg = loaded
		cfg.ConfigFilePath = strings.TrimSpace(os.Getenv("GATEWAY_CONFIG_FILE"))
	}

	if host := strings.TrimSpace(os.Getenv("HOST")); host != "" {
		cfg.Host = host
	}

	if portRaw := strings.TrimSpace(os.Getenv("PORT")); portRaw != "" {
		port, err := parsePort(portRaw, "PORT")
		if err != nil {
			return Config{}, err
		}
		cfg.Port = port
	}

	if settingsFilePath := strings.TrimSpace(os.Getenv("SETTINGS_FILE")); settingsFilePath != "" {
		cfg.SettingsFilePath = settingsFilePath
	}
	if apiKey := strings.TrimSpace(os.Getenv("GATEWAY_API_KEY")); apiKey != "" {
		cfg.APIKey = apiKey
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("invalid port value: %d", cfg.Port)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return Config{}, fmt.Errorf("GATEWAY_API_KEY is required")
	}

	return cfg, nil
}

func readFromTOML(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	var doc fileConfig
	if err := toml.Unmarshal(content, &doc); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	cfg := Config{
		Host:             "0.0.0.0",
		Port:             8080,
		SettingsFilePath: "data/settings.json",
		APIKey:           strings.TrimSpace(doc.APIKey),
		ConfigFilePath:   path,
	}
	if host := strings.TrimSpace(doc.Host); host != "" {
		cfg.Host = host
	}
	if doc.Port != 0 {
		cfg.Port = doc.Port
	}
	if settingsFilePath := strings.TrimSpace(doc.SettingsFilePath); settingsFilePath != "" {
		cfg.SettingsFilePath = settingsFilePath
	}
	if !filepath.IsAbs(cfg.SettingsFilePath) {
		cfg.SettingsFilePath = filepath.Clean(cfg.SettingsFilePath)
	}
	return cfg, nil
}

func parsePort(raw string, source string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid %s value: %q", source, raw)
	}
	return port, nil
}
