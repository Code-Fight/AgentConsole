package codex

func (c *AppServerClient) StartTurn(threadID string, prompt string) (map[string]any, error) {
	var out map[string]any
	if err := c.runner.Call("turn/start", map[string]any{"threadId": threadID, "input": prompt}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
