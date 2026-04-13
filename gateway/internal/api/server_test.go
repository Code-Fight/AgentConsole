package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"code-agent-gateway/gateway/internal/settings"
	gatewayws "code-agent-gateway/gateway/internal/websocket"
	"github.com/coder/websocket"
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

func TestServerRuntimeControlEndpointsUseExpectedCommands(t *testing.T) {
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
			return protocol.CommandCompletedPayload{
				CommandName: name,
				Result:      mustMarshalJSON(t, map[string]any{}),
			}, nil
		},
	}

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), sender, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodPost, "/machines/machine-01/runtime/stop", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("runtime stop returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/machines/machine-01/runtime/start", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("runtime start returned %d", rec.Code)
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].machineID != "machine-01" || calls[0].name != "runtime.stop" {
		t.Fatalf("unexpected stop call: %+v", calls[0])
	}
	if _, ok := calls[0].payload.(protocol.RuntimeStopCommandPayload); !ok {
		t.Fatalf("unexpected runtime.stop payload type: %T", calls[0].payload)
	}
	if calls[1].machineID != "machine-01" || calls[1].name != "runtime.start" {
		t.Fatalf("unexpected start call: %+v", calls[1])
	}
	if _, ok := calls[1].payload.(protocol.RuntimeStartCommandPayload); !ok {
		t.Fatalf("unexpected runtime.start payload type: %T", calls[1].payload)
	}
}

func TestServerEnvironmentMutationEndpointsUseExpectedCommands(t *testing.T) {
	idx := runtimeindex.NewStore()
	idx.ReplaceSnapshot("machine-01", nil, []domain.EnvironmentResource{
		{
			ResourceID:  "skill-a",
			MachineID:   "machine-01",
			Kind:        domain.EnvironmentKindSkill,
			DisplayName: "Skill A",
			Status:      domain.EnvironmentResourceStatusEnabled,
		},
		{
			ResourceID:  "github",
			MachineID:   "machine-01",
			Kind:        domain.EnvironmentKindMCP,
			DisplayName: "GitHub MCP",
			Status:      domain.EnvironmentResourceStatusEnabled,
		},
		{
			ResourceID:  "plugin-a",
			MachineID:   "machine-01",
			Kind:        domain.EnvironmentKindPlugin,
			DisplayName: "Plugin A",
			Status:      domain.EnvironmentResourceStatusEnabled,
			Details: map[string]any{
				"marketplacePath": "/tmp/codex/marketplace",
				"pluginName":      "plugin-a",
			},
		},
	})

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
			return protocol.CommandCompletedPayload{
				CommandName: name,
				Result:      mustMarshalJSON(t, map[string]any{}),
			}, nil
		},
	}

	handler := NewServer(registry.NewStore(), idx, routing.NewRouter(), sender, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodPost, "/environment/skills/skill-a/enable", bytes.NewBufferString(`{"machineId":"machine-01"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("enable returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/environment/skills/skill-a/disable", bytes.NewBufferString(`{"machineId":"machine-01"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/environment/plugins/plugin-a", bytes.NewBufferString(`{"machineId":"machine-01"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("plugin uninstall returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/environment/mcps", bytes.NewBufferString(`{"machineId":"machine-01","resourceId":"github","config":{"command":"npx","args":["-y","@modelcontextprotocol/server-github"]}}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mcp upsert returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/environment/mcps/github/disable", bytes.NewBufferString(`{"machineId":"machine-01"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mcp disable returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/environment/mcps/github", bytes.NewBufferString(`{"machineId":"machine-01"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mcp remove returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/environment/plugins/plugin-a/install", bytes.NewBufferString(`{"machineId":"machine-01"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("plugin install returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/environment/plugins/plugin-a/disable", bytes.NewBufferString(`{"machineId":"machine-01"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("plugin disable returned %d", rec.Code)
	}

	if len(calls) != 8 {
		t.Fatalf("expected 8 calls, got %d", len(calls))
	}
	if calls[0].machineID != "machine-01" || calls[0].name != "environment.skill.enable" {
		t.Fatalf("unexpected enable call: %+v", calls[0])
	}
	enablePayload, ok := calls[0].payload.(protocol.EnvironmentSkillSetEnabledCommandPayload)
	if !ok {
		t.Fatalf("unexpected enable payload type: %T", calls[0].payload)
	}
	if enablePayload.SkillID != "skill-a" || !enablePayload.Enabled {
		t.Fatalf("unexpected enable payload: %+v", enablePayload)
	}

	if calls[1].machineID != "machine-01" || calls[1].name != "environment.skill.disable" {
		t.Fatalf("unexpected disable call: %+v", calls[1])
	}
	disablePayload, ok := calls[1].payload.(protocol.EnvironmentSkillSetEnabledCommandPayload)
	if !ok {
		t.Fatalf("unexpected disable payload type: %T", calls[1].payload)
	}
	if disablePayload.SkillID != "skill-a" || disablePayload.Enabled {
		t.Fatalf("unexpected disable payload: %+v", disablePayload)
	}

	if calls[2].machineID != "machine-01" || calls[2].name != "environment.plugin.uninstall" {
		t.Fatalf("unexpected plugin uninstall call: %+v", calls[2])
	}
	uninstallPayload, ok := calls[2].payload.(protocol.EnvironmentPluginUninstallCommandPayload)
	if !ok {
		t.Fatalf("unexpected uninstall payload type: %T", calls[2].payload)
	}
	if uninstallPayload.PluginID != "plugin-a" {
		t.Fatalf("unexpected uninstall payload: %+v", uninstallPayload)
	}

	if calls[3].machineID != "machine-01" || calls[3].name != "environment.mcp.upsert" {
		t.Fatalf("unexpected mcp upsert call: %+v", calls[3])
	}
	mcpUpsertPayload, ok := calls[3].payload.(protocol.EnvironmentMCPUpsertCommandPayload)
	if !ok {
		t.Fatalf("unexpected mcp upsert payload type: %T", calls[3].payload)
	}
	if mcpUpsertPayload.ServerID != "github" {
		t.Fatalf("unexpected mcp upsert payload: %+v", mcpUpsertPayload)
	}

	if calls[4].machineID != "machine-01" || calls[4].name != "environment.mcp.disable" {
		t.Fatalf("unexpected mcp disable call: %+v", calls[4])
	}
	mcpDisablePayload, ok := calls[4].payload.(protocol.EnvironmentMCPSetEnabledCommandPayload)
	if !ok {
		t.Fatalf("unexpected mcp disable payload type: %T", calls[4].payload)
	}
	if mcpDisablePayload.ServerID != "github" || mcpDisablePayload.Enabled {
		t.Fatalf("unexpected mcp disable payload: %+v", mcpDisablePayload)
	}

	if calls[5].machineID != "machine-01" || calls[5].name != "environment.mcp.remove" {
		t.Fatalf("unexpected mcp remove call: %+v", calls[5])
	}
	mcpRemovePayload, ok := calls[5].payload.(protocol.EnvironmentMCPRemoveCommandPayload)
	if !ok {
		t.Fatalf("unexpected mcp remove payload type: %T", calls[5].payload)
	}
	if mcpRemovePayload.ServerID != "github" {
		t.Fatalf("unexpected mcp remove payload: %+v", mcpRemovePayload)
	}

	if calls[6].machineID != "machine-01" || calls[6].name != "environment.plugin.install" {
		t.Fatalf("unexpected plugin install call: %+v", calls[6])
	}
	pluginInstallPayload, ok := calls[6].payload.(protocol.EnvironmentPluginInstallCommandPayload)
	if !ok {
		t.Fatalf("unexpected plugin install payload type: %T", calls[6].payload)
	}
	if pluginInstallPayload.PluginID != "plugin-a" || pluginInstallPayload.MarketplacePath != "/tmp/codex/marketplace" || pluginInstallPayload.PluginName != "plugin-a" {
		t.Fatalf("unexpected plugin install payload: %+v", pluginInstallPayload)
	}

	if calls[7].machineID != "machine-01" || calls[7].name != "environment.plugin.disable" {
		t.Fatalf("unexpected plugin disable call: %+v", calls[7])
	}
	pluginDisablePayload, ok := calls[7].payload.(protocol.EnvironmentPluginSetEnabledCommandPayload)
	if !ok {
		t.Fatalf("unexpected plugin disable payload type: %T", calls[7].payload)
	}
	if pluginDisablePayload.PluginID != "plugin-a" || pluginDisablePayload.Enabled {
		t.Fatalf("unexpected plugin disable payload: %+v", pluginDisablePayload)
	}
}

func TestServerEnvironmentMutationRejectsMissingMachineID(t *testing.T) {
	idx := runtimeindex.NewStore()
	idx.ReplaceSnapshot("machine-01", nil, []domain.EnvironmentResource{
		{
			ResourceID:  "skill-a",
			MachineID:   "machine-01",
			Kind:        domain.EnvironmentKindSkill,
			DisplayName: "Skill A",
			Status:      domain.EnvironmentResourceStatusEnabled,
		},
	})

	handler := NewServer(registry.NewStore(), idx, routing.NewRouter(), &fakeCommandSender{}, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodPost, "/environment/skills/skill-a/enable", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "machineId is required\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestServerEnvironmentMutationTargetsRequestedMachine(t *testing.T) {
	idx := runtimeindex.NewStore()
	idx.ReplaceSnapshot("machine-01", nil, []domain.EnvironmentResource{
		{
			ResourceID:  "skill-a",
			MachineID:   "machine-01",
			Kind:        domain.EnvironmentKindSkill,
			DisplayName: "Skill A 1",
			Status:      domain.EnvironmentResourceStatusEnabled,
		},
	})
	idx.ReplaceSnapshot("machine-02", nil, []domain.EnvironmentResource{
		{
			ResourceID:  "skill-a",
			MachineID:   "machine-02",
			Kind:        domain.EnvironmentKindSkill,
			DisplayName: "Skill A 2",
			Status:      domain.EnvironmentResourceStatusDisabled,
		},
	})

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
			return protocol.CommandCompletedPayload{
				CommandName: name,
				Result:      mustMarshalJSON(t, map[string]any{}),
			}, nil
		},
	}

	handler := NewServer(registry.NewStore(), idx, routing.NewRouter(), sender, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodPost, "/environment/skills/skill-a/enable", bytes.NewBufferString(`{"machineId":"machine-02"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].machineID != "machine-02" {
		t.Fatalf("expected machine-02, got %+v", calls[0])
	}
}

func TestServerRoutesApprovalResponseToResolvedMachine(t *testing.T) {
	reg := registry.NewStore()
	reg.UpsertPendingApproval("machine-01", protocol.ApprovalRequiredPayload{
		RequestID: "approval-1",
		ThreadID:  "thread-01",
		TurnID:    "turn-01",
		Kind:      "command",
		Command:   "go test ./...",
	})

	sender := &fakeCommandSender{
		resolveApprovalMachine: func(requestID string) (string, bool) {
			if requestID != "approval-1" {
				t.Fatalf("requestID = %q", requestID)
			}
			return "machine-01", true
		},
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			if machineID != "machine-01" {
				t.Fatalf("machineID = %q", machineID)
			}
			if name != "approval.respond" {
				t.Fatalf("name = %q", name)
			}

			commandPayload, ok := payload.(protocol.ApprovalRespondCommandPayload)
			if !ok {
				t.Fatalf("payload type = %T", payload)
			}
			if commandPayload.RequestID != "approval-1" || commandPayload.Decision != "accept" {
				t.Fatalf("unexpected payload: %+v", commandPayload)
			}
			if commandPayload.ThreadID != "thread-01" || commandPayload.TurnID != "turn-01" {
				t.Fatalf("expected stored thread context in payload: %+v", commandPayload)
			}
			if len(commandPayload.Answers) != 2 ||
				commandPayload.Answers["question-1"] != "release" ||
				commandPayload.Answers["question-2"] != "Need the release branch" {
				t.Fatalf("expected approval answers in payload: %+v", commandPayload)
			}

			return protocol.CommandCompletedPayload{
				CommandName: "approval.respond",
				Result: mustMarshalJSON(t, protocol.ApprovalRespondCommandResult{
					RequestID: "approval-1",
					Decision:  "accept",
				}),
			}, nil
		},
	}

	handler := NewServer(reg, runtimeindex.NewStore(), routing.NewRouter(), sender, http.NotFoundHandler(), http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodPost, "/approvals/approval-1/respond", bytes.NewBufferString(`{"decision":"accept","answers":{"question-1":"release","question-2":"Need the release branch"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with %s", rec.Code, rec.Body.String())
	}

	var body struct {
		RequestID string `json:"requestId"`
		Decision  string `json:"decision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body.RequestID != "approval-1" || body.Decision != "accept" {
		t.Fatalf("unexpected response body: %+v", body)
	}
	if _, ok := reg.PendingApproval("approval-1"); ok {
		t.Fatal("expected successful approval response to clear durable pending approval state")
	}
}

func TestServerThreadDetailIncludesActiveTurnIDFromSenderState(t *testing.T) {
	idx := runtimeindex.NewStore()
	thread := domain.Thread{
		ThreadID:  "thread-01",
		MachineID: "machine-01",
		Status:    domain.ThreadStatusActive,
		Title:     "Investigate flaky test",
	}
	idx.ReplaceSnapshot("machine-01", []domain.Thread{thread}, nil)

	router := routing.NewRouter()
	router.TrackThread("thread-01", "machine-01")

	handler := NewServer(registry.NewStore(), idx, router, &fakeCommandSender{
		activeTurnID: func(threadID string) (string, bool) {
			if threadID != "thread-01" {
				t.Fatalf("threadID = %q", threadID)
			}
			return "turn-active-1", true
		},
	}, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodGet, "/threads/thread-01", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Thread       domain.Thread `json:"thread"`
		ActiveTurnID string        `json:"activeTurnId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body.Thread.ThreadID != "thread-01" || body.ActiveTurnID != "turn-active-1" {
		t.Fatalf("unexpected thread detail: %+v", body)
	}
}

func TestServerDeleteThreadBroadcastsThreadUpdatedInvalidation(t *testing.T) {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	router := routing.NewRouter()
	clientHub := gatewayws.NewClientHubWithStores(reg, idx, router)
	consoleHub := gatewayws.NewConsoleHub()
	clientHub.SetConsoleHub(consoleHub)

	thread := domain.Thread{
		ThreadID:  "thread-01",
		MachineID: "machine-01",
		Status:    domain.ThreadStatusIdle,
		Title:     "Investigate flaky test",
	}
	idx.ReplaceSnapshot("machine-01", []domain.Thread{thread}, nil)
	router.TrackThread("thread-01", "machine-01")

	server := httptest.NewServer(NewServer(reg, idx, router, clientHub, clientHub.Handler(), consoleHub.Handler()))
	defer server.Close()

	conn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	req := httptest.NewRequest(http.MethodDelete, "/threads/thread-01", nil)
	rec := httptest.NewRecorder()
	NewServer(reg, idx, router, clientHub, clientHub.Handler(), consoleHub.Handler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with %s", rec.Code, rec.Body.String())
	}

	readCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("expected thread.updated broadcast after delete: %v", err)
	}

	var envelope protocol.Envelope
	if err := transport.Decode(data, &envelope); err != nil {
		t.Fatalf("decode envelope failed: %v", err)
	}
	if envelope.Name != "thread.updated" {
		t.Fatalf("expected thread.updated, got %q", envelope.Name)
	}

	var payload protocol.ThreadUpdatedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}
	if payload.MachineID != "machine-01" || payload.ThreadID != "thread-01" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestServerThreadDetailIncludesPendingApprovalsWhenThreadIsOffline(t *testing.T) {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	router := routing.NewRouter()
	clientHub := gatewayws.NewClientHubWithStores(reg, idx, router)

	thread := domain.Thread{
		ThreadID:  "thread-01",
		MachineID: "machine-01",
		Status:    domain.ThreadStatusIdle,
		Title:     "Investigate flaky test",
	}
	idx.ReplaceSnapshot("machine-01", []domain.Thread{thread}, nil)
	router.TrackThread(thread.ThreadID, thread.MachineID)

	handler := NewServer(reg, idx, router, &fakeCommandSender{
		send: func(_ context.Context, _ string, _ string, _ any) (protocol.CommandCompletedPayload, error) {
			t.Fatal("thread detail fallback should not call the runtime for an offline thread")
			return protocol.CommandCompletedPayload{}, nil
		},
	}, clientHub.Handler(), http.NotFoundHandler())

	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := writeClientEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T10:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeClientEnvelope(t, conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategoryEvent,
		Name:      "approval.required",
		RequestID: "approval-1",
		MachineID: "machine-01",
		Timestamp: "2026-04-09T10:00:01Z",
		Payload: mustMarshalJSON(t, protocol.ApprovalRequiredPayload{
			RequestID: "approval-1",
			ThreadID:  "thread-01",
			TurnID:    "turn-01",
			ItemID:    "item-01",
			Kind:      "command",
			Command:   "go test ./...",
		}),
	}); err != nil {
		t.Fatal(err)
	}

	if err := conn.Close(websocket.StatusNormalClosure, "offline"); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		machines := reg.List()
		return len(machines) == 1 && machines[0].Status == domain.MachineStatusOffline
	})

	req := httptest.NewRequest(http.MethodGet, "/threads/thread-01", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Thread           domain.Thread                      `json:"thread"`
		PendingApprovals []protocol.ApprovalRequiredPayload `json:"pendingApprovals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body.Thread.ThreadID != "thread-01" || body.Thread.Status != domain.ThreadStatusUnknown {
		t.Fatalf("unexpected thread: %+v", body.Thread)
	}
	if len(body.PendingApprovals) != 1 {
		t.Fatalf("expected 1 pending approval, got %+v", body.PendingApprovals)
	}
	if body.PendingApprovals[0].RequestID != expectedApprovalRequestID("machine-01", "approval-1") || body.PendingApprovals[0].Command != "go test ./..." {
		t.Fatalf("unexpected pending approval: %+v", body.PendingApprovals[0])
	}
}

func TestServerThreadDetailReturnsLiveReadFailureForOnlineMachine(t *testing.T) {
	reg := registry.NewStore()
	reg.Upsert(domain.Machine{
		ID:     "machine-01",
		Name:   "machine-01",
		Status: domain.MachineStatusOnline,
	})

	idx := runtimeindex.NewStore()
	idx.ReplaceSnapshot("machine-01", []domain.Thread{
		{
			ThreadID:  "thread-01",
			MachineID: "machine-01",
			Status:    domain.ThreadStatusIdle,
			Title:     "stale snapshot",
		},
	}, nil)

	router := routing.NewRouter()
	router.TrackThread("thread-01", "machine-01")

	handler := NewServer(reg, idx, router, &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			if machineID != "machine-01" || name != "thread.read" {
				t.Fatalf("unexpected live read route %q %q", machineID, name)
			}
			return protocol.CommandCompletedPayload{}, context.DeadlineExceeded
		},
	}, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodGet, "/threads/thread-01", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestServerGetsMachineByIDAndDeletesThreadThroughArchiveShim(t *testing.T) {
	reg := registry.NewStore()
	reg.Upsert(domain.Machine{
		ID:     "machine-01",
		Name:   "Dev Mac",
		Status: domain.MachineStatusOnline,
	})

	idx := runtimeindex.NewStore()
	idx.ReplaceSnapshot("machine-01", []domain.Thread{
		{
			ThreadID:  "thread-01",
			MachineID: "machine-01",
			Status:    domain.ThreadStatusIdle,
			Title:     "Investigate flaky test",
		},
	}, nil)

	router := routing.NewRouter()
	router.TrackThread("thread-01", "machine-01")

	handler := NewServer(reg, idx, router, &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			if machineID != "machine-01" {
				t.Fatalf("machineID = %q", machineID)
			}
			if name != "thread.archive" {
				t.Fatalf("name = %q", name)
			}
			commandPayload, ok := payload.(protocol.ThreadArchiveCommandPayload)
			if !ok {
				t.Fatalf("payload type = %T", payload)
			}
			if commandPayload.ThreadID != "thread-01" {
				t.Fatalf("threadID = %q", commandPayload.ThreadID)
			}
			return protocol.CommandCompletedPayload{
				CommandName: "thread.archive",
				Result: mustMarshalJSON(t, protocol.ThreadArchiveCommandResult{
					ThreadID: "thread-01",
				}),
			}, nil
		},
	}, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodGet, "/machines/machine-01", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("machine detail returned %d", rec.Code)
	}

	var machineBody struct {
		Machine domain.Machine `json:"machine"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &machineBody); err != nil {
		t.Fatalf("invalid machine json: %v", err)
	}
	if machineBody.Machine.ID != "machine-01" || machineBody.Machine.Name != "Dev Mac" {
		t.Fatalf("unexpected machine: %+v", machineBody.Machine)
	}

	req = httptest.NewRequest(http.MethodDelete, "/threads/thread-01", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("delete returned %d with %s", rec.Code, rec.Body.String())
	}

	var deleteBody struct {
		ThreadID string `json:"threadId"`
		Deleted  bool   `json:"deleted"`
		Archived bool   `json:"archived"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &deleteBody); err != nil {
		t.Fatalf("invalid delete json: %v", err)
	}
	if deleteBody.ThreadID != "thread-01" || !deleteBody.Deleted || !deleteBody.Archived {
		t.Fatalf("unexpected delete body: %+v", deleteBody)
	}

	req = httptest.NewRequest(http.MethodGet, "/threads", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var threadsBody struct {
		Items []domain.Thread `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &threadsBody); err != nil {
		t.Fatalf("invalid threads json: %v", err)
	}
	if len(threadsBody.Items) != 0 {
		t.Fatalf("expected deleted thread to be hidden from listing, got %+v", threadsBody.Items)
	}

	req = httptest.NewRequest(http.MethodGet, "/threads/thread-01", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected deleted thread detail to return 404, got %d", rec.Code)
	}
}

type fakeCommandSender struct {
	send                   func(ctx context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error)
	resolveApprovalMachine func(requestID string) (string, bool)
	activeTurnID           func(threadID string) (string, bool)
}

func (s *fakeCommandSender) SendCommand(ctx context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
	if s.send == nil {
		return protocol.CommandCompletedPayload{}, nil
	}

	return s.send(ctx, machineID, name, payload)
}

func (s *fakeCommandSender) ResolveApprovalMachine(requestID string) (string, bool) {
	if s.resolveApprovalMachine == nil {
		return "", false
	}
	return s.resolveApprovalMachine(requestID)
}

func (s *fakeCommandSender) ActiveTurnID(threadID string) (string, bool) {
	if s.activeTurnID == nil {
		return "", false
	}
	return s.activeTurnID(threadID)
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return raw
}

func expectedApprovalRequestID(machineID string, rawRequestID string) string {
	return "apr." +
		base64.RawURLEncoding.EncodeToString([]byte(machineID)) +
		"." +
		base64.RawURLEncoding.EncodeToString([]byte(rawRequestID))
}

func writeClientEnvelope(t *testing.T, conn *websocket.Conn, envelope protocol.Envelope) error {
	t.Helper()

	raw, err := transport.Encode(envelope)
	if err != nil {
		t.Fatalf("encode envelope failed: %v", err)
	}
	return conn.Write(context.Background(), websocket.MessageText, raw)
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
	t.Fatal("condition not met before timeout")
}

func TestServerSettingsEndpointsManageGlobalAndMachineDocuments(t *testing.T) {
	settingsStore := settings.NewMemoryStore([]domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	handler := NewServerWithSettings(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), &fakeCommandSender{}, settingsStore, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodPut, "/settings/agents/codex/global", bytes.NewBufferString(`{"content":"model = \"gpt-5.4\"\n"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("global put returned %d with %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/settings/agents/codex/global", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("global get returned %d", rec.Code)
	}
	var globalBody struct {
		Document *domain.AgentConfigDocument `json:"document"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &globalBody); err != nil {
		t.Fatal(err)
	}
	if globalBody.Document == nil || globalBody.Document.Content != "model = \"gpt-5.4\"\n" {
		t.Fatalf("unexpected global response: %+v", globalBody)
	}

	req = httptest.NewRequest(http.MethodPut, "/settings/machines/machine-01/agents/codex", bytes.NewBufferString(`{"content":"model = \"gpt-5.2\"\n"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("machine put returned %d with %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/settings/machines/machine-01/agents/codex", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("machine get returned %d", rec.Code)
	}
	var assignment domain.MachineAgentConfigAssignment
	if err := json.Unmarshal(rec.Body.Bytes(), &assignment); err != nil {
		t.Fatal(err)
	}
	if assignment.MachineOverride == nil || assignment.MachineOverride.Content != "model = \"gpt-5.2\"\n" || assignment.UsesGlobalDefault {
		t.Fatalf("unexpected machine assignment: %+v", assignment)
	}

	req = httptest.NewRequest(http.MethodDelete, "/settings/machines/machine-01/agents/codex", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("machine delete returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/settings/machines/machine-01/agents/codex", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("machine get after delete returned %d", rec.Code)
	}
	var fallbackAssignment domain.MachineAgentConfigAssignment
	if err := json.Unmarshal(rec.Body.Bytes(), &fallbackAssignment); err != nil {
		t.Fatal(err)
	}
	if fallbackAssignment.MachineOverride != nil || !fallbackAssignment.UsesGlobalDefault || fallbackAssignment.GlobalDefault == nil {
		t.Fatalf("expected fallback to global default, got %+v", fallbackAssignment)
	}
}

type consoleSettingsErrorStore struct {
	*settings.MemoryStore
	getErr error
}

func (s *consoleSettingsErrorStore) GetConsolePreferences() (domain.ConsolePreferences, bool, error) {
	if s.getErr != nil {
		return domain.ConsolePreferences{}, false, s.getErr
	}
	return s.MemoryStore.GetConsolePreferences()
}

func TestConsoleSettingsPutEchoesRequestWhenReadbackFails(t *testing.T) {
	settingsStore := &consoleSettingsErrorStore{
		MemoryStore: settings.NewMemoryStore([]domain.AgentDescriptor{
			{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
		}),
		getErr: errors.New("read failed"),
	}
	handler := NewServerWithSettings(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), nil, settingsStore, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodPut, "/settings/console", bytes.NewBufferString(`{
  "preferences": {
    "consoleUrl": "http://localhost:3100",
    "apiKey": "test-key",
    "profile": "dev",
    "safetyPolicy": "strict",
    "lastThreadId": "thread-123"
  }
}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("console settings put returned %d with %s", rec.Code, rec.Body.String())
	}

	var putBody struct {
		Preferences *struct {
			ConsoleURL   string `json:"consoleUrl"`
			APIKey       string `json:"apiKey"`
			Profile      string `json:"profile"`
			SafetyPolicy string `json:"safetyPolicy"`
			LastThreadID string `json:"lastThreadId"`
		} `json:"preferences"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &putBody); err != nil {
		t.Fatalf("invalid console settings json: %v", err)
	}
	if putBody.Preferences == nil || putBody.Preferences.LastThreadID != "thread-123" {
		t.Fatalf("unexpected console preferences: %+v", putBody.Preferences)
	}
}

func TestCapabilitiesEndpointReflectsDependencies(t *testing.T) {
	settingsStore := settings.NewMemoryStore([]domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	handler := NewServerWithSettings(
		registry.NewStore(),
		runtimeindex.NewStore(),
		routing.NewRouter(),
		nil,
		settingsStore,
		http.NotFoundHandler(),
		http.NotFoundHandler(),
	)

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("capabilities returned %d with %s", rec.Code, rec.Body.String())
	}

	var body struct {
		ThreadHub                   bool `json:"threadHub"`
		ThreadWorkspace             bool `json:"threadWorkspace"`
		Approvals                   bool `json:"approvals"`
		StartTurn                   bool `json:"startTurn"`
		SteerTurn                   bool `json:"steerTurn"`
		InterruptTurn               bool `json:"interruptTurn"`
		MachineInstallAgent         bool `json:"machineInstallAgent"`
		MachineRemoveAgent          bool `json:"machineRemoveAgent"`
		EnvironmentSyncCatalog      bool `json:"environmentSyncCatalog"`
		EnvironmentRestartBridge    bool `json:"environmentRestartBridge"`
		EnvironmentOpenMarketplace  bool `json:"environmentOpenMarketplace"`
		EnvironmentMutateResources  bool `json:"environmentMutateResources"`
		EnvironmentWriteMcp         bool `json:"environmentWriteMcp"`
		SettingsEditGatewayEndpoint bool `json:"settingsEditGatewayEndpoint"`
		SettingsEditConsoleProfile  bool `json:"settingsEditConsoleProfile"`
		SettingsEditSafetyPolicy    bool `json:"settingsEditSafetyPolicy"`
		SettingsGlobalDefault       bool `json:"settingsGlobalDefault"`
		SettingsMachineOverride     bool `json:"settingsMachineOverride"`
		SettingsApplyMachine        bool `json:"settingsApplyMachine"`
		DashboardMetrics            bool `json:"dashboardMetrics"`
		AgentLifecycle              bool `json:"agentLifecycle"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid capabilities json: %v", err)
	}

	if !body.ThreadHub || !body.ThreadWorkspace {
		t.Fatalf("expected thread hub/workspace to be enabled: %+v", body)
	}
	if body.Approvals || body.StartTurn || body.SteerTurn || body.InterruptTurn {
		t.Fatalf("expected command-driven capabilities to be disabled: %+v", body)
	}
	if body.EnvironmentMutateResources || body.EnvironmentWriteMcp {
		t.Fatalf("expected environment write capabilities to be disabled: %+v", body)
	}
	if !body.SettingsGlobalDefault || !body.SettingsMachineOverride {
		t.Fatalf("expected settings read/write capabilities to be enabled: %+v", body)
	}
	if body.SettingsApplyMachine {
		t.Fatalf("expected settings apply to be disabled without sender: %+v", body)
	}
	if body.MachineInstallAgent || body.MachineRemoveAgent || body.DashboardMetrics || body.AgentLifecycle {
		t.Fatalf("unexpected future capabilities enabled: %+v", body)
	}
}

func TestCapabilitiesEndpointEnablesCommandBackedFeatures(t *testing.T) {
	settingsStore := settings.NewMemoryStore([]domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	sender := &fakeCommandSender{
		resolveApprovalMachine: func(requestID string) (string, bool) {
			return "machine-01", true
		},
	}

	handler := NewServerWithSettings(
		registry.NewStore(),
		runtimeindex.NewStore(),
		routing.NewRouter(),
		sender,
		settingsStore,
		http.NotFoundHandler(),
		http.NotFoundHandler(),
	)

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("capabilities returned %d with %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Approvals                  bool `json:"approvals"`
		StartTurn                  bool `json:"startTurn"`
		SteerTurn                  bool `json:"steerTurn"`
		InterruptTurn              bool `json:"interruptTurn"`
		EnvironmentMutateResources bool `json:"environmentMutateResources"`
		EnvironmentWriteMcp        bool `json:"environmentWriteMcp"`
		SettingsApplyMachine       bool `json:"settingsApplyMachine"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid capabilities json: %v", err)
	}

	if !body.Approvals || !body.StartTurn || !body.SteerTurn || !body.InterruptTurn {
		t.Fatalf("expected command capabilities enabled: %+v", body)
	}
	if !body.EnvironmentMutateResources || !body.EnvironmentWriteMcp {
		t.Fatalf("expected environment write capabilities enabled: %+v", body)
	}
	if !body.SettingsApplyMachine {
		t.Fatalf("expected settings apply enabled: %+v", body)
	}
}

func TestConsoleSettingsEndpointsPersistPreferences(t *testing.T) {
	settingsStore := settings.NewMemoryStore([]domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	handler := NewServerWithSettings(
		registry.NewStore(),
		runtimeindex.NewStore(),
		routing.NewRouter(),
		nil,
		settingsStore,
		http.NotFoundHandler(),
		http.NotFoundHandler(),
	)

	req := httptest.NewRequest(http.MethodGet, "/settings/console", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("console settings get returned %d", rec.Code)
	}

	var getBody struct {
		Preferences *struct {
			ConsoleURL   string `json:"consoleUrl"`
			APIKey       string `json:"apiKey"`
			Profile      string `json:"profile"`
			SafetyPolicy string `json:"safetyPolicy"`
			LastThreadID string `json:"lastThreadId"`
		} `json:"preferences"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("invalid console settings json: %v", err)
	}
	if getBody.Preferences != nil {
		t.Fatalf("expected empty console preferences, got %+v", getBody.Preferences)
	}

	req = httptest.NewRequest(http.MethodPut, "/settings/console", bytes.NewBufferString(`{
  "preferences": {
    "consoleUrl": "http://localhost:3100",
    "apiKey": "test-key",
    "profile": "dev",
    "safetyPolicy": "strict",
    "lastThreadId": "thread-123"
  }
}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("console settings put returned %d with %s", rec.Code, rec.Body.String())
	}

	var putBody struct {
		Preferences *struct {
			ConsoleURL   string `json:"consoleUrl"`
			APIKey       string `json:"apiKey"`
			Profile      string `json:"profile"`
			SafetyPolicy string `json:"safetyPolicy"`
			LastThreadID string `json:"lastThreadId"`
		} `json:"preferences"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &putBody); err != nil {
		t.Fatalf("invalid console settings json: %v", err)
	}
	if putBody.Preferences == nil || putBody.Preferences.LastThreadID != "thread-123" {
		t.Fatalf("unexpected console preferences: %+v", putBody.Preferences)
	}

	req = httptest.NewRequest(http.MethodGet, "/settings/console", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("console settings get returned %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("invalid console settings json: %v", err)
	}
	if getBody.Preferences == nil || getBody.Preferences.ConsoleURL != "http://localhost:3100" {
		t.Fatalf("expected persisted console preferences, got %+v", getBody.Preferences)
	}
}

func TestServerApplySettingsUsesMachineOverrideBeforeGlobalDefault(t *testing.T) {
	settingsStore := settings.NewMemoryStore([]domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	if err := settingsStore.PutGlobal(domain.AgentTypeCodex, domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.4\"\n",
	}); err != nil {
		t.Fatal(err)
	}
	if err := settingsStore.PutMachine("machine-01", domain.AgentTypeCodex, domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   "model = \"gpt-5.2\"\n",
	}); err != nil {
		t.Fatal(err)
	}

	var call struct {
		machineID string
		name      string
		payload   protocol.AgentConfigApplyCommandPayload
	}
	sender := &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			call.machineID = machineID
			call.name = name
			commandPayload, ok := payload.(protocol.AgentConfigApplyCommandPayload)
			if !ok {
				t.Fatalf("unexpected payload type: %T", payload)
			}
			call.payload = commandPayload
			return protocol.CommandCompletedPayload{
				CommandName: name,
				Result: mustMarshalJSON(t, protocol.AgentConfigApplyCommandResult{
					AgentType: "codex",
					FilePath:  "/tmp/.codex/config.toml",
					Source:    commandPayload.Source,
				}),
			}, nil
		},
	}

	handler := NewServerWithSettings(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), sender, settingsStore, http.NotFoundHandler(), http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodPost, "/settings/machines/machine-01/agents/codex/apply", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("apply returned %d with %s", rec.Code, rec.Body.String())
	}
	if call.machineID != "machine-01" || call.name != "agent.config.apply" {
		t.Fatalf("unexpected command call: %+v", call)
	}
	if call.payload.Source != "machine" || call.payload.Document.Content != "model = \"gpt-5.2\"\n" {
		t.Fatalf("expected machine override payload, got %+v", call.payload)
	}

	if err := settingsStore.DeleteMachine("machine-01", domain.AgentTypeCodex); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/settings/machines/machine-01/agents/codex/apply", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("apply without override returned %d with %s", rec.Code, rec.Body.String())
	}
	if call.payload.Source != "global" || call.payload.Document.Content != "model = \"gpt-5.4\"\n" {
		t.Fatalf("expected global default payload, got %+v", call.payload)
	}
}

func TestServerSettingsRejectInvalidTOML(t *testing.T) {
	settingsStore := settings.NewMemoryStore([]domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	handler := NewServerWithSettings(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), &fakeCommandSender{}, settingsStore, http.NotFoundHandler(), http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodPut, "/settings/agents/codex/global", bytes.NewBufferString(`{"content":"model = ["}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body != "content must be valid toml\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestServerThreadDetailFallsBackToRuntimeIndexWhenRouterHasNotTrackedThreadYet(t *testing.T) {
	idx := runtimeindex.NewStore()
	idx.UpsertThread("machine-01", domain.Thread{
		ThreadID:  "thread-01",
		MachineID: "machine-01",
		Status:    domain.ThreadStatusIdle,
		Title:     "Fallback thread",
	})

	handler := NewServer(registry.NewStore(), idx, routing.NewRouter(), &fakeCommandSender{}, http.NotFoundHandler(), http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/threads/thread-01", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Thread domain.Thread `json:"thread"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Thread.ThreadID != "thread-01" || body.Thread.MachineID != "machine-01" {
		t.Fatalf("unexpected body: %+v", body)
	}
}
