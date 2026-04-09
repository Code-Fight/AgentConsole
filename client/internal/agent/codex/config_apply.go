package codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
	toml "github.com/pelletier/go-toml/v2"
)

func (c *AppServerClient) ApplyConfig(document domain.AgentConfigDocument) (agenttypes.ApplyConfigResult, error) {
	homeDir := c.homeDir
	if homeDir == nil {
		homeDir = resolveUserHomeDir
	}
	return applyConfigDocument(homeDir, document)
}

func (a *FakeAdapter) ApplyConfig(document domain.AgentConfigDocument) (agenttypes.ApplyConfigResult, error) {
	return applyConfigDocument(resolveUserHomeDir, document)
}

func applyConfigDocument(resolveHomeDir func() (string, error), document domain.AgentConfigDocument) (agenttypes.ApplyConfigResult, error) {
	if document.AgentType != domain.AgentTypeCodex {
		return agenttypes.ApplyConfigResult{}, fmt.Errorf("unsupported agentType %q", document.AgentType)
	}
	if document.Format != "" && document.Format != domain.AgentConfigFormatTOML {
		return agenttypes.ApplyConfigResult{}, fmt.Errorf("unsupported config format %q", document.Format)
	}
	if strings.TrimSpace(document.Content) == "" {
		return agenttypes.ApplyConfigResult{}, fmt.Errorf("config content is required")
	}

	if err := validateTOML(document.Content); err != nil {
		return agenttypes.ApplyConfigResult{}, err
	}

	homeDir, err := resolveHomeDir()
	if err != nil {
		return agenttypes.ApplyConfigResult{}, err
	}

	configPath := filepath.Join(homeDir, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return agenttypes.ApplyConfigResult{}, err
	}
	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(document.Content), 0o600); err != nil {
		return agenttypes.ApplyConfigResult{}, err
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		return agenttypes.ApplyConfigResult{}, err
	}

	return agenttypes.ApplyConfigResult{
		AgentType: domain.AgentTypeCodex,
		FilePath:  configPath,
	}, nil
}

func validateTOML(content string) error {
	var out map[string]any
	if err := toml.Unmarshal([]byte(content), &out); err != nil {
		return fmt.Errorf("invalid toml: %w", err)
	}
	return nil
}

func resolveUserHomeDir() (string, error) {
	return os.UserHomeDir()
}
