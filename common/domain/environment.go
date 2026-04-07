package domain

type EnvironmentKind string

const (
	EnvironmentKindSkill  EnvironmentKind = "skill"
	EnvironmentKindMCP    EnvironmentKind = "mcp"
	EnvironmentKindPlugin EnvironmentKind = "plugin"
)

type EnvironmentResource struct {
	ID              string          `json:"id"`
	Kind            EnvironmentKind `json:"kind"`
	DisplayName     string          `json:"displayName"`
	Status          string          `json:"status"`
	RestartRequired bool            `json:"restartRequired"`
	LastObservedAt  string          `json:"lastObservedAt"`
}
