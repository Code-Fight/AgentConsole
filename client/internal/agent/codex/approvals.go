package codex

func (c *AppServerClient) RespondApproval(requestID string, decision string) error {
	var out map[string]any
	return c.runner.Call("approval/respond", map[string]any{"requestId": requestID, "decision": decision}, &out)
}
