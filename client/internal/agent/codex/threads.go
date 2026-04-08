package codex

import "code-agent-gateway/common/domain"

func (c *AppServerClient) ReadThread(threadID string) (domain.Thread, error) {
	var response threadReadResponse
	if err := c.runner.Call("thread/read", map[string]any{
		"threadId":     threadID,
		"includeTurns": false,
	}, &response); err != nil {
		return domain.Thread{}, err
	}
	return response.Thread.toDomain(), nil
}

func (c *AppServerClient) ResumeThread(threadID string) (domain.Thread, error) {
	var response threadResumeResponse
	if err := c.runner.Call("thread/resume", map[string]any{
		"threadId":               threadID,
		"persistExtendedHistory": false,
	}, &response); err != nil {
		return domain.Thread{}, err
	}
	return response.Thread.toDomain(), nil
}

func (c *AppServerClient) ArchiveThread(threadID string) error {
	var out map[string]any
	return c.runner.Call("thread/archive", map[string]any{"threadId": threadID}, &out)
}
