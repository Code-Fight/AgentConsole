package domain

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type ThreadStatus string
type TurnStatus string
type ApprovalStatus string

const (
	ThreadStatusNotLoaded ThreadStatus = "notLoaded"
	ThreadStatusIdle      ThreadStatus = "idle"
	ThreadStatusActive    ThreadStatus = "active"
	ThreadStatusUnknown   ThreadStatus = "unknown"
	ThreadStatusError     ThreadStatus = "systemError"

	TurnStatusCompleted   TurnStatus = "completed"
	TurnStatusInterrupted TurnStatus = "interrupted"
	TurnStatusFailed      TurnStatus = "failed"

	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

type Thread struct {
	ThreadID  string       `json:"threadId"`
	MachineID string       `json:"machineId"`
	AgentID   string       `json:"agentId,omitempty"`
	Status    ThreadStatus `json:"status"`
	Title     string       `json:"title"`
}

type ThreadRoute struct {
	MachineID string `json:"machineId"`
	AgentID   string `json:"agentId"`
}

const publicThreadIDPrefix = "th"

func PublicThreadID(agentID string, runtimeThreadID string) string {
	if strings.TrimSpace(agentID) == "" || strings.TrimSpace(runtimeThreadID) == "" {
		return strings.TrimSpace(runtimeThreadID)
	}

	return publicThreadIDPrefix + "." +
		base64.RawURLEncoding.EncodeToString([]byte(agentID)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(runtimeThreadID))
}

func DecodePublicThreadID(publicThreadID string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(publicThreadID), ".")
	if len(parts) != 3 || parts[0] != publicThreadIDPrefix {
		return "", "", false
	}

	decodedAgentID, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || strings.TrimSpace(string(decodedAgentID)) == "" {
		return "", "", false
	}
	decodedThreadID, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || strings.TrimSpace(string(decodedThreadID)) == "" {
		return "", "", false
	}

	return string(decodedAgentID), string(decodedThreadID), true
}

func ResolveRuntimeThread(agentID string, threadID string) (string, string, error) {
	if decodedAgentID, decodedThreadID, ok := DecodePublicThreadID(threadID); ok {
		if strings.TrimSpace(agentID) != "" && decodedAgentID != agentID {
			return "", "", fmt.Errorf("threadId agent mismatch")
		}
		return decodedAgentID, decodedThreadID, nil
	}

	if strings.TrimSpace(agentID) == "" {
		return "", "", fmt.Errorf("agentId is required")
	}
	if strings.TrimSpace(threadID) == "" {
		return "", "", fmt.Errorf("threadId is required")
	}

	return agentID, threadID, nil
}

type Turn struct {
	TurnID   string     `json:"turnId"`
	ThreadID string     `json:"threadId"`
	Status   TurnStatus `json:"status"`
}

type ApprovalRequest struct {
	RequestID string         `json:"requestId"`
	ThreadID  string         `json:"threadId"`
	TurnID    string         `json:"turnId"`
	ItemID    string         `json:"itemId"`
	Kind      string         `json:"kind"`
	Status    ApprovalStatus `json:"status"`
}
