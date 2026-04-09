package codex

import (
	"fmt"
	"strings"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

func (c *AppServerClient) ListEnvironment() ([]domain.EnvironmentResource, error) {
	var skills skillsListResponse
	if err := c.runner.Call("skills/list", map[string]any{}, &skills); err != nil {
		return nil, err
	}

	var mcps mcpServerStatusListResponse
	if err := c.runner.Call("mcpServerStatus/list", map[string]any{}, &mcps); err != nil {
		return nil, err
	}

	var plugins pluginListResponse
	if err := c.runner.Call("plugin/list", map[string]any{}, &plugins); err != nil {
		return nil, err
	}

	config, err := c.readConfig()
	if err != nil {
		config = map[string]any{}
	}
	mcpConfigs := nestedConfigMap(config, "mcp_servers")

	lastObservedAt := c.now().UTC().Format(timeLayoutRFC3339)
	environment := make([]domain.EnvironmentResource, 0, len(skills.Data)+len(mcps.Data)+len(plugins.Marketplaces))
	for _, entry := range skills.Data {
		for _, skill := range entry.Skills {
			resourceID := strings.TrimSpace(skill.Path)
			if resourceID == "" {
				resourceID = strings.TrimSpace(skill.Name)
			}
			environment = append(environment, domain.EnvironmentResource{
				ResourceID:      resourceID,
				Kind:            domain.EnvironmentKindSkill,
				DisplayName:     skill.Name,
				Status:          enabledStatus(skill.Enabled),
				RestartRequired: false,
				LastObservedAt:  lastObservedAt,
				Details: map[string]any{
					"path":    resourceID,
					"enabled": skill.Enabled,
				},
			})
		}
	}
	for _, mcp := range mcps.Data {
		resourceID := strings.TrimSpace(mcp.ID)
		if resourceID == "" {
			resourceID = strings.TrimSpace(mcp.Name)
		}
		displayName := strings.TrimSpace(mcp.DisplayName)
		if displayName == "" {
			displayName = strings.TrimSpace(mcp.Name)
		}
		if displayName == "" {
			displayName = resourceID
		}
		details := buildMCPDetails(mcp, nestedConfigMap(mcpConfigs, resourceID))
		environment = append(environment, domain.EnvironmentResource{
			ResourceID:      resourceID,
			Kind:            domain.EnvironmentKindMCP,
			DisplayName:     displayName,
			Status:          mcpStatus(mcp),
			RestartRequired: false,
			LastObservedAt:  lastObservedAt,
			Details:         details,
		})
	}
	for _, marketplace := range plugins.Marketplaces {
		for _, plugin := range marketplace.Plugins {
			details := map[string]any{
				"marketplaceName": marketplace.Name,
				"marketplacePath": marketplace.Path,
				"pluginName":      firstNonEmpty(plugin.Name, plugin.ID),
				"installed":       plugin.Installed,
				"enabled":         plugin.Enabled,
			}
			if strings.TrimSpace(plugin.InstallPolicy) != "" {
				details["installPolicy"] = plugin.InstallPolicy
			}
			if strings.TrimSpace(plugin.AuthPolicy) != "" {
				details["authPolicy"] = plugin.AuthPolicy
			}
			pluginName := firstNonEmpty(plugin.Name, plugin.ID)
			if strings.TrimSpace(marketplace.Path) != "" && pluginName != "" {
				if pluginDetail, err := c.readPluginDetail(marketplace.Path, pluginName); err == nil {
					for key, value := range buildPluginDetails(pluginDetail) {
						details[key] = value
					}
				} else {
					details["detailError"] = err.Error()
				}
			}
			environment = append(environment, domain.EnvironmentResource{
				ResourceID:      plugin.ID,
				Kind:            domain.EnvironmentKindPlugin,
				DisplayName:     pluginName,
				Status:          pluginStatus(plugin),
				RestartRequired: c.isRestartRequired(domain.EnvironmentKindPlugin, plugin.ID),
				LastObservedAt:  lastObservedAt,
				Details:         details,
			})
		}
	}

	return environment, nil
}

func (c *AppServerClient) SetSkillEnabled(nameOrPath string, enabled bool) error {
	payload := map[string]any{
		"enabled": enabled,
	}
	if isPathLikeResourceID(nameOrPath) {
		payload["path"] = nameOrPath
	} else {
		payload["name"] = nameOrPath
	}

	var response skillsConfigWriteResponse
	return c.runner.Call("skills/config/write", payload, &response)
}

func (c *AppServerClient) UpsertMCPServer(serverID string, config map[string]any) error {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return fmt.Errorf("serverID is required")
	}

	if err := c.writeConfigValue("mcp_servers."+serverID, cloneAnyMap(config), "replace"); err != nil {
		return err
	}

	return c.reloadMCPServers()
}

func (c *AppServerClient) RemoveMCPServer(serverID string) error {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return fmt.Errorf("serverID is required")
	}

	config, err := c.readConfig()
	if err != nil {
		return err
	}
	servers := nestedConfigMap(config, "mcp_servers")
	if _, ok := servers[serverID]; !ok {
		return fmt.Errorf("mcp server %q not found", serverID)
	}
	delete(servers, serverID)

	if err := c.writeConfigValue("mcp_servers", servers, "replace"); err != nil {
		return err
	}

	return c.reloadMCPServers()
}

func (c *AppServerClient) SetMCPServerEnabled(serverID string, enabled bool) error {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return fmt.Errorf("serverID is required")
	}

	config, err := c.readConfig()
	if err != nil {
		return err
	}
	servers := nestedConfigMap(config, "mcp_servers")
	serverConfig, ok := servers[serverID]
	if !ok {
		return fmt.Errorf("mcp server %q not found", serverID)
	}
	updated := cloneAnyMap(serverConfig)
	updated["enabled"] = enabled

	if err := c.writeConfigValue("mcp_servers."+serverID, updated, "replace"); err != nil {
		return err
	}

	return c.reloadMCPServers()
}

func (c *AppServerClient) InstallPlugin(params agenttypes.InstallPluginParams) error {
	pluginID := strings.TrimSpace(params.PluginID)
	marketplacePath := strings.TrimSpace(params.MarketplacePath)
	pluginName := strings.TrimSpace(params.PluginName)
	if marketplacePath == "" || pluginName == "" {
		return fmt.Errorf("marketplacePath and pluginName are required")
	}

	var response pluginInstallResponse
	if err := c.runner.Call("plugin/install", map[string]any{
		"marketplacePath": marketplacePath,
		"pluginName":      pluginName,
	}, &response); err != nil {
		return err
	}

	c.markRestartRequired(domain.EnvironmentKindPlugin, firstNonEmpty(pluginID, pluginName))
	return nil
}

func (c *AppServerClient) SetPluginEnabled(pluginID string, enabled bool) error {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return fmt.Errorf("pluginID is required")
	}

	config, err := c.readConfig()
	if err != nil {
		return err
	}
	plugins := nestedConfigMap(config, "plugins")
	pluginConfig := cloneAnyMap(nestedConfigMap(plugins, pluginID))
	pluginConfig["enabled"] = enabled

	if err := c.writeConfigValue("plugins."+pluginID, pluginConfig, "replace"); err != nil {
		return err
	}

	c.markRestartRequired(domain.EnvironmentKindPlugin, pluginID)
	return nil
}

func (c *AppServerClient) UninstallPlugin(pluginID string) error {
	var response map[string]any
	if err := c.runner.Call("plugin/uninstall", map[string]any{
		"pluginId": pluginID,
	}, &response); err != nil {
		return err
	}

	c.markRestartRequired(domain.EnvironmentKindPlugin, pluginID)
	return nil
}

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"

func enabledStatus(enabled bool) domain.EnvironmentResourceStatus {
	if enabled {
		return domain.EnvironmentResourceStatusEnabled
	}
	return domain.EnvironmentResourceStatusDisabled
}

func pluginStatus(plugin pluginSummary) domain.EnvironmentResourceStatus {
	if !plugin.Installed {
		return domain.EnvironmentResourceStatusUnknown
	}
	return enabledStatus(plugin.Enabled)
}

func mcpStatus(server mcpServerStatusRecord) domain.EnvironmentResourceStatus {
	switch strings.ToLower(strings.TrimSpace(server.Status)) {
	case "auth_required", "needs_auth":
		return domain.EnvironmentResourceStatusAuthRequired
	case "error", "failed":
		return domain.EnvironmentResourceStatusError
	case "disabled":
		return domain.EnvironmentResourceStatusDisabled
	case "enabled", "connected", "running":
		return domain.EnvironmentResourceStatusEnabled
	}

	if server.NeedsAuth {
		return domain.EnvironmentResourceStatusAuthRequired
	}
	if strings.TrimSpace(server.Error) != "" {
		return domain.EnvironmentResourceStatusError
	}
	return enabledStatus(server.Enabled)
}

func isPathLikeResourceID(resourceID string) bool {
	trimmed := strings.TrimSpace(resourceID)
	if trimmed == "" {
		return false
	}
	if strings.ContainsAny(trimmed, `/\`) {
		return true
	}
	return len(trimmed) > 1 && trimmed[1] == ':'
}

func (c *AppServerClient) readConfig() (map[string]any, error) {
	var response configReadResponse
	if err := c.runner.Call("config/read", map[string]any{}, &response); err != nil {
		return nil, err
	}
	if response.Config == nil {
		return map[string]any{}, nil
	}
	return cloneAnyMap(response.Config), nil
}

func (c *AppServerClient) writeConfigValue(keyPath string, value any, mergeStrategy string) error {
	var response configWriteResponse
	return c.runner.Call("config/value/write", map[string]any{
		"keyPath":       keyPath,
		"value":         value,
		"mergeStrategy": mergeStrategy,
	}, &response)
}

func (c *AppServerClient) reloadMCPServers() error {
	var response map[string]any
	return c.runner.Call("config/mcpServer/reload", map[string]any{}, &response)
}

func (c *AppServerClient) readPluginDetail(marketplacePath string, pluginName string) (pluginDetail, error) {
	var response pluginReadResponse
	if err := c.runner.Call("plugin/read", map[string]any{
		"marketplacePath": marketplacePath,
		"pluginName":      pluginName,
	}, &response); err != nil {
		return pluginDetail{}, err
	}
	return response.Plugin, nil
}

func (c *AppServerClient) markRestartRequired(kind domain.EnvironmentKind, resourceID string) {
	if kind == "" || strings.TrimSpace(resourceID) == "" {
		return
	}

	c.restartMu.Lock()
	c.restartRequired[string(kind)+":"+resourceID] = true
	c.restartMu.Unlock()
}

func (c *AppServerClient) isRestartRequired(kind domain.EnvironmentKind, resourceID string) bool {
	if kind == "" || strings.TrimSpace(resourceID) == "" {
		return false
	}

	c.restartMu.RLock()
	defer c.restartMu.RUnlock()
	return c.restartRequired[string(kind)+":"+resourceID]
}

func buildMCPDetails(server mcpServerStatusRecord, config map[string]any) map[string]any {
	details := cloneAnyMap(config)
	if len(details) > 0 {
		details["config"] = cloneAnyMap(config)
	}
	if strings.TrimSpace(server.Status) != "" {
		details["status"] = server.Status
	}
	if strings.TrimSpace(server.Error) != "" {
		details["error"] = server.Error
	}
	details["enabled"] = server.Enabled
	details["needsAuth"] = server.NeedsAuth
	return details
}

func buildPluginDetails(detail pluginDetail) map[string]any {
	details := map[string]any{
		"marketplaceName":   detail.MarketplaceName,
		"marketplacePath":   detail.MarketplacePath,
		"description":       detail.Description,
		"bundledSkills":     summarizeSkillNames(detail.Skills),
		"bundledApps":       summarizeAppNames(detail.Apps),
		"bundledMcpServers": append([]string(nil), detail.MCPServers...),
	}
	if strings.TrimSpace(detail.Summary.InstallPolicy) != "" {
		details["installPolicy"] = detail.Summary.InstallPolicy
	}
	if strings.TrimSpace(detail.Summary.AuthPolicy) != "" {
		details["authPolicy"] = detail.Summary.AuthPolicy
	}
	return details
}

func summarizeSkillNames(skills []skillSummary) []string {
	items := make([]string, 0, len(skills))
	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}
		items = append(items, name)
	}
	return items
}

func summarizeAppNames(apps []appSummary) []string {
	items := make([]string, 0, len(apps))
	for _, app := range apps {
		name := strings.TrimSpace(app.Name)
		if name == "" {
			name = strings.TrimSpace(app.ID)
		}
		if name == "" {
			continue
		}
		items = append(items, name)
	}
	return items
}

func nestedConfigMap(root map[string]any, key string) map[string]any {
	if root == nil {
		return map[string]any{}
	}
	value, ok := root[key]
	if !ok {
		return map[string]any{}
	}
	return cloneAnyMap(value)
}

func cloneAnyMap(value any) map[string]any {
	source, ok := value.(map[string]any)
	if !ok || len(source) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(source))
	for key, item := range source {
		cloned[key] = cloneJSONValue(item)
	}
	return cloned
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		cloned := make([]any, 0, len(typed))
		for _, item := range typed {
			cloned = append(cloned, cloneJSONValue(item))
		}
		return cloned
	default:
		return typed
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
