package websocket

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestClientHubAcceptsRegisterMessage(t *testing.T) {
	hub := NewClientHub()
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	if err := conn.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.register","machineId":"machine-01","timestamp":"2026-04-07T10:00:00Z","version":"v1","payload":{}}`)); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if hub.Count() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if hub.Count() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.Count())
	}

	time.Sleep(20 * time.Millisecond)
	if err := conn.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.heartbeat","machineId":"machine-01","timestamp":"2026-04-07T10:00:05Z","version":"v1","payload":{}}`)); err != nil {
		t.Fatalf("expected second system frame write to succeed, got %v", err)
	}
}
