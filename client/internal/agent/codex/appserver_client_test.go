package codex

import (
	"encoding/json"
	"errors"
	"testing"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type fakeRunner struct {
	call                 func(method string, payload any, out any) error
	notificationHandler  func(jsonRPCNotification)
	serverRequestHandler func(jsonRPCServerRequest)
	respond              func(id json.RawMessage, result any, rpcErr *jsonRPCError) error
}

func (r *fakeRunner) Call(method string, payload any, out any) error {
	if r.call != nil {
		return r.call(method, payload, out)
	}
	return nil
}

func (r *fakeRunner) SetNotificationHandler(handler func(jsonRPCNotification)) {
	r.notificationHandler = handler
}

func (r *fakeRunner) SetServerRequestHandler(handler func(jsonRPCServerRequest)) {
	r.serverRequestHandler = handler
}

func (r *fakeRunner) Respond(id json.RawMessage, result any, rpcErr *jsonRPCError) error {
	if r.respond != nil {
		return r.respond(id, result, rpcErr)
	}
	return nil
}

func (r *fakeRunner) emitNotification(t *testing.T, method string, params any) {
	t.Helper()

	if r.notificationHandler == nil {
		t.Fatal("notification handler not registered")
	}

	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal notification params failed: %v", err)
	}

	r.notificationHandler(jsonRPCNotification{
		Method: method,
		Params: raw,
	})
}

func (r *fakeRunner) emitServerRequest(t *testing.T, id string, method string, params any) {
	t.Helper()

	if r.serverRequestHandler == nil {
		t.Fatal("server request handler not registered")
	}

	rawParams, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal server request params failed: %v", err)
	}

	rawID, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("marshal server request id failed: %v", err)
	}

	r.serverRequestHandler(jsonRPCServerRequest{
		ID:     rawID,
		Method: method,
		Params: rawParams,
	})
}

func TestClientListThreads(t *testing.T) {
	tests := []struct {
		name    string
		callErr error
	}{
		{
			name: "maps records",
		},
		{
			name:    "propagates runner error",
			callErr: errors.New("runner boom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					if method != "thread/list" {
						t.Fatalf("unexpected method: %s", method)
					}
					if tt.callErr != nil {
						return tt.callErr
					}
					response := out.(*threadListResponse)
					response.Data = []threadRecord{{
						ID:     "thread-1",
						Name:   "Investigate flaky test",
						Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
					}}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			threads, err := client.ListThreads()
			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(threads) != 1 {
				t.Fatalf("unexpected thread count: %d", len(threads))
			}
			if threads[0].ThreadID != "thread-1" {
				t.Fatalf("unexpected threads: %+v", threads)
			}
		})
	}
}

func TestClientListEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		callErr error
	}{
		{
			name: "maps records",
		},
		{
			name:    "propagates runner error",
			callErr: errors.New("environment list failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					calls = append(calls, method)
					if tt.callErr != nil {
						return tt.callErr
					}
					switch method {
					case "skills/list":
						response := out.(*skillsListResponse)
						response.Data = []skillsListEntry{
							{
								Cwd: "/tmp/project",
								Skills: []skillMetadata{
									{Name: "skill-a", Enabled: true, Path: "/tmp/project/.codex/skills/skill-a/SKILL.md"},
								},
							},
						}
					case "mcpServerStatus/list":
						response := out.(*mcpServerStatusListResponse)
						response.Data = []mcpServerStatusRecord{
							{Name: "github", Enabled: true},
						}
					case "plugin/list":
						response := out.(*pluginListResponse)
						response.Marketplaces = []pluginMarketplaceEntry{
							{
								Name: "local",
								Plugins: []pluginSummary{
									{ID: "plugin-a", Name: "Plugin A", Enabled: true, Installed: true},
								},
							},
						}
					default:
						t.Fatalf("unexpected method: %s", method)
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			environment, err := client.ListEnvironment()
			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(calls) != 3 || calls[0] != "skills/list" || calls[1] != "mcpServerStatus/list" || calls[2] != "plugin/list" {
				t.Fatalf("unexpected calls: %#v", calls)
			}
			if len(environment) != 3 {
				t.Fatalf("unexpected environment count: %d", len(environment))
			}
			if environment[0].ResourceID != "skill-a" || environment[0].Kind != domain.EnvironmentKindSkill {
				t.Fatalf("unexpected skill environment: %+v", environment[0])
			}
			if environment[1].ResourceID != "github" || environment[1].Kind != domain.EnvironmentKindMCP {
				t.Fatalf("unexpected mcp environment: %+v", environment[1])
			}
			if environment[2].ResourceID != "plugin-a" || environment[2].Kind != domain.EnvironmentKindPlugin {
				t.Fatalf("unexpected environment: %+v", environment)
			}
		})
	}
}

func TestClientSetSkillEnabledUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name       string
		nameOrPath string
		enabled    bool
		callErr    error
	}{
		{
			name:       "enables a skill by name",
			nameOrPath: "skill-a",
			enabled:    true,
		},
		{
			name:       "propagates runner error",
			nameOrPath: "/tmp/project/.codex/skills/skill-a/SKILL.md",
			enabled:    false,
			callErr:    errors.New("skill update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}

					response := out.(*skillsConfigWriteResponse)
					response.Data = []skillMetadata{
						{Name: "skill-a", Path: tt.nameOrPath, Enabled: tt.enabled},
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			err := client.SetSkillEnabled(tt.nameOrPath, tt.enabled)
			if gotMethod != "skills/config/write" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["nameOrPath"] != tt.nameOrPath || payloadMap["enabled"] != tt.enabled {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientUninstallPluginUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name     string
		pluginID string
		callErr  error
	}{
		{
			name:     "uses plugin/uninstall payload keys",
			pluginID: "plugin-a",
		},
		{
			name:     "propagates runner error",
			pluginID: "plugin-b",
			callErr:  errors.New("plugin uninstall failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			err := client.UninstallPlugin(tt.pluginID)
			if gotMethod != "plugin/uninstall" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["pluginId"] != tt.pluginID {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientCreateThreadUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		callErr error
	}{
		{
			name:  "uses thread/start payload keys",
			title: "Investigate flaky test",
		},
		{
			name:    "propagates runner error",
			title:   "Build failure",
			callErr: errors.New("thread start failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					response := out.(*threadStartResponse)
					response.Thread = threadRecord{
						ID:     "thread-1",
						Name:   tt.title,
						Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			thread, err := client.CreateThread(agenttypes.CreateThreadParams{Title: tt.title})
			if gotMethod != "thread/start" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["title"] != tt.title {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}
			if payloadMap["experimentalRawEvents"] != false || payloadMap["persistExtendedHistory"] != false {
				t.Fatalf("expected thread/start defaults in payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if thread.ThreadID != "thread-1" || thread.Title != tt.title {
				t.Fatalf("unexpected thread: %+v", thread)
			}
		})
	}
}

func TestClientStartTurnUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name     string
		threadID string
		prompt   string
		callErr  error
	}{
		{
			name:     "uses turn/start payload keys",
			threadID: "thread-1",
			prompt:   "run tests",
		},
		{
			name:     "propagates runner error",
			threadID: "thread-2",
			prompt:   "build",
			callErr:  errors.New("turn start failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					response := out.(*turnStartResponse)
					response.Turn = turnRecord{ID: "turn-1"}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			result, err := client.StartTurn(agenttypes.StartTurnParams{
				ThreadID: tt.threadID,
				Input:    tt.prompt,
			})
			if gotMethod != "turn/start" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			inputs, ok := payloadMap["input"].([]map[string]any)
			if !ok {
				t.Fatalf("unexpected input payload type: %T", payloadMap["input"])
			}
			if payloadMap["threadId"] != tt.threadID || len(inputs) != 1 || inputs[0]["text"] != tt.prompt {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}
			if inputs[0]["type"] != "text" {
				t.Fatalf("unexpected input payload: %#v", inputs[0])
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.TurnID != "turn-1" || result.ThreadID != tt.threadID {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

func TestAppServerClientTranslatesNotificationsIntoTurnEvents(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)

	events := make([]agenttypes.RuntimeTurnEvent, 0, 4)
	client.SetTurnEventHandler(func(event agenttypes.RuntimeTurnEvent) {
		events = append(events, event)
	})

	runner.emitNotification(t, "turn/started", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
	})
	runner.emitNotification(t, "item/agentMessage/delta", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"delta":    "hello",
	})
	runner.emitNotification(t, "item/agentMessage/delta", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"delta":    " world",
	})
	runner.emitNotification(t, "turn/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
	})

	if len(events) != 4 {
		t.Fatalf("unexpected event count: %d", len(events))
	}
	if events[0].Type != agenttypes.RuntimeTurnEventTypeStarted || events[0].ThreadID != "thread-1" || events[0].TurnID != "turn-1" {
		t.Fatalf("unexpected started event: %+v", events[0])
	}
	if events[1].Type != agenttypes.RuntimeTurnEventTypeDelta || events[1].Sequence != 1 || events[1].Delta != "hello" {
		t.Fatalf("unexpected first delta event: %+v", events[1])
	}
	if events[2].Type != agenttypes.RuntimeTurnEventTypeDelta || events[2].Sequence != 2 || events[2].Delta != " world" {
		t.Fatalf("unexpected second delta event: %+v", events[2])
	}
	if events[3].Type != agenttypes.RuntimeTurnEventTypeCompleted {
		t.Fatalf("unexpected completed event: %+v", events[3])
	}
	if events[3].Turn.ThreadID != "thread-1" || events[3].Turn.TurnID != "turn-1" || events[3].Turn.Status != domain.TurnStatusCompleted {
		t.Fatalf("unexpected completed turn: %+v", events[3].Turn)
	}
}

func TestAppServerClientRespondApprovalWritesStoredServerRequestResponse(t *testing.T) {
	var gotID string
	var gotResult map[string]any

	runner := &fakeRunner{
		respond: func(id json.RawMessage, result any, rpcErr *jsonRPCError) error {
			if rpcErr != nil {
				t.Fatalf("unexpected rpc error: %+v", rpcErr)
			}
			if err := json.Unmarshal(id, &gotID); err != nil {
				t.Fatalf("unmarshal id failed: %v", err)
			}

			raw, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("marshal result failed: %v", err)
			}
			if err := json.Unmarshal(raw, &gotResult); err != nil {
				t.Fatalf("unmarshal result failed: %v", err)
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	runner.emitServerRequest(t, "approval-1", "item/commandExecution/requestApproval", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"itemId":   "item-1",
		"command":  "go test ./...",
	})

	if err := client.RespondApproval("approval-1", "accept"); err != nil {
		t.Fatal(err)
	}

	if gotID != "approval-1" {
		t.Fatalf("unexpected response id: %q", gotID)
	}
	if gotResult["decision"] != "accept" {
		t.Fatalf("unexpected response payload: %#v", gotResult)
	}
}

func TestAppServerClientRespondApprovalMapsPermissionsRequests(t *testing.T) {
	tests := []struct {
		name            string
		decision        string
		wantScope       string
		wantPermissions map[string]any
	}{
		{
			name:      "accept grants requested permissions for the session",
			decision:  "accept",
			wantScope: "session",
			wantPermissions: map[string]any{
				"fs.read": true,
				"net":     true,
			},
		},
		{
			name:            "decline returns an empty grant",
			decision:        "decline",
			wantScope:       "turn",
			wantPermissions: map[string]any{},
		},
		{
			name:            "cancel returns an empty grant",
			decision:        "cancel",
			wantScope:       "turn",
			wantPermissions: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotID string
			var gotResult map[string]any

			runner := &fakeRunner{
				respond: func(id json.RawMessage, result any, rpcErr *jsonRPCError) error {
					if rpcErr != nil {
						t.Fatalf("unexpected rpc error: %+v", rpcErr)
					}
					if err := json.Unmarshal(id, &gotID); err != nil {
						t.Fatalf("unmarshal id failed: %v", err)
					}

					raw, err := json.Marshal(result)
					if err != nil {
						t.Fatalf("marshal result failed: %v", err)
					}
					if err := json.Unmarshal(raw, &gotResult); err != nil {
						t.Fatalf("unmarshal result failed: %v", err)
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			runner.emitServerRequest(t, "approval-1", "item/permissions/requestApproval", map[string]any{
				"threadId":    "thread-1",
				"turnId":      "turn-1",
				"itemId":      "item-1",
				"session":     "session-1",
				"permissions": map[string]any{"fs.read": true, "net": true},
			})

			if err := client.RespondApproval("approval-1", tt.decision); err != nil {
				t.Fatal(err)
			}

			if gotID != "approval-1" {
				t.Fatalf("unexpected response id: %q", gotID)
			}
			if gotResult["scope"] != tt.wantScope {
				t.Fatalf("unexpected scope: %#v", gotResult)
			}
			gotPermissions, ok := gotResult["permissions"].(map[string]any)
			if !ok {
				t.Fatalf("unexpected permissions payload: %#v", gotResult)
			}
			if len(gotPermissions) != len(tt.wantPermissions) {
				t.Fatalf("unexpected permissions length: %#v", gotResult)
			}
			for key, want := range tt.wantPermissions {
				if gotPermissions[key] != want {
					t.Fatalf("permissions[%q] = %#v, want %#v", key, gotPermissions[key], want)
				}
			}
		})
	}
}

func TestClientListEnvironmentMapsMcpInventory(t *testing.T) {
	var calls []string
	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			calls = append(calls, method)
			switch method {
			case "skills/list":
				response := out.(*skillsListResponse)
				response.Data = nil
			case "mcpServerStatus/list":
				response := out.(*mcpServerStatusListResponse)
				response.Data = []mcpServerStatusRecord{
					{Name: "github", Enabled: true},
					{Name: "postgres", Enabled: false},
				}
			case "plugin/list":
				response := out.(*pluginListResponse)
				response.Marketplaces = nil
			default:
				t.Fatalf("unexpected method: %s", method)
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	environment, err := client.ListEnvironment()
	if err != nil {
		t.Fatal(err)
	}

	if len(calls) != 3 || calls[0] != "skills/list" || calls[1] != "mcpServerStatus/list" || calls[2] != "plugin/list" {
		t.Fatalf("unexpected calls: %#v", calls)
	}
	if len(environment) != 2 {
		t.Fatalf("unexpected environment count: %d", len(environment))
	}
	if environment[0].Kind != domain.EnvironmentKindMCP || environment[0].ResourceID != "github" || environment[0].Status != domain.EnvironmentResourceStatusEnabled {
		t.Fatalf("unexpected first mcp entry: %+v", environment[0])
	}
	if environment[1].Kind != domain.EnvironmentKindMCP || environment[1].ResourceID != "postgres" || environment[1].Status != domain.EnvironmentResourceStatusDisabled {
		t.Fatalf("unexpected second mcp entry: %+v", environment[1])
	}
}

func TestClientReadThreadUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name     string
		threadID string
		callErr  error
	}{
		{
			name:     "uses thread/read payload keys",
			threadID: "thread-1",
		},
		{
			name:     "propagates runner error",
			threadID: "thread-2",
			callErr:  errors.New("thread read failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					response := out.(*threadReadResponse)
					response.Thread = threadRecord{
						ID:     tt.threadID,
						Name:   "Investigate flaky test",
						Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			thread, err := client.ReadThread(tt.threadID)
			if gotMethod != "thread/read" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["threadId"] != tt.threadID || payloadMap["includeTurns"] != false {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if thread.ThreadID != tt.threadID || thread.Title != "Investigate flaky test" {
				t.Fatalf("unexpected thread: %+v", thread)
			}
		})
	}
}

func TestClientResumeThreadUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name     string
		threadID string
		callErr  error
	}{
		{
			name:     "uses thread/resume payload keys",
			threadID: "thread-1",
		},
		{
			name:     "propagates runner error",
			threadID: "thread-2",
			callErr:  errors.New("thread resume failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					response := out.(*threadResumeResponse)
					response.Thread = threadRecord{
						ID:     tt.threadID,
						Name:   "Resumed thread",
						Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			thread, err := client.ResumeThread(tt.threadID)
			if gotMethod != "thread/resume" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["threadId"] != tt.threadID || payloadMap["persistExtendedHistory"] != false {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if thread.ThreadID != tt.threadID || thread.Title != "Resumed thread" {
				t.Fatalf("unexpected thread: %+v", thread)
			}
		})
	}
}

func TestClientArchiveThreadUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name     string
		threadID string
		callErr  error
	}{
		{
			name:     "uses thread/archive payload keys",
			threadID: "thread-1",
		},
		{
			name:     "propagates runner error",
			threadID: "thread-2",
			callErr:  errors.New("thread archive failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			err := client.ArchiveThread(tt.threadID)
			if gotMethod != "thread/archive" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["threadId"] != tt.threadID {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientSteerTurnUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name     string
		threadID string
		turnID   string
		input    string
		callErr  error
	}{
		{
			name:     "uses turn/steer payload keys",
			threadID: "thread-1",
			turnID:   "turn-1",
			input:    "try a different fix",
		},
		{
			name:     "propagates runner error",
			threadID: "thread-2",
			turnID:   "turn-2",
			input:    "stop and summarize",
			callErr:  errors.New("turn steer failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			result, err := client.SteerTurn(agenttypes.SteerTurnParams{
				ThreadID: tt.threadID,
				TurnID:   tt.turnID,
				Input:    tt.input,
			})
			if gotMethod != "turn/steer" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			inputs, ok := payloadMap["input"].([]map[string]any)
			if !ok {
				t.Fatalf("unexpected input payload type: %T", payloadMap["input"])
			}
			if payloadMap["threadId"] != tt.threadID || payloadMap["expectedTurnId"] != tt.turnID {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}
			if len(inputs) != 1 || inputs[0]["text"] != tt.input || inputs[0]["type"] != "text" {
				t.Fatalf("unexpected input payload: %#v", inputs)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.ThreadID != tt.threadID || result.TurnID != tt.turnID {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

func TestClientInterruptTurnUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name     string
		threadID string
		turnID   string
		callErr  error
	}{
		{
			name:     "uses turn/interrupt payload keys",
			threadID: "thread-1",
			turnID:   "turn-1",
		},
		{
			name:     "propagates runner error",
			threadID: "thread-2",
			turnID:   "turn-2",
			callErr:  errors.New("turn interrupt failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			turn, err := client.InterruptTurn(agenttypes.InterruptTurnParams{
				ThreadID: tt.threadID,
				TurnID:   tt.turnID,
			})
			if gotMethod != "turn/interrupt" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["threadId"] != tt.threadID || payloadMap["turnId"] != tt.turnID {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if turn.ThreadID != tt.threadID || turn.TurnID != tt.turnID || turn.Status != domain.TurnStatusInterrupted {
				t.Fatalf("unexpected turn: %+v", turn)
			}
		})
	}
}
