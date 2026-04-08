package gateway

import (
	"testing"
	"time"
)

func TestSessionFrames(t *testing.T) {
	var sent []string

	session := NewSession("machine-01", func(msg []byte) error {
		sent = append(sent, string(msg))
		return nil
	}, func() time.Time {
		return time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	})

	if err := session.Register(); err != nil {
		t.Fatal(err)
	}
	if err := session.Heartbeat(); err != nil {
		t.Fatal(err)
	}

	if len(sent) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(sent))
	}
}
