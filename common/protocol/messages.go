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

type ThreadSnapshotPayload struct {
	Threads []domain.Thread `json:"threads"`
}

type EnvironmentSnapshotPayload struct {
	Environment []domain.EnvironmentResource `json:"environment"`
}

type ThreadCreateCommandPayload struct {
	Title string `json:"title,omitempty"`
}

type ThreadReadCommandPayload struct {
	ThreadID string `json:"threadId"`
}

type ThreadResumeCommandPayload struct {
	ThreadID string `json:"threadId"`
}

type ThreadArchiveCommandPayload struct {
	ThreadID string `json:"threadId"`
}

type TurnStartCommandPayload struct {
	ThreadID string `json:"threadId"`
	Input    string `json:"input"`
}

type TurnSteerCommandPayload struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	Input    string `json:"input"`
}

type TurnInterruptCommandPayload struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type RuntimeStartCommandPayload struct{}

type RuntimeStopCommandPayload struct{}

type EnvironmentSkillSetEnabledCommandPayload struct {
	SkillID string `json:"skillId"`
	Enabled bool   `json:"enabled"`
}

type EnvironmentPluginUninstallCommandPayload struct {
	PluginID string `json:"pluginId"`
}

type ApprovalRespondCommandPayload struct {
	RequestID string `json:"requestId"`
	ThreadID  string `json:"threadId,omitempty"`
	TurnID    string `json:"turnId,omitempty"`
	Decision  string `json:"decision"`
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

type EnvironmentSkillSetEnabledCommandResult struct {
	SkillID string `json:"skillId"`
	Enabled bool   `json:"enabled"`
}

type EnvironmentPluginUninstallCommandResult struct {
	PluginID string `json:"pluginId"`
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
	RequestID string `json:"requestId"`
	ThreadID  string `json:"threadId,omitempty"`
	TurnID    string `json:"turnId,omitempty"`
	ItemID    string `json:"itemId,omitempty"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason,omitempty"`
	Command   string `json:"command,omitempty"`
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
