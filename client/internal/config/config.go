package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	RuntimeModeAppServer = "appserver"
	RuntimeModeFake      = "fake"
)

type Config struct {
	MachineID        string
	MachineName      string
	GatewayURL       string
	RuntimeMode      string
	CodexBin         string
	ManagedAgentsDir string
}

func Read() (Config, error) {
	machineID, err := loadOrCreateMachineID()
	if err != nil {
		return Config{}, fmt.Errorf("load machine identity: %w", err)
	}
	machineName := resolveMachineName(machineID)

	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "ws://localhost:8080/ws/client"
	}

	runtimeMode := os.Getenv("CODEX_RUNTIME_MODE")
	switch runtimeMode {
	case "", RuntimeModeAppServer:
		runtimeMode = RuntimeModeAppServer
	case RuntimeModeFake:
	default:
		runtimeMode = RuntimeModeAppServer
	}

	codexBin := os.Getenv("CODEX_BIN")
	if codexBin == "" {
		codexBin = "codex"
	}

	managedAgentsDir := os.Getenv("MANAGED_AGENTS_DIR")
	if managedAgentsDir == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil && homeDir != "" {
			managedAgentsDir = filepath.Join(homeDir, ".code-agent-gateway", "machines", machineID, "agents")
		} else {
			managedAgentsDir = filepath.Join(".", ".code-agent-gateway", "machines", machineID, "agents")
		}
	}

	return Config{
		MachineID:        machineID,
		MachineName:      machineName,
		GatewayURL:       gatewayURL,
		RuntimeMode:      runtimeMode,
		CodexBin:         codexBin,
		ManagedAgentsDir: managedAgentsDir,
	}, nil
}
