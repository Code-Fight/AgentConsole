package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Host             string
	Port             int
	SettingsFilePath string
}

func Read() (Config, error) {
	host := strings.TrimSpace(os.Getenv("HOST"))
	if host == "" {
		host = "0.0.0.0"
	}

	portRaw := strings.TrimSpace(os.Getenv("PORT"))
	settingsFilePath := strings.TrimSpace(os.Getenv("SETTINGS_FILE"))
	if settingsFilePath == "" {
		settingsFilePath = "data/settings.json"
	}
	if portRaw == "" {
		return Config{Host: host, Port: 8080, SettingsFilePath: settingsFilePath}, nil
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("invalid PORT value: %q", portRaw)
	}

	return Config{Host: host, Port: port, SettingsFilePath: settingsFilePath}, nil
}
