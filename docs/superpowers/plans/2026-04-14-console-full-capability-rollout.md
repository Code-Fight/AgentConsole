# Console Full Capability Rollout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement every capability recorded in `docs/2026-04-13-console-gateway-capability-checklist.csv` by wiring existing Gateway/Client capabilities into the active Console and adding the missing control-plane and runtime capabilities where they do not yet exist.

**Architecture:** The rollout has to start by fixing the active Console entrypoint. Right now `console/src/main.tsx` renders `console/src/design-source/App.tsx` directly, so the user-visible UI is mostly local state and mock data even where Gateway-backed pages already exist elsewhere in the repo. The implementation therefore proceeds in four layers: first establish a Gateway-backed active Console host, then wire all already-implemented northbound/southbound capabilities into that host, then add missing control-plane APIs and Console preferences, and finally introduce the missing machine-agent lifecycle model and the remaining management capabilities.

**Tech Stack:** React 19, TypeScript, Vite, Go Gateway API, Go client runtime/supervisor code, Codex App Server integration, Vitest, Playwright, Go test, dev integration Docker environment.

---

### Task 1: Replace The Active Mock Console Entry With A Gateway-Backed Host

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/main.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-host/app-root.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/App.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/MachinePanel.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/ThreadItem.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/SessionChat.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-host/use-console-host.ts`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-host/console-host-router.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/app/shell.test.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/tests/console-smoke.spec.ts`

- [ ] **Step 1: Write failing frontend tests that assert the active app no longer renders mock-only flows**

Add or update tests so they fail while `main.tsx` still mounts `design-source/App.tsx` directly:
- a unit test that expects the app root to fetch `/threads` and `/machines`
- a smoke test that verifies thread list data comes from mocked HTTP routes rather than bundled `mockData`
- a negative assertion that locally generated fake assistant replies are not used in the active workspace

- [ ] **Step 2: Run the failing frontend tests**

Run:
```bash
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/app/shell.test.tsx
corepack pnpm playwright test console/tests/console-smoke.spec.ts
```

Expected:
- shell test fails because `DesignSourceAppRoot` still renders the local-state app
- smoke test fails because the workspace is still driven by mock sessions and `setTimeout` reply logic

- [ ] **Step 3: Introduce a Gateway-backed host that treats `design-source` components as presentational building blocks**

Implementation requirements:
- stop mounting `App` as a self-contained local-state app from `main.tsx`
- move active state, routing, and side effects into a `design-host` layer
- change `design-source/App.tsx` from a local-state root into a composable shell that receives page data and callbacks via props
- keep the imported look and component tree as intact as possible while removing its ownership of `mockData`

- [ ] **Step 4: Re-run unit and smoke tests**

Run:
```bash
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/app/shell.test.tsx
corepack pnpm playwright test console/tests/console-smoke.spec.ts
```

Expected:
- PASS for the updated host-level tests
- the app root now depends on Gateway-backed hooks instead of bundled mock state

- [ ] **Step 5: Commit**

```bash
git add console/src/main.tsx console/src/design-host/app-root.tsx console/src/design-source/App.tsx console/src/design-source/components/MachinePanel.tsx console/src/design-source/components/ThreadItem.tsx console/src/design-source/components/SessionChat.tsx console/src/design-host/use-console-host.ts console/src/design-host/console-host-router.tsx console/src/app/shell.test.tsx console/tests/console-smoke.spec.ts
git commit -m "feat: activate gateway-backed console host"
```

### Task 2: Publish Capability Snapshot And Persisted Console Preferences From Gateway

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/capability.go`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/console_preferences.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/settings/store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/settings/memory_store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/settings/file_store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/common/api/types.ts`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-console-preferences.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/capabilities.ts`

- [ ] **Step 1: Write failing Gateway tests for `/capabilities` and persisted Console preferences**

Add tests for:
- `GET /capabilities`
- `GET /settings/console`
- `PUT /settings/console`

Cover fields needed by the CSV:
- `consoleUrl`
- `apiKey`
- `profile`
- `safetyPolicy`
- `lastThreadId`

- [ ] **Step 2: Run the failing Gateway tests**

Run:
```bash
go test ./gateway/internal/api -run 'TestCapabilities|TestConsoleSettings' -count=1
```

Expected:
- FAIL because neither the capability snapshot nor console preference endpoints exist

- [ ] **Step 3: Extend the settings store and API layer**

Implementation requirements:
- add a shared `CapabilitySnapshot` domain model
- add a shared `ConsolePreferences` domain model
- persist Console preferences in `gateway/internal/settings` alongside existing agent config data
- return capability flags based on real backend support, not hardcoded frontend values
- keep the settings store file backward compatible with existing JSON on disk

- [ ] **Step 4: Connect Console capability gating and preference loading**

Implementation requirements:
- replace hardcoded capability assumptions in the active host with a capability fetch
- load and save Console preferences from the new Gateway endpoints
- use `lastThreadId` as the new source of truth for restoring the last active thread when the Console opens

- [ ] **Step 5: Run verification**

Run:
```bash
go test ./gateway/internal/settings ./gateway/internal/api -count=1
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/common/api/http.test.ts src/app/shell.test.tsx
```

Expected:
- PASS for new API and store tests
- PASS for Console capability/preference wiring tests

- [ ] **Step 6: Commit**

```bash
git add common/domain/capability.go common/domain/console_preferences.go gateway/internal/settings/store.go gateway/internal/settings/memory_store.go gateway/internal/settings/file_store.go gateway/internal/api/server.go gateway/internal/api/server_test.go console/src/common/api/types.ts console/src/gateway/use-console-preferences.ts console/src/gateway/capabilities.ts
git commit -m "feat: add console preferences and capability snapshot"
```

### Task 3: Wire Existing Thread Hub And Workspace Capabilities Into The Active Console

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/App.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/MachinePanel.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/ThreadItem.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/SessionChat.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-thread-hub.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-thread-workspace.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/thread-view-model.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/threads-page.test.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/thread-workspace-page.test.tsx`

- [ ] **Step 1: Write failing tests for the active design-source thread hub and workspace**

Cover:
- thread list rendering from `/threads`
- machine inventory rendering from `/machines`
- selecting a thread loads `/threads/{threadId}` and `/machines/{machineId}`
- sending a prompt starts a real turn request
- websocket `turn.delta`, `approval.required`, `approval.resolved`, `turn.completed`, and `turn.failed` update the active workspace

- [ ] **Step 2: Run the failing Console tests**

Run:
```bash
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/pages/threads-page.test.tsx src/pages/thread-workspace-page.test.tsx
```

Expected:
- FAIL because the active `design-source` components still use local sessions and fake assistant replies

- [ ] **Step 3: Connect thread list, thread selection, start turn, and realtime workspace behavior**

Implementation requirements:
- adapt `MachinePanel` to render thread data from the Gateway thread hub adapter
- adapt `SessionChat` to render live workspace messages, approvals, active-turn state, and machine status
- remove the local `setTimeout` agent reply flow
- store the current thread selection back into `lastThreadId`

- [ ] **Step 4: Add thread rename on top of existing thread metadata**

Implementation requirements:
- add a northbound rename endpoint, preferably `PATCH /threads/{threadId}`
- persist renamed titles at the Gateway control plane so titles survive refresh even when Codex list/read returns blank names
- keep the existing client-side cached-title fallback logic intact

- [ ] **Step 5: Run verification**

Run:
```bash
go test ./gateway/internal/api -run TestThread -count=1
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/pages/threads-page.test.tsx src/pages/thread-workspace-page.test.tsx
corepack pnpm playwright test console/tests/console-smoke.spec.ts
```

Expected:
- PASS for thread API tests
- PASS for active Console hub/workspace tests
- smoke route exercises real thread/workspace flow instead of mock data

- [ ] **Step 6: Commit**

```bash
git add console/src/design-source/App.tsx console/src/design-source/components/MachinePanel.tsx console/src/design-source/components/ThreadItem.tsx console/src/design-source/components/SessionChat.tsx console/src/gateway/use-thread-hub.ts console/src/gateway/use-thread-workspace.ts console/src/gateway/thread-view-model.ts console/src/pages/threads-page.test.tsx console/src/pages/thread-workspace-page.test.tsx gateway/internal/api/server.go gateway/internal/api/server_test.go
git commit -m "feat: connect active console thread workflows"
```

### Task 4: Connect Existing Environment And Settings Backends To The Active Console

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/Environment.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/Settings.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-environment-page.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-settings-page.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/environment-page.test.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/settings-page.test.tsx`

- [ ] **Step 1: Write failing tests for active Environment and Settings integration**

Cover:
- environment lists load from `/environment/skills`, `/environment/mcps`, `/environment/plugins`
- existing MCP create/update/delete and enable/disable actions work from the active UI
- settings list agent types from `/settings/agents`
- global default load/save works
- machine override load/save/delete/apply works

- [ ] **Step 2: Run the failing Console tests**

Run:
```bash
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/pages/environment-page.test.tsx src/pages/settings-page.test.tsx
```

Expected:
- FAIL because the active design-source Environment and Settings components still use local form state and `console.log`

- [ ] **Step 3: Replace local forms with Gateway-backed adapter state**

Implementation requirements:
- remove the local `skills`, `mcps`, `plugins`, and settings document state from the active root where Gateway already owns truth
- map design-source dialogs and action buttons onto existing environment/settings hooks
- keep unsupported actions explicit until their backend slices land in later tasks

- [ ] **Step 4: Run verification**

Run:
```bash
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/pages/environment-page.test.tsx src/pages/settings-page.test.tsx
corepack pnpm build
```

Expected:
- PASS for Environment and Settings tests
- successful production build

- [ ] **Step 5: Commit**

```bash
git add console/src/design-source/components/Environment.tsx console/src/design-source/components/Settings.tsx console/src/gateway/use-environment-page.ts console/src/gateway/use-settings-page.ts console/src/pages/environment-page.test.tsx console/src/pages/settings-page.test.tsx
git commit -m "feat: connect active console environment and settings"
```

### Task 5: Add Missing Environment Write Capabilities For Skills And Normalize Plugin Install Semantics

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/protocol/messages.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/types/interfaces.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/manager/manager.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/codex/environment.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/codex/appserver_client_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/cmd/client/main.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/Environment.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/environment-page.test.tsx`

- [ ] **Step 1: Decide and document the real semantics for local-only environment affordances**

Normalize the current mock-only controls into real product actions:
- `Add skill` becomes `create/install skill scaffold on target machine`
- `Delete skill` becomes `remove installed skill scaffold from target machine`
- `Add plugin record` becomes real plugin install, with the UI updated to collect the install inputs the backend actually needs

- [ ] **Step 2: Write failing Go and Console tests for the new environment actions**

Cover:
- `POST /environment/skills`
- `DELETE /environment/skills/{id}`
- revised plugin install path from the active UI

- [ ] **Step 3: Implement the new southbound commands and northbound API**

Implementation requirements:
- extend runtime interfaces and protocol messages for skill create/delete
- implement Codex-side file/config mutations for skill scaffold create/delete
- revise the active Environment dialogs so they submit real install/delete data instead of only local labels

- [ ] **Step 4: Run verification**

Run:
```bash
go test ./client/internal/agent/codex ./client/internal/agent/manager ./gateway/internal/api -count=1
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/pages/environment-page.test.tsx
```

Expected:
- PASS for new skill create/delete and plugin install tests
- PASS for active Environment UI tests

- [ ] **Step 5: Commit**

```bash
git add common/protocol/messages.go client/internal/agent/types/interfaces.go client/internal/agent/manager/manager.go client/internal/agent/codex/environment.go client/internal/agent/codex/appserver_client_test.go client/cmd/client/main.go gateway/internal/api/server.go gateway/internal/api/server_test.go console/src/design-source/components/Environment.tsx console/src/pages/environment-page.test.tsx
git commit -m "feat: add missing environment write capabilities"
```

### Task 6: Introduce Managed Agents, Agent-Scoped Routing, And Machine Agent Lifecycle

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/agent_instance.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/machine.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/thread.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/environment.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/protocol/messages.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/registry/store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/runtimeindex/store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/routing/router.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/websocket/client_hub.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/config/config.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/types/interfaces.go`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/manager/supervisor.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/registry/registry.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/cmd/client/main.go`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/codex/instance_runner.go`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/codex/instance_runner_test.go`

- [ ] **Step 1: Write failing domain and API tests for managed machine agents**

Cover:
- machine summaries include agent inventory
- create thread requires `agentId`
- threads and environment resources retain `agentId`
- `POST /machines/{machineId}/agents`
- `DELETE /machines/{machineId}/agents/{agentId}`
- per-agent config read/save in Machines UI

- [ ] **Step 2: Run the failing Go tests**

Run:
```bash
go test ./common/... ./gateway/internal/... ./client/... -run 'Agent|Machine|Thread' -count=1
```

Expected:
- FAIL because the current system still assumes one runtime named `codex` per machine

- [ ] **Step 3: Refactor the client into a small supervisor over multiple managed agent runtimes**

Implementation requirements:
- each managed agent gets its own identity, config document, and isolated working home/config directory
- the client process becomes a machine-level supervisor that can start, stop, install, delete, and route commands to agent instances
- thread creation and environment mutations become agent-scoped
- router and runtime index track `threadId -> machineId + agentId`, not only `threadId -> machineId`

- [ ] **Step 4: Connect the active Machines page and agent-aware create-thread flow**

Implementation requirements:
- install/delete/edit agent config on the active Machines page must call the new APIs
- the active create-thread dialog in `MachinePanel` must submit both `machineId` and `agentId`
- workspace/thread metadata must retain the chosen agent identity for later routing and rendering

- [ ] **Step 5: Run verification**

Run:
```bash
go test ./client/... ./gateway/internal/... -count=1
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/pages/machines-page.test.tsx src/pages/threads-page.test.tsx
```

Expected:
- PASS for new managed-agent domain, API, and supervisor tests
- PASS for Machines and create-thread Console tests

- [ ] **Step 6: Commit**

```bash
git add common/domain/agent_instance.go common/domain/machine.go common/domain/thread.go common/domain/environment.go common/protocol/messages.go gateway/internal/registry/store.go gateway/internal/runtimeindex/store.go gateway/internal/routing/router.go gateway/internal/websocket/client_hub.go gateway/internal/api/server.go gateway/internal/api/server_test.go client/internal/config/config.go client/internal/agent/types/interfaces.go client/internal/agent/manager/supervisor.go client/internal/agent/registry/registry.go client/cmd/client/main.go client/internal/agent/codex/instance_runner.go client/internal/agent/codex/instance_runner_test.go console/src/design-source/components/MachinePanel.tsx console/src/design-source/components/Machines.tsx
git commit -m "feat: add managed agents and agent-scoped routing"
```

### Task 7: Implement Remaining Console Settings Surfaces And Wire Them To Gateway Preferences

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/console_preferences.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/settings/store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/settings/memory_store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/settings/file_store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/Settings.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/settings-page.test.tsx`

- [ ] **Step 1: Write failing tests for the remaining settings affordances**

Cover:
- `Console URL`
- `API Key`
- `Console Profile`
- `Safety Policy`
- persistence across reload through Gateway settings endpoints

- [ ] **Step 2: Run the failing tests**

Run:
```bash
go test ./gateway/internal/settings ./gateway/internal/api -run TestConsoleSettings -count=1
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/pages/settings-page.test.tsx
```

Expected:
- FAIL for fields that are still only local component state

- [ ] **Step 3: Extend Console preference persistence and wire the active Settings screen**

Implementation requirements:
- use the same `settings/console` document added in Task 2
- fill out remaining fields not needed by earlier phases
- remove all `console.log`-only save paths from the active Settings component

- [ ] **Step 4: Run verification**

Run:
```bash
go test ./gateway/internal/settings ./gateway/internal/api -count=1
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test src/pages/settings-page.test.tsx
```

Expected:
- PASS for persisted settings behavior

- [ ] **Step 5: Commit**

```bash
git add common/domain/console_preferences.go gateway/internal/settings/store.go gateway/internal/settings/memory_store.go gateway/internal/settings/file_store.go gateway/internal/api/server.go console/src/design-source/components/Settings.tsx console/src/pages/settings-page.test.tsx
git commit -m "feat: wire remaining console settings preferences"
```

### Task 8: Add Overview Metrics And Close The Final CSV Gaps

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/common/domain/metrics.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/runtimeindex/store.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/runtimeindex/store_test.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/App.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design-source/components/Overview.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/tests/console-smoke.spec.ts`

- [ ] **Step 1: Write failing tests for overview metrics and final CSV completion**

Cover:
- `GET /overview/metrics`
- an active Console route or navigation entry that renders overview metrics
- a final regression check that every CSV row is either real Gateway-backed UI or a newly implemented backend capability

- [ ] **Step 2: Run the failing tests**

Run:
```bash
go test ./gateway/internal/runtimeindex ./gateway/internal/api -run TestOverviewMetrics -count=1
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm playwright test console/tests/console-smoke.spec.ts
```

Expected:
- FAIL because overview metrics are still absent

- [ ] **Step 3: Implement the overview endpoint and expose it in the active Console**

Implementation requirements:
- aggregate metrics from registry and runtime index
- add an overview management surface to the active design-source app only if it remains the chosen primary shell after Tasks 1-7
- if the host is route-based by then, expose the overview route there instead

- [ ] **Step 4: Run the full verification matrix**

Run:
```bash
go test ./...
cd /Users/zfcode/Documents/DEV/CodingAgentGateway/console
corepack pnpm test
corepack pnpm build
corepack pnpm playwright test
cd /Users/zfcode/Documents/DEV/CodingAgentGateway
./testenv/settings-e2e/run.sh
./testenv/dev-integration/run.sh up
./testenv/dev-integration/run.sh down
```

Expected:
- all Go tests pass
- all Console unit tests pass
- production build succeeds
- Playwright and settings e2e pass
- dev integration stack boots and shuts down cleanly

- [ ] **Step 5: Update the CSV and README to reflect the finished state**

Files:
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/docs/2026-04-13-console-gateway-capability-checklist.csv`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/README.md`

- [ ] **Step 6: Commit**

```bash
git add common/domain/metrics.go gateway/internal/runtimeindex/store.go gateway/internal/runtimeindex/store_test.go gateway/internal/api/server.go console/src/design-source/App.tsx console/src/design-source/components/Overview.tsx console/tests/console-smoke.spec.ts docs/2026-04-13-console-gateway-capability-checklist.csv console/README.md
git commit -m "feat: complete console capability rollout"
```

### Spec Coverage Notes

- The plan covers every row currently present in `docs/2026-04-13-console-gateway-capability-checklist.csv`.
- It also accounts for two structural constraints exposed by the active `design-source` app that are larger than the original CSV wording:
  - thread creation is already agent-aware in the current UI, so routing and thread metadata must become agent-scoped
  - Environment dialogs already ask the user to target machine + agent, so resource operations must become agent-aware or be redesigned to a machine-scoped UX during implementation

### Risks

- The largest risk is Task 6. The current client runtime model is single-runtime-per-machine (`client/internal/config/config.go`, `client/cmd/client/main.go`, `client/internal/agent/types/interfaces.go`), so true managed agent lifecycle is a real architecture expansion rather than an API-only patch.
- The current active Console and the existing Gateway-backed adapter layer diverged. The implementation should not try to keep both as competing app roots after Task 1.
- Some mock-only design-source affordances need semantic normalization, not literal backend reproduction. `Add plugin record` and `Add skill` should become real install/create flows, not remain fake local list entries.

