package codex

import (
	"fmt"
	"strings"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

var defaultThreadRuntimeApprovalPolicies = []string{
	"untrusted",
	"on-failure",
	"on-request",
	"never",
}

var defaultThreadRuntimeSandboxModes = []string{
	"workspace-write",
	"danger-full-access",
	"read-only",
}

func (c *AppServerClient) ReadThreadRuntimeSettings(threadID string) (domain.ThreadRuntimeSettings, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return domain.ThreadRuntimeSettings{}, fmt.Errorf("threadID is required")
	}

	options := c.readThreadRuntimeOptions()
	prefs := c.displayRuntimePreferences(c.threadRuntimePreferencesForDisplay(threadID))
	options = ensureCurrentModelOption(options, prefs.Model)

	return domain.ThreadRuntimeSettings{
		ThreadID:    threadID,
		Preferences: prefs,
		Options:     options,
	}, nil
}

func (c *AppServerClient) UpdateThreadRuntimeSettings(params agenttypes.UpdateThreadRuntimeSettingsParams) (domain.ThreadRuntimeSettings, error) {
	threadID := strings.TrimSpace(params.ThreadID)
	if threadID == "" {
		return domain.ThreadRuntimeSettings{}, fmt.Errorf("threadID is required")
	}

	options := c.readThreadRuntimeOptions()
	current := c.threadRuntimePreferencesForUpdate(threadID)
	next := normalizeThreadRuntimePreferences(current)

	if params.Patch.Model != nil {
		model := strings.TrimSpace(*params.Patch.Model)
		if model != "" && len(options.Models) > 0 && !containsModelOption(options.Models, model) {
			return domain.ThreadRuntimeSettings{}, fmt.Errorf("unsupported model %q", model)
		}
		next.Model = model
	}
	if params.Patch.ApprovalPolicy != nil {
		policy := normalizeApprovalPolicy(*params.Patch.ApprovalPolicy)
		if policy == "" && strings.TrimSpace(*params.Patch.ApprovalPolicy) != "" {
			return domain.ThreadRuntimeSettings{}, fmt.Errorf("unsupported approval policy %q", strings.TrimSpace(*params.Patch.ApprovalPolicy))
		}
		if policy != "" && !containsString(options.ApprovalPolicies, policy) {
			return domain.ThreadRuntimeSettings{}, fmt.Errorf("unsupported approval policy %q", policy)
		}
		next.ApprovalPolicy = policy
	}
	if params.Patch.SandboxMode != nil {
		mode := normalizeSandboxMode(*params.Patch.SandboxMode)
		if mode == "" && strings.TrimSpace(*params.Patch.SandboxMode) != "" {
			return domain.ThreadRuntimeSettings{}, fmt.Errorf("unsupported sandbox mode %q", strings.TrimSpace(*params.Patch.SandboxMode))
		}
		if mode != "" && !containsString(options.SandboxModes, mode) {
			return domain.ThreadRuntimeSettings{}, fmt.Errorf("unsupported sandbox mode %q", mode)
		}
		next.SandboxMode = mode
	}

	c.setThreadRuntimeDesired(threadID, next)

	display := c.displayRuntimePreferences(next)
	options = ensureCurrentModelOption(options, display.Model)

	return domain.ThreadRuntimeSettings{
		ThreadID:    threadID,
		Preferences: display,
		Options:     options,
	}, nil
}

func (c *AppServerClient) seedThreadRuntimeState(threadID string, prefs domain.ThreadRuntimePreferences) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return
	}
	next := normalizeThreadRuntimePreferences(prefs)
	if isEmptyThreadRuntimePreferences(next) {
		return
	}

	c.threadMu.Lock()
	c.threadRuntimeDesired[threadID] = next
	c.threadRuntimeApplied[threadID] = next
	c.threadMu.Unlock()
}

func (c *AppServerClient) setThreadRuntimeDesired(threadID string, prefs domain.ThreadRuntimePreferences) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return
	}
	next := normalizeThreadRuntimePreferences(prefs)

	c.threadMu.Lock()
	c.threadRuntimeDesired[threadID] = next
	c.threadMu.Unlock()
}

func (c *AppServerClient) setThreadRuntimeApplied(threadID string, prefs domain.ThreadRuntimePreferences) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return
	}
	next := normalizeThreadRuntimePreferences(prefs)

	c.threadMu.Lock()
	c.threadRuntimeApplied[threadID] = next
	c.threadMu.Unlock()
}

func (c *AppServerClient) turnStartRuntimeOverride(threadID string) (map[string]any, *domain.ThreadRuntimePreferences) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil, nil
	}

	c.threadMu.RLock()
	desired, hasDesired := c.threadRuntimeDesired[threadID]
	applied, hasApplied := c.threadRuntimeApplied[threadID]
	c.threadMu.RUnlock()
	if !hasDesired {
		return nil, nil
	}

	desired = normalizeThreadRuntimePreferences(desired)
	applied = normalizeThreadRuntimePreferences(applied)
	if hasApplied && desired == applied {
		return nil, nil
	}

	payload := map[string]any{}
	if desired.Model != "" {
		payload["model"] = desired.Model
	}
	if desired.ApprovalPolicy != "" {
		payload["approvalPolicy"] = desired.ApprovalPolicy
	}
	if policy, ok := sandboxPolicyPayloadForMode(desired.SandboxMode); ok {
		payload["sandboxPolicy"] = policy
	}
	if len(payload) == 0 {
		return nil, nil
	}

	next := desired
	return payload, &next
}

func (c *AppServerClient) threadRuntimePreferencesForUpdate(threadID string) domain.ThreadRuntimePreferences {
	c.threadMu.RLock()
	defer c.threadMu.RUnlock()

	if prefs, ok := c.threadRuntimeDesired[threadID]; ok {
		return prefs
	}
	if prefs, ok := c.threadRuntimeApplied[threadID]; ok {
		return prefs
	}
	return domain.ThreadRuntimePreferences{}
}

func (c *AppServerClient) threadRuntimePreferencesForDisplay(threadID string) domain.ThreadRuntimePreferences {
	c.threadMu.RLock()
	defer c.threadMu.RUnlock()

	if prefs, ok := c.threadRuntimeDesired[threadID]; ok {
		return prefs
	}
	if prefs, ok := c.threadRuntimeApplied[threadID]; ok {
		return prefs
	}
	return domain.ThreadRuntimePreferences{}
}

func (c *AppServerClient) displayRuntimePreferences(current domain.ThreadRuntimePreferences) domain.ThreadRuntimePreferences {
	display := normalizeThreadRuntimePreferences(current)
	if display.Model != "" && display.ApprovalPolicy != "" && display.SandboxMode != "" {
		return display
	}

	defaults := c.defaultThreadRuntimePreferences()
	if display.Model == "" {
		display.Model = defaults.Model
	}
	if display.ApprovalPolicy == "" {
		display.ApprovalPolicy = defaults.ApprovalPolicy
	}
	if display.SandboxMode == "" {
		display.SandboxMode = defaults.SandboxMode
	}
	return normalizeThreadRuntimePreferences(display)
}

func (c *AppServerClient) defaultThreadRuntimePreferences() domain.ThreadRuntimePreferences {
	config, err := c.readConfig()
	if err != nil {
		return domain.ThreadRuntimePreferences{}
	}
	defaults := domain.ThreadRuntimePreferences{
		Model:          strings.TrimSpace(stringFromAny(config["model"])),
		ApprovalPolicy: normalizeApprovalPolicy(config["approval_policy"]),
		SandboxMode:    normalizeSandboxMode(config["sandbox_mode"]),
	}

	profileName := strings.TrimSpace(stringFromAny(config["profile"]))
	if profileName != "" {
		profiles := nestedConfigMap(config, "profiles")
		profile := nestedConfigMap(profiles, profileName)
		if defaults.Model == "" {
			defaults.Model = strings.TrimSpace(stringFromAny(profile["model"]))
		}
		if defaults.ApprovalPolicy == "" {
			defaults.ApprovalPolicy = normalizeApprovalPolicy(profile["approval_policy"])
		}
	}

	return normalizeThreadRuntimePreferences(defaults)
}

func (c *AppServerClient) readThreadRuntimeOptions() domain.ThreadRuntimeOptions {
	options := domain.ThreadRuntimeOptions{
		ApprovalPolicies: append([]string(nil), defaultThreadRuntimeApprovalPolicies...),
		SandboxModes:     append([]string(nil), defaultThreadRuntimeSandboxModes...),
	}

	var modelsResponse modelListResponse
	if err := c.runner.Call("model/list", map[string]any{}, &modelsResponse); err == nil {
		models := make([]domain.ThreadRuntimeModelOption, 0, len(modelsResponse.Data))
		for _, item := range modelsResponse.Data {
			modelID := firstNonEmpty(strings.TrimSpace(item.Model), strings.TrimSpace(item.ID))
			if modelID == "" {
				continue
			}
			models = append(models, domain.ThreadRuntimeModelOption{
				ID:          modelID,
				DisplayName: firstNonEmpty(strings.TrimSpace(item.DisplayName), modelID),
				IsDefault:   item.IsDefault,
			})
		}
		options.Models = models
	}

	var requirementsResponse configRequirementsReadResponse
	if err := c.runner.Call("configRequirements/read", map[string]any{}, &requirementsResponse); err == nil && requirementsResponse.Requirements != nil {
		if parsed := parseApprovalPolicies(requirementsResponse.Requirements.AllowedApprovalPolicies); len(parsed) > 0 {
			options.ApprovalPolicies = parsed
		}
		if parsed := parseSandboxModes(requirementsResponse.Requirements.AllowedSandboxModes); len(parsed) > 0 {
			options.SandboxModes = parsed
		}
	}

	return options
}

func parseApprovalPolicies(values []any) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		policy := normalizeApprovalPolicy(value)
		if policy == "" || containsString(items, policy) {
			continue
		}
		items = append(items, policy)
	}
	return items
}

func parseSandboxModes(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		mode := normalizeSandboxMode(value)
		if mode == "" || containsString(items, mode) {
			continue
		}
		items = append(items, mode)
	}
	return items
}

func ensureCurrentModelOption(options domain.ThreadRuntimeOptions, currentModel string) domain.ThreadRuntimeOptions {
	currentModel = strings.TrimSpace(currentModel)
	if currentModel == "" || containsModelOption(options.Models, currentModel) {
		return options
	}

	options.Models = append(options.Models, domain.ThreadRuntimeModelOption{
		ID:          currentModel,
		DisplayName: currentModel,
	})
	return options
}

func containsModelOption(options []domain.ThreadRuntimeModelOption, model string) bool {
	for _, option := range options {
		if strings.TrimSpace(option.ID) == model {
			return true
		}
	}
	return false
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}

func normalizeThreadRuntimePreferences(prefs domain.ThreadRuntimePreferences) domain.ThreadRuntimePreferences {
	return domain.ThreadRuntimePreferences{
		Model:          strings.TrimSpace(prefs.Model),
		ApprovalPolicy: normalizeApprovalPolicy(prefs.ApprovalPolicy),
		SandboxMode:    normalizeSandboxMode(prefs.SandboxMode),
	}
}

func isEmptyThreadRuntimePreferences(prefs domain.ThreadRuntimePreferences) bool {
	return strings.TrimSpace(prefs.Model) == "" &&
		strings.TrimSpace(prefs.ApprovalPolicy) == "" &&
		strings.TrimSpace(prefs.SandboxMode) == ""
}

func normalizeApprovalPolicy(value any) string {
	switch typed := value.(type) {
	case string:
		policy := strings.ToLower(strings.TrimSpace(typed))
		switch policy {
		case "untrusted", "unless-trusted", "unless_trusted":
			return "untrusted"
		case "on-failure", "on_failure", "onfailure":
			return "on-failure"
		case "on-request", "on_request", "onrequest":
			return "on-request"
		case "never":
			return "never"
		case "granular":
			return "granular"
		default:
			return ""
		}
	case map[string]any:
		if kind := strings.TrimSpace(stringFromAny(typed["type"])); kind != "" {
			return normalizeApprovalPolicy(kind)
		}
		if len(typed) > 0 {
			return "granular"
		}
	}

	return ""
}

func normalizeSandboxMode(value any) string {
	switch typed := value.(type) {
	case string:
		mode := strings.ToLower(strings.TrimSpace(typed))
		switch mode {
		case "readonly", "read-only", "read_only", "readonlymode", "readOnly":
			return "read-only"
		case "workspacewrite", "workspace-write", "workspace_write", "workspaceWrite":
			return "workspace-write"
		case "dangerfullaccess", "danger-full-access", "danger_full_access", "dangerFullAccess":
			return "danger-full-access"
		case "externalsandbox", "external-sandbox", "external_sandbox", "externalSandbox":
			return "danger-full-access"
		default:
			return ""
		}
	case map[string]any:
		return normalizeSandboxMode(stringFromAny(typed["type"]))
	}

	return ""
}

func sandboxPolicyPayloadForMode(mode string) (map[string]any, bool) {
	switch normalizeSandboxMode(mode) {
	case "read-only":
		return map[string]any{"type": "readOnly"}, true
	case "workspace-write":
		return map[string]any{"type": "workspaceWrite"}, true
	case "danger-full-access":
		return map[string]any{"type": "dangerFullAccess"}, true
	default:
		return nil, false
	}
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}
