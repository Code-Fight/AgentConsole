package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/routing"
	"code-agent-gateway/gateway/internal/runtimeindex"
)

func TestServerServesEmptyControlPlaneViews(t *testing.T) {
	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), nil, http.NotFoundHandler())

	for _, path := range []string{"/health", "/machines", "/threads", "/environment/skills", "/environment/mcps", "/environment/plugins"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, rec.Code)
		}

		if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("%s content-type = %q", path, got)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s invalid json: %v", path, err)
		}

		if path == "/health" {
			ok, exists := body["ok"].(bool)
			if !exists || !ok {
				t.Fatalf("%s unexpected body: %v", path, body)
			}
			continue
		}

		items, exists := body["items"].([]any)
		if !exists {
			t.Fatalf("%s items is not a json array: %T (%v)", path, body["items"], body["items"])
		}
		if len(items) != 0 {
			t.Fatalf("%s expected empty items, got: %v", path, items)
		}
	}
}

func TestServerMountsClientWebsocketRoute(t *testing.T) {
	called := false
	wsHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), nil, wsHandler)
	req := httptest.NewRequest(http.MethodGet, "/ws/client", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("/ws/client returned %d", rec.Code)
	}
	if !called {
		t.Fatal("/ws/client handler was not invoked")
	}
}

func TestServerCreatesThreadThroughCommandSender(t *testing.T) {
	sender := &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			if machineID != "machine-01" {
				t.Fatalf("machineID = %q", machineID)
			}
			if name != "thread.create" {
				t.Fatalf("name = %q", name)
			}

			commandPayload, ok := payload.(protocol.ThreadCreateCommandPayload)
			if !ok {
				t.Fatalf("payload type = %T", payload)
			}
			if commandPayload.Title != "Investigate flaky test" {
				t.Fatalf("title = %q", commandPayload.Title)
			}

			return protocol.CommandCompletedPayload{
				CommandName: "thread.create",
				Result: mustMarshalJSON(t, protocol.ThreadCreateCommandResult{
					Thread: domain.Thread{
						ThreadID:  "thread-01",
						MachineID: "machine-01",
						Status:    domain.ThreadStatusIdle,
						Title:     "Investigate flaky test",
					},
				}),
			}, nil
		},
	}

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), sender, http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodPost, "/threads", bytes.NewBufferString(`{"machineId":"machine-01","title":"Investigate flaky test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var body struct {
		Thread domain.Thread `json:"thread"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if body.Thread.ThreadID != "thread-01" || body.Thread.MachineID != "machine-01" {
		t.Fatalf("unexpected thread: %+v", body.Thread)
	}
}

func TestServerStartsTurnOnResolvedMachine(t *testing.T) {
	router := routing.NewRouter()
	router.TrackThread("thread-01", "machine-01")

	sender := &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			if machineID != "machine-01" {
				t.Fatalf("machineID = %q", machineID)
			}
			if name != "turn.start" {
				t.Fatalf("name = %q", name)
			}

			commandPayload, ok := payload.(protocol.TurnStartCommandPayload)
			if !ok {
				t.Fatalf("payload type = %T", payload)
			}
			if commandPayload.ThreadID != "thread-01" || commandPayload.Input != "run tests" {
				t.Fatalf("unexpected payload: %+v", commandPayload)
			}

			return protocol.CommandCompletedPayload{
				CommandName: "turn.start",
				Result: mustMarshalJSON(t, protocol.TurnStartCommandResult{
					TurnID:   "turn-01",
					ThreadID: "thread-01",
				}),
			}, nil
		},
	}

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), router, sender, http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodPost, "/threads/thread-01/turns", bytes.NewBufferString(`{"input":"run tests"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	var body struct {
		Turn protocol.TurnStartCommandResult `json:"turn"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if body.Turn.TurnID != "turn-01" || body.Turn.ThreadID != "thread-01" {
		t.Fatalf("unexpected turn: %+v", body.Turn)
	}
}

type fakeCommandSender struct {
	send func(ctx context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error)
}

func (s *fakeCommandSender) SendCommand(ctx context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
	if s.send == nil {
		return protocol.CommandCompletedPayload{}, nil
	}

	return s.send(ctx, machineID, name, payload)
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return raw
}
