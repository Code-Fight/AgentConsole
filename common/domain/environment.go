package domain

type EnvironmentKind string
type EnvironmentResourceStatus string

const (
	EnvironmentKindSkill  EnvironmentKind = "skill"
	EnvironmentKindMCP    EnvironmentKind = "mcp"
	EnvironmentKindPlugin EnvironmentKind = "plugin"

	EnvironmentResourceStatusUnknown      EnvironmentResourceStatus = "unknown"
	EnvironmentResourceStatusEnabled      EnvironmentResourceStatus = "enabled"
	EnvironmentResourceStatusDisabled     EnvironmentResourceStatus = "disabled"
	EnvironmentResourceStatusAuthRequired EnvironmentResourceStatus = "auth_required"
	EnvironmentResourceStatusError        EnvironmentResourceStatus = "error"
)

type EnvironmentResource struct {
	ResourceID      string                    `json:"resourceId"`
	MachineID       string                    `json:"machineId"`
	Kind            EnvironmentKind           `json:"kind"`
	DisplayName     string                    `json:"displayName"`
	Status          EnvironmentResourceStatus `json:"status"`
	RestartRequired bool                      `json:"restartRequired"`
	LastObservedAt  string                    `json:"lastObservedAt"`
	Details         map[string]any            `json:"details,omitempty"`
}
