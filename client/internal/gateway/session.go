package gateway

import (
	"encoding/json"
	"sync"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
)

type Sender func([]byte) error

type Session struct {
	machineID string
	send      Sender
	now       func() time.Time
	sendMu    sync.Mutex
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
	return s.sendEnvelope(protocol.CategorySystem, "client.register", "", struct{}{})
}

func (s *Session) Heartbeat() error {
	return s.sendEnvelope(protocol.CategorySystem, "client.heartbeat", "", struct{}{})
}

func (s *Session) MachineSnapshot(machine domain.Machine) error {
	if machine.ID == "" {
		machine.ID = s.machineID
	}
	if machine.Status == "" {
		machine.Status = domain.MachineStatusOnline
	}

	return s.sendEnvelope(protocol.CategorySnapshot, "machine.snapshot", "", protocol.MachineSnapshotPayload{
		Machine: machine,
	})
}

func (s *Session) ThreadSnapshot(threads []domain.Thread) error {
	normalized := make([]domain.Thread, 0, len(threads))
	for _, item := range threads {
		thread := item
		if thread.MachineID == "" {
			thread.MachineID = s.machineID
		}
		normalized = append(normalized, thread)
	}

	return s.sendEnvelope(protocol.CategorySnapshot, "thread.snapshot", "", protocol.ThreadSnapshotPayload{
		Threads: normalized,
	})
}

func (s *Session) EnvironmentSnapshot(environment []domain.EnvironmentResource) error {
	normalized := make([]domain.EnvironmentResource, 0, len(environment))
	for _, item := range environment {
		resource := item
		if resource.MachineID == "" {
			resource.MachineID = s.machineID
		}
		normalized = append(normalized, resource)
	}

	return s.sendEnvelope(protocol.CategorySnapshot, "environment.snapshot", "", protocol.EnvironmentSnapshotPayload{
		Environment: normalized,
	})
}

func (s *Session) CommandCompleted(requestID string, commandName string, result any) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return s.sendEnvelope(protocol.CategoryEvent, "command.completed", requestID, protocol.CommandCompletedPayload{
		CommandName: commandName,
		Result:      resultJSON,
	})
}

func (s *Session) CommandRejected(requestID string, commandName string, reason string, threadID string) error {
	return s.sendEnvelope(protocol.CategoryEvent, "command.rejected", requestID, protocol.CommandRejectedPayload{
		CommandName: commandName,
		Reason:      reason,
		ThreadID:    threadID,
	})
}

func (s *Session) TurnDelta(requestID string, payload protocol.TurnDeltaPayload) error {
	return s.sendEnvelope(protocol.CategoryEvent, "turn.delta", requestID, payload)
}

func (s *Session) TurnCompleted(requestID string, payload protocol.TurnCompletedPayload) error {
	return s.sendEnvelope(protocol.CategoryEvent, "turn.completed", requestID, payload)
}

func (s *Session) TurnFailed(requestID string, payload protocol.TurnCompletedPayload) error {
	return s.sendEnvelope(protocol.CategoryEvent, "turn.failed", requestID, payload)
}

func (s *Session) sendEnvelope(category protocol.Category, name string, requestID string, payload any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	frame := protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  category,
		Name:      name,
		RequestID: requestID,
		MachineID: s.machineID,
		Timestamp: s.now().Format(time.RFC3339),
		Payload:   payloadJSON,
	}

	encoded, err := transport.Encode(frame)
	if err != nil {
		return err
	}

	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	return s.send(encoded)
}
