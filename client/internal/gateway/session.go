package gateway

import (
	"encoding/json"
	"time"

	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
)

type Sender func([]byte) error

type Session struct {
	machineID string
	send      Sender
	now       func() time.Time
}

func NewSession(machineID string, send Sender, now func() time.Time) *Session {
	if now == nil {
		now = time.Now
	}

	return &Session{
		machineID: machineID,
		send:      send,
		now:       now,
	}
}

func (s *Session) Register() error {
	return s.sendSystem(protocol.CategorySystem, "client.register")
}

func (s *Session) Heartbeat() error {
	return s.sendSystem(protocol.CategorySystem, "client.heartbeat")
}

func (s *Session) sendSystem(category protocol.Category, name string) error {
	frame := protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  category,
		Name:      name,
		MachineID: s.machineID,
		Timestamp: s.now().Format(time.RFC3339),
		Payload:   json.RawMessage(`{}`),
	}

	encoded, err := transport.Encode(frame)
	if err != nil {
		return err
	}

	return s.send(encoded)
}
