package codex

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type InstanceLayout struct {
	AgentID      string
	RootDir      string
	HomeDir      string
	CodexHomeDir string
	ConfigPath   string
}

func ValidateAgentID(agentID string) error {
	trimmedAgentID := strings.TrimSpace(agentID)
	if trimmedAgentID == "" {
		return fmt.Errorf("agentID is required")
	}
	if strings.Contains(trimmedAgentID, "..") {
		return fmt.Errorf("agentID must not contain '..'")
	}
	if strings.ContainsAny(trimmedAgentID, `/\`) {
		return fmt.Errorf("agentID must not contain path separators")
	}
	for idx, r := range trimmedAgentID {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			if idx == 0 && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return fmt.Errorf("agentID must start with a letter or digit")
			}
			continue
		}
		return fmt.Errorf("agentID contains unsupported characters")
	}
	return nil
}

func NewInstanceLayout(rootDir string, agentID string) (InstanceLayout, error) {
	trimmedRoot := strings.TrimSpace(rootDir)
	trimmedAgentID := strings.TrimSpace(agentID)
	if err := ValidateAgentID(trimmedAgentID); err != nil {
		return InstanceLayout{}, err
	}
	instanceRoot := filepath.Join(trimmedRoot, trimmedAgentID)
	homeDir := filepath.Join(instanceRoot, "home")
	return InstanceLayout{
		AgentID:      trimmedAgentID,
		RootDir:      instanceRoot,
		HomeDir:      homeDir,
		CodexHomeDir: filepath.Join(instanceRoot, "codex-home"),
		ConfigPath:   filepath.Join(homeDir, ".codex", "config.toml"),
	}, nil
}

func (l InstanceLayout) ApplyConfig(document domain.AgentConfigDocument) (agenttypes.ApplyConfigResult, error) {
	return applyConfigDocument(func() (string, error) {
		if err := os.MkdirAll(l.HomeDir, 0o755); err != nil {
			return "", err
		}
		if err := os.MkdirAll(l.CodexHomeDir, 0o755); err != nil {
			return "", err
		}
		return l.HomeDir, nil
	}, document)
}

func NewIsolatedAppServerClient(ctx context.Context, codexBin string, layout InstanceLayout) (*AppServerClient, func() error, error) {
	runner, err := newStdioRunner(ctx, codexBin, func(ctx context.Context, name string, args ...string) (stdioProcess, error) {
		if err := os.MkdirAll(layout.RootDir, 0o755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(layout.HomeDir, 0o755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(layout.CodexHomeDir, 0o755); err != nil {
			return nil, err
		}
		if err := seedSharedCodexFiles(layout); err != nil {
			return nil, err
		}

		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = layout.RootDir
		cmd.Env = append(os.Environ(),
			"HOME="+layout.HomeDir,
			"CODEX_HOME="+layout.CodexHomeDir,
		)
		return cmd, nil
	})
	if err != nil {
		return nil, nil, err
	}

	client := NewAppServerClient(runner)
	client.homeDir = func() (string, error) {
		return layout.HomeDir, nil
	}
	if err := client.Initialize(); err != nil {
		_ = runner.Close()
		return nil, nil, err
	}
	return client, runner.Close, nil
}

func seedSharedCodexFiles(layout InstanceLayout) error {
	homeDir, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(homeDir) == "" {
		return nil
	}

	pairs := []struct {
		source string
		target string
	}{
		{
			source: filepath.Join(homeDir, ".codex", "auth.json"),
			target: filepath.Join(layout.CodexHomeDir, "auth.json"),
		},
		{
			source: filepath.Join(homeDir, ".codex", "config.toml"),
			target: layout.ConfigPath,
		},
	}

	for _, pair := range pairs {
		if err := copyIfPresent(pair.source, pair.target); err != nil {
			return err
		}
	}

	return nil
}

func copyIfPresent(sourcePath string, targetPath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if sourceInfo.IsDir() {
		return nil
	}

	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	if targetData, err := os.ReadFile(targetPath); err == nil && string(targetData) == string(sourceData) {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, sourceData, 0o600); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	return os.Rename(tempPath, targetPath)
}
