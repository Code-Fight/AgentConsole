package domain

type CapabilitySnapshot struct {
	ThreadHub                   bool `json:"threadHub"`
	ThreadWorkspace             bool `json:"threadWorkspace"`
	Approvals                   bool `json:"approvals"`
	StartTurn                   bool `json:"startTurn"`
	SteerTurn                   bool `json:"steerTurn"`
	InterruptTurn               bool `json:"interruptTurn"`
	MachineInstallAgent         bool `json:"machineInstallAgent"`
	MachineRemoveAgent          bool `json:"machineRemoveAgent"`
	EnvironmentSyncCatalog      bool `json:"environmentSyncCatalog"`
	EnvironmentRestartBridge    bool `json:"environmentRestartBridge"`
	EnvironmentOpenMarketplace  bool `json:"environmentOpenMarketplace"`
	EnvironmentMutateResources  bool `json:"environmentMutateResources"`
	EnvironmentWriteMcp         bool `json:"environmentWriteMcp"`
	EnvironmentWriteSkills      bool `json:"environmentWriteSkills"`
	SettingsEditGatewayEndpoint bool `json:"settingsEditGatewayEndpoint"`
	SettingsEditConsoleProfile  bool `json:"settingsEditConsoleProfile"`
	SettingsEditSafetyPolicy    bool `json:"settingsEditSafetyPolicy"`
	SettingsGlobalDefault       bool `json:"settingsGlobalDefault"`
	SettingsMachineOverride     bool `json:"settingsMachineOverride"`
	SettingsApplyMachine        bool `json:"settingsApplyMachine"`
	DashboardMetrics            bool `json:"dashboardMetrics"`
	AgentLifecycle              bool `json:"agentLifecycle"`
}
