package codex

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

func NewInstanceLayout(rootDir string, agentID string) InstanceLayout {
	trimmedRoot := strings.TrimSpace(rootDir)
	trimmedAgentID := strings.TrimSpace(agentID)
	instanceRoot := filepath.Join(trimmedRoot, trimmedAgentID)
	homeDir := filepath.Join(instanceRoot, "home")
	return InstanceLayout{
		AgentID:      trimmedAgentID,
		RootDir:      instanceRoot,
		HomeDir:      homeDir,
		CodexHomeDir: filepath.Join(instanceRoot, "codex-home"),
		ConfigPath:   filepath.Join(homeDir, ".codex", "config.toml"),
	}
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
