package types

import (
	"context"

	"code-agent-gateway/common/domain"
)

type ManagedAgentSpec struct {
	AgentID     string
	AgentType   domain.AgentType
	DisplayName string
}

type RuntimeFactory interface {
	Start(ctx context.Context, spec ManagedAgentSpec) (Runtime, func() error, error)
}

type CreateThreadParams struct {
	Title string
}

type StartTurnParams struct {
	ThreadID string
	Input    string
}

type UpdateThreadRuntimeSettingsParams struct {
	ThreadID string
	Patch    domain.ThreadRuntimePreferencePatch
}

type SteerTurnParams struct {
	ThreadID string
	TurnID   string
	Input    string
}

type InterruptTurnParams struct {
	ThreadID string
	TurnID   string
}

type CreateSkillParams struct {
	Name        string
	Description string
}

type InstallPluginParams struct {
	PluginID        string
	MarketplacePath string
	PluginName      string
}

type ApplyConfigResult struct {
	AgentType domain.AgentType
	FilePath  string
}

type TurnDelta struct {
	Sequence int
	Delta    string
}

type RuntimeTurnEventType string

const (
	RuntimeTurnEventTypeStarted   RuntimeTurnEventType = "turn.started"
	RuntimeTurnEventTypeDelta     RuntimeTurnEventType = "turn.delta"
	RuntimeTurnEventTypeCompleted RuntimeTurnEventType = "turn.completed"
	RuntimeTurnEventTypeFailed    RuntimeTurnEventType = "turn.failed"
)

type RuntimeTurnEvent struct {
	Type         RuntimeTurnEventType
	RequestID    string
	ThreadID     string
	TurnID       string
	Sequence     int
	Delta        string
	ErrorMessage string
	Turn         domain.Turn
}

type RuntimeTurnEventSource interface {
	SetTurnEventHandler(func(RuntimeTurnEvent))
}

type RuntimeApprovalRequest struct {
	RequestID string
	ThreadID  string
	TurnID    string
	ItemID    string
	Kind      string
	Reason    string
	Command   string
}

type RuntimeApprovalEventSource interface {
	SetApprovalHandler(func(RuntimeApprovalRequest))
}

type RuntimeApprovalResponder interface {
	RespondApproval(requestID string, decision string, answers map[string]any) error
}

type RuntimeSkillConfigurator interface {
	SetSkillEnabled(nameOrPath string, enabled bool) error
}

type RuntimeSkillManager interface {
	CreateSkill(params CreateSkillParams) (string, error)
	DeleteSkill(nameOrPath string) error
}

type RuntimeMCPManager interface {
	UpsertMCPServer(serverID string, config map[string]any) error
	RemoveMCPServer(serverID string) error
	SetMCPServerEnabled(serverID string, enabled bool) error
	ReloadMCPServers() error
}

type RuntimePluginManager interface {
	InstallPlugin(params InstallPluginParams) error
	SetPluginEnabled(pluginID string, enabled bool) error
	UninstallPlugin(pluginID string) error
}

type RuntimeConfigManager interface {
	ApplyConfig(document domain.AgentConfigDocument) (ApplyConfigResult, error)
}

type RuntimeThreadRuntimeManager interface {
	ReadThreadRuntimeSettings(threadID string) (domain.ThreadRuntimeSettings, error)
	UpdateThreadRuntimeSettings(params UpdateThreadRuntimeSettingsParams) (domain.ThreadRuntimeSettings, error)
}

type StartTurnResult struct {
	TurnID   string
	ThreadID string
	Deltas   []TurnDelta
}

type SteerTurnResult struct {
	TurnID   string
	ThreadID string
	Deltas   []TurnDelta
}

type Runtime interface {
	ListThreads() ([]domain.Thread, error)
	ListEnvironment() ([]domain.EnvironmentResource, error)
	CreateThread(params CreateThreadParams) (domain.Thread, error)
	ReadThread(threadID string) (domain.Thread, error)
	ResumeThread(threadID string) (domain.Thread, error)
	ArchiveThread(threadID string) error
	StartTurn(params StartTurnParams) (StartTurnResult, error)
	SteerTurn(params SteerTurnParams) (SteerTurnResult, error)
	InterruptTurn(params InterruptTurnParams) (domain.Turn, error)
}
