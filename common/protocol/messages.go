package protocol

import (
	"encoding/json"
	"errors"
	"strings"

	"code-agent-gateway/common/domain"
)

type Category string

const (
	CategorySystem   Category = "system"
	CategoryCommand  Category = "command"
	CategoryEvent    Category = "event"
	CategorySnapshot Category = "snapshot"
)

type MachineSnapshotPayload struct {
	Machine domain.Machine `json:"machine"`
}

type ThreadSnapshotPayload struct {
	Threads []domain.Thread `json:"threads"`
}

type EnvironmentSnapshotPayload struct {
	Environment []domain.EnvironmentResource `json:"environment"`
}

type Envelope struct {
	Version   string          `json:"version"`
	Category  Category        `json:"category"`
	Name      string          `json:"name"`
	RequestID string          `json:"requestId,omitempty"`
	MachineID string          `json:"machineId,omitempty"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func (e Envelope) Validate() error {
	if e.Category == CategoryCommand && strings.TrimSpace(e.RequestID) == "" {
		return errors.New("requestId is required for command envelopes")
	}

	return nil
}

func (e Envelope) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}

	type envelopeAlias Envelope
	return json.Marshal(envelopeAlias(e))
}
