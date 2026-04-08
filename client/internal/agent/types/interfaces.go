package types

import "code-agent-gateway/common/domain"

type CreateThreadParams struct {
	Title string
}

type StartTurnParams struct {
	ThreadID string
	Input    string
}

type TurnDelta struct {
	Sequence int
	Delta    string
}

type StartTurnResult struct {
	TurnID   string
	ThreadID string
	Deltas   []TurnDelta
}

type Runtime interface {
	ListThreads() ([]domain.Thread, error)
	ListEnvironment() ([]domain.EnvironmentResource, error)
	CreateThread(params CreateThreadParams) (domain.Thread, error)
	StartTurn(params StartTurnParams) (StartTurnResult, error)
}
