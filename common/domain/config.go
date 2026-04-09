package domain

type AgentType string
type AgentConfigFormat string

const (
	AgentTypeCodex AgentType = "codex"

	AgentConfigFormatTOML AgentConfigFormat = "toml"
)

type AgentDescriptor struct {
	AgentType   AgentType `json:"agentType"`
	DisplayName string    `json:"displayName"`
}

type AgentConfigDocument struct {
	AgentType AgentType         `json:"agentType"`
	Format    AgentConfigFormat `json:"format"`
	Content   string            `json:"content"`
	UpdatedAt string            `json:"updatedAt,omitempty"`
	UpdatedBy string            `json:"updatedBy,omitempty"`
	Version   int64             `json:"version,omitempty"`
}

type MachineAgentConfigAssignment struct {
	MachineID         string               `json:"machineId"`
	AgentType         AgentType            `json:"agentType"`
	GlobalDefault     *AgentConfigDocument `json:"globalDefault,omitempty"`
	MachineOverride   *AgentConfigDocument `json:"machineOverride,omitempty"`
	UsesGlobalDefault bool                 `json:"usesGlobalDefault"`
}
