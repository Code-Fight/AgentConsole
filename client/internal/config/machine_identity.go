package config

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type machineIdentityDocument struct {
	MachineID string `json:"machineId"`
	Hostname  string `json:"hostname,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func loadOrCreateMachineID() (string, error) {
	path, err := machineIdentityPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err == nil {
		var document machineIdentityDocument
		if err := json.Unmarshal(data, &document); err != nil {
			return "", fmt.Errorf("decode %s: %w", path, err)
		}
		if strings.TrimSpace(document.MachineID) == "" {
			return "", fmt.Errorf("decode %s: machineId is required", path)
		}
		return strings.TrimSpace(document.MachineID), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	hostname, _ := os.Hostname()
	normalizedHost := normalizeMachineIDHost(hostname)
	document := machineIdentityDocument{
		Hostname:  strings.TrimSpace(hostname),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	uuid, err := newUUIDv4()
	if err != nil {
		return "", err
	}
	document.MachineID = fmt.Sprintf("%s_%s", normalizedHost, uuid)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create identity dir: %w", err)
	}

	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return "", err
	}
	encoded = append(encoded, '\n')

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, encoded, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return "", fmt.Errorf("rename %s to %s: %w", tempPath, path, err)
	}

	return document.MachineID, nil
}

func resolveMachineName(machineID string) string {
	if name := strings.TrimSpace(os.Getenv("MACHINE_NAME")); name != "" {
		return name
	}
	if hostname, err := os.Hostname(); err == nil && strings.TrimSpace(hostname) != "" {
		return strings.TrimSpace(hostname)
	}
	return machineID
}

func machineIdentityPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(homeDir) == "" {
		return "", fmt.Errorf("user home directory is empty")
	}
	return filepath.Join(homeDir, ".code-agent-gateway", "machine.json"), nil
}

func normalizeMachineIDHost(hostname string) string {
	raw := strings.ToLower(strings.TrimSpace(hostname))
	if raw == "" {
		return "machine"
	}

	var builder strings.Builder
	lastWasDash := false
	for _, r := range raw {
		isLowerAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLowerAlpha || isDigit {
			builder.WriteRune(r)
			lastWasDash = false
			continue
		}
		if !lastWasDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastWasDash = true
		}
	}

	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		return "machine"
	}
	return normalized
}

func newUUIDv4() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate machine id uuid: %w", err)
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}
