package domain

type OverviewMetrics struct {
	OnlineMachines   int `json:"onlineMachines"`
	ActiveThreads    int `json:"activeThreads"`
	PendingApprovals int `json:"pendingApprovals"`
	RunningAgents    int `json:"runningAgents"`
	EnvironmentItems int `json:"environmentItems"`
}
