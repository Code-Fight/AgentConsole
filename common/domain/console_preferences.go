package domain

type ConsolePreferences struct {
	Profile      string            `json:"profile"`
	SafetyPolicy string            `json:"safetyPolicy"`
	LastThreadID string            `json:"lastThreadId"`
	ThreadTitles map[string]string `json:"threadTitles,omitempty"`
}
