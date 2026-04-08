package websocket

import (
	"context"
	"errors"
	"net/url"
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

func TestConsoleHubFiltersEventsBySubscribedThreadID(t *testing.T) {
	hub := NewConsoleHub()
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	thread1URL := "ws" + server.URL[4:] + "/ws?threadId=" + url.QueryEscape("thread-01")
	thread2URL := "ws" + server.URL[4:] + "/ws?threadId=" + url.QueryEscape("thread-02")

	thread1Conn, _, err := cws.Dial(context.Background(), thread1URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer thread1Conn.Close(cws.StatusNormalClosure, "done")

	thread2Conn, _, err := cws.Dial(context.Background(), thread2URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer thread2Conn.Close(cws.StatusNormalClosure, "done")

	thread1Received := make(chan protocol.Envelope, 1)

	go readConsoleEnvelope(t, thread1Conn, thread1Received)

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
	case envelope := <-thread1Received:
		if envelope.Name != "turn.delta" {
			t.Fatalf("name = %q", envelope.Name)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for matching thread event")
	}

	readCtx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_, data, err := thread2Conn.Read(readCtx)
	if err == nil {
		var envelope protocol.Envelope
		if err := transport.Decode(data, &envelope); err != nil {
			t.Fatalf("decode broadcast failed: %v", err)
		}
		t.Fatalf("unexpected event delivered to non-matching subscriber: %+v", envelope)
	}
	if !isTimeoutError(err) {
		t.Fatalf("expected timeout when reading non-matching event, got %v", err)
	}
}

func readConsoleEnvelope(t *testing.T, conn *cws.Conn, out chan<- protocol.Envelope) {
	t.Helper()

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

	out <- envelope
}

func isTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || cws.CloseStatus(err) == -1
}
