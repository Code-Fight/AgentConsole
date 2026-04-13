package domain

type ConsolePreferences struct {
	ConsoleURL   string `json:"consoleUrl"`
	APIKey       string `json:"apiKey"`
	Profile      string `json:"profile"`
	SafetyPolicy string `json:"safetyPolicy"`
	LastThreadID string `json:"lastThreadId"`
}
