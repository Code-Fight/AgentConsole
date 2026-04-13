# Gateway Capability Gap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the Gateway capability gaps exposed by the new design-driven Console so unsupported UI actions can become real product capabilities instead of permanent disabled states.

**Architecture:** Deliver the gaps in three vertical slices: capability introspection, management operations, and metrics/overview data. Each slice starts by extending shared domain/types, then adds Gateway API handlers, then wires client/runtime behavior, and finally enables the matching Console capability flag.

**Tech Stack:** Go Gateway API, Go client runtime abstractions, Codex appserver integration, React Console adapters, Vitest, Go test, Playwright.

---

### Task 1: Publish A First-Class Capability Map From Gateway

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/capability.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/common/api/types.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/capabilities.ts`

- [ ] **Step 1: Write the failing Gateway API test for `/capabilities`**

```go
func TestCapabilitiesEndpointReturnsConsoleFeatureFlags(t *testing.T) {
	server := httptest.NewServer(NewServerWithSettings(reg, idx, router, sender, nil, http.NotFoundHandler(), http.NotFoundHandler()))
	defer server.Close()

	resp, err := http.Get(server.URL + "/capabilities")
	if err != nil {
		t.Fatalf("get capabilities: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run the failing Gateway test**

Run: `go test ./gateway/internal/api -run TestCapabilitiesEndpointReturnsConsoleFeatureFlags -count=1`
Expected: FAIL because the endpoint does not exist yet.

- [ ] **Step 3: Add the shared capability model and endpoint**

```go
// /common/domain/capability.go
package domain

type CapabilitySnapshot struct {
	DashboardMetrics bool `json:"dashboardMetrics"`
	AgentLifecycle   bool `json:"agentLifecycle"`
	EnvironmentWrite bool `json:"environmentWrite"`
}
```

```go
// /gateway/internal/api/server.go
mux.HandleFunc("GET /capabilities", func(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.CapabilitySnapshot{
		DashboardMetrics: false,
		AgentLifecycle:   false,
		EnvironmentWrite: true,
	})
})
```

- [ ] **Step 4: Run the Gateway test and Console type checks**

Run: `go test ./gateway/internal/api -run TestCapabilitiesEndpointReturnsConsoleFeatureFlags -count=1 && corepack pnpm test src/common/api/http.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add common/domain/capability.go gateway/internal/api/server.go gateway/internal/api/server_test.go console/src/common/api/types.ts console/src/gateway/capabilities.ts
git commit -m "feat: publish gateway capability snapshot"
```

### Task 2: Implement Agent Lifecycle Management For Machines Page Actions

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/types/interfaces.go`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/agent_lifecycle.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/machines-page.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/machines-page.test.tsx`

- [ ] **Step 1: Write the failing machines API test for creating and deleting managed agents**

```go
func TestMachineAgentLifecycleEndpoints(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/machines/machine-01/agents", strings.NewReader(`{"agentType":"codex","displayName":"Codex"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	NewServerWithSettings(reg, idx, router, sender, nil, http.NotFoundHandler(), http.NotFoundHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run the failing Gateway test**

Run: `go test ./gateway/internal/api -run TestMachineAgentLifecycleEndpoints -count=1`
Expected: FAIL because the endpoints and runtime contracts do not exist yet.

- [ ] **Step 3: Extend runtime interfaces and add Gateway handlers**

```go
// /client/internal/agent/types/interfaces.go
type RuntimeAgentManager interface {
	InstallAgent(machineID string, agentType domain.AgentType, displayName string) error
	DeleteAgent(machineID string, agentID string) error
}
```

```go
// /gateway/internal/api/server.go
mux.HandleFunc("POST /machines/{machineId}/agents", func(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusAccepted, map[string]any{"machineId": r.PathValue("machineId")})
})

mux.HandleFunc("DELETE /machines/{machineId}/agents/{agentId}", func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})
```

- [ ] **Step 4: Run the targeted Go and Console tests**

Run: `go test ./gateway/internal/api -run TestMachineAgentLifecycleEndpoints -count=1 && corepack pnpm test src/pages/machines-page.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add client/internal/agent/types/interfaces.go common/domain/agent_lifecycle.go gateway/internal/api/server.go gateway/internal/api/server_test.go console/src/pages/machines-page.tsx console/src/pages/machines-page.test.tsx
git commit -m "feat: add machine agent lifecycle endpoints"
```

### Task 3: Finish The Environment Management Gaps Exposed By The New Design

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/codex/environment.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/codex/appserver_client_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/environment-page.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/environment-page.test.tsx`

- [ ] **Step 1: Write the failing environment test for every enabled design action**

```tsx
test("supports all design-exposed environment actions without disabled stub fallbacks", async () => {
  render(<EnvironmentPage />);

  expect(await screen.findByRole("button", { name: "Install" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "Save MCP" })).toBeEnabled();
});
```

- [ ] **Step 2: Run the environment test to verify action gaps**

Run: `corepack pnpm test src/pages/environment-page.test.tsx`
Expected: FAIL for any still-disabled design action.

- [ ] **Step 3: Complete missing Gateway and client environment mutations**

```go
// /gateway/internal/api/server.go
mux.HandleFunc("POST /environment/mcps", func(w http.ResponseWriter, r *http.Request) {
	// decode machineId/resourceId/config and send command to target machine
	w.WriteHeader(http.StatusAccepted)
})
```

```go
// /client/internal/agent/codex/environment.go
func (c *AppServerClient) UpsertMCPServer(serverID string, config map[string]any) error {
	return c.writeConfigValue("mcp_servers."+serverID, config, "replace")
}
```

- [ ] **Step 4: Run Go and Console environment tests**

Run: `go test ./client/internal/agent/codex ./gateway/internal/api -count=1 && corepack pnpm test src/pages/environment-page.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gateway/internal/api/server.go gateway/internal/api/server_test.go client/internal/agent/codex/environment.go client/internal/agent/codex/appserver_client_test.go console/src/pages/environment-page.tsx console/src/pages/environment-page.test.tsx
git commit -m "feat: close environment management capability gaps"
```

### Task 4: Add Metrics Snapshots For The Design Overview Surface

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/metrics.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/runtimeindex/store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/runtimeindex/store_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/overview-page.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/overview-page.test.tsx`

- [ ] **Step 1: Write the failing metrics endpoint test**

```go
func TestOverviewMetricsEndpointReturnsAggregatedSnapshot(t *testing.T) {
	resp, err := http.Get(server.URL + "/overview/metrics")
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run the failing metrics test**

Run: `go test ./gateway/internal/api -run TestOverviewMetricsEndpointReturnsAggregatedSnapshot -count=1`
Expected: FAIL because the endpoint and aggregation model do not exist.

- [ ] **Step 3: Add runtime-index aggregation and expose it northbound**

```go
// /common/domain/metrics.go
package domain

type OverviewMetrics struct {
	OnlineMachines int `json:"onlineMachines"`
	ActiveThreads  int `json:"activeThreads"`
	PendingApprovals int `json:"pendingApprovals"`
}
```

```go
// /gateway/internal/api/server.go
mux.HandleFunc("GET /overview/metrics", func(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, idx.OverviewMetrics())
})
```

- [ ] **Step 4: Run verification for Go and Console overview tests**

Run: `go test ./gateway/internal/runtimeindex ./gateway/internal/api -count=1 && corepack pnpm test src/pages/overview-page.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add common/domain/metrics.go gateway/internal/runtimeindex/store.go gateway/internal/runtimeindex/store_test.go gateway/internal/api/server.go gateway/internal/api/server_test.go console/src/pages/overview-page.tsx console/src/pages/overview-page.test.tsx
git commit -m "feat: add overview metrics snapshot"
```

### Task 5: Turn Off Disabled Placeholders As Real Capabilities Land

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/capabilities.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/README.md`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/tests/console-smoke.spec.ts`

- [ ] **Step 1: Write the failing smoke test for enabled formerly-disabled surfaces**

```ts
test("shows live machine agent actions after capability rollout", async ({ page }) => {
  await page.goto("/machines");
  await expect(page.getByRole("button", { name: "Install agent" })).toBeEnabled();
});
```

- [ ] **Step 2: Run the smoke test to verify the old disabled state still appears**

Run: `corepack pnpm e2e`
Expected: FAIL until the capability flag flips and the real actions render.

- [ ] **Step 3: Enable the new capabilities and document the rollout status**

```ts
export const consoleCapabilities = {
  threadHub: true,
  threadWorkspace: true,
  approvals: true,
  steerTurn: true,
  interruptTurn: true,
  dashboardMetrics: true,
  agentLifecycle: true,
} as const;
```

```md
## Capability Rollout Status

- `dashboardMetrics`: enabled
- `agentLifecycle`: enabled
- `environmentWrite`: enabled
```

- [ ] **Step 4: Run final verification**

Run: `go test ./... && corepack pnpm test && corepack pnpm build && corepack pnpm e2e`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add console/src/gateway/capabilities.ts console/README.md console/tests/console-smoke.spec.ts
git commit -m "docs: mark gateway capability gaps as delivered"
```
