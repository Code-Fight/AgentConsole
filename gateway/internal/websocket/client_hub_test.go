package websocket

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
	"code-agent-gateway/gateway/internal/registry"
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
	hub := NewClientHubWithStores(reg, idx)
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
