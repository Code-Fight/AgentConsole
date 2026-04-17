package codex

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

func TestClientListThreadsIncludesRecentlyCreatedThreadWhenListLagsBehind(t *testing.T) {
	var calls []string
	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			calls = append(calls, method)
			switch method {
			case "thread/start":
				response := out.(*threadStartResponse)
				response.Thread = threadRecord{
					ID:     "thread-1",
					Name:   "Investigate flaky test",
					Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
				}
			case "thread/list":
				response := out.(*threadListResponse)
				response.Data = nil
			default:
				t.Fatalf("unexpected method: %s", method)
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	thread, err := client.CreateThread(agenttypes.CreateThreadParams{Title: "Investigate flaky test"})
	if err != nil {
		t.Fatal(err)
	}
	if thread.ThreadID != "thread-1" {
		t.Fatalf("unexpected created thread: %+v", thread)
	}

	threads, err := client.ListThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || threads[0].ThreadID != "thread-1" {
		t.Fatalf("expected cached thread to remain visible, got %+v", threads)
	}
	if len(calls) != 2 || calls[0] != "thread/start" || calls[1] != "thread/list" {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestClientCreateThreadFallsBackToRequestedTitleWhenResponseIsBlank(t *testing.T) {
	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			if method != "thread/start" {
				t.Fatalf("unexpected method: %s", method)
			}
			response := out.(*threadStartResponse)
			response.Thread = threadRecord{
				ID:     "thread-blank-title",
				Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	thread, err := client.CreateThread(agenttypes.CreateThreadParams{Title: "Investigate flaky test"})
	if err != nil {
		t.Fatal(err)
	}

	if thread.Title != "Investigate flaky test" {
		t.Fatalf("expected requested title fallback, got %+v", thread)
	}
}

func TestClientListThreadsPreservesCachedTitleWhenListEntryIsBlank(t *testing.T) {
	var calls []string
	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			calls = append(calls, method)
			switch method {
			case "thread/start":
				response := out.(*threadStartResponse)
				response.Thread = threadRecord{
					ID:     "thread-1",
					Name:   "Investigate flaky test",
					Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
				}
			case "thread/list":
				response := out.(*threadListResponse)
				response.Data = []threadRecord{
					{
						ID:     "thread-1",
						Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
					},
				}
			default:
				t.Fatalf("unexpected method: %s", method)
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	if _, err := client.CreateThread(agenttypes.CreateThreadParams{Title: "Investigate flaky test"}); err != nil {
		t.Fatal(err)
	}

	threads, err := client.ListThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 {
		t.Fatalf("unexpected thread count: %d", len(threads))
	}
	if threads[0].Title != "Investigate flaky test" {
		t.Fatalf("expected cached title to be preserved, got %+v", threads[0])
	}
	if len(calls) != 2 || calls[0] != "thread/start" || calls[1] != "thread/list" {
		t.Fatalf("unexpected calls: %#v", calls)
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
								Path: "/tmp/codex/marketplace",
								Plugins: []pluginSummary{
									{ID: "plugin-a", Name: "plugin-a", Enabled: true, Installed: true},
								},
							},
						}
					case "config/read":
						response := out.(*configReadResponse)
						response.Config = map[string]any{
							"mcp_servers": map[string]any{
								"github": map[string]any{
									"command": "npx",
									"args":    []any{"-y", "@modelcontextprotocol/server-github"},
								},
							},
						}
					case "plugin/read":
						response := out.(*pluginReadResponse)
						response.Plugin = pluginDetail{
							MarketplaceName: "local",
							MarketplacePath: "/tmp/codex/marketplace",
							Summary: pluginSummary{
								ID:        "plugin-a",
								Name:      "plugin-a",
								Installed: true,
								Enabled:   true,
							},
							Description: "Plugin A description",
							Skills: []skillSummary{
								{Name: "skill-a", Description: "Skill A"},
							},
							MCPServers: []string{"github"},
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
			if len(calls) != 5 ||
				calls[0] != "skills/list" ||
				calls[1] != "mcpServerStatus/list" ||
				calls[2] != "plugin/list" ||
				calls[3] != "config/read" ||
				calls[4] != "plugin/read" {
				t.Fatalf("unexpected calls: %#v", calls)
			}
			if len(environment) != 3 {
				t.Fatalf("unexpected environment count: %d", len(environment))
			}
			if environment[0].ResourceID != "/tmp/project/.codex/skills/skill-a/SKILL.md" || environment[0].Kind != domain.EnvironmentKindSkill {
				t.Fatalf("unexpected skill environment: %+v", environment[0])
			}
			if environment[0].DisplayName != "skill-a" {
				t.Fatalf("unexpected skill display name: %+v", environment[0])
			}
			if environment[1].ResourceID != "github" || environment[1].Kind != domain.EnvironmentKindMCP {
				t.Fatalf("unexpected mcp environment: %+v", environment[1])
			}
			if environment[1].Details["command"] != "npx" {
				t.Fatalf("expected mcp config details, got %+v", environment[1])
			}
			if environment[2].ResourceID != "plugin-a" || environment[2].Kind != domain.EnvironmentKindPlugin {
				t.Fatalf("unexpected environment: %+v", environment)
			}
			if environment[2].Details["marketplacePath"] != "/tmp/codex/marketplace" {
				t.Fatalf("expected plugin marketplace details, got %+v", environment[2])
			}
			if environment[2].Details["description"] != "Plugin A description" {
				t.Fatalf("expected plugin description details, got %+v", environment[2])
			}
		})
	}
}

func TestClientSetSkillEnabledUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name        string
		nameOrPath  string
		selectorKey string
		enabled     bool
		callErr     error
	}{
		{
			name:        "enables a skill by name",
			nameOrPath:  "skill-a",
			selectorKey: "name",
			enabled:     true,
		},
		{
			name:        "disables a skill by path",
			nameOrPath:  "/tmp/project/.codex/skills/skill-a/SKILL.md",
			selectorKey: "path",
			enabled:     false,
		},
		{
			name:        "propagates runner error",
			nameOrPath:  "/tmp/project/.codex/skills/skill-a/SKILL.md",
			selectorKey: "path",
			enabled:     false,
			callErr:     errors.New("skill update failed"),
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
			if payloadMap[tt.selectorKey] != tt.nameOrPath || payloadMap["enabled"] != tt.enabled {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}
			if len(payloadMap) != 2 {
				t.Fatalf("unexpected payload keys: %#v", payloadMap)
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

func TestClientCreateSkillScaffoldWritesDefaultFile(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)
	tmpDir := t.TempDir()
	client.homeDir = func() (string, error) {
		return tmpDir, nil
	}

	skillID, err := client.CreateSkill(agenttypes.CreateSkillParams{
		Name:        "Debug Helper!!",
		Description: "Assists debugging\nHandles tricky cases",
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := filepath.Join(tmpDir, ".codex", "skills", "debug-helper", "SKILL.md")
	if skillID != expectedPath {
		t.Fatalf("expected skill path %q, got %q", expectedPath, skillID)
	}

	contents, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read scaffold file: %v", err)
	}

	expected := `---
name: "Debug Helper!!"
description: "Assists debugging\nHandles tricky cases"
---

# Debug Helper!!

Assists debugging
Handles tricky cases

## Usage

Add task-specific instructions here.
`
	if string(contents) != expected {
		t.Fatalf("unexpected scaffold contents:\n%s", string(contents))
	}
}

func TestClientCreateSkillScaffoldDoesNotOverwriteExistingSkill(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)
	tmpDir := t.TempDir()
	client.homeDir = func() (string, error) {
		return tmpDir, nil
	}

	skillDir := filepath.Join(tmpDir, ".codex", "skills", "debug-helper")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("original"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := client.CreateSkill(agenttypes.CreateSkillParams{
		Name:        "Debug Helper",
		Description: "Updated",
	})
	if err == nil {
		t.Fatal("expected error when skill scaffold exists")
	}

	contents, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(contents) != "original" {
		t.Fatalf("expected original contents preserved, got %q", string(contents))
	}
}

func TestClientDeleteSkillScaffoldRemovesDirectory(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)
	tmpDir := t.TempDir()
	client.homeDir = func() (string, error) {
		return tmpDir, nil
	}

	tests := []struct {
		name      string
		selector  string
		skillSlug string
	}{
		{
			name:      "delete by path",
			selector:  filepath.Join(tmpDir, ".codex", "skills", "skill-a", "SKILL.md"),
			skillSlug: "skill-a",
		},
		{
			name:      "delete by name",
			selector:  "Skill A",
			skillSlug: "skill-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skillDir := filepath.Join(tmpDir, ".codex", "skills", tt.skillSlug)
			if err := os.MkdirAll(skillDir, 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("stub"), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}

			if err := client.DeleteSkill(tt.selector); err != nil {
				t.Fatalf("delete skill: %v", err)
			}

			if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
				t.Fatalf("expected skill dir removed, stat err: %v", err)
			}
		})
	}
}

func TestClientUpsertMCPUsesConfigWriteAndReload(t *testing.T) {
	var calls []string
	var gotPayloads []any

	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			calls = append(calls, method)
			gotPayloads = append(gotPayloads, payload)
			switch method {
			case "config/value/write":
				response := out.(*configWriteResponse)
				response.Version = "v1"
				response.FilePath = "/tmp/codex/config.toml"
			case "config/mcpServer/reload":
			default:
				t.Fatalf("unexpected method: %s", method)
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	err := client.UpsertMCPServer("github", map[string]any{
		"command": "npx",
		"args":    []any{"-y", "@modelcontextprotocol/server-github"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(calls) != 2 || calls[0] != "config/value/write" || calls[1] != "config/mcpServer/reload" {
		t.Fatalf("unexpected calls: %#v", calls)
	}

	payload, ok := gotPayloads[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", gotPayloads[0])
	}
	if payload["keyPath"] != "mcp_servers.github" {
		t.Fatalf("unexpected keyPath payload: %#v", payload)
	}
	if payload["mergeStrategy"] != "replace" {
		t.Fatalf("unexpected merge strategy payload: %#v", payload)
	}
}

func TestClientSetMCPEnabledUsesExistingConfigAndReload(t *testing.T) {
	var calls []string
	var gotPayload any

	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			calls = append(calls, method)
			switch method {
			case "config/read":
				response := out.(*configReadResponse)
				response.Config = map[string]any{
					"mcp_servers": map[string]any{
						"github": map[string]any{
							"command": "npx",
							"enabled": true,
						},
					},
				}
			case "config/value/write":
				gotPayload = payload
				response := out.(*configWriteResponse)
				response.Version = "v1"
				response.FilePath = "/tmp/codex/config.toml"
			case "config/mcpServer/reload":
			default:
				t.Fatalf("unexpected method: %s", method)
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	if err := client.SetMCPServerEnabled("github", false); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 3 || calls[0] != "config/read" || calls[1] != "config/value/write" || calls[2] != "config/mcpServer/reload" {
		t.Fatalf("unexpected calls: %#v", calls)
	}
	payload, ok := gotPayload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", gotPayload)
	}
	value, ok := payload["value"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected value payload: %#v", payload)
	}
	if value["enabled"] != false {
		t.Fatalf("expected updated enabled flag, got %#v", value)
	}
}

func TestClientRemoveMCPRewritesConfigAndReloads(t *testing.T) {
	var calls []string
	var gotPayload any

	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			calls = append(calls, method)
			switch method {
			case "config/read":
				response := out.(*configReadResponse)
				response.Config = map[string]any{
					"mcp_servers": map[string]any{
						"github":   map[string]any{"command": "npx"},
						"postgres": map[string]any{"command": "docker"},
					},
				}
			case "config/value/write":
				gotPayload = payload
				response := out.(*configWriteResponse)
				response.Version = "v1"
				response.FilePath = "/tmp/codex/config.toml"
			case "config/mcpServer/reload":
			default:
				t.Fatalf("unexpected method: %s", method)
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	if err := client.RemoveMCPServer("github"); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 3 {
		t.Fatalf("unexpected calls: %#v", calls)
	}
	payload, ok := gotPayload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", gotPayload)
	}
	if payload["keyPath"] != "mcp_servers" {
		t.Fatalf("unexpected keyPath payload: %#v", payload)
	}
	value, ok := payload["value"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected value payload: %#v", payload)
	}
	if _, exists := value["github"]; exists {
		t.Fatalf("expected removed server to be absent, got %#v", value)
	}
	if _, exists := value["postgres"]; !exists {
		t.Fatalf("expected other server to remain, got %#v", value)
	}
}

func TestClientInstallPluginUsesExpectedMethodAndPayload(t *testing.T) {
	var gotMethod string
	var gotPayload any

	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			gotMethod = method
			gotPayload = payload
			response := out.(*pluginInstallResponse)
			response.AuthPolicy = "never"
			return nil
		},
	}

	client := NewAppServerClient(runner)
	err := client.InstallPlugin(agenttypes.InstallPluginParams{
		PluginID:        "plugin-a",
		MarketplacePath: "/tmp/codex/marketplace",
		PluginName:      "plugin-a",
	})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != "plugin/install" {
		t.Fatalf("unexpected method: %s", gotMethod)
	}
	payload, ok := gotPayload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", gotPayload)
	}
	if payload["marketplacePath"] != "/tmp/codex/marketplace" || payload["pluginName"] != "plugin-a" {
		t.Fatalf("unexpected install payload: %#v", payload)
	}
}

func TestClientSetPluginEnabledWritesConfig(t *testing.T) {
	var calls []string
	var gotPayload any

	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			calls = append(calls, method)
			switch method {
			case "config/read":
				response := out.(*configReadResponse)
				response.Config = map[string]any{
					"plugins": map[string]any{
						"plugin-a": map[string]any{
							"enabled": true,
						},
					},
				}
			case "config/value/write":
				gotPayload = payload
				response := out.(*configWriteResponse)
				response.Version = "v1"
				response.FilePath = "/tmp/codex/config.toml"
			default:
				t.Fatalf("unexpected method: %s", method)
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	if err := client.SetPluginEnabled("plugin-a", false); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 2 || calls[0] != "config/read" || calls[1] != "config/value/write" {
		t.Fatalf("unexpected calls: %#v", calls)
	}
	payload, ok := gotPayload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", gotPayload)
	}
	if payload["keyPath"] != "plugins.plugin-a" {
		t.Fatalf("unexpected keyPath payload: %#v", payload)
	}
	value, ok := payload["value"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected value payload: %#v", payload)
	}
	if value["enabled"] != false {
		t.Fatalf("expected updated enabled flag, got %#v", value)
	}
}

func TestAppServerClientSupportsToolUserInputApprovalRequests(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)

	var got agenttypes.RuntimeApprovalRequest
	client.SetApprovalHandler(func(event agenttypes.RuntimeApprovalRequest) {
		got = event
	})

	runner.emitServerRequest(t, "approval-1", "item/tool/requestUserInput", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"itemId":   "item-1",
		"questions": []map[string]any{
			{
				"id":       "question-1",
				"question": "Pick an option",
				"options": []map[string]any{
					{"value": "option-a", "label": "Option A"},
					{"value": "option-b", "label": "Option B"},
				},
			},
		},
	})

	if got.RequestID != "approval-1" {
		t.Fatalf("unexpected approval request: %+v", got)
	}
	if got.ThreadID != "thread-1" || got.TurnID != "turn-1" || got.ItemID != "item-1" {
		t.Fatalf("unexpected approval scope: %+v", got)
	}
	if got.Kind != "tool_user_input" {
		t.Fatalf("unexpected approval kind: %+v", got)
	}
	if got.Reason != "Pick an option" {
		t.Fatalf("unexpected approval reason: %+v", got)
	}
}

func TestAppServerClientPreservesToolUserInputQuestionMetadata(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)

	runner.emitServerRequest(t, "approval-1", "item/tool/requestUserInput", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"itemId":   "item-1",
		"questions": []map[string]any{
			{
				"id":       "question-1",
				"header":   "Git branch",
				"question": "Pick a branch",
				"options": []map[string]any{
					{"value": "main", "label": "Main"},
					{"value": "release", "label": "Release"},
				},
			},
		},
	})

	pending, ok := client.pendingApprovals["approval-1"]
	if !ok {
		t.Fatal("expected pending approval to be stored")
	}
	if len(pending.request.UserInputQuestions) != 1 {
		t.Fatalf("unexpected questions: %+v", pending.request.UserInputQuestions)
	}

	question := pending.request.UserInputQuestions[0]
	if question.Key != "question-1" || question.Header != "Git branch" || question.Text != "Pick a branch" {
		t.Fatalf("unexpected question metadata: %+v", question)
	}
	if len(question.Options) != 2 || question.Options[0] != "main" || question.Options[1] != "release" {
		t.Fatalf("unexpected question options: %+v", question)
	}
}

func TestAppServerClientRespondApprovalMapsToolUserInputResponses(t *testing.T) {
	tests := []struct {
		name       string
		decision   string
		answers    map[string]any
		request    map[string]any
		wantAnswer map[string]any
	}{
		{
			name:     "accept chooses the first option when no answers are provided",
			decision: "accept",
			request: map[string]any{
				"threadId": "thread-1",
				"turnId":   "turn-1",
				"itemId":   "item-1",
				"questions": []map[string]any{
					{
						"id":       "question-1",
						"question": "Pick an option",
						"options": []map[string]any{
							{"value": "option-a", "label": "Option A"},
							{"value": "option-b", "label": "Option B"},
						},
					},
				},
			},
			wantAnswer: map[string]any{"question-1": "option-a"},
		},
		{
			name:     "accept uses provided answers",
			decision: "accept",
			answers: map[string]any{
				"question-1": "release",
				"question-2": "Need the release branch",
			},
			request: map[string]any{
				"threadId": "thread-1",
				"turnId":   "turn-1",
				"itemId":   "item-1",
				"questions": []map[string]any{
					{
						"id":       "question-1",
						"question": "Pick an option",
						"options": []map[string]any{
							{"value": "option-a", "label": "Option A"},
							{"value": "option-b", "label": "Option B"},
						},
					},
					{
						"id":       "question-2",
						"question": "Why?",
					},
				},
			},
			wantAnswer: map[string]any{
				"question-1": "release",
				"question-2": "Need the release branch",
			},
		},
		{
			name:     "accept answers freeform questions with an empty string",
			decision: "accept",
			request: map[string]any{
				"threadId": "thread-1",
				"turnId":   "turn-1",
				"itemId":   "item-1",
				"questions": []map[string]any{
					{
						"id":       "question-1",
						"question": "Why?",
					},
				},
			},
			wantAnswer: map[string]any{"question-1": ""},
		},
		{
			name:     "decline sends an empty answers object",
			decision: "decline",
			request: map[string]any{
				"threadId": "thread-1",
				"turnId":   "turn-1",
				"itemId":   "item-1",
				"questions": []map[string]any{
					{
						"id":       "question-1",
						"question": "Pick an option",
						"options": []map[string]any{
							{"value": "option-a", "label": "Option A"},
						},
					},
				},
			},
			wantAnswer: map[string]any{},
		},
		{
			name:     "cancel sends an empty answers object",
			decision: "cancel",
			request: map[string]any{
				"threadId": "thread-1",
				"turnId":   "turn-1",
				"itemId":   "item-1",
				"questions": []map[string]any{
					{
						"id":       "question-1",
						"question": "Pick an option",
						"options": []map[string]any{
							{"value": "option-a", "label": "Option A"},
						},
					},
				},
			},
			wantAnswer: map[string]any{},
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
			runner.emitServerRequest(t, "approval-1", "item/tool/requestUserInput", tt.request)

			if err := client.RespondApproval("approval-1", tt.decision, tt.answers); err != nil {
				t.Fatal(err)
			}

			if gotID != "approval-1" {
				t.Fatalf("unexpected response id: %q", gotID)
			}
			gotAnswers, ok := gotResult["answers"].(map[string]any)
			if !ok {
				t.Fatalf("unexpected response payload: %#v", gotResult)
			}
			if len(gotAnswers) != len(tt.wantAnswer) {
				t.Fatalf("unexpected response payload: %#v", gotResult)
			}
			for key, want := range tt.wantAnswer {
				if gotAnswers[key] != want {
					t.Fatalf("answers[%q] = %#v, want %#v", key, gotAnswers[key], want)
				}
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

func TestAppServerClientEmitsCompletedAgentMessageTextWhenNoDeltaArrives(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)

	events := make([]agenttypes.RuntimeTurnEvent, 0, 3)
	client.SetTurnEventHandler(func(event agenttypes.RuntimeTurnEvent) {
		events = append(events, event)
	})

	runner.emitNotification(t, "turn/started", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
	})
	runner.emitNotification(t, "item/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"id":   "msg-1",
			"type": "agentMessage",
			"text": "hello",
		},
	})
	runner.emitNotification(t, "turn/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
	})

	if len(events) != 3 {
		t.Fatalf("unexpected event count: %d", len(events))
	}
	if events[1].Type != agenttypes.RuntimeTurnEventTypeDelta || events[1].Sequence != 1 || events[1].Delta != "hello" {
		t.Fatalf("unexpected completed agent message event: %+v", events[1])
	}
	if events[2].Type != agenttypes.RuntimeTurnEventTypeCompleted {
		t.Fatalf("unexpected completed event: %+v", events[2])
	}
}

func TestAppServerClientEmitsOnlyMissingCompletedAgentMessageSuffix(t *testing.T) {
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
		"itemId":   "msg-1",
		"delta":    "Down",
	})
	runner.emitNotification(t, "item/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"id":   "msg-1",
			"type": "agentMessage",
			"text": "Downstream unavailable",
		},
	})
	runner.emitNotification(t, "turn/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
	})

	if len(events) != 4 {
		t.Fatalf("unexpected event count: %d", len(events))
	}
	if events[1].Type != agenttypes.RuntimeTurnEventTypeDelta || events[1].Sequence != 1 || events[1].Delta != "Down" {
		t.Fatalf("unexpected first delta event: %+v", events[1])
	}
	if events[2].Type != agenttypes.RuntimeTurnEventTypeDelta || events[2].Sequence != 2 || events[2].Delta != "stream unavailable" {
		t.Fatalf("unexpected completed suffix event: %+v", events[2])
	}
}

func TestAppServerClientIncludesErrorMessageOnFailedTurnEvent(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)

	events := make([]agenttypes.RuntimeTurnEvent, 0, 2)
	client.SetTurnEventHandler(func(event agenttypes.RuntimeTurnEvent) {
		events = append(events, event)
	})

	runner.emitNotification(t, "turn/started", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
	})
	runner.emitNotification(t, "error", map[string]any{
		"threadId":  "thread-1",
		"turnId":    "turn-1",
		"willRetry": false,
		"error": map[string]any{
			"message": "Downstream unavailable",
		},
	})
	runner.emitNotification(t, "turn/completed", map[string]any{
		"threadId": "thread-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "failed",
		},
	})

	if len(events) != 2 {
		t.Fatalf("unexpected event count: %d", len(events))
	}
	if events[1].Type != agenttypes.RuntimeTurnEventTypeFailed {
		t.Fatalf("unexpected failed event: %+v", events[1])
	}
	if events[1].ErrorMessage != "Downstream unavailable" {
		t.Fatalf("unexpected failed error message: %+v", events[1])
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

	if err := client.RespondApproval("approval-1", "accept", nil); err != nil {
		t.Fatal(err)
	}

	if gotID != "approval-1" {
		t.Fatalf("unexpected response id: %q", gotID)
	}
	if gotResult["decision"] != "accept" {
		t.Fatalf("unexpected response payload: %#v", gotResult)
	}
}

func TestAppServerClientServerRequestResolvedEmitsApprovalResolvedWithStoredContext(t *testing.T) {
	runner := &fakeRunner{}
	client := NewAppServerClient(runner)

	var resolvedEvents []ApprovalResolvedEvent
	client.SetApprovalResolvedHandler(func(event ApprovalResolvedEvent) {
		resolvedEvents = append(resolvedEvents, event)
	})

	runner.emitServerRequest(t, "approval-1", "item/commandExecution/requestApproval", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"itemId":   "item-1",
		"command":  "go test ./...",
	})

	runner.emitNotification(t, "serverRequest/resolved", map[string]any{
		"requestId": "approval-1",
		"decision":  "accept",
	})

	if len(resolvedEvents) != 1 {
		t.Fatalf("expected 1 resolved event, got %d", len(resolvedEvents))
	}
	if resolvedEvents[0].RequestID != "approval-1" ||
		resolvedEvents[0].ThreadID != "thread-1" ||
		resolvedEvents[0].TurnID != "turn-1" ||
		resolvedEvents[0].Decision != "accept" {
		t.Fatalf("unexpected resolved event: %+v", resolvedEvents[0])
	}
	if _, ok := client.pendingApprovals["approval-1"]; ok {
		t.Fatal("expected pending approval to be removed after resolution")
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

			if err := client.RespondApproval("approval-1", tt.decision, nil); err != nil {
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
			case "config/read":
				response := out.(*configReadResponse)
				response.Config = map[string]any{}
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

	if len(calls) != 4 || calls[0] != "skills/list" || calls[1] != "mcpServerStatus/list" || calls[2] != "plugin/list" || calls[3] != "config/read" {
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
			if payloadMap["threadId"] != tt.threadID || payloadMap["includeTurns"] != true {
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

func TestClientReadThreadMapsHistoricalMessages(t *testing.T) {
	runner := &fakeRunner{
		call: func(method string, payload any, out any) error {
			if method != "thread/read" {
				t.Fatalf("unexpected method: %s", method)
			}
			response := out.(*threadReadResponse)
			response.Thread = threadRecord{
				ID:     "thread-1",
				Name:   "Investigate flaky test",
				Status: threadStatusRecord{Type: domain.ThreadStatusIdle},
				Turns: []turnRecord{
					{
						ID:     "turn-1",
						Status: domain.TurnStatusCompleted,
						Items: []threadItemRecord{
							{
								ID:   "user-1",
								Type: "userMessage",
								Content: []userInputRecord{
									{Type: "text", Text: "hello"},
								},
							},
							{
								ID:   "agent-1",
								Type: "agentMessage",
								Text: "hi there",
							},
						},
					},
					{
						ID:     "turn-2",
						Status: domain.TurnStatusFailed,
						Error: &turnErrorRecord{
							Message: "Downstream unavailable",
						},
					},
				},
			}
			return nil
		},
	}

	client := NewAppServerClient(runner)
	thread, err := client.ReadThread("thread-1")
	if err != nil {
		t.Fatal(err)
	}

	if len(thread.Messages) != 3 {
		t.Fatalf("unexpected message count: %+v", thread.Messages)
	}
	if thread.Messages[0].Kind != domain.ThreadMessageKindUser || thread.Messages[0].Text != "hello" {
		t.Fatalf("unexpected user message: %+v", thread.Messages[0])
	}
	if thread.Messages[1].Kind != domain.ThreadMessageKindAgent || thread.Messages[1].Text != "hi there" {
		t.Fatalf("unexpected agent message: %+v", thread.Messages[1])
	}
	if thread.Messages[2].Kind != domain.ThreadMessageKindSystem || thread.Messages[2].Text != "Turn turn-2 failed: Downstream unavailable" {
		t.Fatalf("unexpected failed system message: %+v", thread.Messages[2])
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
