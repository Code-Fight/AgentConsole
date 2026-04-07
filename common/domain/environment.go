package domain

type EnvironmentKind string
type EnvironmentResourceStatus string

const (
	EnvironmentKindSkill  EnvironmentKind = "skill"
	EnvironmentKindMCP    EnvironmentKind = "mcp"
	EnvironmentKindPlugin EnvironmentKind = "plugin"

	EnvironmentResourceStatusUnknown    EnvironmentResourceStatus = "unknown"
	EnvironmentResourceStatusHealthy    EnvironmentResourceStatus = "healthy"
	EnvironmentResourceStatusDegraded   EnvironmentResourceStatus = "degraded"
	EnvironmentResourceStatusUnhealthy  EnvironmentResourceStatus = "unhealthy"
	EnvironmentResourceStatusRestarting EnvironmentResourceStatus = "restarting"
)

type EnvironmentResource struct {
	ResourceID      string                    `json:"resourceId"`
	MachineID       string                    `json:"machineId"`
	Kind            EnvironmentKind           `json:"kind"`
	DisplayName     string                    `json:"displayName"`
	Status          EnvironmentResourceStatus `json:"status"`
	RestartRequired bool                      `json:"restartRequired"`
	LastObservedAt  string                    `json:"lastObservedAt"`
}
