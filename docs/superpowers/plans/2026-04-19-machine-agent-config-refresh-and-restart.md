# Machine Agent Config Refresh + Restart Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure `/machines` agent config editing always loads latest machine-side config before editing, and on save performs `config write -> agent restart` so the new config takes effect immediately.

**Architecture:** Keep existing machine-agent config read/write APIs; add a new restart API (`POST /machines/{machineId}/agents/{agentId}/restart`) mapped to a new southbound command (`machine.agent.restart`). In console, enforce load-success gate before save and run save+restart sequentially with explicit partial-success feedback.

**Tech Stack:** Go (`net/http`, go test), shared protocol model in `common`, React + TypeScript + Vitest in `console`.

---

## File Structure

- Modify: `common/protocol/messages.go`
  Responsibility: add restart command payload/result contracts shared by gateway/client.
- Modify: `gateway/internal/api/server.go`
  Responsibility: add northbound restart endpoint and forward command.
- Modify: `gateway/internal/api/server_test.go`
  Responsibility: validate restart endpoint command routing and error handling.
- Modify: `client/internal/agent/manager/supervisor.go`
  Responsibility: add `RestartAgent(agentID)` runtime lifecycle primitive.
- Modify: `client/internal/agent/manager/supervisor_test.go`
  Responsibility: verify restart semantics for running/stopped agents.
- Modify: `client/cmd/client/main.go`
  Responsibility: handle `machine.agent.restart` command and refresh snapshot.
- Modify: `client/cmd/client/main_test.go`
  Responsibility: verify command handling success/failure paths.
- Modify: `console/src/features/machines/components/machines-screen.tsx`
  Responsibility: enforce config load gate, show load/save/restart errors, avoid fallback fake config.
- Modify: `console/src/features/machines/hooks/use-machines-page.ts`
  Responsibility: perform `PUT config` then `POST restart` sequence and return structured outcome.
- Modify: `console/src/features/machines/pages/machines-page.test.tsx`
  Responsibility: verify UI/API behavior for load gate and save+restart flow.

### Task 1: Shared Protocol + Gateway Restart API

**Files:**
- Modify: `common/protocol/messages.go`
- Modify: `gateway/internal/api/server.go`
- Test: `gateway/internal/api/server_test.go`

- [ ] **Step 1: Write the failing gateway API test for restart endpoint**

```go
func TestServerMachineAgentRestartEndpointUsesExpectedCommand(t *testing.T) {
	type recordedCall struct {
		machineID string
		name      string
		payload   any
	}
	var calls []recordedCall

	sender := &fakeCommandSender{
		send: func(_ context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error) {
			calls = append(calls, recordedCall{machineID: machineID, name: name, payload: payload})
			return protocol.CommandCompletedPayload{
				CommandName: name,
				Result: mustMarshalJSON(t, protocol.MachineAgentRestartCommandResult{AgentID: "agent-01"}),
			}, nil
		},
	}

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), routing.NewRouter(), sender, http.NotFoundHandler(), http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodPost, "/machines/machine-01/agents/agent-01/restart", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("restart returned %d", rec.Code)
	}
	if len(calls) != 1 || calls[0].name != "machine.agent.restart" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}
```

- [ ] **Step 2: Run test to confirm failure before implementation**

Run: `go test ./gateway/internal/api -run TestServerMachineAgentRestartEndpointUsesExpectedCommand -count=1`
Expected: FAIL with 404 or missing handler behavior.

- [ ] **Step 3: Add restart payload/result contract to shared protocol**

```go
type MachineAgentRestartCommandPayload struct {
	AgentID string `json:"agentId"`
}

type MachineAgentRestartCommandResult struct {
	AgentID string `json:"agentId"`
}
```

- [ ] **Step 4: Add gateway restart northbound route and command forwarding**

```go
mux.HandleFunc("POST /machines/{machineId}/agents/{agentId}/restart", func(w http.ResponseWriter, r *http.Request) {
	if sender == nil {
		http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
		return
	}
	machineID := strings.TrimSpace(r.PathValue("machineId"))
	agentID := strings.TrimSpace(r.PathValue("agentId"))
	if machineID == "" {
		http.Error(w, "machineId is required", http.StatusBadRequest)
		return
	}
	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	completed, err := sender.SendCommand(r.Context(), machineID, "machine.agent.restart", protocol.MachineAgentRestartCommandPayload{AgentID: agentID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	var result protocol.MachineAgentRestartCommandResult
	if err := transport.Decode(completed.Result, &result); err != nil {
		http.Error(w, "invalid machine.agent.restart result", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"machineId": machineID, "agentId": result.AgentID, "status": "restarted"})
})
```

- [ ] **Step 5: Run gateway API tests**

Run: `go test ./gateway/internal/api -run "MachineAgentRestart|MachineAgentConfig" -count=1`
Expected: PASS for restart and existing config endpoint tests.

- [ ] **Step 6: Commit Task 1 changes**

```bash
git add common/protocol/messages.go gateway/internal/api/server.go gateway/internal/api/server_test.go
git commit -m "feat(gateway): add machine agent restart endpoint"
```

### Task 2: Client Supervisor Restart Primitive

**Files:**
- Modify: `client/internal/agent/manager/supervisor.go`
- Test: `client/internal/agent/manager/supervisor_test.go`

- [ ] **Step 1: Write failing tests for restart behavior in supervisor**

```go
func TestSupervisorRestartAgentRestartsRunningAgent(t *testing.T) {
	managedAgentsDir := t.TempDir()
	startCount := 0
	cleanupCount := 0
	supervisor, err := NewSupervisor(context.Background(), managedAgentsDir, agentregistry.New(), map[domain.AgentType]agenttypes.RuntimeFactory{
		domain.AgentTypeCodex: supervisorRuntimeFactory(func(context.Context, agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			startCount++
			return noopRuntime{}, func() error { cleanupCount++; return nil }, nil
		}),
	})
	if err != nil { t.Fatal(err) }

	if err := supervisor.RestartAgent("agent-01"); err != nil { t.Fatal(err) }
	if startCount < 2 { t.Fatalf("expected restart to start runtime again, got %d", startCount) }
	if cleanupCount < 1 { t.Fatalf("expected restart to cleanup prior runtime, got %d", cleanupCount) }
}
```

- [ ] **Step 2: Run supervisor tests to confirm failure**

Run: `go test ./client/internal/agent/manager -run TestSupervisorRestartAgentRestartsRunningAgent -count=1`
Expected: FAIL with `RestartAgent` undefined.

- [ ] **Step 3: Implement `RestartAgent(agentID)` in supervisor**

```go
func (s *Supervisor) RestartAgent(agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if err := codex.ValidateAgentID(agentID); err != nil {
		return err
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	record, ok := s.record(agentID)
	if !ok {
		return fmt.Errorf("agent %q is not installed", agentID)
	}

	s.mu.RLock()
	_, running := s.cleanups[agentID]
	s.mu.RUnlock()
	if running {
		if err := s.stopAgent(agentID); err != nil {
			return err
		}
	}
	return s.startAgent(record)
}
```

- [ ] **Step 4: Add stopped-agent restart test and run manager package tests**

```go
func TestSupervisorRestartAgentStartsStoppedAgent(t *testing.T) {
	managedAgentsDir := t.TempDir()
	supervisor, err := NewSupervisor(context.Background(), managedAgentsDir, agentregistry.New(), map[domain.AgentType]agenttypes.RuntimeFactory{
		domain.AgentTypeCodex: supervisorRuntimeFactory(func(context.Context, agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			return noopRuntime{}, func() error { return nil }, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := supervisor.StopAll(); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.RestartAgent("agent-01"); err != nil {
		t.Fatal(err)
	}

	agents := supervisor.AgentInstances()
	if len(agents) != 1 {
		t.Fatalf("expected one managed agent, got %d", len(agents))
	}
	if agents[0].Status != domain.AgentInstanceStatusRunning {
		t.Fatalf("expected running status after restart, got %+v", agents[0])
	}
}
```

Run: `go test ./client/internal/agent/manager -count=1`
Expected: PASS with new restart tests.

- [ ] **Step 5: Commit Task 2 changes**

```bash
git add client/internal/agent/manager/supervisor.go client/internal/agent/manager/supervisor_test.go
git commit -m "feat(client): add supervisor restart for single agent"
```

### Task 3: Client Command Handler for `machine.agent.restart`

**Files:**
- Modify: `client/cmd/client/main.go`
- Test: `client/cmd/client/main_test.go`

- [ ] **Step 1: Add failing command-handler tests for restart**

```go
func TestHandleCommandEnvelopeRestartsMachineAgent(t *testing.T) {
	registry := agentregistry.New()
	supervisor, err := manager.NewSupervisor(context.Background(), t.TempDir(), registry, map[domain.AgentType]agenttypes.RuntimeFactory{
		domain.AgentTypeCodex: runtimeFactoryFunc(func(context.Context, agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			return &notifyingRuntime{}, func() error { return nil }, nil
		}),
	})
	if err != nil { t.Fatal(err) }
	mgr := manager.New(registry)
	session, sent := newRecordingSession()

	err = handleCommandEnvelope(session, mgr, registry, supervisor, protocol.Envelope{
		Name:      "machine.agent.restart",
		RequestID: "req-restart-1",
		Payload:   []byte(`{"agentId":"agent-01"}`),
	})
	if err != nil { t.Fatal(err) }
	if len(*sent) == 0 { t.Fatal("expected command.completed") }
}
```

- [ ] **Step 2: Run restart command test to confirm failure**

Run: `go test ./client/cmd/client -run TestHandleCommandEnvelopeRestartsMachineAgent -count=1`
Expected: FAIL with unknown command rejection or missing case.

- [ ] **Step 3: Implement command case in client main loop**

```go
case "machine.agent.restart":
	var payload protocol.MachineAgentRestartCommandPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
	}
	if supervisor == nil {
		return session.CommandRejected(envelope.RequestID, envelope.Name, "runtime supervisor unavailable", "")
	}
	if err := supervisor.RestartAgent(payload.AgentID); err != nil {
		return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
	}
	if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.MachineAgentRestartCommandResult{AgentID: payload.AgentID}); err != nil {
		return err
	}
	return sendLiveSnapshot(session, buildMachineSnapshot(session.machineID, session.machineName, supervisor), mgr, registry)
```

- [ ] **Step 4: Run client command tests**

Run: `go test ./client/cmd/client -run "machine.agent.restart|runtime.stop|runtime.start" -count=1`
Expected: PASS and restart command emits `command.completed` plus snapshot refresh.

- [ ] **Step 5: Commit Task 3 changes**

```bash
git add client/cmd/client/main.go client/cmd/client/main_test.go
git commit -m "feat(client): handle machine agent restart command"
```

### Task 4: Console Load Gate + Save-Then-Restart UX

**Files:**
- Modify: `console/src/features/machines/hooks/use-machines-page.ts`
- Modify: `console/src/features/machines/components/machines-screen.tsx`
- Test: `console/src/features/machines/pages/machines-page.test.tsx`

- [ ] **Step 1: Add failing frontend tests for load gate and restart call sequence**

```ts
test("does not allow save when config load fails", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);
    const method = init?.method ?? "GET";
    if (path === "/capabilities") return jsonResponse(capabilities);
    if (path === "/threads") return jsonResponse({ items: [] });
    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-01",
            name: "Machine 01",
            status: "online",
            runtimeStatus: "running",
            agents: [{ agentId: "agent-01", agentType: "codex", displayName: "Primary Codex", status: "running" }],
          },
        ],
      });
    }
    if (path === "/machines/machine-01/agents/agent-01/config" && method === "GET") {
      return new Response("boom", { status: 502 });
    }
    throw new Error(`Unexpected request: ${method} ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
  render(<MemoryRouter><MachinesPage /></MemoryRouter>);

  fireEvent.click((await screen.findAllByTitle("编辑配置"))[0]);
  expect(await screen.findByText("无法加载该 Agent 的最新配置，请重试。")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "保存" })).toBeDisabled();
});

test("save triggers config PUT then restart POST in order", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);
    const method = init?.method ?? "GET";
    if (path === "/capabilities") return jsonResponse(capabilities);
    if (path === "/threads") return jsonResponse({ items: [] });
    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-01",
            name: "Machine 01",
            status: "online",
            runtimeStatus: "running",
            agents: [{ agentId: "agent-01", agentType: "codex", displayName: "Primary Codex", status: "running" }],
          },
        ],
      });
    }
    if (path === "/machines/machine-01/agents/agent-01/config" && method === "GET") {
      return jsonResponse({ document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" } });
    }
    if (path === "/machines/machine-01/agents/agent-01/config" && method === "PUT") {
      return jsonResponse({ document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.5\"\n" } });
    }
    if (path === "/machines/machine-01/agents/agent-01/restart" && method === "POST") {
      return jsonResponse({ machineId: "machine-01", agentId: "agent-01", status: "restarted" });
    }
    throw new Error(`Unexpected request: ${method} ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
  render(<MemoryRouter><MachinesPage /></MemoryRouter>);

  fireEvent.click((await screen.findAllByTitle("编辑配置"))[0]);
  await screen.findByDisplayValue("model = \"gpt-5.4\"\n");
  fireEvent.change(screen.getByRole("textbox"), { target: { value: "model = \"gpt-5.5\"\n" } });
  fireEvent.click(screen.getByRole("button", { name: "保存" }));

  await waitFor(() => {
    const paths = fetchMock.mock.calls.map(([input]) => getPath(input as RequestInfo | URL));
    const putIndex = paths.findIndex((path) => path === "/machines/machine-01/agents/agent-01/config");
    const restartIndex = paths.findIndex((path) => path === "/machines/machine-01/agents/agent-01/restart");
    expect(putIndex).toBeGreaterThan(-1);
    expect(restartIndex).toBeGreaterThan(putIndex);
  });
});
```

- [ ] **Step 2: Run machines page tests to confirm failure**

Run: `pnpm --dir console test -- src/features/machines/pages/machines-page.test.tsx`
Expected: FAIL because dialog currently allows fallback config and has no restart call.

- [ ] **Step 3: Update hook to return structured save outcome with restart stage**

```ts
const handleUpdateAgentConfig = useCallback(async (machineId: string, agentId: string, config: string) => {
  await http(`/machines/${encodeURIComponent(machineId)}/agents/${encodeURIComponent(agentId)}/config`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content: config }),
  });

  try {
    await http(`/machines/${encodeURIComponent(machineId)}/agents/${encodeURIComponent(agentId)}/restart`, {
      method: "POST",
    });
    await loadMachinesPageData();
    return { saved: true, restarted: true as const };
  } catch (error) {
    await loadMachinesPageData();
    return { saved: true, restarted: false as const, error: error instanceof Error ? error.message : "restart failed" };
  }
}, [loadMachinesPageData, remoteEnabled]);
```

- [ ] **Step 4: Update edit dialog to enforce load success gate and show partial-success errors**

```tsx
const [isLoadingConfig, setIsLoadingConfig] = useState(false);
const [loadError, setLoadError] = useState<string | null>(null);
const [saveError, setSaveError] = useState<string | null>(null);

useEffect(() => {
  if (!open) return;
  setIsLoadingConfig(true);
  setLoadError(null);
  void http<{ document?: { content?: string } }>(`/machines/${encodeURIComponent(machine.id)}/agents/${encodeURIComponent(agent.id)}/config`)
    .then((response) => setConfig(response.document?.content ?? ""))
    .catch(() => setLoadError("无法加载该 Agent 的最新配置，请重试。"))
    .finally(() => setIsLoadingConfig(false));
}, [open, machine.id, agent.id]);

<button disabled={isLoadingConfig || !!loadError || config.trim() === ""}>保存</button>
```

- [ ] **Step 5: Re-run console tests**

Run: `pnpm --dir console test -- src/features/machines/pages/machines-page.test.tsx`
Expected: PASS with load gate and restart sequence assertions.

- [ ] **Step 6: Commit Task 4 changes**

```bash
git add console/src/features/machines/hooks/use-machines-page.ts console/src/features/machines/components/machines-screen.tsx console/src/features/machines/pages/machines-page.test.tsx
git commit -m "feat(console): enforce config load gate and restart after save"
```

### Task 5: End-to-End Verification and Cleanup

**Files:**
- Modify: `gateway/internal/api/server_test.go`
- Modify: `client/cmd/client/main_test.go`
- Modify: `console/src/features/machines/pages/machines-page.test.tsx`

- [ ] **Step 1: Run focused Go test suites for changed backend/client packages**

Run: `go test ./common/protocol ./gateway/internal/api ./client/internal/agent/manager ./client/cmd/client -count=1`
Expected: PASS across all changed packages.

- [ ] **Step 2: Run focused console suite**

Run: `pnpm --dir console test -- src/features/machines/pages/machines-page.test.tsx`
Expected: PASS for all machines page behaviors.

- [ ] **Step 3: Run broader console regression for related pages**

Run: `pnpm --dir console test -- src/features/machines src/features/settings`
Expected: PASS without regression on settings/machines capabilities.

- [ ] **Step 4: Final manual API sanity check (optional but recommended)**

Run:

```bash
curl -X POST "http://127.0.0.1:8080/machines/${MACHINE_ID}/agents/${AGENT_ID}/restart" \
  -H "Authorization: Bearer ${GATEWAY_API_KEY}"
```

Expected: `200` with JSON containing `machineId`, `agentId`, `status: "restarted"`.

- [ ] **Step 5: Commit verification fixes (if any)**

```bash
git add -A
git commit -m "test: cover machine agent config restart flow"
```
