package codex

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ApprovalRequest struct {
	RequestID string
	ThreadID  string
	TurnID    string
	ItemID    string
	Kind      string
	Reason    string
	Command   string
}

type pendingApprovalRequest struct {
	id      json.RawMessage
	request ApprovalRequest
}

func (c *AppServerClient) RespondApproval(requestID string, decision string) error {
	if !isSupportedApprovalDecision(decision) {
		return fmt.Errorf("unsupported approval decision %q", decision)
	}

	responder, ok := c.runner.(serverRequestRunner)
	if !ok {
		return fmt.Errorf("runner does not support server request responses")
	}

	c.approvalMu.RLock()
	pending, ok := c.pendingApprovals[requestID]
	c.approvalMu.RUnlock()
	if !ok {
		return fmt.Errorf("approval request %q not found", requestID)
	}

	if err := responder.Respond(pending.id, map[string]any{"decision": decision}, nil); err != nil {
		return err
	}

	c.deletePendingApproval(requestID)
	return nil
}

func approvalKindFromMethod(method string) (string, bool) {
	switch method {
	case "item/commandExecution/requestApproval":
		return "command", true
	case "item/fileChange/requestApproval":
		return "file_change", true
	case "item/permissions/requestApproval":
		return "permissions", false
	default:
		return "unknown", false
	}
}

func isSupportedApprovalDecision(decision string) bool {
	switch strings.TrimSpace(decision) {
	case "accept", "decline", "cancel":
		return true
	default:
		return false
	}
}
