package domain

type ThreadStatus string
type TurnStatus string
type ApprovalStatus string

const (
	ThreadStatusNotLoaded ThreadStatus = "notLoaded"
	ThreadStatusIdle      ThreadStatus = "idle"
	ThreadStatusActive    ThreadStatus = "active"
	ThreadStatusError     ThreadStatus = "systemError"

	TurnStatusCompleted   TurnStatus = "completed"
	TurnStatusInterrupted TurnStatus = "interrupted"
	TurnStatusFailed      TurnStatus = "failed"

	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

type Thread struct {
	ID        string       `json:"id"`
	MachineID string       `json:"machineId"`
	Status    ThreadStatus `json:"status"`
	Title     string       `json:"title"`
}

type Turn struct {
	ID       string     `json:"id"`
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
