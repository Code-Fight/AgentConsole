package codex

func (c *AppServerClient) SteerTurn(threadID, turnID, input string) error {
	var out map[string]any
	return c.runner.Call("turn/steer", map[string]any{
		"threadId":       threadID,
		"expectedTurnId": turnID,
		"input": []map[string]any{
			{
				"type":          "text",
				"text":          input,
				"text_elements": []any{},
			},
		},
	}, &out)
}

func (c *AppServerClient) InterruptTurn(threadID, turnID string) error {
	var out map[string]any
	return c.runner.Call("turn/interrupt", map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
	}, &out)
}
