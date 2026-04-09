package codex

import (
	"strings"

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

	lastObservedAt := c.now().UTC().Format(timeLayoutRFC3339)
	environment := make([]domain.EnvironmentResource, 0, len(skills.Data)+len(mcps.Data)+len(plugins.Marketplaces))
	for _, entry := range skills.Data {
		for _, skill := range entry.Skills {
			environment = append(environment, domain.EnvironmentResource{
				ResourceID:      skill.Name,
				Kind:            domain.EnvironmentKindSkill,
				DisplayName:     skill.Name,
				Status:          enabledStatus(skill.Enabled),
				RestartRequired: false,
				LastObservedAt:  lastObservedAt,
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
		environment = append(environment, domain.EnvironmentResource{
			ResourceID:      resourceID,
			Kind:            domain.EnvironmentKindMCP,
			DisplayName:     displayName,
			Status:          mcpStatus(mcp),
			RestartRequired: false,
			LastObservedAt:  lastObservedAt,
		})
	}
	for _, marketplace := range plugins.Marketplaces {
		for _, plugin := range marketplace.Plugins {
			environment = append(environment, domain.EnvironmentResource{
				ResourceID:      plugin.ID,
				Kind:            domain.EnvironmentKindPlugin,
				DisplayName:     plugin.Name,
				Status:          pluginStatus(plugin),
				RestartRequired: false,
				LastObservedAt:  lastObservedAt,
			})
		}
	}

	return environment, nil
}

func (c *AppServerClient) SetSkillEnabled(nameOrPath string, enabled bool) error {
	var response skillsConfigWriteResponse
	return c.runner.Call("skills/config/write", map[string]any{
		"nameOrPath": nameOrPath,
		"enabled":    enabled,
	}, &response)
}

func (c *AppServerClient) UninstallPlugin(pluginID string) error {
	var response map[string]any
	return c.runner.Call("plugin/uninstall", map[string]any{
		"pluginId": pluginID,
	}, &response)
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
