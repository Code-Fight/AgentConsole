package domain

type AgentInstanceStatus string

const (
	AgentInstanceStatusRunning AgentInstanceStatus = "running"
	AgentInstanceStatusStopped AgentInstanceStatus = "stopped"
	AgentInstanceStatusError   AgentInstanceStatus = "error"
)

type AgentInstance struct {
	AgentID     string              `json:"agentId"`
	AgentType   AgentType           `json:"agentType"`
	DisplayName string              `json:"displayName"`
	Status      AgentInstanceStatus `json:"status"`
}
