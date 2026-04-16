package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/gateway/internal/api"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/routing"
	"code-agent-gateway/gateway/internal/runtimeindex"
	"code-agent-gateway/gateway/internal/settings"
	ws "code-agent-gateway/gateway/internal/websocket"
)

func TestSettingsSystemScenarios(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T, system *liveSettingsSystem)
	}{
		{
			name: "put and get global default",
			run: func(t *testing.T, system *liveSettingsSystem) {
				rec := system.request(http.MethodPut, "/settings/agents/codex/global", `{"content":"model = \"gpt-5.4\"\n"}`)
				if rec.Code != http.StatusOK {
					t.Fatalf("put global returned %d: %s", rec.Code, rec.Body)
				}
				rec = system.request(http.MethodGet, "/settings/agents/codex/global", "")
				if rec.Code != http.StatusOK {
					t.Fatalf("get global returned %d", rec.Code)
				}
				var body struct {
					Document *domain.AgentConfigDocument `json:"document"`
				}
				if err := json.Unmarshal([]byte(rec.Body), &body); err != nil {
					t.Fatal(err)
				}
				if body.Document == nil || body.Document.Content != "model = \"gpt-5.4\"\n" {
					t.Fatalf("unexpected global document: %+v", body)
				}
			},
		},
		{
			name: "put and get machine override",
			run: func(t *testing.T, system *liveSettingsSystem) {
				rec := system.request(http.MethodPut, system.machineAgentPath("codex"), `{"content":"model = \"gpt-5.2\"\n"}`)
				if rec.Code != http.StatusOK {
					t.Fatalf("put machine override returned %d: %s", rec.Code, rec.Body)
				}
				rec = system.request(http.MethodGet, system.machineAgentPath("codex"), "")
				if rec.Code != http.StatusOK {
					t.Fatalf("get machine override returned %d", rec.Code)
				}
				var body domain.MachineAgentConfigAssignment
				if err := json.Unmarshal([]byte(rec.Body), &body); err != nil {
					t.Fatal(err)
				}
				if body.MachineOverride == nil || body.MachineOverride.Content != "model = \"gpt-5.2\"\n" {
					t.Fatalf("unexpected machine assignment: %+v", body)
				}
			},
		},
		{
			name: "delete machine override falls back to global default",
			run: func(t *testing.T, system *liveSettingsSystem) {
				system.mustStoreGlobal(t, "model = \"gpt-5.4\"\n")
				system.mustStoreMachine(t, system.machineID, "model = \"gpt-5.2\"\n")
				rec := system.request(http.MethodDelete, system.machineAgentPath("codex"), "")
				if rec.Code != http.StatusNoContent {
					t.Fatalf("delete machine override returned %d", rec.Code)
				}
				rec = system.request(http.MethodGet, system.machineAgentPath("codex"), "")
				var body domain.MachineAgentConfigAssignment
				if err := json.Unmarshal([]byte(rec.Body), &body); err != nil {
					t.Fatal(err)
				}
				if !body.UsesGlobalDefault || body.MachineOverride != nil || body.GlobalDefault == nil {
					t.Fatalf("unexpected fallback assignment: %+v", body)
				}
			},
		},
		{
			name: "apply global default writes config file",
			run: func(t *testing.T, system *liveSettingsSystem) {
				system.mustStoreGlobal(t, "model = \"gpt-5.4\"\n")
				rec := system.request(http.MethodPost, system.machineApplyPath("codex"), "")
				if rec.Code != http.StatusOK {
					t.Fatalf("apply returned %d: %s", rec.Code, rec.Body)
				}
				if got := system.readAppliedConfigFile(t, rec.Body); got != "model = \"gpt-5.4\"\n" {
					t.Fatalf("unexpected config file: %q", got)
				}
			},
		},
		{
			name: "apply machine override writes config file",
			run: func(t *testing.T, system *liveSettingsSystem) {
				system.mustStoreGlobal(t, "model = \"gpt-5.4\"\n")
				system.mustStoreMachine(t, system.machineID, "model = \"gpt-5.2\"\n")
				rec := system.request(http.MethodPost, system.machineApplyPath("codex"), "")
				if rec.Code != http.StatusOK {
					t.Fatalf("apply returned %d: %s", rec.Code, rec.Body)
				}
				if got := system.readAppliedConfigFile(t, rec.Body); got != "model = \"gpt-5.2\"\n" {
					t.Fatalf("unexpected config file: %q", got)
				}
			},
		},
		{
			name: "apply without any config returns conflict",
			run: func(t *testing.T, system *liveSettingsSystem) {
				rec := system.request(http.MethodPost, system.machineApplyPath("codex"), "")
				if rec.Code != http.StatusConflict {
					t.Fatalf("expected 409, got %d", rec.Code)
				}
			},
		},
		{
			name: "rejects empty content",
			run: func(t *testing.T, system *liveSettingsSystem) {
				rec := system.request(http.MethodPut, "/settings/agents/codex/global", `{"content":""}`)
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("expected 400, got %d", rec.Code)
				}
			},
		},
		{
			name: "rejects invalid json body",
			run: func(t *testing.T, system *liveSettingsSystem) {
				rec := system.request(http.MethodPut, "/settings/agents/codex/global", "{")
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("expected 400, got %d", rec.Code)
				}
			},
		},
		{
			name: "rejects invalid toml content",
			run: func(t *testing.T, system *liveSettingsSystem) {
				rec := system.request(http.MethodPut, "/settings/agents/codex/global", `{"content":"model = ["}`)
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("expected 400, got %d", rec.Code)
				}
			},
		},
		{
			name: "rejects unsupported agent type",
			run: func(t *testing.T, system *liveSettingsSystem) {
				rec := system.request(http.MethodGet, "/settings/agents/claude_code/global", "")
				if rec.Code != http.StatusNotFound {
					t.Fatalf("expected 404, got %d", rec.Code)
				}
			},
		},
		{
			name: "returns supported agent list",
			run: func(t *testing.T, system *liveSettingsSystem) {
				rec := system.request(http.MethodGet, "/settings/agents", "")
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d", rec.Code)
				}
				var body struct {
					Items []domain.AgentDescriptor `json:"items"`
				}
				if err := json.Unmarshal([]byte(rec.Body), &body); err != nil {
					t.Fatal(err)
				}
				if len(body.Items) != 1 || body.Items[0].AgentType != domain.AgentTypeCodex {
					t.Fatalf("unexpected agent list: %+v", body)
				}
			},
		},
	}

	if len(cases) < 10 {
		t.Fatalf("expected at least 10 system scenarios, got %d", len(cases))
	}

	clientBinary := buildClientBinary(t)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			system := newLiveSettingsSystem(t, clientBinary)
			defer system.close()
			tt.run(t, system)
		})
	}
}

type liveSettingsSystem struct {
	server        *httptest.Server
	settingsStore settings.Store
	clientHome    string
	machineID     string
	cancel        context.CancelFunc
	clientOutput  *bytes.Buffer
}

type httpResult struct {
	Code int
	Body string
}

func newLiveSettingsSystem(t *testing.T, clientBinary string) *liveSettingsSystem {
	t.Helper()

	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	router := routing.NewRouter()
	consoleHub := ws.NewConsoleHub()
	clientHub := ws.NewClientHubWithStores(reg, idx, router)
	clientHub.SetConsoleHub(consoleHub)
	settingsStore := settings.NewMemoryStore([]domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})

	server := httptest.NewServer(api.NewServerWithSettings(reg, idx, router, clientHub, settingsStore, clientHub.Handler(), consoleHub.Handler()))
	clientHome := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	output := &bytes.Buffer{}
	command := exec.CommandContext(ctx, clientBinary)
	command.Dir = repoRoot(t)
	command.Env = append(os.Environ(),
		"MACHINE_NAME=Settings System Test Client",
		"GATEWAY_URL=ws"+server.URL[4:]+"/ws/client",
		"CODEX_RUNTIME_MODE=fake",
		"HOME="+clientHome,
	)
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		cancel()
		server.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = command.Wait()
		server.Close()
	})

	machineID := waitForMachineRegistration(t, server.URL+"/machines")

	return &liveSettingsSystem{
		server:        server,
		settingsStore: settingsStore,
		clientHome:    clientHome,
		machineID:     machineID,
		cancel:        cancel,
		clientOutput:  output,
	}
}

func (s *liveSettingsSystem) close() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.server != nil {
		s.server.Close()
	}
}

func (s *liveSettingsSystem) request(method string, path string, body string) httpResult {
	request, err := http.NewRequest(method, s.server.URL+path, strings.NewReader(body))
	if err != nil {
		return httpResult{Code: 0, Body: err.Error()}
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return httpResult{Code: 0, Body: err.Error()}
	}
	defer response.Body.Close()
	content, _ := io.ReadAll(response.Body)
	return httpResult{
		Code: response.StatusCode,
		Body: string(content),
	}
}

func (s *liveSettingsSystem) machineAgentPath(agentType string) string {
	return fmt.Sprintf("/settings/machines/%s/agents/%s", s.machineID, agentType)
}

func (s *liveSettingsSystem) machineApplyPath(agentType string) string {
	return fmt.Sprintf("/settings/machines/%s/agents/%s/apply", s.machineID, agentType)
}

func (s *liveSettingsSystem) mustStoreGlobal(t *testing.T, content string) {
	t.Helper()
	if err := s.settingsStore.PutGlobal(domain.AgentTypeCodex, domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   content,
	}); err != nil {
		t.Fatal(err)
	}
}

func (s *liveSettingsSystem) mustStoreMachine(t *testing.T, machineID string, content string) {
	t.Helper()
	if err := s.settingsStore.PutMachine(machineID, domain.AgentTypeCodex, domain.AgentConfigDocument{
		AgentType: domain.AgentTypeCodex,
		Format:    domain.AgentConfigFormatTOML,
		Content:   content,
	}); err != nil {
		t.Fatal(err)
	}
}

func (s *liveSettingsSystem) readAppliedConfigFile(t *testing.T, responseBody string) string {
	t.Helper()
	var body struct {
		FilePath string `json:"filePath"`
	}
	if err := json.Unmarshal([]byte(responseBody), &body); err != nil {
		t.Fatalf("decode apply response failed: %v\nbody:\n%s", err, responseBody)
	}
	if strings.TrimSpace(body.FilePath) == "" {
		t.Fatalf("apply response missing filePath:\n%s", responseBody)
	}
	content, err := os.ReadFile(body.FilePath)
	if err != nil {
		t.Fatalf("read config file failed: %v\nclient output:\n%s", err, s.clientOutput.String())
	}
	return string(content)
}

func buildClientBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "client")
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("locate go binary: %v", err)
	}
	command := exec.Command(goBinary, "build", "-o", path, "./client/cmd/client")
	command.Dir = repoRoot(t)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build client failed: %v\n%s", err, string(output))
	}
	return path
}

func repoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(cwd, "..", "..", ".."))
}

func waitForMachineRegistration(t *testing.T, machinesURL string) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(machinesURL)
		if err == nil {
			var body struct {
				Items []domain.Machine `json:"items"`
			}
			if decodeErr := json.NewDecoder(response.Body).Decode(&body); decodeErr == nil {
				_ = response.Body.Close()
				for _, machine := range body.Items {
					if machine.ID != "" && machine.Status == domain.MachineStatusOnline {
						return machine.ID
					}
				}
			} else {
				_ = response.Body.Close()
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("no online machine registered in time")
	return ""
}
