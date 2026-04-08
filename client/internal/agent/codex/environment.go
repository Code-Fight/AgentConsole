package codex

func (c *AppServerClient) ListEnvironment() ([]map[string]any, error) {
	var out []map[string]any
	if err := c.runner.Call("environment/list", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
