package config

import "os"

const (
	RuntimeModeAppServer = "appserver"
	RuntimeModeFake      = "fake"
)

type Config struct {
	MachineID   string
	GatewayURL  string
	RuntimeMode string
	CodexBin    string
}

func Read() Config {
	machineID := os.Getenv("MACHINE_ID")
	if machineID == "" {
		machineID = "machine-01"
	}

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

	return Config{
		MachineID:   machineID,
		GatewayURL:  gatewayURL,
		RuntimeMode: runtimeMode,
		CodexBin:    codexBin,
	}
}
