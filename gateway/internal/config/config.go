package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Host string
	Port int
}

func Read() (Config, error) {
	host := strings.TrimSpace(os.Getenv("HOST"))
	if host == "" {
		host = "0.0.0.0"
	}

	portRaw := strings.TrimSpace(os.Getenv("PORT"))
	if portRaw == "" {
		return Config{Host: host, Port: 8080}, nil
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("invalid PORT value: %q", portRaw)
	}

	return Config{Host: host, Port: port}, nil
}
