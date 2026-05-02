package websocket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	hub := NewClientHubWithStores(registry.NewStore(), nil)
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	if err := conn.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.register","machineId":"machine-01","timestamp":"2026-04-07T10:00:00Z","version":"v1","payload":{"name":"Workstation Alpha"}}`)); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, hub, 1, 1*time.Second)

	waitForCondition(t, 1*time.Second, func() bool {
		machines := hub.registry.List()
		return len(machines) == 1 && machines[0].Name == "Workstation Alpha"
	})

	time.Sleep(20 * time.Millisecond)
	if err := conn.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.heartbeat","machineId":"machine-01","timestamp":"2026-04-07T10:00:05Z","version":"v1","payload":{}}`)); err != nil {
		t.Fatalf("expected second system frame write to succeed, got %v", err)
	}
}

func TestClientHubReconnectKeepsLatestConnectionOwner(t *testing.T) {
	hub := NewClientHubWithStores(registry.NewStore(), nil)
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn1, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "done")

	if err := conn1.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.register","machineId":"machine-01","timestamp":"2026-04-07T10:00:00Z","version":"v1","payload":{"name":"Workstation Alpha"}}`)); err != nil {
		t.Fatal(err)
	}
	waitForCount(t, hub, 1, 1*time.Second)

	conn2, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "done")

	if err := conn2.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.register","machineId":"machine-01","timestamp":"2026-04-07T10:00:05Z","version":"v1","payload":{"name":"Workstation Beta"}}`)); err != nil {
		t.Fatal(err)
	}
	waitForCount(t, hub, 1, 1*time.Second)

	waitForCondition(t, 1*time.Second, func() bool {
		machines := hub.registry.List()
		return len(machines) == 1 && machines[0].Name == "Workstation Beta"
	})

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

func TestClientHubTracksActiveTurnsForThreadRecovery(t *testing.T) {
	idx := runtimeindex.NewStore()
	hub := NewClientHubWithStores(registry.NewStore(), idx)

	hub.upsertThreadSnapshot("machine-01", domain.Thread{
		ThreadID:  "thread-01",
		MachineID: "machine-01",
		Status:    domain.ThreadStatusIdle,
		Title:     "Investigate flaky test",
	})

	hub.setActiveTurn("machine-01", "thread-01", "turn-01", "2026-04-20T10:00:00Z")

	activeTurnID, ok := hub.ActiveTurnID("thread-01")
	if !ok || activeTurnID != "turn-01" {
		t.Fatalf("expected active turn to be tracked, got %q ok=%v", activeTurnID, ok)
	}

	threads := idx.Threads()
	if len(threads) != 1 || threads[0].Status != domain.ThreadStatusActive {
		t.Fatalf("expected active thread snapshot, got %+v", threads)
	}
	if threads[0].LastActivityAt != "2026-04-20T10:00:00Z" {
		t.Fatalf("expected active turn timestamp, got %+v", threads[0])
	}

	hub.clearActiveTurn("machine-01", "thread-01", "turn-01", "2026-04-20T10:01:00Z")

	if activeTurnID, ok := hub.ActiveTurnID("thread-01"); ok || activeTurnID != "" {
		t.Fatalf("expected active turn to clear, got %q ok=%v", activeTurnID, ok)
	}

	threads = idx.Threads()
	if len(threads) != 1 || threads[0].Status != domain.ThreadStatusIdle {
		t.Fatalf("expected idle thread snapshot after clear, got %+v", threads)
	}
	if threads[0].LastActivityAt != "2026-04-20T10:01:00Z" {
		t.Fatalf("expected clear turn timestamp, got %+v", threads[0])
	}
}

func TestClientHubAppliesTimelineLifecycleAndApprovalState(t *testing.T) {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	hub := NewClientHubWithStores(reg, idx)
	hub.upsertThreadSnapshot("machine-01", domain.Thread{
		ThreadID:  "thread-01",
		MachineID: "machine-01",
		Status:    domain.ThreadStatusIdle,
		Title:     "Timeline",
	})

	started := protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "timeline.event",
		MachineID: "machine-01",
		Timestamp: "2026-04-20T10:00:00Z",
		Payload: mustMarshalJSON(t, protocol.TimelineEventPayload{Event: domain.AgentTimelineEvent{
			SchemaVersion: domain.AgentTimelineSchemaVersion,
			EventID:       "event-started",
			Sequence:      1,
			ThreadID:      "thread-01",
			TurnID:        "turn-01",
			EventType:     domain.AgentTimelineEventTurnStarted,
			Status:        domain.AgentTimelineStatusRunning,
		}}),
	}
	hub.applyTurnLifecycleEvent(started)
	if activeTurnID, ok := hub.ActiveTurnID("thread-01"); !ok || activeTurnID != "turn-01" {
		t.Fatalf("expected timeline turn to become active, got %q ok=%v", activeTurnID, ok)
	}

	approval := normalizeTimelineEnvelope(protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "timeline.event",
		MachineID: "machine-01",
		Timestamp: "2026-04-20T10:00:01Z",
		Payload: mustMarshalJSON(t, protocol.TimelineEventPayload{Event: domain.AgentTimelineEvent{
			SchemaVersion: domain.AgentTimelineSchemaVersion,
			EventID:       "event-approval",
			Sequence:      2,
			ThreadID:      "thread-01",
			TurnID:        "turn-01",
			ItemID:        "command-01",
			EventType:     domain.AgentTimelineEventApprovalRequested,
			ItemType:      domain.AgentTimelineItemCommand,
			Approval: &domain.AgentTimelineApproval{
				RequestID: "approval-01",
				Kind:      "command",
				Title:     "go test ./...",
			},
		}}),
	})
	hub.applyTimelineEvent(approval)
	approvals := reg.PendingApprovalsForThread("thread-01")
	if len(approvals) != 1 || approvals[0].RequestID != expectedApprovalRequestID("machine-01", "approval-01") {
		t.Fatalf("unexpected timeline pending approvals: %+v", approvals)
	}

	completed := protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "timeline.event",
		MachineID: "machine-01",
		Timestamp: "2026-04-20T10:00:02Z",
		Payload: mustMarshalJSON(t, protocol.TimelineEventPayload{Event: domain.AgentTimelineEvent{
			SchemaVersion: domain.AgentTimelineSchemaVersion,
			EventID:       "event-completed",
			Sequence:      3,
			ThreadID:      "thread-01",
			TurnID:        "turn-01",
			EventType:     domain.AgentTimelineEventTurnCompleted,
			Status:        domain.AgentTimelineStatusCompleted,
		}}),
	}
	hub.applyTurnLifecycleEvent(completed)
	if activeTurnID, ok := hub.ActiveTurnID("thread-01"); ok || activeTurnID != "" {
		t.Fatalf("expected timeline turn to clear, got %q ok=%v", activeTurnID, ok)
	}
}

func TestClientHubIgnoresMessagesFromSupersededConnection(t *testing.T) {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	consoleHub := NewConsoleHub()
	hub := NewClientHubWithStores(reg, idx)
	hub.SetConsoleHub(consoleHub)

	mux := http.NewServeMux()
	mux.Handle("/ws/client", hub.Handler())
	mux.Handle("/ws", consoleHub.Handler())

	server := httptest.NewServer(mux)
	defer server.Close()

	consoleConn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws?threadId=thread-01", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer consoleConn.Close(websocket.StatusNormalClosure, "done")

	conn1, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "done")

	conn2, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "done")

	register := func(conn *websocket.Conn, timestamp string) {
		t.Helper()
		if err := writeEnvelope(t, conn, protocol.Envelope{
			Version:   version.CurrentProtocolVersion,
			Category:  protocol.CategorySystem,
			Name:      "client.register",
			MachineID: "machine-01",
			Timestamp: timestamp,
			Payload:   []byte(`{}`),
		}); err != nil {
			t.Fatal(err)
		}
	}

	register(conn1, "2026-04-08T10:00:00Z")
	waitForCount(t, hub, 1, 1*time.Second)
	register(conn2, "2026-04-08T10:00:01Z")
	waitForCount(t, hub, 1, 1*time.Second)

	if err := writeEnvelope(t, conn2, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "thread.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:02Z",
		Payload:   []byte(`{"threads":[{"threadId":"thread-01","title":"fresh","status":"idle"}]}`),
	}); err != nil {
		t.Fatal(err)
	}
	waitForCondition(t, 2*time.Second, func() bool {
		threads := idx.Threads()
		return len(threads) == 1 && threads[0].Title == "fresh"
	})

	if err := writeEnvelope(t, conn1, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "thread.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:03Z",
		Payload:   []byte(`{"threads":[{"threadId":"thread-01","title":"stale","status":"idle"}]}`),
	}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	threads := idx.Threads()
	if len(threads) != 1 || threads[0].Title != "fresh" {
		t.Fatalf("stale snapshot should be ignored, got %+v", threads)
	}

	if err := writeEnvelope(t, conn1, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "turn.delta",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:04Z",
		Payload:   []byte(`{"threadId":"thread-01","turnId":"turn-01","sequence":1,"delta":"stale"}`),
	}); err != nil {
		t.Fatal(err)
	}

	readResult := make(chan error, 1)
	go func() {
		_, _, err := consoleConn.Read(context.Background())
		readResult <- err
	}()

	select {
	case err := <-readResult:
		if err == nil {
			t.Fatal("expected stale event to be ignored")
		}
	case <-time.After(150 * time.Millisecond):
	}

	if err := consoleConn.Close(websocket.StatusNormalClosure, "reconnect-console"); err != nil {
		t.Fatal(err)
	}

	consoleConn, _, err = websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws?threadId=thread-01", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer consoleConn.Close(websocket.StatusNormalClosure, "done")

	if err := writeEnvelope(t, conn2, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "turn.delta",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:05Z",
		Payload:   []byte(`{"threadId":"thread-01","turnId":"turn-01","sequence":2,"delta":"fresh"}`),
	}); err != nil {
		t.Fatal(err)
	}

	_, data, err := consoleConn.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var envelope protocol.Envelope
	if err := transport.Decode(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Name != "turn.delta" {
		t.Fatalf("unexpected forwarded envelope: %+v", envelope)
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

	route, ok := router.ResolveThread("thread-01")
	if !ok || route.MachineID != "machine-01" {
		t.Fatalf("router resolved (%+v, %v)", route, ok)
	}
}

func TestClientHubKeepsConnectionForLargeEnvironmentSnapshot(t *testing.T) {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	hub := NewClientHubWithStores(reg, idx, routing.NewRouter())
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
		Timestamp: "2026-04-09T11:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	waitForCount(t, hub, 1, 1*time.Second)

	largeBlob := strings.Repeat("x", 64*1024)
	payload := mustMarshalJSON(t, map[string]any{
		"environment": []map[string]any{
			{
				"resourceId":      "plugin-01",
				"kind":            "plugin",
				"displayName":     "Plugin One",
				"status":          "enabled",
				"restartRequired": false,
				"lastObservedAt":  "2026-04-09T11:00:01Z",
				"details": map[string]any{
					"description": largeBlob,
				},
			},
		},
	})

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "environment.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T11:00:01Z",
		Payload:   payload,
	}); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		items := idx.Environment(domain.EnvironmentKindPlugin)
		return len(items) == 1 && items[0].ResourceID == "plugin-01"
	})

	time.Sleep(50 * time.Millisecond)
	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.heartbeat",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T11:00:02Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatalf("expected connection to remain open after large snapshot, got %v", err)
	}
}

func TestClientHubEmitsNorthboundUpdateEvents(t *testing.T) {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	consoleHub := NewConsoleHub()
	hub := NewClientHubWithStores(reg, idx, routing.NewRouter())
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

	if err := writeEnvelope(t, clientConn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T11:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(t, clientConn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "thread.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T11:00:01Z",
		Payload:   []byte(`{"threads":[{"threadId":"thread-01","status":"idle","title":"One"}]}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(t, clientConn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "environment.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T11:00:02Z",
		Payload:   []byte(`{"environment":[{"resourceId":"skill-01","kind":"skill","displayName":"Skill A","status":"enabled","restartRequired":false,"lastObservedAt":"2026-04-09T11:00:02Z"}]}`),
	}); err != nil {
		t.Fatal(err)
	}

	expectedNames := []string{"machine.updated", "thread.updated", "resource.changed"}
	for _, expectedName := range expectedNames {
		_, data, err := consoleConn.Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		var envelope protocol.Envelope
		if err := transport.Decode(data, &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Name != expectedName {
			t.Fatalf("expected %q, got %+v", expectedName, envelope)
		}
		if envelope.Category != protocol.CategoryEvent {
			t.Fatalf("category = %q", envelope.Category)
		}
	}
}

func TestClientHubKeepsPendingApprovalRoutingAndRegistryStateAcrossDisconnect(t *testing.T) {
	reg := registry.NewStore()
	hub := NewClientHubWithStores(reg, runtimeindex.NewStore(), routing.NewRouter())
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T10:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "approval.required",
		RequestID: "approval-1",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T10:00:01Z",
		Payload:   []byte(`{"requestId":"approval-1","threadId":"thread-01","kind":"command","command":"go test ./..."}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := conn.Close(websocket.StatusNormalClosure, "offline"); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		_, ok := hub.ResolveApprovalMachine(expectedApprovalRequestID("machine-01", "approval-1"))
		return ok
	})

	machineID, ok := hub.ResolveApprovalMachine(expectedApprovalRequestID("machine-01", "approval-1"))
	if !ok || machineID != "machine-01" {
		t.Fatalf("unexpected approval route: machineID=%q ok=%v", machineID, ok)
	}

	approvals := reg.PendingApprovalsForThread("thread-01")
	if len(approvals) != 1 || approvals[0].RequestID != expectedApprovalRequestID("machine-01", "approval-1") {
		t.Fatalf("unexpected stored approvals: %+v", approvals)
	}
}

func TestClientHubDisconnectPreservesUnknownThreadsAndClearsRoutes(t *testing.T) {
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
		Name:      "thread.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:01Z",
		Payload:   []byte(`{"threads":[{"threadId":"thread-01","status":"idle","title":"One"}]}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "environment.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:02Z",
		Payload:   []byte(`{"environment":[{"resourceId":"skill-01","kind":"skill","displayName":"Skill A","status":"enabled","restartRequired":false,"lastObservedAt":"2026-04-08T10:00:02Z"}]}`),
	}); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		return len(idx.Threads()) == 1 && len(idx.Environment(domain.EnvironmentKindSkill)) == 1
	})
	waitForCondition(t, 2*time.Second, func() bool {
		route, ok := router.ResolveThread("thread-01")
		return ok && route.MachineID == "machine-01"
	})

	if err := conn.Close(websocket.StatusNormalClosure, "disconnect"); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		machines := reg.List()
		return len(machines) == 1 && machines[0].ID == "machine-01" && machines[0].Status == domain.MachineStatusOffline
	})
	waitForCondition(t, 2*time.Second, func() bool {
		threads := idx.Threads()
		if len(threads) != 1 {
			return false
		}
		if threads[0].ThreadID != "thread-01" || threads[0].Status != domain.ThreadStatusUnknown {
			return false
		}
		_, ok := router.ResolveThread("thread-01")
		return !ok
	})
}

func TestClientHubTracksApprovalRequestOwnership(t *testing.T) {
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

	if err := writeEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "approval.required",
		RequestID: "approval-1",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:01Z",
		Payload:   []byte(`{"requestId":"approval-1","threadId":"thread-1","turnId":"turn-1","itemId":"item-1","kind":"command","command":"go test ./..."}`),
	}); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		machineID, ok := hub.ResolveApprovalMachine(expectedApprovalRequestID("machine-01", "approval-1"))
		return ok && machineID == "machine-01"
	})
}

func TestClientHubScopesApprovalIDsAcrossMachines(t *testing.T) {
	reg := registry.NewStore()
	hub := NewClientHubWithStores(reg, runtimeindex.NewStore(), routing.NewRouter())
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn1, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "done")

	conn2, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "done")

	if err := writeEnvelope(t, conn1, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeEnvelope(t, conn2, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-02",
		Timestamp: "2026-04-08T10:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(t, conn1, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "approval.required",
		RequestID: "approval-1",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:01Z",
		Payload:   []byte(`{"requestId":"approval-1","threadId":"thread-1","kind":"command","command":"go test ./..."}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeEnvelope(t, conn2, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "approval.required",
		RequestID: "approval-1",
		MachineID: "machine-02",
		Timestamp: "2026-04-08T10:00:02Z",
		Payload:   []byte(`{"requestId":"approval-1","threadId":"thread-2","kind":"command","command":"go test ./..."}`),
	}); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		approvals1 := reg.PendingApprovalsForThread("thread-1")
		approvals2 := reg.PendingApprovalsForThread("thread-2")
		return len(approvals1) == 1 && len(approvals2) == 1
	})

	approval1 := reg.PendingApprovalsForThread("thread-1")[0]
	approval2 := reg.PendingApprovalsForThread("thread-2")[0]
	if approval1.RequestID == "approval-1" || approval2.RequestID == "approval-1" {
		t.Fatalf("expected public scoped approval ids, got %+v %+v", approval1, approval2)
	}
	if approval1.RequestID == approval2.RequestID {
		t.Fatalf("expected machine-scoped approval ids to be unique, got %q", approval1.RequestID)
	}

	machine1, ok1 := hub.ResolveApprovalMachine(approval1.RequestID)
	machine2, ok2 := hub.ResolveApprovalMachine(approval2.RequestID)
	if !ok1 || machine1 != "machine-01" || !ok2 || machine2 != "machine-02" {
		t.Fatalf("unexpected approval routes: machine1=%q ok1=%v machine2=%q ok2=%v", machine1, ok1, machine2, ok2)
	}
}

func expectedApprovalRequestID(machineID string, rawRequestID string) string {
	return "apr." +
		base64.RawURLEncoding.EncodeToString([]byte(machineID)) +
		"." +
		base64.RawURLEncoding.EncodeToString([]byte(rawRequestID))
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

func TestClientHubSendCommandReturnsRejectedCommandError(t *testing.T) {
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

		payload, err := json.Marshal(protocol.CommandRejectedPayload{
			CommandName: "thread.create",
			Reason:      "unsupported command",
		})
		if err != nil {
			t.Errorf("marshal rejection failed: %v", err)
			return
		}

		if err := writeEnvelope(t, conn, protocol.Envelope{
			Version:   version.CurrentProtocolVersion,
			Category:  protocol.CategoryEvent,
			Name:      "command.rejected",
			RequestID: envelope.RequestID,
			MachineID: "machine-01",
			Timestamp: "2026-04-08T10:00:01Z",
			Payload:   payload,
		}); err != nil {
			t.Errorf("write rejection failed: %v", err)
		}
	}()

	_, err = hub.SendCommand(context.Background(), "machine-01", "thread.create", protocol.ThreadCreateCommandPayload{Title: "One"})
	if err == nil {
		t.Fatal("expected command rejection error")
	}
	if got := err.Error(); got != `command "thread.create" rejected: unsupported command` {
		t.Fatalf("unexpected error: %q", got)
	}

	<-done
}

func TestClientHubSendCommandFailsWhenTargetMachineDisconnects(t *testing.T) {
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

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		if _, _, err := conn.Read(context.Background()); err != nil {
			t.Errorf("read command failed: %v", err)
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "disconnect before response")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := hub.SendCommand(ctx, "machine-01", "thread.create", protocol.ThreadCreateCommandPayload{Title: "One"})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected disconnect error")
		}
		if err == context.DeadlineExceeded {
			t.Fatalf("expected disconnect error, got deadline exceeded")
		}
		if !strings.Contains(err.Error(), `machine "machine-01" disconnected`) {
			t.Fatalf("expected machine disconnect error, got %v", err)
		}
	case <-time.After(150 * time.Millisecond):
		t.Fatal("SendCommand did not fail promptly after disconnect")
	}

	<-readDone
}

func TestClientHubSendCommandUsesBoundedTimeoutWithoutCallerDeadline(t *testing.T) {
	hub := NewClientHub()
	hub.commandTimeout = 20 * time.Millisecond
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

	done := make(chan error, 1)
	go func() {
		_, err := hub.SendCommand(context.Background(), "machine-01", "thread.create", protocol.ThreadCreateCommandPayload{Title: "One"})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if err != context.DeadlineExceeded {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("SendCommand did not finish within bounded timeout")
	}
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

	if err := writeEnvelope(t, clientConn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T10:00:19Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, hub, 1, 1*time.Second)
	if err := consoleConn.Close(websocket.StatusNormalClosure, "reset-console"); err != nil {
		t.Fatal(err)
	}

	consoleConn, _, err = websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer consoleConn.Close(websocket.StatusNormalClosure, "done")

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
