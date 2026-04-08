package types

import "code-agent-gateway/common/domain"

type CreateThreadParams struct {
	Title string
}

type StartTurnParams struct {
	ThreadID string
	Input    string
}

type SteerTurnParams struct {
	ThreadID string
	TurnID   string
	Input    string
}

type InterruptTurnParams struct {
	ThreadID string
	TurnID   string
}

type TurnDelta struct {
	Sequence int
	Delta    string
}

type RuntimeTurnEventType string

const (
	RuntimeTurnEventTypeStarted   RuntimeTurnEventType = "turn.started"
	RuntimeTurnEventTypeDelta     RuntimeTurnEventType = "turn.delta"
	RuntimeTurnEventTypeCompleted RuntimeTurnEventType = "turn.completed"
)

type RuntimeTurnEvent struct {
	Type      RuntimeTurnEventType
	RequestID string
	ThreadID  string
	TurnID    string
	Sequence  int
	Delta     string
	Turn      domain.Turn
}

type RuntimeTurnEventSource interface {
	SetTurnEventHandler(func(RuntimeTurnEvent))
}

type StartTurnResult struct {
	TurnID   string
	ThreadID string
	Deltas   []TurnDelta
}

type SteerTurnResult struct {
	TurnID   string
	ThreadID string
	Deltas   []TurnDelta
}

type Runtime interface {
	ListThreads() ([]domain.Thread, error)
	ListEnvironment() ([]domain.EnvironmentResource, error)
	CreateThread(params CreateThreadParams) (domain.Thread, error)
	ReadThread(threadID string) (domain.Thread, error)
	ResumeThread(threadID string) (domain.Thread, error)
	ArchiveThread(threadID string) error
	StartTurn(params StartTurnParams) (StartTurnResult, error)
	SteerTurn(params SteerTurnParams) (SteerTurnResult, error)
	InterruptTurn(params InterruptTurnParams) (domain.Turn, error)
}
