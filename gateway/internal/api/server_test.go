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
	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), nil, http.NotFoundHandler(), http.NotFoundHandler())

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

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), nil, wsHandler, http.NotFoundHandler())
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

func TestServerMountsConsoleWebsocketRoute(t *testing.T) {
	called := false
	wsHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusSwitchingProtocols)
	})

	handler := NewServer(
		registry.NewStore(),
		runtimeindex.NewStore(),
		routing.NewRouter(),
		nil,
		http.NotFoundHandler(),
		wsHandler,
	)
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSwitchingProtocols {
		t.Fatalf("/ws returned %d", rec.Code)
	}
	if !called {
		t.Fatal("/ws handler was not invoked")
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

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), sender, http.NotFoundHandler(), http.NotFoundHandler())
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

func TestServerCreateThreadUpdatesRouterBeforeNextTurnRequest(t *testing.T) {
	router := routing.NewRouter()

	sender := &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			switch name {
			case "thread.create":
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
			case "turn.start":
				commandPayload, ok := payload.(protocol.TurnStartCommandPayload)
				if !ok {
					t.Fatalf("payload type = %T", payload)
				}
				if machineID != "machine-01" {
					t.Fatalf("machineID = %q", machineID)
				}
				if commandPayload.ThreadID != "thread-01" {
					t.Fatalf("threadID = %q", commandPayload.ThreadID)
				}

				return protocol.CommandCompletedPayload{
					CommandName: "turn.start",
					Result: mustMarshalJSON(t, protocol.TurnStartCommandResult{
						TurnID:   "turn-01",
						ThreadID: "thread-01",
					}),
				}, nil
			default:
				t.Fatalf("unexpected command %q", name)
				return protocol.CommandCompletedPayload{}, nil
			}
		},
	}

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), router, sender, http.NotFoundHandler(), http.NotFoundHandler())

	createReq := httptest.NewRequest(http.MethodPost, "/threads", bytes.NewBufferString(`{"machineId":"machine-01","title":"Investigate flaky test"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create thread returned %d", createRec.Code)
	}

	turnReq := httptest.NewRequest(http.MethodPost, "/threads/thread-01/turns", bytes.NewBufferString(`{"input":"run tests"}`))
	turnReq.Header.Set("Content-Type", "application/json")
	turnRec := httptest.NewRecorder()
	handler.ServeHTTP(turnRec, turnReq)

	if turnRec.Code != http.StatusAccepted {
		t.Fatalf("expected immediate turn start to succeed, got %d with %s", turnRec.Code, turnRec.Body.String())
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

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), router, sender, http.NotFoundHandler(), http.NotFoundHandler())
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

func TestServerReadsThreadFromResolvedMachine(t *testing.T) {
	router := routing.NewRouter()
	router.TrackThread("thread-01", "machine-01")

	sender := &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			if machineID != "machine-01" {
				t.Fatalf("machineID = %q", machineID)
			}
			if name != "thread.read" {
				t.Fatalf("name = %q", name)
			}

			commandPayload, ok := payload.(protocol.ThreadReadCommandPayload)
			if !ok {
				t.Fatalf("payload type = %T", payload)
			}
			if commandPayload.ThreadID != "thread-01" {
				t.Fatalf("threadID = %q", commandPayload.ThreadID)
			}

			return protocol.CommandCompletedPayload{
				CommandName: "thread.read",
				Result: mustMarshalJSON(t, protocol.ThreadReadCommandResult{
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

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), router, sender, http.NotFoundHandler(), http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/threads/thread-01", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
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

func TestServerThreadAndTurnControlEndpointsUseExpectedCommands(t *testing.T) {
	router := routing.NewRouter()
	router.TrackThread("thread-01", "machine-01")

	type recordedCall struct {
		machineID string
		name      string
		payload   any
	}

	var calls []recordedCall
	sender := &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			calls = append(calls, recordedCall{
				machineID: machineID,
				name:      name,
				payload:   payload,
			})

			switch name {
			case "thread.resume":
				return protocol.CommandCompletedPayload{
					CommandName: name,
					Result: mustMarshalJSON(t, protocol.ThreadResumeCommandResult{
						Thread: domain.Thread{
							ThreadID:  "thread-01",
							MachineID: "machine-01",
							Status:    domain.ThreadStatusIdle,
							Title:     "Investigate flaky test",
						},
					}),
				}, nil
			case "thread.archive":
				return protocol.CommandCompletedPayload{
					CommandName: name,
					Result: mustMarshalJSON(t, protocol.ThreadArchiveCommandResult{
						ThreadID: "thread-01",
					}),
				}, nil
			case "turn.steer":
				return protocol.CommandCompletedPayload{
					CommandName: name,
					Result: mustMarshalJSON(t, protocol.TurnSteerCommandResult{
						ThreadID: "thread-01",
						TurnID:   "turn-01",
					}),
				}, nil
			case "turn.interrupt":
				return protocol.CommandCompletedPayload{
					CommandName: name,
					Result: mustMarshalJSON(t, protocol.TurnInterruptCommandResult{
						Turn: domain.Turn{
							ThreadID: "thread-01",
							TurnID:   "turn-01",
							Status:   domain.TurnStatusInterrupted,
						},
					}),
				}, nil
			default:
				t.Fatalf("unexpected command %q", name)
				return protocol.CommandCompletedPayload{}, nil
			}
		},
	}

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), router, sender, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodPost, "/threads/thread-01/resume", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("resume returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/threads/thread-01/archive", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("archive returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/threads/thread-01/turns/turn-01/steer", bytes.NewBufferString(`{"input":"try a smaller patch"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("steer returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/threads/thread-01/turns/turn-01/interrupt", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("interrupt returned %d", rec.Code)
	}

	if len(calls) != 4 {
		t.Fatalf("expected 4 calls, got %d", len(calls))
	}

	if calls[0].machineID != "machine-01" || calls[0].name != "thread.resume" {
		t.Fatalf("unexpected resume call: %+v", calls[0])
	}
	resumePayload, ok := calls[0].payload.(protocol.ThreadResumeCommandPayload)
	if !ok {
		t.Fatalf("resume payload type = %T", calls[0].payload)
	}
	if resumePayload.ThreadID != "thread-01" {
		t.Fatalf("unexpected resume payload: %+v", resumePayload)
	}

	if calls[1].machineID != "machine-01" || calls[1].name != "thread.archive" {
		t.Fatalf("unexpected archive call: %+v", calls[1])
	}
	archivePayload, ok := calls[1].payload.(protocol.ThreadArchiveCommandPayload)
	if !ok {
		t.Fatalf("archive payload type = %T", calls[1].payload)
	}
	if archivePayload.ThreadID != "thread-01" {
		t.Fatalf("unexpected archive payload: %+v", archivePayload)
	}

	if calls[2].machineID != "machine-01" || calls[2].name != "turn.steer" {
		t.Fatalf("unexpected steer call: %+v", calls[2])
	}
	steerPayload, ok := calls[2].payload.(protocol.TurnSteerCommandPayload)
	if !ok {
		t.Fatalf("steer payload type = %T", calls[2].payload)
	}
	if steerPayload.ThreadID != "thread-01" || steerPayload.TurnID != "turn-01" || steerPayload.Input != "try a smaller patch" {
		t.Fatalf("unexpected steer payload: %+v", steerPayload)
	}

	if calls[3].machineID != "machine-01" || calls[3].name != "turn.interrupt" {
		t.Fatalf("unexpected interrupt call: %+v", calls[3])
	}
	interruptPayload, ok := calls[3].payload.(protocol.TurnInterruptCommandPayload)
	if !ok {
		t.Fatalf("interrupt payload type = %T", calls[3].payload)
	}
	if interruptPayload.ThreadID != "thread-01" || interruptPayload.TurnID != "turn-01" {
		t.Fatalf("unexpected interrupt payload: %+v", interruptPayload)
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
