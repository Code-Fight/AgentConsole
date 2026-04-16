# Machine Identity And Name Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist a client-generated `machineId`, surface a friendly `machine.name`, and verify the full flow in `testenv/dev-integration`.

**Architecture:** The client becomes the source of machine identity by storing a generated `machineId` under its local state directory and resolving a runtime `machineName` from `MACHINE_NAME` or hostname. Gateway continues to route by `machineId`, but it now accepts the friendly name during registration and snapshots. Console keeps its existing data model and consistently prefers `machine.name` for display.

**Tech Stack:** Go, React, Vitest, Docker Compose

---

### Task 1: Lock the expected client identity behavior in tests

**Files:**
- Modify: `client/internal/config/config_test.go`
- Modify: `client/cmd/client/main_test.go`

- [ ] Add failing config tests for persisted `machineId`, `MACHINE_NAME`, hostname fallback, and stable managed-agent path derivation.
- [ ] Run `go test ./client/internal/config ./client/cmd/client -run 'TestRead|TestBuildMachineSnapshot|TestRunClient'` from repo root and verify the new assertions fail for the expected reasons.

### Task 2: Implement persisted client identity and friendly machine naming

**Files:**
- Modify: `client/internal/config/config.go`
- Create: `client/internal/config/machine_identity.go`
- Modify: `client/cmd/client/main.go`

- [ ] Implement a small identity store that loads or creates `machineId` as `hostname_uuid` under the client state directory.
- [ ] Extend config loading to resolve `MachineName` from `MACHINE_NAME`, hostname, then `MachineID`.
- [ ] Thread `MachineName` through machine snapshot construction and client session registration.
- [ ] Run the targeted Go tests again and verify they pass.

### Task 3: Lock gateway register/name propagation in tests

**Files:**
- Modify: `client/internal/gateway/session_test.go`
- Modify: `gateway/internal/websocket/client_hub_test.go`
- Modify: `gateway/cmd/gateway/main_test.go`

- [ ] Add failing tests proving `client.register` carries the friendly name and Gateway stores/broadcasts it immediately.
- [ ] Run `go test ./client/internal/gateway ./gateway/internal/websocket ./gateway/cmd/gateway -run 'TestSession|TestClientHub|TestBuildServerHandler'` and verify the new assertions fail first.

### Task 4: Implement gateway register payload handling and Console-facing display consistency

**Files:**
- Modify: `common/protocol/messages.go`
- Modify: `client/internal/gateway/session.go`
- Modify: `gateway/internal/websocket/client_hub.go`
- Modify: `console/src/pages/threads-page.test.tsx`
- Modify: `console/src/pages/thread-workspace-page.test.tsx`
- Modify: `console/src/pages/settings-page.test.tsx`

- [ ] Add a typed register payload containing the machine name.
- [ ] Decode the register payload in Gateway and use it to seed `machine.name` before the first snapshot.
- [ ] Add or update Console tests to prove UI copies prefer `machine.name` and only fall back to `machine.id` when absent.
- [ ] Run the targeted Go and frontend tests and verify they pass.

### Task 5: Update dev-integration and validate the end-to-end behavior

**Files:**
- Modify: `testenv/dev-integration/docker-compose.yml`
- Modify: `testenv/dev-integration/run.sh`
- Modify: `testenv/dev-integration/README.md`
- Modify: `testenv/settings-e2e/docker-compose.yml`

- [ ] Remove the old `MACHINE_ID` dependency from dev/test environments, add `MACHINE_NAME`, and persist client local state across container restarts.
- [ ] Update the helper script to wait on a friendly machine name instead of a fixed `machineId`.
- [ ] Run focused automated tests for the touched packages.
- [ ] Run `./testenv/dev-integration/run.sh up`, inspect `/machines` and the Console output, then run `./testenv/dev-integration/run.sh down`.
- [ ] Capture the exact verification results so the final report can point to concrete evidence.
