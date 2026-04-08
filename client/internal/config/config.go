package config

import "os"

type Config struct {
	MachineID  string
	GatewayURL string
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

	return Config{
		MachineID:  machineID,
		GatewayURL: gatewayURL,
	}
}
