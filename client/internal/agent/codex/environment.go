package codex

import "code-agent-gateway/common/domain"

func (c *AppServerClient) ListEnvironment() ([]domain.EnvironmentResource, error) {
	var records []EnvironmentRecord
	if err := c.runner.Call("environment/list", map[string]any{}, &records); err != nil {
		return nil, err
	}

	environment := make([]domain.EnvironmentResource, 0, len(records))
	for _, record := range records {
		environment = append(environment, record.toDomain())
	}
	return environment, nil
}
