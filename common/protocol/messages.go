package protocol

import "encoding/json"

type Category string

const (
	CategorySystem   Category = "system"
	CategoryCommand  Category = "command"
	CategoryEvent    Category = "event"
	CategorySnapshot Category = "snapshot"
)

type Envelope struct {
	Version   string          `json:"version"`
	Category  Category        `json:"category"`
	Name      string          `json:"name"`
	RequestID string          `json:"requestId,omitempty"`
	MachineID string          `json:"machineId,omitempty"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}
