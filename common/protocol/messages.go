package protocol

import (
	"encoding/json"
	"errors"
	"strings"

	"code-agent-gateway/common/domain"
)

type Category string

const (
	CategorySystem   Category = "system"
	CategoryCommand  Category = "command"
	CategoryEvent    Category = "event"
	CategorySnapshot Category = "snapshot"
)

type MachineSnapshotPayload struct {
	Machine domain.Machine `json:"machine"`
}

type ClientRegisterPayload struct {
	Name string `json:"name,omitempty"`
}

type ThreadSnapshotPayload struct {
	Threads []domain.Thread `json:"threads"`
}

type EnvironmentSnapshotPayload struct {
	Environment []domain.EnvironmentResource `json:"environment"`
}

type MachineUpdatedPayload struct {
	Machine domain.Machine `json:"machine"`
}

type ThreadUpdatedPayload struct {
	MachineID string         `json:"machineId"`
	AgentID   string         `json:"agentId,omitempty"`
	ThreadID  string         `json:"threadId,omitempty"`
	Thread    *domain.Thread `json:"thread,omitempty"`
}

type ResourceChangedPayload struct {
	MachineID  string                      `json:"machineId"`
	AgentID    string                      `json:"agentId,omitempty"`
	Kind       domain.EnvironmentKind      `json:"kind,omitempty"`
	ResourceID string                      `json:"resourceId,omitempty"`
	Resource   *domain.EnvironmentResource `json:"resource,omitempty"`
	Action     string                      `json:"action,omitempty"`
}

type ThreadCreateCommandPayload struct {
	AgentID string `json:"agentId"`
	Title   string `json:"title,omitempty"`
}

type ThreadReadCommandPayload struct {
	ThreadID string `json:"threadId"`
	AgentID  string `json:"agentId,omitempty"`
}

type ThreadResumeCommandPayload struct {
	ThreadID string `json:"threadId"`
	AgentID  string `json:"agentId,omitempty"`
}

type ThreadArchiveCommandPayload struct {
	ThreadID string `json:"threadId"`
	AgentID  string `json:"agentId,omitempty"`
}

type TurnStartCommandPayload struct {
	ThreadID string `json:"threadId"`
	AgentID  string `json:"agentId,omitempty"`
	Input    string `json:"input"`
}

type TurnSteerCommandPayload struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	AgentID  string `json:"agentId,omitempty"`
	Input    string `json:"input"`
}

type TurnInterruptCommandPayload struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	AgentID  string `json:"agentId,omitempty"`
}

type RuntimeStartCommandPayload struct{}

type RuntimeStopCommandPayload struct{}

type AgentConfigApplyCommandPayload struct {
	AgentType string                     `json:"agentType"`
	Source    string                     `json:"source,omitempty"`
	Document  domain.AgentConfigDocument `json:"document"`
}

type EnvironmentSkillSetEnabledCommandPayload struct {
	SkillID string `json:"skillId"`
	AgentID string `json:"agentId,omitempty"`
	Enabled bool   `json:"enabled"`
}

type EnvironmentSkillCreateCommandPayload struct {
	AgentID     string `json:"agentId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type EnvironmentSkillDeleteCommandPayload struct {
	SkillID string `json:"skillId"`
	AgentID string `json:"agentId,omitempty"`
}

type EnvironmentMCPUpsertCommandPayload struct {
	ServerID string         `json:"serverId"`
	AgentID  string         `json:"agentId,omitempty"`
	Config   map[string]any `json:"config"`
}

type EnvironmentMCPRemoveCommandPayload struct {
	ServerID string `json:"serverId"`
	AgentID  string `json:"agentId,omitempty"`
}

type EnvironmentMCPSetEnabledCommandPayload struct {
	ServerID string `json:"serverId"`
	AgentID  string `json:"agentId,omitempty"`
	Enabled  bool   `json:"enabled"`
}

type EnvironmentPluginInstallCommandPayload struct {
	PluginID        string `json:"pluginId"`
	AgentID         string `json:"agentId,omitempty"`
	MarketplacePath string `json:"marketplacePath"`
	PluginName      string `json:"pluginName"`
}

type EnvironmentPluginSetEnabledCommandPayload struct {
	PluginID string `json:"pluginId"`
	AgentID  string `json:"agentId,omitempty"`
	Enabled  bool   `json:"enabled"`
}

type EnvironmentPluginUninstallCommandPayload struct {
	PluginID string `json:"pluginId"`
	AgentID  string `json:"agentId,omitempty"`
}

type EnvironmentRefreshCommandPayload struct{}

type EnvironmentRefreshCommandResult struct{}

type EnvironmentMCPReloadCommandPayload struct{}

type EnvironmentMCPReloadCommandResult struct{}

type MachineAgentInstallCommandPayload struct {
	AgentType   string `json:"agentType"`
	DisplayName string `json:"displayName"`
}

type MachineAgentDeleteCommandPayload struct {
	AgentID string `json:"agentId"`
}

type MachineAgentConfigReadCommandPayload struct {
	AgentID string `json:"agentId"`
}

type MachineAgentConfigWriteCommandPayload struct {
	AgentID  string                     `json:"agentId"`
	Document domain.AgentConfigDocument `json:"document"`
}

type ApprovalQuestionPayload struct {
	ID      string   `json:"id"`
	Header  string   `json:"header,omitempty"`
	Text    string   `json:"text,omitempty"`
	Options []string `json:"options,omitempty"`
}

type ApprovalRespondCommandPayload struct {
	RequestID string         `json:"requestId"`
	ThreadID  string         `json:"threadId,omitempty"`
	TurnID    string         `json:"turnId,omitempty"`
	Decision  string         `json:"decision"`
	Answers   map[string]any `json:"answers,omitempty"`
}

type CommandCompletedPayload struct {
	CommandName string          `json:"commandName"`
	Result      json.RawMessage `json:"result,omitempty"`
}

type CommandRejectedPayload struct {
	CommandName string `json:"commandName"`
	Reason      string `json:"reason,omitempty"`
	ThreadID    string `json:"threadId,omitempty"`
}

type ThreadCreateCommandResult struct {
	Thread domain.Thread `json:"thread"`
}

type ThreadReadCommandResult struct {
	Thread domain.Thread `json:"thread"`
}

type ThreadResumeCommandResult struct {
	Thread domain.Thread `json:"thread"`
}

type ThreadArchiveCommandResult struct {
	ThreadID string `json:"threadId"`
}

type TurnStartCommandResult struct {
	TurnID   string `json:"turnId"`
	ThreadID string `json:"threadId"`
}

type TurnSteerCommandResult struct {
	TurnID   string `json:"turnId"`
	ThreadID string `json:"threadId"`
}

type TurnInterruptCommandResult struct {
	Turn domain.Turn `json:"turn"`
}

type RuntimeStartCommandResult struct{}

type RuntimeStopCommandResult struct{}

type AgentConfigApplyCommandResult struct {
	AgentType string `json:"agentType"`
	FilePath  string `json:"filePath"`
	Source    string `json:"source,omitempty"`
}

type EnvironmentSkillSetEnabledCommandResult struct {
	SkillID string `json:"skillId"`
	Enabled bool   `json:"enabled"`
}

type EnvironmentSkillCreateCommandResult struct {
	SkillID string `json:"skillId"`
}

type EnvironmentSkillDeleteCommandResult struct {
	SkillID string `json:"skillId"`
}

type EnvironmentMCPUpsertCommandResult struct {
	ServerID string `json:"serverId"`
}

type EnvironmentMCPRemoveCommandResult struct {
	ServerID string `json:"serverId"`
}

type EnvironmentMCPSetEnabledCommandResult struct {
	ServerID string `json:"serverId"`
	Enabled  bool   `json:"enabled"`
}

type EnvironmentPluginInstallCommandResult struct {
	PluginID string `json:"pluginId"`
}

type EnvironmentPluginSetEnabledCommandResult struct {
	PluginID string `json:"pluginId"`
	Enabled  bool   `json:"enabled"`
}

type EnvironmentPluginUninstallCommandResult struct {
	PluginID string `json:"pluginId"`
}

type MachineAgentInstallCommandResult struct {
	Agent domain.AgentInstance `json:"agent"`
}

type MachineAgentDeleteCommandResult struct {
	AgentID string `json:"agentId"`
}

type MachineAgentConfigReadCommandResult struct {
	Document domain.AgentConfigDocument `json:"document"`
}

type MachineAgentConfigWriteCommandResult struct {
	Document domain.AgentConfigDocument `json:"document"`
}

type ApprovalRespondCommandResult struct {
	RequestID string `json:"requestId"`
	Decision  string `json:"decision"`
}

type TurnDeltaPayload struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	Sequence int    `json:"sequence"`
	Delta    string `json:"delta"`
}

type TurnStartedPayload struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type TurnCompletedPayload struct {
	Turn domain.Turn `json:"turn"`
}

type ApprovalRequiredPayload struct {
	RequestID string                    `json:"requestId"`
	ThreadID  string                    `json:"threadId,omitempty"`
	TurnID    string                    `json:"turnId,omitempty"`
	ItemID    string                    `json:"itemId,omitempty"`
	Kind      string                    `json:"kind"`
	Reason    string                    `json:"reason,omitempty"`
	Command   string                    `json:"command,omitempty"`
	Questions []ApprovalQuestionPayload `json:"questions,omitempty"`
}

type ApprovalResolvedPayload struct {
	RequestID string `json:"requestId"`
	ThreadID  string `json:"threadId,omitempty"`
	TurnID    string `json:"turnId,omitempty"`
	Decision  string `json:"decision"`
}

type Envelope struct {
	Version   string          `json:"version"`
	Category  Category        `json:"category"`
	Name      string          `json:"name"`
	RequestID string          `json:"requestId,omitempty"`
	MachineID string          `json:"machineId,omitempty"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func (e Envelope) Validate() error {
	if e.Category == CategoryCommand && strings.TrimSpace(e.RequestID) == "" {
		return errors.New("requestId is required for command envelopes")
	}

	return nil
}

func (e Envelope) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}

	type envelopeAlias Envelope
	return json.Marshal(envelopeAlias(e))
}
