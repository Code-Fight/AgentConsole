package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/routing"
	"code-agent-gateway/gateway/internal/runtimeindex"
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

func TestClientHubIngestsSnapshotsIntoRegistryAndRuntimeIndex(t *testing.T) {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	router := routing.NewRouter()
	hub := NewClientHubWithStores(reg, idx, router)
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "machine.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:01Z",
		Payload:   []byte(`{"machine":{"id":"machine-01","name":"Dev Mac","status":"online"}}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "thread.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:02Z",
		Payload: []byte(`{
			"threads":[
				{"threadId":"thread-01","machineId":"machine-01","status":"idle","title":"One"}
			]
		}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "environment.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:03Z",
		Payload: []byte(`{
			"environment":[
				{"resourceId":"skill-01","machineId":"machine-01","kind":"skill","displayName":"Skill A","status":"enabled","restartRequired":false,"lastObservedAt":"2026-04-08T10:00:03Z"}
			]
		}`),
	}); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		machines := reg.List()
		if len(machines) != 1 {
			return false
		}
		return machines[0].ID == "machine-01" && machines[0].Status == domain.MachineStatusOnline
	})

	waitForCondition(t, 2*time.Second, func() bool {
		threads := idx.Threads()
		if len(threads) != 1 {
			return false
		}
		if threads[0].ThreadID != "thread-01" || threads[0].MachineID != "machine-01" {
			return false
		}

		environment := idx.Environment(domain.EnvironmentKindSkill)
		if len(environment) != 1 {
			return false
		}

		return environment[0].ResourceID == "skill-01" && environment[0].MachineID == "machine-01"
	})

	machineID, ok := router.ResolveThread("thread-01")
	if !ok || machineID != "machine-01" {
		t.Fatalf("router resolved (%q, %v)", machineID, ok)
	}
}

func TestClientHubSendCommandRoundTripsCompletedResponse(t *testing.T) {
	hub := NewClientHub()
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, hub, 1, 1*time.Second)

	done := make(chan struct{})
	go func() {
		defer close(done)

		_, data, err := conn.Read(context.Background())
		if err != nil {
			t.Errorf("read command failed: %v", err)
			return
		}

		var envelope protocol.Envelope
		if err := transport.Decode(data, &envelope); err != nil {
			t.Errorf("decode command failed: %v", err)
			return
		}

		if envelope.Category != protocol.CategoryCommand || envelope.Name != "thread.create" {
			t.Errorf("unexpected command envelope: %+v", envelope)
			return
		}

		payload, err := json.Marshal(protocol.CommandCompletedPayload{
			CommandName: "thread.create",
			Result: mustMarshalJSON(t, protocol.ThreadCreateCommandResult{
				Thread: domain.Thread{
					ThreadID:  "thread-01",
					MachineID: "machine-01",
					Status:    domain.ThreadStatusIdle,
					Title:     "One",
				},
			}),
		})
		if err != nil {
			t.Errorf("marshal response failed: %v", err)
			return
		}

		if err := writeEnvelope(t, conn, protocol.Envelope{
			Version:   version.CurrentProtocolVersion,
			Category:  protocol.CategoryEvent,
			Name:      "command.completed",
			RequestID: envelope.RequestID,
			MachineID: "machine-01",
			Timestamp: "2026-04-08T10:00:01Z",
			Payload:   payload,
		}); err != nil {
			t.Errorf("write response failed: %v", err)
		}
	}()

	response, err := hub.SendCommand(context.Background(), "machine-01", "thread.create", protocol.ThreadCreateCommandPayload{Title: "One"})
	if err != nil {
		t.Fatal(err)
	}

	if response.CommandName != "thread.create" {
		t.Fatalf("commandName = %q", response.CommandName)
	}

	var result protocol.ThreadCreateCommandResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode result failed: %v", err)
	}
	if result.Thread.ThreadID != "thread-01" {
		t.Fatalf("unexpected result: %+v", result)
	}

	<-done
}

func TestClientHubFansOutEventEnvelopesToConsoleClients(t *testing.T) {
	consoleHub := NewConsoleHub()
	hub := NewClientHub()
	hub.SetConsoleHub(consoleHub)

	mux := http.NewServeMux()
	mux.Handle("/ws/client", hub.Handler())
	mux.Handle("/ws", consoleHub.Handler())

	server := httptest.NewServer(mux)
	defer server.Close()

	consoleConn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer consoleConn.Close(websocket.StatusNormalClosure, "done")

	clientConn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close(websocket.StatusNormalClosure, "done")

	done := make(chan protocol.Envelope, 1)
	go func() {
		_, data, err := consoleConn.Read(context.Background())
		if err != nil {
			t.Errorf("read console event failed: %v", err)
			return
		}

		var envelope protocol.Envelope
		if err := transport.Decode(data, &envelope); err != nil {
			t.Errorf("decode console event failed: %v", err)
			return
		}

		done <- envelope
	}()

	if err := writeEnvelope(t, clientConn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "turn.delta",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:20Z",
		Payload:   []byte(`{"threadId":"thread-01","turnId":"turn-01","sequence":1,"delta":"hello"}`),
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case envelope := <-done:
		if envelope.Name != "turn.delta" {
			t.Fatalf("name = %q", envelope.Name)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for console event")
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

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition did not become true before timeout")
}

func writeEnvelope(t *testing.T, conn *websocket.Conn, envelope protocol.Envelope) error {
	t.Helper()

	encoded, err := transport.Encode(envelope)
	if err != nil {
		return err
	}

	return conn.Write(context.Background(), websocket.MessageText, encoded)
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	return raw
}
