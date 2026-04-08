package gateway

import "time"

type Sender func([]byte) error

type Session struct {
	machineID string
	send      Sender
	now       func() time.Time
}

func NewSession(machineID string, send Sender, now func() time.Time) *Session {
	return &Session{
		machineID: machineID,
		send:      send,
		now:       now,
	}
}

func (s *Session) Register() error {
	return s.send([]byte(`{"category":"system","name":"client.register","machineId":"` + s.machineID + `"}`))
}

func (s *Session) Heartbeat() error {
	return s.send([]byte(`{"category":"system","name":"client.heartbeat","machineId":"` + s.machineID + `"}`))
}
