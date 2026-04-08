package codex

import (
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

func (c *AppServerClient) SteerTurn(params agenttypes.SteerTurnParams) (agenttypes.SteerTurnResult, error) {
	var out map[string]any
	if err := c.runner.Call("turn/steer", map[string]any{
		"threadId":       params.ThreadID,
		"expectedTurnId": params.TurnID,
		"input": []map[string]any{
			{
				"type":          "text",
				"text":          params.Input,
				"text_elements": []any{},
			},
		},
	}, &out); err != nil {
		return agenttypes.SteerTurnResult{}, err
	}

	return agenttypes.SteerTurnResult{
		TurnID:   params.TurnID,
		ThreadID: params.ThreadID,
	}, nil
}

func (c *AppServerClient) InterruptTurn(params agenttypes.InterruptTurnParams) (domain.Turn, error) {
	var out map[string]any
	if err := c.runner.Call("turn/interrupt", map[string]any{
		"threadId": params.ThreadID,
		"turnId":   params.TurnID,
	}, &out); err != nil {
		return domain.Turn{}, err
	}

	return domain.Turn{
		TurnID:   params.TurnID,
		ThreadID: params.ThreadID,
		Status:   domain.TurnStatusInterrupted,
	}, nil
}
