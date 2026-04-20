package codex

import (
	"strings"

	"code-agent-gateway/common/domain"
)

func (c *AppServerClient) ReadThread(threadID string) (domain.Thread, error) {
	var response threadReadResponse
	if err := c.runner.Call("thread/read", map[string]any{
		"threadId":     threadID,
		"includeTurns": true,
	}, &response); err != nil {
		return domain.Thread{}, err
	}
	thread := response.Thread.toDomain()
	if strings.TrimSpace(thread.Title) == "" {
		for _, cached := range c.cachedThreads() {
			if cached.ThreadID == thread.ThreadID {
				thread = mergeRememberedThread(cached, thread)
				break
			}
		}
	}
	c.rememberThread(thread)
	return thread, nil
}

func (c *AppServerClient) ResumeThread(threadID string) (domain.Thread, error) {
	var response threadResumeResponse
	if err := c.runner.Call("thread/resume", map[string]any{
		"threadId":               threadID,
		"persistExtendedHistory": false,
	}, &response); err != nil {
		return domain.Thread{}, err
	}
	thread := response.Thread.toDomain()
	if strings.TrimSpace(thread.Title) == "" {
		for _, cached := range c.cachedThreads() {
			if cached.ThreadID == thread.ThreadID {
				thread = mergeRememberedThread(cached, thread)
				break
			}
		}
	}
	c.rememberThread(thread)
	c.seedThreadRuntimeState(thread.ThreadID, domain.ThreadRuntimePreferences{
		Model:          strings.TrimSpace(response.Model),
		ApprovalPolicy: normalizeApprovalPolicy(response.ApprovalPolicy),
		SandboxMode:    normalizeSandboxMode(response.Sandbox),
	})
	return thread, nil
}

func (c *AppServerClient) ArchiveThread(threadID string) error {
	var out map[string]any
	if err := c.runner.Call("thread/archive", map[string]any{"threadId": threadID}, &out); err != nil {
		return err
	}
	c.forgetThread(threadID)
	return nil
}
