package domain

type ThreadRuntimePreferences struct {
	Model          string `json:"model,omitempty"`
	ApprovalPolicy string `json:"approvalPolicy,omitempty"`
	SandboxMode    string `json:"sandboxMode,omitempty"`
}

type ThreadRuntimePreferencePatch struct {
	Model          *string `json:"model,omitempty"`
	ApprovalPolicy *string `json:"approvalPolicy,omitempty"`
	SandboxMode    *string `json:"sandboxMode,omitempty"`
}

type ThreadRuntimeModelOption struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	IsDefault   bool   `json:"isDefault,omitempty"`
}

type ThreadRuntimeOptions struct {
	Models           []ThreadRuntimeModelOption `json:"models,omitempty"`
	ApprovalPolicies []string                   `json:"approvalPolicies,omitempty"`
	SandboxModes     []string                   `json:"sandboxModes,omitempty"`
}

type ThreadRuntimeSettings struct {
	ThreadID    string                   `json:"threadId"`
	Preferences ThreadRuntimePreferences `json:"preferences"`
	Options     ThreadRuntimeOptions     `json:"options"`
}
