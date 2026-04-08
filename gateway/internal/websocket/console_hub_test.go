package websocket

import (
	"context"
	"testing"
	"time"

	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
	"net/http/httptest"

	cws "github.com/coder/websocket"
)

func TestConsoleHubBroadcastsEventsToConnectedClients(t *testing.T) {
	hub := NewConsoleHub()
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn, _, err := cws.Dial(context.Background(), "ws"+server.URL[4:]+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(cws.StatusNormalClosure, "done")

	done := make(chan protocol.Envelope, 1)
	go func() {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			t.Errorf("read broadcast failed: %v", err)
			return
		}

		var envelope protocol.Envelope
		if err := transport.Decode(data, &envelope); err != nil {
			t.Errorf("decode broadcast failed: %v", err)
			return
		}

		done <- envelope
	}()

	if err := hub.Broadcast(protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "turn.delta",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T14:00:00Z",
		Payload:   []byte(`{"threadId":"thread-01","turnId":"turn-01","sequence":1,"delta":"hello"}`),
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case envelope := <-done:
		if envelope.Name != "turn.delta" {
			t.Fatalf("name = %q", envelope.Name)
		}
		if string(envelope.Payload) != `{"threadId":"thread-01","turnId":"turn-01","sequence":1,"delta":"hello"}` {
			t.Fatalf("payload = %s", string(envelope.Payload))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}
