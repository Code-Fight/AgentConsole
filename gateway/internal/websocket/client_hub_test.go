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

	waitForCount(t, hub, 1, 1*time.Second)

	time.Sleep(20 * time.Millisecond)
	if err := conn.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.heartbeat","machineId":"machine-01","timestamp":"2026-04-07T10:00:05Z","version":"v1","payload":{}}`)); err != nil {
		t.Fatalf("expected second system frame write to succeed, got %v", err)
	}
}

func TestClientHubReconnectKeepsLatestConnectionOwner(t *testing.T) {
	hub := NewClientHub()
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn1, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "done")

	if err := conn1.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.register","machineId":"machine-01","timestamp":"2026-04-07T10:00:00Z","version":"v1","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	waitForCount(t, hub, 1, 1*time.Second)

	conn2, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "done")

	if err := conn2.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.register","machineId":"machine-01","timestamp":"2026-04-07T10:00:05Z","version":"v1","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	waitForCount(t, hub, 1, 1*time.Second)

	if err := conn1.Close(websocket.StatusNormalClosure, "old-conn-closed"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	if hub.Count() != 1 {
		t.Fatalf("expected latest client mapping to remain, got %d", hub.Count())
	}

	if err := conn2.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.heartbeat","machineId":"machine-01","timestamp":"2026-04-07T10:00:10Z","version":"v1","payload":{}}`)); err != nil {
		t.Fatalf("expected heartbeat on latest connection to succeed, got %v", err)
	}
}

func waitForCount(t *testing.T, hub *ClientHub, expected int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if hub.Count() == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected %d clients, got %d", expected, hub.Count())
}
