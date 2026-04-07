# Code Agent Gateway V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Codex-first, single-tenant gateway with a responsive H5 console, one machine client per host, unified thread control, and unified skill/MCP/plugin operations.

**Architecture:** Use a TypeScript monorepo with three deployables: `apps/gateway`, `apps/client-codex`, and `apps/console`. Put canonical domain types and northbound/southbound protocol schemas in shared packages so the H5, Gateway, and machine client agree on resource identity and event shapes. Keep V1 state in memory at the Gateway; recover control-plane views from machine re-registration and snapshot replay.

**Tech Stack:** `pnpm` workspace, TypeScript, Node.js 22, Fastify, `ws`, Zod, React, Vite, Zustand, Vitest, React Testing Library, Playwright

---

## Scope Check

Although the spec spans three deployables, this remains one implementation plan because V1 only works if the shared protocol, Gateway, machine client, and H5 land as one coherent vertical slice. Each task below ends in a working, testable increment.

## File Structure

- Create: `package.json`
- Create: `pnpm-workspace.yaml`
- Create: `tsconfig.base.json`
- Create: `.gitignore`
- Create: `vitest.workspace.ts`
- Create: `playwright.config.ts`
- Create: `apps/gateway/package.json`
- Create: `apps/gateway/tsconfig.json`
- Create: `apps/gateway/src/server.ts`
- Create: `apps/gateway/src/app.ts`
- Create: `apps/gateway/src/config.ts`
- Create: `apps/gateway/src/store/registry-store.ts`
- Create: `apps/gateway/src/store/thread-store.ts`
- Create: `apps/gateway/src/services/command-broker.ts`
- Create: `apps/gateway/src/http/routes/machines-routes.ts`
- Create: `apps/gateway/src/http/routes/threads-routes.ts`
- Create: `apps/gateway/src/http/routes/environment-routes.ts`
- Create: `apps/gateway/src/ws/client-session-hub.ts`
- Create: `apps/gateway/src/ws/console-session-hub.ts`
- Create: `apps/gateway/src/app.test.ts`
- Create: `apps/client-codex/package.json`
- Create: `apps/client-codex/tsconfig.json`
- Create: `apps/client-codex/src/main.ts`
- Create: `apps/client-codex/src/config.ts`
- Create: `apps/client-codex/src/gateway/gateway-session.ts`
- Create: `apps/client-codex/src/gateway/command-dispatcher.ts`
- Create: `apps/client-codex/src/codex-adapter/adapter.ts`
- Create: `apps/client-codex/src/codex-adapter/fake-codex-adapter.ts`
- Create: `apps/client-codex/src/codex-adapter/codex-app-server-adapter.ts`
- Create: `apps/client-codex/src/codex-config/config-store.ts`
- Create: `apps/client-codex/src/snapshot/snapshot-builder.ts`
- Create: `apps/client-codex/src/gateway/gateway-session.test.ts`
- Create: `apps/console/package.json`
- Create: `apps/console/tsconfig.json`
- Create: `apps/console/vite.config.ts`
- Create: `apps/console/src/main.tsx`
- Create: `apps/console/src/app/router.tsx`
- Create: `apps/console/src/app/shell.tsx`
- Create: `apps/console/src/app/http-client.ts`
- Create: `apps/console/src/app/ws-client.ts`
- Create: `apps/console/src/app/store.ts`
- Create: `apps/console/src/features/overview/overview-page.tsx`
- Create: `apps/console/src/features/machines/machines-page.tsx`
- Create: `apps/console/src/features/threads/threads-page.tsx`
- Create: `apps/console/src/features/threads/thread-workspace-page.tsx`
- Create: `apps/console/src/features/environment/environment-page.tsx`
- Create: `apps/console/src/styles.css`
- Create: `apps/console/src/app/shell.test.tsx`
- Create: `packages/domain/package.json`
- Create: `packages/domain/tsconfig.json`
- Create: `packages/domain/src/index.ts`
- Create: `packages/domain/src/machine.ts`
- Create: `packages/domain/src/thread.ts`
- Create: `packages/domain/src/environment.ts`
- Create: `packages/domain/src/machine.test.ts`
- Create: `packages/protocol/package.json`
- Create: `packages/protocol/tsconfig.json`
- Create: `packages/protocol/src/index.ts`
- Create: `packages/protocol/src/resources.ts`
- Create: `packages/protocol/src/commands.ts`
- Create: `packages/protocol/src/events.ts`
- Create: `packages/protocol/src/schemas.test.ts`
- Create: `tests/integration/thread-lifecycle.test.ts`
- Create: `tests/integration/thread-lifecycle-harness.ts`
- Create: `tests/integration/environment-management.test.ts`
- Create: `tests/integration/environment-management-harness.ts`
- Create: `tests/e2e/console-smoke.spec.ts`

## Task 1: Bootstrap Workspace And Shared Domain Primitives

**Files:**
- Create: `package.json`
- Create: `pnpm-workspace.yaml`
- Create: `tsconfig.base.json`
- Create: `.gitignore`
- Create: `vitest.workspace.ts`
- Create: `packages/domain/package.json`
- Create: `packages/domain/tsconfig.json`
- Create: `packages/domain/src/index.ts`
- Create: `packages/domain/src/machine.ts`
- Test: `packages/domain/src/machine.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// packages/domain/src/machine.test.ts
import { describe, expect, it } from "vitest";
import { buildMachineId, buildRuntimeId } from "./machine";

describe("machine identity helpers", () => {
  it("builds stable machine and runtime ids", () => {
    expect(buildMachineId({ hostname: "mac-mini-01", agentKind: "codex" })).toBe(
      "mac-mini-01:codex",
    );
    expect(buildRuntimeId({ machineId: "mac-mini-01:codex", runtimeKind: "codex" })).toBe(
      "mac-mini-01:codex/runtime/codex",
    );
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest packages/domain/src/machine.test.ts`
Expected: FAIL with `Cannot find module './machine'` or `buildMachineId is not defined`

- [ ] **Step 3: Write minimal implementation**

```json
// package.json
{
  "name": "code-agent-gateway",
  "private": true,
  "packageManager": "pnpm@10.0.0",
  "scripts": {
    "build": "pnpm -r build",
    "test": "vitest run",
    "test:watch": "vitest",
    "e2e": "playwright test"
  },
  "devDependencies": {
    "@playwright/test": "^1.54.0",
    "typescript": "^5.9.0",
    "vitest": "^3.2.0"
  }
}
```

```yaml
# pnpm-workspace.yaml
packages:
  - apps/*
  - packages/*
  - tests/*
```

```json
// tsconfig.base.json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "declaration": true,
    "composite": true,
    "baseUrl": ".",
    "paths": {
      "@cag/domain": ["packages/domain/src/index.ts"],
      "@cag/protocol": ["packages/protocol/src/index.ts"]
    }
  }
}
```

```gitignore
# .gitignore
node_modules
dist
coverage
playwright-report
.superpowers
```

```ts
// vitest.workspace.ts
import { defineWorkspace } from "vitest/config";

export default defineWorkspace([
  "packages/*",
  "apps/*",
  "tests/*",
]);
```

```json
// packages/domain/package.json
{
  "name": "@cag/domain",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./src/index.ts",
  "scripts": {
    "build": "tsc -p tsconfig.json",
    "test": "vitest run src/machine.test.ts"
  }
}
```

```json
// packages/domain/tsconfig.json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": "src"
  },
  "include": ["src"]
}
```

```ts
// packages/domain/src/machine.ts
export function buildMachineId(input: { hostname: string; agentKind: string }): string {
  return `${input.hostname}:${input.agentKind}`;
}

export function buildRuntimeId(input: { machineId: string; runtimeKind: string }): string {
  return `${input.machineId}/runtime/${input.runtimeKind}`;
}
```

```ts
// packages/domain/src/thread.ts
export type ThreadStatus = "created" | "ready" | "running" | "waiting_input" | "archived";

export type ThreadSummary = {
  threadId: string;
  machineId: string;
  title: string;
  status: ThreadStatus;
};
```

```ts
// packages/domain/src/environment.ts
export type EnvironmentKind = "skill" | "mcp" | "plugin";

export type EnvironmentResource = {
  resourceId: string;
  machineId: string;
  kind: EnvironmentKind;
  displayName: string;
  status: "enabled" | "disabled" | "auth_required" | "error" | "unknown";
};
```

```ts
// packages/domain/src/index.ts
export * from "./machine";
export * from "./thread";
export * from "./environment";
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest packages/domain/src/machine.test.ts`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
test -d .git || git init
git add package.json pnpm-workspace.yaml tsconfig.base.json .gitignore vitest.workspace.ts packages/domain
git commit -m "chore: bootstrap workspace and domain package"
```

## Task 2: Define Shared Resource, Command, And Event Schemas

**Files:**
- Create: `packages/protocol/package.json`
- Create: `packages/protocol/tsconfig.json`
- Create: `packages/protocol/src/index.ts`
- Create: `packages/protocol/src/resources.ts`
- Create: `packages/protocol/src/commands.ts`
- Create: `packages/protocol/src/events.ts`
- Test: `packages/protocol/src/schemas.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// packages/protocol/src/schemas.test.ts
import { describe, expect, it } from "vitest";
import { commandEnvelopeSchema, resourceSnapshotSchema } from "./index";

describe("protocol schemas", () => {
  it("accepts a start thread command and a plugin snapshot", () => {
    expect(
      commandEnvelopeSchema.parse({
        commandId: "cmd-1",
        type: "command.startThread",
        payload: { machineId: "mac-mini-01:codex", title: "Investigate flaky test" },
      }),
    ).toMatchObject({ type: "command.startThread" });

    expect(
      resourceSnapshotSchema.parse({
        resourceId: "plugin:figma",
        kind: "plugin",
        machineId: "mac-mini-01:codex",
        displayName: "Figma Plugin",
        scope: "user",
        status: "enabled",
        source: "plugin-bundle",
        restartRequired: false,
        lastObservedAt: "2026-04-07T09:00:00.000Z",
      }),
    ).toMatchObject({ kind: "plugin" });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest packages/protocol/src/schemas.test.ts`
Expected: FAIL with `Cannot find module './index'`

- [ ] **Step 3: Write minimal implementation**

```json
// packages/protocol/package.json
{
  "name": "@cag/protocol",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./src/index.ts",
  "dependencies": {
    "zod": "^4.0.0"
  },
  "scripts": {
    "build": "tsc -p tsconfig.json",
    "test": "vitest run src/schemas.test.ts"
  }
}
```

```json
// packages/protocol/tsconfig.json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": "src"
  },
  "include": ["src"]
}
```

```ts
// packages/protocol/src/resources.ts
import { z } from "zod";

export const resourceKindSchema = z.enum(["skill", "mcp", "plugin"]);
export const resourceSnapshotSchema = z.object({
  resourceId: z.string(),
  kind: resourceKindSchema,
  machineId: z.string(),
  displayName: z.string(),
  scope: z.enum(["system", "user", "repo", "project-config", "plugin-bundled"]),
  status: z.enum(["enabled", "disabled", "auth_required", "error", "unknown"]),
  source: z.enum(["builtin", "curated", "local-path", "config-entry", "plugin-bundle"]),
  restartRequired: z.boolean(),
  lastObservedAt: z.string(),
});
```

```ts
// packages/protocol/src/commands.ts
import { z } from "zod";

export const commandEnvelopeSchema = z.discriminatedUnion("type", [
  z.object({
    commandId: z.string(),
    type: z.literal("command.startThread"),
    payload: z.object({
      machineId: z.string(),
      title: z.string(),
    }),
  }),
  z.object({
    commandId: z.string(),
    type: z.literal("command.startTurn"),
    payload: z.object({
      threadId: z.string(),
      prompt: z.string(),
    }),
  }),
  z.object({
    commandId: z.string(),
    type: z.literal("command.toggleSkill"),
    payload: z.object({
      machineId: z.string(),
      resourceId: z.string(),
      enabled: z.boolean(),
    }),
  }),
]);
```

```ts
// packages/protocol/src/events.ts
import { z } from "zod";

export const eventEnvelopeSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("event.commandAccepted"),
    commandId: z.string(),
    machineId: z.string(),
  }),
  z.object({
    type: z.literal("event.turn.delta"),
    threadId: z.string(),
    turnId: z.string(),
    delta: z.string(),
  }),
  z.object({
    type: z.literal("event.resource.changed"),
    machineId: z.string(),
    resourceId: z.string(),
  }),
]);
```

```ts
// packages/protocol/src/index.ts
export * from "./commands";
export * from "./events";
export * from "./resources";
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest packages/protocol/src/schemas.test.ts`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
git add packages/protocol
git commit -m "feat: add shared protocol schemas"
```

## Task 3: Build The Gateway Skeleton With In-Memory Registry And HTTP Surface

**Files:**
- Create: `apps/gateway/package.json`
- Create: `apps/gateway/tsconfig.json`
- Create: `apps/gateway/src/config.ts`
- Create: `apps/gateway/src/store/registry-store.ts`
- Create: `apps/gateway/src/store/thread-store.ts`
- Create: `apps/gateway/src/http/routes/machines-routes.ts`
- Create: `apps/gateway/src/http/routes/threads-routes.ts`
- Create: `apps/gateway/src/http/routes/environment-routes.ts`
- Create: `apps/gateway/src/app.ts`
- Create: `apps/gateway/src/server.ts`
- Test: `apps/gateway/src/app.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// apps/gateway/src/app.test.ts
import { describe, expect, it } from "vitest";
import { buildGatewayApp } from "./app";

describe("gateway app", () => {
  it("serves health and an empty machine list", async () => {
    const app = buildGatewayApp();

    const health = await app.inject({ method: "GET", url: "/health" });
    const machines = await app.inject({ method: "GET", url: "/machines" });

    expect(health.statusCode).toBe(200);
    expect(JSON.parse(health.body)).toEqual({ ok: true });
    expect(JSON.parse(machines.body)).toEqual({ items: [] });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest apps/gateway/src/app.test.ts`
Expected: FAIL with `Cannot find module './app'`

- [ ] **Step 3: Write minimal implementation**

```json
// apps/gateway/package.json
{
  "name": "@cag/gateway",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "dependencies": {
    "@cag/protocol": "workspace:*",
    "fastify": "^5.0.0"
  },
  "scripts": {
    "dev": "tsx watch src/server.ts",
    "build": "tsc -p tsconfig.json",
    "test": "vitest run src/app.test.ts"
  }
}
```

```json
// apps/gateway/tsconfig.json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": "src"
  },
  "include": ["src"]
}
```

```ts
// apps/gateway/src/config.ts
export function readGatewayConfig() {
  return {
    port: Number(process.env.PORT ?? 3000),
    host: process.env.HOST ?? "0.0.0.0",
  };
}
```

```ts
// apps/gateway/src/store/registry-store.ts
export type MachineSummary = {
  machineId: string;
  hostname: string;
  agentKind: "codex";
  status: "online" | "offline" | "unknown";
};

export class RegistryStore {
  #machines = new Map<string, MachineSummary>();

  listMachines(): MachineSummary[] {
    return [...this.#machines.values()];
  }

  upsertMachine(machine: MachineSummary): void {
    this.#machines.set(machine.machineId, machine);
  }
}
```

```ts
// apps/gateway/src/store/thread-store.ts
export type ThreadSummary = {
  threadId: string;
  machineId: string;
  title: string;
  status: "ready" | "running" | "waiting_input" | "archived";
};

export class ThreadStore {
  #threads = new Map<string, ThreadSummary>();

  listThreads(): ThreadSummary[] {
    return [...this.#threads.values()];
  }

  upsertThread(thread: ThreadSummary): void {
    this.#threads.set(thread.threadId, thread);
  }
}
```

```ts
// apps/gateway/src/http/routes/machines-routes.ts
import { FastifyInstance } from "fastify";
import { RegistryStore } from "../../store/registry-store";

export function registerMachineRoutes(app: FastifyInstance, store: RegistryStore): void {
  app.get("/machines", async () => ({ items: store.listMachines() }));
}
```

```ts
// apps/gateway/src/http/routes/threads-routes.ts
import { FastifyInstance } from "fastify";
import { ThreadStore } from "../../store/thread-store";

export function registerThreadRoutes(app: FastifyInstance, store: ThreadStore): void {
  app.get("/threads", async () => ({ items: store.listThreads() }));
}
```

```ts
// apps/gateway/src/http/routes/environment-routes.ts
import { FastifyInstance } from "fastify";

export function registerEnvironmentRoutes(app: FastifyInstance): void {
  app.get("/skills", async () => ({ items: [] }));
  app.get("/mcps", async () => ({ items: [] }));
  app.get("/plugins", async () => ({ items: [] }));
}
```

```ts
// apps/gateway/src/app.ts
import Fastify from "fastify";
import { registerEnvironmentRoutes } from "./http/routes/environment-routes";
import { registerMachineRoutes } from "./http/routes/machines-routes";
import { registerThreadRoutes } from "./http/routes/threads-routes";
import { RegistryStore } from "./store/registry-store";
import { ThreadStore } from "./store/thread-store";

export function buildGatewayApp() {
  const app = Fastify();
  const registryStore = new RegistryStore();
  const threadStore = new ThreadStore();

  app.get("/health", async () => ({ ok: true }));
  registerMachineRoutes(app, registryStore);
  registerThreadRoutes(app, threadStore);
  registerEnvironmentRoutes(app);

  return app;
}
```

```ts
// apps/gateway/src/server.ts
import { buildGatewayApp } from "./app";
import { readGatewayConfig } from "./config";

const app = buildGatewayApp();
const config = readGatewayConfig();
await app.listen({ port: config.port, host: config.host });
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest apps/gateway/src/app.test.ts`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
git add apps/gateway
git commit -m "feat: add gateway http skeleton"
```

## Task 4: Add Machine Client Session Management And A Fake Codex Adapter

**Files:**
- Create: `apps/client-codex/package.json`
- Create: `apps/client-codex/tsconfig.json`
- Create: `apps/client-codex/src/config.ts`
- Create: `apps/client-codex/src/gateway/gateway-session.ts`
- Create: `apps/client-codex/src/gateway/command-dispatcher.ts`
- Create: `apps/client-codex/src/codex-adapter/adapter.ts`
- Create: `apps/client-codex/src/codex-adapter/fake-codex-adapter.ts`
- Create: `apps/client-codex/src/snapshot/snapshot-builder.ts`
- Create: `apps/client-codex/src/main.ts`
- Test: `apps/client-codex/src/gateway/gateway-session.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// apps/client-codex/src/gateway/gateway-session.test.ts
import { describe, expect, it } from "vitest";
import { GatewaySession } from "./gateway-session";

describe("GatewaySession", () => {
  it("emits register and heartbeat frames", async () => {
    const frames: unknown[] = [];

    const session = new GatewaySession({
      machineId: "mac-mini-01:codex",
      send: (frame) => {
        frames.push(frame);
      },
      now: () => new Date("2026-04-07T09:00:00.000Z"),
    });

    session.register();
    session.heartbeat();

    expect(frames).toEqual([
      expect.objectContaining({ type: "client.register" }),
      expect.objectContaining({ type: "client.heartbeat" }),
    ]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest apps/client-codex/src/gateway/gateway-session.test.ts`
Expected: FAIL with `Cannot find module './gateway-session'`

- [ ] **Step 3: Write minimal implementation**

```json
// apps/client-codex/package.json
{
  "name": "@cag/client-codex",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "dependencies": {
    "@cag/domain": "workspace:*",
    "@cag/protocol": "workspace:*"
  },
  "scripts": {
    "dev": "tsx watch src/main.ts",
    "build": "tsc -p tsconfig.json",
    "test": "vitest run src/gateway/gateway-session.test.ts"
  }
}
```

```json
// apps/client-codex/tsconfig.json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": "src"
  },
  "include": ["src"]
}
```

```ts
// apps/client-codex/src/config.ts
export function readClientConfig() {
  return {
    machineId: process.env.MACHINE_ID ?? "mac-mini-01:codex",
    gatewayUrl: process.env.GATEWAY_URL ?? "ws://localhost:3000/client",
  };
}
```

```ts
// apps/client-codex/src/codex-adapter/adapter.ts
export interface CodexAdapter {
  listThreads(): Promise<Array<{ threadId: string; title: string; status: string }>>;
  getEnvironmentResources(): Promise<
    Array<{ resourceId: string; kind: "skill" | "mcp" | "plugin"; displayName: string; status: string }>
  >;
}
```

```ts
// apps/client-codex/src/codex-adapter/codex-app-server-adapter.ts
import { CodexAdapter } from "./adapter";

type AppServerRunner = {
  request<T>(method: string, payload?: unknown): Promise<T>;
};

export class CodexAppServerAdapter implements CodexAdapter {
  constructor(private readonly runner: AppServerRunner) {}

  async listThreads() {
    return this.runner.request<Array<{ threadId: string; title: string; status: string }>>("thread/list");
  }

  async getEnvironmentResources() {
    return this.runner.request<
      Array<{ resourceId: string; kind: "skill" | "mcp" | "plugin"; displayName: string; status: string }>
    >("environment/list");
  }
}
```

```ts
// apps/client-codex/src/codex-adapter/fake-codex-adapter.ts
import { CodexAdapter } from "./adapter";

export class FakeCodexAdapter implements CodexAdapter {
  async listThreads() {
    return [];
  }

  async getEnvironmentResources() {
    return [];
  }
}
```

```ts
// apps/client-codex/src/snapshot/snapshot-builder.ts
import { CodexAdapter } from "../codex-adapter/adapter";

export async function buildSnapshot(adapter: CodexAdapter) {
  return {
    threads: await adapter.listThreads(),
    resources: await adapter.getEnvironmentResources(),
  };
}
```

```ts
// apps/client-codex/src/gateway/gateway-session.ts
type FrameSender = (frame: unknown) => void;

export class GatewaySession {
  constructor(
    private readonly input: {
      machineId: string;
      send: FrameSender;
      now: () => Date;
    },
  ) {}

  register(): void {
    this.input.send({
      type: "client.register",
      machineId: this.input.machineId,
      connectedAt: this.input.now().toISOString(),
    });
  }

  heartbeat(): void {
    this.input.send({
      type: "client.heartbeat",
      machineId: this.input.machineId,
      sentAt: this.input.now().toISOString(),
    });
  }
}
```

```ts
// apps/client-codex/src/main.ts
import { readClientConfig } from "./config";
import { FakeCodexAdapter } from "./codex-adapter/fake-codex-adapter";
import { buildSnapshot } from "./snapshot/snapshot-builder";

const config = readClientConfig();
const adapter = new FakeCodexAdapter();
console.log(config.gatewayUrl, await buildSnapshot(adapter));
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest apps/client-codex/src/gateway/gateway-session.test.ts`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
git add apps/client-codex
git commit -m "feat: add machine client session and fake adapter"
```

## Task 5: Implement Thread Lifecycle Commands Through Gateway, Client, And Fake Adapter

**Files:**
- Create: `apps/gateway/src/services/command-broker.ts`
- Create: `apps/gateway/src/ws/client-session-hub.ts`
- Create: `apps/gateway/src/ws/console-session-hub.ts`
- Modify: `apps/gateway/src/app.ts`
- Modify: `apps/gateway/src/http/routes/threads-routes.ts`
- Modify: `apps/gateway/src/store/thread-store.ts`
- Modify: `apps/client-codex/src/codex-adapter/adapter.ts`
- Modify: `apps/client-codex/src/codex-adapter/fake-codex-adapter.ts`
- Modify: `apps/client-codex/src/gateway/command-dispatcher.ts`
- Test: `tests/integration/thread-lifecycle.test.ts`
- Create: `tests/integration/thread-lifecycle-harness.ts`

- [ ] **Step 1: Write the failing test**

```ts
// tests/integration/thread-lifecycle.test.ts
import { describe, expect, it } from "vitest";
import { InMemoryHarness } from "./thread-lifecycle-harness";

describe("thread lifecycle", () => {
  it("creates a thread, starts a turn, and streams deltas", async () => {
    const harness = await InMemoryHarness.start();

    const thread = await harness.createThread({
      machineId: "mac-mini-01:codex",
      title: "Fix flaky payment retries",
    });

    const deltas = await harness.startTurn({
      threadId: thread.threadId,
      prompt: "Inspect failing tests first",
    });

    expect(thread.status).toBe("ready");
    expect(deltas).toEqual(["Inspecting test output", "Found flaky retry timing"]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest tests/integration/thread-lifecycle.test.ts`
Expected: FAIL with `Cannot find module './thread-lifecycle-harness'` or missing create/start turn support

- [ ] **Step 3: Write minimal implementation**

```ts
// apps/client-codex/src/codex-adapter/adapter.ts
export interface CodexAdapter {
  createThread(input: { title: string }): Promise<{ threadId: string; title: string; status: "ready" }>;
  startTurn(input: { threadId: string; prompt: string }): AsyncIterable<string>;
  listThreads(): Promise<Array<{ threadId: string; title: string; status: string }>>;
  getEnvironmentResources(): Promise<
    Array<{ resourceId: string; kind: "skill" | "mcp" | "plugin"; displayName: string; status: string }>
  >;
}
```

```ts
// apps/client-codex/src/codex-adapter/fake-codex-adapter.ts
import { CodexAdapter } from "./adapter";

export class FakeCodexAdapter implements CodexAdapter {
  #threads = new Map<string, { threadId: string; title: string; status: "ready" }>();

  async createThread(input: { title: string }) {
    const thread = {
      threadId: `thread-${this.#threads.size + 1}`,
      title: input.title,
      status: "ready" as const,
    };
    this.#threads.set(thread.threadId, thread);
    return thread;
  }

  async *startTurn(input: { threadId: string; prompt: string }) {
    yield "Inspecting test output";
    yield "Found flaky retry timing";
  }

  async listThreads() {
    return [...this.#threads.values()];
  }

  async getEnvironmentResources() {
    return [];
  }
}
```

```ts
// apps/gateway/src/app.ts
import Fastify from "fastify";
import { registerEnvironmentRoutes } from "./http/routes/environment-routes";
import { registerMachineRoutes } from "./http/routes/machines-routes";
import { registerThreadRoutes } from "./http/routes/threads-routes";
import { CommandBroker } from "./services/command-broker";
import { RegistryStore } from "./store/registry-store";
import { ThreadStore } from "./store/thread-store";

const defaultBroker = new CommandBroker({
  async createThread(input) {
    return { threadId: `dev-${input.title}`, title: input.title, status: "ready" as const };
  },
  async startTurn() {
    return [];
  },
});

export function buildGatewayApp(input: { broker?: CommandBroker } = {}) {
  const app = Fastify();
  const registryStore = new RegistryStore();
  const threadStore = new ThreadStore();
  const broker = input.broker ?? defaultBroker;

  app.get("/health", async () => ({ ok: true }));
  registerMachineRoutes(app, registryStore);
  registerThreadRoutes(app, threadStore, broker);
  registerEnvironmentRoutes(app);

  return app;
}
```

```ts
// apps/gateway/src/services/command-broker.ts
export interface MachineCommandChannel {
  createThread(input: { machineId: string; title: string }): Promise<{ threadId: string; title: string; status: "ready" }>;
  startTurn(input: { threadId: string; prompt: string }): Promise<string[]>;
}

export class CommandBroker {
  constructor(private readonly channel: MachineCommandChannel) {}

  createThread(input: { machineId: string; title: string }) {
    return this.channel.createThread(input);
  }

  startTurn(input: { threadId: string; prompt: string }) {
    return this.channel.startTurn(input);
  }
}
```

```ts
// apps/gateway/src/ws/client-session-hub.ts
import { RegistryStore } from "../store/registry-store";

export class ClientSessionHub {
  constructor(private readonly registryStore: RegistryStore) {}

  register(machine: { machineId: string; hostname: string; agentKind: "codex" }) {
    this.registryStore.upsertMachine({ ...machine, status: "online" });
  }
}
```

```ts
// apps/gateway/src/ws/console-session-hub.ts
export class ConsoleSessionHub {
  broadcastThreadDelta(input: { threadId: string; delta: string }) {
    return input;
  }
}
```

```ts
// apps/gateway/src/http/routes/threads-routes.ts
import { FastifyInstance } from "fastify";
import { ThreadStore } from "../../store/thread-store";
import { CommandBroker } from "../../services/command-broker";

export function registerThreadRoutes(
  app: FastifyInstance,
  store: ThreadStore,
  broker: CommandBroker,
): void {
  app.get("/threads", async () => ({ items: store.listThreads() }));

  app.post("/threads", async (request) => {
    const body = request.body as { machineId: string; title: string };
    const thread = await broker.createThread(body);
    store.upsertThread({ ...thread, machineId: body.machineId });
    return thread;
  });

  app.post("/threads/:threadId/turns", async (request) => {
    const body = request.body as { prompt: string };
    const { threadId } = request.params as { threadId: string };
    return { deltas: await broker.startTurn({ threadId, prompt: body.prompt }) };
  });
}
```

```ts
// apps/client-codex/src/gateway/command-dispatcher.ts
import { CodexAdapter } from "../codex-adapter/adapter";

export class CommandDispatcher {
  constructor(private readonly adapter: CodexAdapter) {}

  async createThread(input: { machineId: string; title: string }) {
    return this.adapter.createThread({ title: input.title });
  }

  async startTurn(input: { threadId: string; prompt: string }): Promise<string[]> {
    const deltas: string[] = [];
    for await (const delta of this.adapter.startTurn(input)) {
      deltas.push(delta);
    }
    return deltas;
  }
}
```

```ts
// tests/integration/thread-lifecycle-harness.ts
import { CommandBroker } from "../../apps/gateway/src/services/command-broker";
import { CommandDispatcher } from "../../apps/client-codex/src/gateway/command-dispatcher";
import { FakeCodexAdapter } from "../../apps/client-codex/src/codex-adapter/fake-codex-adapter";

export class InMemoryHarness {
  static async start() {
    const adapter = new FakeCodexAdapter();
    const dispatcher = new CommandDispatcher(adapter);
    const broker = new CommandBroker({
      createThread: (input) => dispatcher.createThread(input),
      startTurn: (input) => dispatcher.startTurn(input),
    });

    return new InMemoryHarness(broker);
  }

  constructor(private readonly broker: CommandBroker) {}

  createThread(input: { machineId: string; title: string }) {
    return this.broker.createThread(input);
  }

  startTurn(input: { threadId: string; prompt: string }) {
    return this.broker.startTurn(input);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest tests/integration/thread-lifecycle.test.ts`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
git add apps/gateway apps/client-codex tests/integration/thread-lifecycle.test.ts
git commit -m "feat: implement thread lifecycle vertical slice"
```

## Task 6: Implement Skill, MCP, And Plugin Management Through Config Snapshots

**Files:**
- Create: `apps/client-codex/src/codex-config/config-store.ts`
- Modify: `apps/client-codex/src/codex-adapter/adapter.ts`
- Modify: `apps/client-codex/src/codex-adapter/fake-codex-adapter.ts`
- Modify: `apps/client-codex/src/gateway/command-dispatcher.ts`
- Modify: `apps/gateway/src/http/routes/environment-routes.ts`
- Modify: `apps/gateway/src/app.ts`
- Test: `tests/integration/environment-management.test.ts`
- Create: `tests/integration/environment-management-harness.ts`

- [ ] **Step 1: Write the failing test**

```ts
// tests/integration/environment-management.test.ts
import { describe, expect, it } from "vitest";
import { EnvironmentHarness } from "./environment-management-harness";

describe("environment management", () => {
  it("toggles a skill, upserts an MCP server, and installs a plugin", async () => {
    const harness = await EnvironmentHarness.start();

    await harness.toggleSkill({ resourceId: "skill:reviewer", enabled: false });
    await harness.upsertMcp({ resourceId: "mcp:github", command: "npx", args: ["-y", "@modelcontextprotocol/server-github"] });
    await harness.installPlugin({ resourceId: "plugin:figma", displayName: "Figma Plugin" });

    const snapshot = await harness.getEnvironment();

    expect(snapshot.skills.find((item) => item.resourceId === "skill:reviewer")?.status).toBe("disabled");
    expect(snapshot.mcps.find((item) => item.resourceId === "mcp:github")?.status).toBe("enabled");
    expect(snapshot.plugins.find((item) => item.resourceId === "plugin:figma")?.status).toBe("enabled");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest tests/integration/environment-management.test.ts`
Expected: FAIL with missing environment commands or missing config store

- [ ] **Step 3: Write minimal implementation**

```ts
// apps/client-codex/src/codex-config/config-store.ts
export type SkillConfig = { resourceId: string; displayName: string; status: "enabled" | "disabled" };
export type McpConfig = { resourceId: string; displayName: string; command: string; args: string[]; status: "enabled" | "disabled" };
export type PluginConfig = { resourceId: string; displayName: string; status: "enabled" | "disabled" };

export class ConfigStore {
  skills: SkillConfig[] = [{ resourceId: "skill:reviewer", displayName: "Reviewer", status: "enabled" }];
  mcps: McpConfig[] = [];
  plugins: PluginConfig[] = [];
}
```

```ts
// apps/client-codex/src/codex-adapter/adapter.ts
export interface CodexAdapter {
  createThread(input: { title: string }): Promise<{ threadId: string; title: string; status: "ready" }>;
  startTurn(input: { threadId: string; prompt: string }): AsyncIterable<string>;
  toggleSkill(input: { resourceId: string; enabled: boolean }): Promise<void>;
  upsertMcp(input: { resourceId: string; displayName: string; command: string; args: string[] }): Promise<void>;
  installPlugin(input: { resourceId: string; displayName: string }): Promise<void>;
  listThreads(): Promise<Array<{ threadId: string; title: string; status: string }>>;
  getEnvironmentResources(): Promise<
    Array<{ resourceId: string; kind: "skill" | "mcp" | "plugin"; displayName: string; status: string }>
  >;
}
```

```ts
// apps/client-codex/src/codex-adapter/fake-codex-adapter.ts
import { ConfigStore } from "../codex-config/config-store";
import { CodexAdapter } from "./adapter";

export class FakeCodexAdapter implements CodexAdapter {
  #config = new ConfigStore();
  #threads = new Map<string, { threadId: string; title: string; status: "ready" }>();

  async createThread(input: { title: string }) {
    const thread = { threadId: `thread-${this.#threads.size + 1}`, title: input.title, status: "ready" as const };
    this.#threads.set(thread.threadId, thread);
    return thread;
  }

  async *startTurn() {
    yield "Inspecting test output";
    yield "Found flaky retry timing";
  }

  async toggleSkill(input: { resourceId: string; enabled: boolean }) {
    const skill = this.#config.skills.find((item) => item.resourceId === input.resourceId);
    if (skill) skill.status = input.enabled ? "enabled" : "disabled";
  }

  async upsertMcp(input: { resourceId: string; displayName: string; command: string; args: string[] }) {
    this.#config.mcps = this.#config.mcps.filter((item) => item.resourceId !== input.resourceId);
    this.#config.mcps.push({ ...input, status: "enabled" });
  }

  async installPlugin(input: { resourceId: string; displayName: string }) {
    this.#config.plugins = this.#config.plugins.filter((item) => item.resourceId !== input.resourceId);
    this.#config.plugins.push({ ...input, status: "enabled" });
  }

  async listThreads() {
    return [...this.#threads.values()];
  }

  async getEnvironmentResources() {
    return [
      ...this.#config.skills.map((item) => ({ ...item, kind: "skill" as const })),
      ...this.#config.mcps.map((item) => ({ ...item, kind: "mcp" as const })),
      ...this.#config.plugins.map((item) => ({ ...item, kind: "plugin" as const })),
    ];
  }
}
```

```ts
// apps/gateway/src/http/routes/environment-routes.ts
import { FastifyInstance } from "fastify";
import { CommandBroker } from "../../services/command-broker";

export function registerEnvironmentRoutes(app: FastifyInstance, broker: CommandBroker): void {
  app.get("/skills", async () => ({ items: await broker.listEnvironment("skill") }));
  app.get("/mcps", async () => ({ items: await broker.listEnvironment("mcp") }));
  app.get("/plugins", async () => ({ items: await broker.listEnvironment("plugin") }));

  app.post("/skills/:resourceId/disable", async (request) => {
    const { resourceId } = request.params as { resourceId: string };
    await broker.toggleSkill({ resourceId, enabled: false });
    return { accepted: true };
  });

  app.post("/mcps", async (request) => {
    await broker.upsertMcp(request.body as { resourceId: string; displayName: string; command: string; args: string[] });
    return { accepted: true };
  });

  app.post("/plugins/:resourceId/install", async (request) => {
    const { resourceId } = request.params as { resourceId: string };
    const body = request.body as { displayName: string };
    await broker.installPlugin({ resourceId, displayName: body.displayName });
    return { accepted: true };
  });
}
```

```ts
// apps/gateway/src/app.ts
import Fastify from "fastify";
import { registerEnvironmentRoutes } from "./http/routes/environment-routes";
import { registerMachineRoutes } from "./http/routes/machines-routes";
import { registerThreadRoutes } from "./http/routes/threads-routes";
import { CommandBroker } from "./services/command-broker";
import { RegistryStore } from "./store/registry-store";
import { ThreadStore } from "./store/thread-store";

const defaultBroker = new CommandBroker({
  async createThread(input) {
    return { threadId: `dev-${input.title}`, title: input.title, status: "ready" as const };
  },
  async startTurn() {
    return [];
  },
  async listEnvironment() {
    return [];
  },
  async toggleSkill() {},
  async upsertMcp() {},
  async installPlugin() {},
});

export function buildGatewayApp(input: { broker?: CommandBroker } = {}) {
  const app = Fastify();
  const registryStore = new RegistryStore();
  const threadStore = new ThreadStore();
  const broker = input.broker ?? defaultBroker;

  app.get("/health", async () => ({ ok: true }));
  registerMachineRoutes(app, registryStore);
  registerThreadRoutes(app, threadStore, broker);
  registerEnvironmentRoutes(app, broker);

  return app;
}
```

```ts
// apps/gateway/src/services/command-broker.ts
type EnvironmentKind = "skill" | "mcp" | "plugin";

export interface MachineCommandChannel {
  createThread(input: { machineId: string; title: string }): Promise<{ threadId: string; title: string; status: "ready" }>;
  startTurn(input: { threadId: string; prompt: string }): Promise<string[]>;
  listEnvironment(kind: EnvironmentKind): Promise<Array<{ resourceId: string; kind: EnvironmentKind; displayName: string; status: string }>>;
  toggleSkill(input: { resourceId: string; enabled: boolean }): Promise<void>;
  upsertMcp(input: { resourceId: string; displayName: string; command: string; args: string[] }): Promise<void>;
  installPlugin(input: { resourceId: string; displayName: string }): Promise<void>;
}

export class CommandBroker {
  constructor(private readonly channel: MachineCommandChannel) {}

  createThread(input: { machineId: string; title: string }) {
    return this.channel.createThread(input);
  }

  startTurn(input: { threadId: string; prompt: string }) {
    return this.channel.startTurn(input);
  }

  listEnvironment(kind: EnvironmentKind) {
    return this.channel.listEnvironment(kind);
  }

  toggleSkill(input: { resourceId: string; enabled: boolean }) {
    return this.channel.toggleSkill(input);
  }

  upsertMcp(input: { resourceId: string; displayName: string; command: string; args: string[] }) {
    return this.channel.upsertMcp(input);
  }

  installPlugin(input: { resourceId: string; displayName: string }) {
    return this.channel.installPlugin(input);
  }
}
```

```ts
// apps/client-codex/src/gateway/command-dispatcher.ts
import { CodexAdapter } from "../codex-adapter/adapter";

export class CommandDispatcher {
  constructor(private readonly adapter: CodexAdapter) {}

  async createThread(input: { machineId: string; title: string }) {
    return this.adapter.createThread({ title: input.title });
  }

  async startTurn(input: { threadId: string; prompt: string }): Promise<string[]> {
    const deltas: string[] = [];
    for await (const delta of this.adapter.startTurn(input)) {
      deltas.push(delta);
    }
    return deltas;
  }

  async listEnvironment(kind: "skill" | "mcp" | "plugin") {
    const resources = await this.adapter.getEnvironmentResources();
    return resources.filter((item) => item.kind === kind);
  }

  toggleSkill(input: { resourceId: string; enabled: boolean }) {
    return this.adapter.toggleSkill(input);
  }

  upsertMcp(input: { resourceId: string; displayName: string; command: string; args: string[] }) {
    return this.adapter.upsertMcp(input);
  }

  installPlugin(input: { resourceId: string; displayName: string }) {
    return this.adapter.installPlugin(input);
  }
}
```

```ts
// tests/integration/environment-management-harness.ts
import { CommandBroker } from "../../apps/gateway/src/services/command-broker";
import { CommandDispatcher } from "../../apps/client-codex/src/gateway/command-dispatcher";
import { FakeCodexAdapter } from "../../apps/client-codex/src/codex-adapter/fake-codex-adapter";

export class EnvironmentHarness {
  static async start() {
    const adapter = new FakeCodexAdapter();
    const dispatcher = new CommandDispatcher(adapter);
    const broker = new CommandBroker({
      createThread: (input) => dispatcher.createThread(input),
      startTurn: (input) => dispatcher.startTurn(input),
      listEnvironment: (kind) => dispatcher.listEnvironment(kind),
      toggleSkill: (input) => dispatcher.toggleSkill(input),
      upsertMcp: (input) => dispatcher.upsertMcp(input),
      installPlugin: (input) => dispatcher.installPlugin(input),
    });

    return new EnvironmentHarness(broker);
  }

  constructor(private readonly broker: CommandBroker) {}

  toggleSkill(input: { resourceId: string; enabled: boolean }) {
    return this.broker.toggleSkill(input);
  }

  upsertMcp(input: { resourceId: string; displayName: string; command: string; args: string[] }) {
    return this.broker.upsertMcp(input);
  }

  installPlugin(input: { resourceId: string; displayName: string }) {
    return this.broker.installPlugin(input);
  }

  async getEnvironment() {
    return {
      skills: await this.broker.listEnvironment("skill"),
      mcps: await this.broker.listEnvironment("mcp"),
      plugins: await this.broker.listEnvironment("plugin"),
    };
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest tests/integration/environment-management.test.ts`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
git add apps/client-codex apps/gateway tests/integration/environment-management.test.ts
git commit -m "feat: add environment resource management"
```

## Task 7: Build The Responsive H5 Shell, Overview, And Machines Screens

**Files:**
- Create: `apps/console/package.json`
- Create: `apps/console/tsconfig.json`
- Create: `apps/console/vite.config.ts`
- Create: `apps/console/src/main.tsx`
- Create: `apps/console/src/app/router.tsx`
- Create: `apps/console/src/app/shell.tsx`
- Create: `apps/console/src/app/http-client.ts`
- Create: `apps/console/src/app/ws-client.ts`
- Create: `apps/console/src/app/store.ts`
- Create: `apps/console/src/features/overview/overview-page.tsx`
- Create: `apps/console/src/features/machines/machines-page.tsx`
- Create: `apps/console/src/styles.css`
- Test: `apps/console/src/app/shell.test.tsx`

- [ ] **Step 1: Write the failing test**

```tsx
// apps/console/src/app/shell.test.tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AppShell } from "./shell";

describe("AppShell", () => {
  it("renders the hybrid workbench layout", () => {
    render(<AppShell />);

    expect(screen.getByText("概览")).toBeInTheDocument();
    expect(screen.getByText("机器")).toBeInTheDocument();
    expect(screen.getByText("中间工作区")).toBeInTheDocument();
    expect(screen.getByText("检查器")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest apps/console/src/app/shell.test.tsx`
Expected: FAIL with `Cannot find module './shell'`

- [ ] **Step 3: Write minimal implementation**

```json
// apps/console/package.json
{
  "name": "@cag/console",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "react-router-dom": "^7.0.0",
    "zustand": "^5.0.0"
  },
  "devDependencies": {
    "@testing-library/react": "^16.0.0",
    "@vitejs/plugin-react": "^5.0.0",
    "vite": "^7.0.0"
  },
  "scripts": {
    "dev": "vite",
    "build": "tsc -p tsconfig.json && vite build",
    "test": "vitest run src/app/shell.test.tsx"
  }
}
```

```json
// apps/console/tsconfig.json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "jsx": "react-jsx",
    "outDir": "dist"
  },
  "include": ["src"]
}
```

```ts
// apps/console/vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 4173,
    host: "127.0.0.1",
  },
});
```

```tsx
// apps/console/src/main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import { router } from "./app/router";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>,
);
```

```ts
// apps/console/src/app/http-client.ts
export async function getJson<T>(path: string): Promise<T> {
  const response = await fetch(path);
  return response.json() as Promise<T>;
}
```

```ts
// apps/console/src/app/ws-client.ts
export function connectGatewaySocket(url: string): WebSocket {
  return new WebSocket(url);
}
```

```tsx
// apps/console/src/app/shell.tsx
import { NavLink, Outlet } from "react-router-dom";

export function AppShell() {
  return (
    <div className="shell">
      <aside className="left-rail">
        <NavLink to="/">概览</NavLink>
        <NavLink to="/machines">机器</NavLink>
        <NavLink to="/threads">线程</NavLink>
        <NavLink to="/environment">环境</NavLink>
      </aside>
      <main className="center-pane">
        <header>中间工作区</header>
        <Outlet />
      </main>
      <aside className="right-rail">检查器</aside>
    </div>
  );
}
```

```tsx
// apps/console/src/features/overview/overview-page.tsx
export function OverviewPage() {
  return <section>概览内容</section>;
}
```

```tsx
// apps/console/src/features/machines/machines-page.tsx
export function MachinesPage() {
  return <section>机器列表</section>;
}
```

```tsx
// apps/console/src/app/router.tsx
import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "./shell";
import { OverviewPage } from "../features/overview/overview-page";
import { MachinesPage } from "../features/machines/machines-page";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: "machines", element: <MachinesPage /> },
    ],
  },
]);
```

```css
/* apps/console/src/styles.css */
.shell {
  display: grid;
  grid-template-columns: 240px 1fr 320px;
  min-height: 100vh;
}

.left-rail,
.right-rail {
  padding: 16px;
  border-right: 1px solid #d4d0c8;
}

.center-pane {
  padding: 20px;
}

@media (max-width: 840px) {
  .shell {
    grid-template-columns: 1fr;
  }

  .left-rail,
  .right-rail {
    display: none;
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest apps/console/src/app/shell.test.tsx`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
git add apps/console
git commit -m "feat: add responsive console shell and overview"
```

## Task 8: Finish Thread Workspace, Environment Screens, And End-To-End Smoke Tests

**Files:**
- Create: `apps/console/src/features/threads/threads-page.tsx`
- Create: `apps/console/src/features/threads/thread-workspace-page.tsx`
- Create: `apps/console/src/features/environment/environment-page.tsx`
- Modify: `apps/console/src/app/router.tsx`
- Modify: `apps/console/src/app/http-client.ts`
- Modify: `apps/console/src/app/ws-client.ts`
- Modify: `apps/console/src/app/store.ts`
- Test: `tests/e2e/console-smoke.spec.ts`
- Create: `playwright.config.ts`

- [ ] **Step 1: Write the failing test**

```ts
// tests/e2e/console-smoke.spec.ts
import { expect, test } from "@playwright/test";

test("console can navigate from thread list to workspace and environment", async ({ page }) => {
  await page.goto("/");

  await page.getByText("线程").click();
  await expect(page.getByText("线程列表")).toBeVisible();

  await page.getByText("打开 thread-1").click();
  await expect(page.getByText("实时对话")).toBeVisible();

  await page.getByText("环境").click();
  await expect(page.getByRole("tab", { name: "Plugins" })).toBeVisible();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm playwright test tests/e2e/console-smoke.spec.ts`
Expected: FAIL because thread pages and environment tabs do not exist yet

- [ ] **Step 3: Write minimal implementation**

```tsx
// apps/console/src/features/threads/threads-page.tsx
import { Link } from "react-router-dom";

export function ThreadsPage() {
  return (
    <section>
      <h1>线程列表</h1>
      <Link to="/threads/thread-1">打开 thread-1</Link>
    </section>
  );
}
```

```tsx
// apps/console/src/features/threads/thread-workspace-page.tsx
export function ThreadWorkspacePage() {
  return (
    <section>
      <h1>实时对话</h1>
      <div>当前上下文</div>
      <div>消息流</div>
      <form>
        <textarea aria-label="prompt" />
        <button type="submit">发送</button>
        <button type="button">中断</button>
      </form>
    </section>
  );
}
```

```tsx
// apps/console/src/features/environment/environment-page.tsx
export function EnvironmentPage() {
  return (
    <section>
      <h1>环境</h1>
      <div role="tablist" aria-label="Environment tabs">
        <button role="tab">Skills</button>
        <button role="tab">MCP</button>
        <button role="tab">Plugins</button>
      </div>
    </section>
  );
}
```

```tsx
// apps/console/src/app/router.tsx
import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "./shell";
import { EnvironmentPage } from "../features/environment/environment-page";
import { MachinesPage } from "../features/machines/machines-page";
import { OverviewPage } from "../features/overview/overview-page";
import { ThreadWorkspacePage } from "../features/threads/thread-workspace-page";
import { ThreadsPage } from "../features/threads/threads-page";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: "machines", element: <MachinesPage /> },
      { path: "threads", element: <ThreadsPage /> },
      { path: "threads/:threadId", element: <ThreadWorkspacePage /> },
      { path: "environment", element: <EnvironmentPage /> },
    ],
  },
]);
```

```ts
// playwright.config.ts
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  use: {
    baseURL: "http://127.0.0.1:4173",
    headless: true,
  },
  webServer: {
    command: "pnpm --filter @cag/console dev --host 127.0.0.1 --port 4173",
    port: 4173,
    reuseExistingServer: true,
  },
});
```

```ts
// apps/console/src/app/store.ts
import { create } from "zustand";

type ConsoleState = {
  activeThreadId?: string;
  setActiveThread(threadId: string): void;
};

export const useConsoleStore = create<ConsoleState>((set) => ({
  activeThreadId: undefined,
  setActiveThread: (threadId) => set({ activeThreadId: threadId }),
}));
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm playwright test tests/e2e/console-smoke.spec.ts`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
git add apps/console tests/e2e/console-smoke.spec.ts
git commit -m "feat: add thread workspace and environment screens"
```

## Self-Review

### Spec coverage

- Gateway + registry: covered by Tasks 3, 5, and 6.
- Machine client and Codex-first adapter: covered by Tasks 4, 5, and 6.
- Thread lifecycle and realtime interaction: covered by Tasks 5 and 8.
- Skill/MCP/plugin management: covered by Tasks 6 and 8.
- Responsive H5 for mobile and web: covered by Tasks 7 and 8.
- Unified interface layer for future agents: covered by Tasks 2, 3, and 4.

### Placeholder scan

- No placeholder markers remain.
- Every task names exact files, code entry points, and verification commands.

### Type consistency

- `machineId`, `threadId`, `commandId`, and resource `kind` values stay consistent across domain, protocol, gateway, client, and console tasks.
- `command.startThread`, `command.startTurn`, and environment management command names match the spec and later UI tasks.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-07-code-agent-gateway-v1.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
