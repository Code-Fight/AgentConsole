package settings

import "code-agent-gateway/common/domain"

type Store interface {
	ListAgentTypes() []domain.AgentDescriptor
	GetGlobal(agentType domain.AgentType) (domain.AgentConfigDocument, bool, error)
	PutGlobal(agentType domain.AgentType, document domain.AgentConfigDocument) error
	GetMachine(machineID string, agentType domain.AgentType) (domain.AgentConfigDocument, bool, error)
	PutMachine(machineID string, agentType domain.AgentType, document domain.AgentConfigDocument) error
	DeleteMachine(machineID string, agentType domain.AgentType) error
}
