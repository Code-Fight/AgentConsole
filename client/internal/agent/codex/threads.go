package codex

func (c *AppServerClient) CreateThread() (map[string]any, error) {
	var out map[string]any
	if err := c.runner.Call("thread/start", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
