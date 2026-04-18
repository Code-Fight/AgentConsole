# Console Feature-Oriented Frontend Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate `console/` from the current `design-source / design-host / gateway` hybrid runtime into a feature-oriented frontend structure under `app/`, `features/`, and `common/` without changing user-visible behavior or visual presentation.

**Architecture:** The migration keeps the current routes and Gateway contracts stable while progressively moving runtime ownership into four feature domains: `threads`, `settings`, `environment`, and `machines`. A new `app` shell owns entry, router, and connection gating, `common` holds truly cross-feature utilities, and a deterministic visual regression environment locks the current Console rendering as the baseline that every migration phase must preserve.

**Tech Stack:** React 19, TypeScript, Vite, React Router 7, Vitest, Playwright, Gateway HTTP/WebSocket APIs, Radix UI, Lucide React, shell scripts under `testing/environments/`

---

## File Map

### App Shell And Routing

- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/entry/main.tsx`
  Responsibility: isolate the React root bootstrap from feature-specific imports.
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/router/index.tsx`
  Responsibility: hold the canonical route tree and expose a test-friendly router factory.
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/providers/index.tsx`
  Responsibility: hold `RouterProvider` and future global providers in one place.
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/layout/app-shell.tsx`
  Responsibility: top-level layout wrapper and `Outlet`.
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/layout/connection-gate.tsx`
  Responsibility: global blocking dialog for missing/invalid Gateway connection outside settings routes.
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/main.tsx`
  Responsibility: delegate to the new app bootstrap.

### Common Cross-Feature Utilities

- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/common/api/http.ts`
  Responsibility: remain the canonical HTTP client after the migration.
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/common/api/ws.ts`
  Responsibility: remain the canonical WebSocket connector after the migration.
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/common/config/capabilities.ts`
  Responsibility: centralize capability fetch, storage, and helpers that multiple features consume.
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/common/ui/console.css`
  Responsibility: own the frozen current Console styles after the runtime stops importing from `design-source/styles/index.css`.

### Threads Feature

- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/api/thread-api.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/model/thread-view-model.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/model/thread-hub-context.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/hooks/use-thread-hub.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/hooks/use-thread-workspace.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/components/thread-panel.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/components/thread-item.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/components/session-chat.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/components/thread-shell.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/pages/threads-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/pages/thread-workspace-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/pages/threads-page.test.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/pages/thread-workspace-page.test.tsx`

### Settings Feature

- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/model/gateway-connection-store.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/hooks/use-console-preferences.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/hooks/use-settings-page.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/components/settings-screen.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/pages/settings-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/pages/settings-page.test.tsx`

### Environment Feature

- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/environment/hooks/use-environment-page.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/environment/components/environment-screen.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/environment/pages/environment-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/environment/pages/environment-page.test.tsx`

### Machines Feature

- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/machines/hooks/use-machines-page.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/machines/components/machines-screen.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/machines/pages/machines-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/machines/pages/machines-page.test.tsx`

### Visual Regression Environment

- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/package.json`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/playwright.visual.config.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/testing/playwright/console-visual-regression.helpers.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/testing/playwright/console-visual-regression.spec.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/testing/environments/console-visual-regression/README.md`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/testing/environments/console-visual-regression/run.sh`

### Legacy Cleanup

- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/design-source`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/design-host`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/design-bridge`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/gateway`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/design`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/pages`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/README.md`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/docs/superpowers/specs/2026-04-18-console-feature-oriented-frontend-architecture-design.md`

---

### Task 1: Freeze Behavioral Baselines And Introduce The Visual Regression Environment

**Files:**
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/package.json`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/shell.test.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/playwright.visual.config.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/testing/playwright/console-visual-regression.helpers.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/testing/playwright/console-visual-regression.spec.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/testing/environments/console-visual-regression/README.md`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/testing/environments/console-visual-regression/run.sh`

- [ ] **Step 1: Replace the old directory assertions with a behavior baseline test and add the failing visual spec scaffold**

```tsx
// /console/src/app/shell.test.tsx
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { clearGatewayConnectionCookies } from "../gateway/gateway-connection-store";
import { DesignSourceAppRoot } from "../design-host/app-root";

describe("console app shell baselines", () => {
  test("keeps settings reachable when gateway cookies are missing", async () => {
    clearGatewayConnectionCookies();
    window.history.pushState({}, "", "/settings");

    render(<DesignSourceAppRoot />);

    expect(await screen.findByLabelText("Gateway URL")).toBeInTheDocument();
    expect(screen.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
  });
});
```

```ts
// /testing/playwright/console-visual-regression.spec.ts
import { expect, test } from "../../console/playwright-test";
import { primeConsoleVisualRoutes, visualScenarios } from "./console-visual-regression.helpers";

for (const scenario of visualScenarios) {
  test(`visual baseline: ${scenario.name}`, async ({ page }) => {
    await primeConsoleVisualRoutes(page, scenario);
    await page.goto(scenario.path);
    await expect(page).toHaveScreenshot(`${scenario.snapshot}.png`, {
      fullPage: true,
      animations: "disabled",
    });
  });
}
```

- [ ] **Step 2: Run the targeted test to confirm the current Console already satisfies the baseline behavior, then run the visual suite to confirm the new harness does not exist yet**

Run: `cd console && corepack pnpm test -- src/app/shell.test.tsx`
Expected: PASS, proving the current Console behavior is now captured by a behavior assertion instead of a directory assertion.

Run: `cd console && corepack pnpm exec playwright test --config playwright.visual.config.ts`
Expected: FAIL because `playwright.visual.config.ts` and the visual helper files do not exist yet.

- [ ] **Step 3: Add deterministic visual-regression infrastructure and replace the old structure assertions with behavior assertions**

```json
// /console/package.json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc -p tsconfig.json && vite build",
    "test": "vitest run",
    "e2e": "playwright test --pass-with-no-tests",
    "e2e:visual": "playwright test --config playwright.visual.config.ts"
  }
}
```

```ts
// /console/playwright.visual.config.ts
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "../testing/playwright",
  testMatch: ["console-visual-regression.spec.ts"],
  outputDir: "./playwright-report/visual-results",
  use: {
    ...devices["Desktop Chrome"],
    baseURL: "http://127.0.0.1:4174",
    viewport: { width: 1440, height: 1024 },
    locale: "zh-CN",
    timezoneId: "Asia/Shanghai",
    colorScheme: "dark",
  },
  webServer: {
    command: "corepack pnpm --ignore-workspace dev --host 127.0.0.1 --port 4174",
    url: "http://127.0.0.1:4174",
    reuseExistingServer: true,
  },
});
```

```ts
// /testing/playwright/console-visual-regression.helpers.ts
import type { Page, Route } from "@playwright/test";

export interface VisualScenario {
  name: string;
  path: string;
  snapshot: string;
}

export const visualScenarios: VisualScenario[] = [
  { name: "thread hub", path: "/", snapshot: "thread-hub" },
  { name: "thread workspace", path: "/threads/thread-1", snapshot: "thread-workspace" },
  { name: "settings", path: "/settings", snapshot: "settings" },
  { name: "environment", path: "/environment", snapshot: "environment" },
  { name: "machines", path: "/machines", snapshot: "machines" },
];

export async function primeConsoleVisualRoutes(page: Page, scenario: VisualScenario) {
  await page.context().addCookies([
    { name: "cag_gateway_url", value: "http://127.0.0.1:4174", url: "http://127.0.0.1:4174" },
    { name: "cag_gateway_api_key", value: "visual-key", url: "http://127.0.0.1:4174" },
  ]);

  await page.route("**/*", async (route: Route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/capabilities") {
      await route.fulfill({ status: 200, body: JSON.stringify({
        threadHub: true, threadWorkspace: true, approvals: true, startTurn: true,
        steerTurn: true, interruptTurn: true, machineInstallAgent: true,
        machineRemoveAgent: true, environmentSyncCatalog: true,
        environmentRestartBridge: true, environmentOpenMarketplace: true,
        environmentMutateResources: true, environmentWriteMcp: true,
        environmentWriteSkills: true, settingsEditGatewayEndpoint: false,
        settingsEditConsoleProfile: false, settingsEditSafetyPolicy: false,
        settingsGlobalDefault: true, settingsMachineOverride: true,
        settingsApplyMachine: true, dashboardMetrics: false, agentLifecycle: true
      }) });
      return;
    }
    if (path === "/settings/console") {
      await route.fulfill({ status: 200, body: JSON.stringify({ preferences: null }) });
      return;
    }
    if (path === "/threads") {
      await route.fulfill({ status: 200, body: JSON.stringify({
        items: [
          {
            threadId: "thread-1",
            machineId: "machine-1",
            agentId: "agent-01",
            status: "idle",
            title: "Gateway Thread 1"
          }
        ]
      }) });
      return;
    }
    if (path === "/threads/thread-1") {
      await route.fulfill({ status: 200, body: JSON.stringify({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          agentId: "agent-01",
          status: "idle",
          title: "Gateway Thread 1"
        },
        activeTurnId: null,
        pendingApprovals: []
      }) });
      return;
    }
    if (path === "/machines") {
      await route.fulfill({ status: 200, body: JSON.stringify({
        items: [
          {
            id: "machine-1",
            name: "Machine One",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-01",
                agentType: "codex",
                displayName: "Primary Codex",
                status: "running"
              }
            ]
          }
        ]
      }) });
      return;
    }
    if (path === "/machines/machine-1") {
      await route.fulfill({ status: 200, body: JSON.stringify({
        machine: {
          id: "machine-1",
          name: "Machine One",
          status: "online",
          runtimeStatus: "running"
        }
      }) });
      return;
    }
    if (path === "/environment/skills") {
      await route.fulfill({ status: 200, body: JSON.stringify({
        items: [
          {
            resourceId: "skill-1",
            machineId: "machine-1",
            kind: "skill",
            displayName: "Debugger",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-18T10:00:00Z"
          }
        ]
      }) });
      return;
    }
    if (path === "/environment/mcps" || path === "/environment/plugins") {
      await route.fulfill({ status: 200, body: JSON.stringify({ items: [] }) });
      return;
    }
    await route.fallback();
  });
}
```

```bash
# /testing/environments/console-visual-regression/run.sh
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${ROOT}/console"
corepack pnpm e2e:visual "$@"
```

```md
<!-- /testing/environments/console-visual-regression/README.md -->
# Console Visual Regression

This environment runs deterministic Playwright screenshot checks for the current Console.

Use it after every feature migration:

```bash
./testing/environments/console-visual-regression/run.sh
```

If a migration intentionally changes visuals, regenerate baselines explicitly with:

```bash
cd console
corepack pnpm e2e:visual --update-snapshots
```
```

- [ ] **Step 4: Run the behavior test, create the first visual baselines from the current Console, and verify both pass**

Run: `cd console && corepack pnpm test -- src/app/shell.test.tsx`
Expected: PASS

Run: `cd console && corepack pnpm e2e:visual --update-snapshots`
Expected: PASS and baseline screenshots written for the current Console rendering.

Run: `./testing/environments/console-visual-regression/run.sh`
Expected: PASS with no diff against the newly recorded baseline.

- [ ] **Step 5: Commit**

```bash
git add console/package.json console/src/app/shell.test.tsx console/playwright.visual.config.ts \
  testing/playwright/console-visual-regression.helpers.ts \
  testing/playwright/console-visual-regression.spec.ts \
  testing/environments/console-visual-regression/README.md \
  testing/environments/console-visual-regression/run.sh
git commit -m "test: add console visual regression baselines"
```

### Task 2: Introduce The New `app/` And `common/` Skeleton Without Changing Page Behavior

**Files:**
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/entry/main.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/router/index.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/providers/index.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/layout/app-shell.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/layout/connection-gate.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/common/config/capabilities.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/common/ui/console.css`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/main.tsx`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/common/api/http.ts`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/common/api/ws.ts`

- [ ] **Step 1: Write the failing route-composition test for the new app shell**

```tsx
// /console/src/app/shell.test.tsx
test("routes root traffic through the new app shell while preserving the current thread hub surface", async () => {
  const router = createAppRouter({ initialEntries: ["/"] });
  render(<AppProviders router={router} />);

  expect(await screen.findByText("Agent Console")).toBeInTheDocument();
  expect(screen.getByText("Gateway Thread 1", { exact: true })).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the shell test to confirm the new `app` bootstrap is not wired yet**

Run: `cd console && corepack pnpm test -- src/app/shell.test.tsx`
Expected: FAIL because the new `createAppRouter` and `AppProviders` do not yet delegate the current routes into an `app` shell.

- [ ] **Step 3: Create the new app skeleton and point the Vite entry at it while keeping legacy pages mounted**

```tsx
// /console/src/app/providers/index.tsx
import type { ComponentProps } from "react";
import { RouterProvider } from "react-router-dom";

type AppProvidersProps = Pick<ComponentProps<typeof RouterProvider>, "router">;

export function AppProviders({ router }: AppProvidersProps) {
  return <RouterProvider router={router} />;
}
```

```tsx
// /console/src/app/layout/app-shell.tsx
import { Outlet } from "react-router-dom";
import { ConnectionGate } from "./connection-gate";

export function AppShell() {
  return (
    <div className="dark fixed inset-0 overflow-hidden relative">
      <Outlet />
      <ConnectionGate />
    </div>
  );
}
```

```tsx
// /console/src/app/layout/connection-gate.tsx
import { useSyncExternalStore } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  getGatewayConnectionState,
  subscribeGatewayConnection,
} from "../../gateway/gateway-connection-store";

export function ConnectionGate() {
  const location = useLocation();
  const navigate = useNavigate();
  const isSettingsRoute = location.pathname === "/settings" || location.pathname.startsWith("/settings/");
  const gatewayState = useSyncExternalStore(
    subscribeGatewayConnection,
    getGatewayConnectionState,
    getGatewayConnectionState,
  );

  if (gatewayState === "ready" || isSettingsRoute) {
    return null;
  }

  return (
    <div role="dialog" aria-modal="true" aria-label="Gateway 连接未配置">
      <p>请先在设置页填写 Gateway URL 与 API Key。</p>
      <button type="button" onClick={() => navigate("/settings")}>
        前往设置
      </button>
    </div>
  );
}
```

```ts
// /console/src/common/config/capabilities.ts
import { useEffect, useState } from "react";
import { http } from "../api/http";

export function useCapabilities(enabled = true) {
  const [snapshot, setSnapshot] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (!enabled) return;
    void http<Record<string, boolean>>("/capabilities").then(setSnapshot);
  }, [enabled]);

  return snapshot;
}
```

```tsx
// /console/src/app/router/index.tsx
import { createBrowserRouter, createMemoryRouter, type Router } from "react-router-dom";
import { AppShell } from "../layout/app-shell";
import { ThreadsPage } from "../../pages/threads-page";
import { ThreadWorkspacePage } from "../../pages/thread-workspace-page";
import { SettingsPage } from "../../pages/settings-page";
import { EnvironmentPage } from "../../pages/environment-page";
import { MachinesPage } from "../../pages/machines-page";

const routes = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <ThreadsPage /> },
      { path: "threads/:threadId", element: <ThreadWorkspacePage /> },
      { path: "machines", element: <MachinesPage /> },
      { path: "environment", element: <EnvironmentPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

export function createAppRouter(options?: { initialEntries?: string[] }): Router {
  if (options?.initialEntries) {
    return createMemoryRouter(routes, { initialEntries: options.initialEntries });
  }
  return createBrowserRouter(routes);
}
```

```css
/* /console/src/common/ui/console.css */
@import "../../design-source/styles/index.css";
```

```tsx
// /console/src/app/entry/main.tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { AppProviders } from "../providers";
import { createAppRouter } from "../router";

export function renderApp(rootElement: HTMLElement) {
  createRoot(rootElement).render(
    <StrictMode>
      <AppProviders router={createAppRouter()} />
    </StrictMode>,
  );
}
```

```tsx
// /console/src/main.tsx
import "./common/ui/console.css";
import { renderApp } from "./app/entry/main";

const rootElement = document.getElementById("root");
if (!rootElement) throw new Error("Root element not found");
renderApp(rootElement);
```

- [ ] **Step 4: Run the behavior suite, build, and the visual suite to verify the top-level skeleton changed but visuals did not**

Run: `cd console && corepack pnpm test -- src/app/shell.test.tsx`
Expected: PASS

Run: `cd console && corepack pnpm build`
Expected: PASS

Run: `./testing/environments/console-visual-regression/run.sh`
Expected: PASS with zero screenshot diffs.

- [ ] **Step 5: Commit**

```bash
git add console/src/app console/src/main.tsx console/src/common/api/http.ts console/src/common/api/ws.ts \
  console/src/common/config/capabilities.ts
git commit -m "refactor: introduce console app and common skeleton"
```

### Task 3: Migrate The Threads Runtime Into `features/threads`

**Files:**
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/api/thread-api.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/model/thread-view-model.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/model/thread-hub-context.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/hooks/use-thread-hub.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/hooks/use-thread-workspace.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/components/thread-item.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/components/thread-panel.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/components/session-chat.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/components/thread-shell.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/pages/threads-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/pages/thread-workspace-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/pages/threads-page.test.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/threads/pages/thread-workspace-page.test.tsx`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/router/index.tsx`

- [ ] **Step 1: Write the failing threads feature tests for hub rendering, workspace deltas, and prompt actions**

```tsx
// /console/src/features/threads/pages/thread-workspace-page.test.tsx
import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ThreadWorkspacePage } from "./thread-workspace-page";

test("renders gateway deltas and supports prompt submission from the feature-local workspace page", async () => {
  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("Gateway Thread 1")).toBeInTheDocument();
  expect(await screen.findByText("hello from gateway")).toBeInTheDocument();

  await waitFor(() => {
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/threads/thread-1/turns"),
      expect.objectContaining({ method: "POST" }),
    );
  });
});
```

```tsx
// /console/src/features/threads/pages/threads-page.test.tsx
test("keeps the current thread panel layout while using the feature-local hub hook", async () => {
  render(
    <MemoryRouter initialEntries={["/"]}>
      <Routes>
        <Route path="/" element={<ThreadsPage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("Gateway Thread 1", { exact: true })).toBeInTheDocument();
  expect(screen.getByText("Machine One")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the new feature-local tests and confirm they fail before the feature exists**

Run: `cd console && corepack pnpm test -- src/features/threads/pages/threads-page.test.tsx src/features/threads/pages/thread-workspace-page.test.tsx`
Expected: FAIL because `features/threads` files do not exist yet.

- [ ] **Step 3: Move thread data access, view-model mapping, and the current visual markup into `features/threads`**

```ts
// /console/src/features/threads/api/thread-api.ts
import { http, buildThreadApiPath } from "../../../common/api/http";
import type { ThreadDetailResponse, ThreadSummary, CreateThreadResponse } from "../../../common/api/types";

export function fetchThreads() {
  return http<{ items: ThreadSummary[] }>("/threads");
}

export function fetchThread(threadId: string) {
  return http<ThreadDetailResponse>(buildThreadApiPath(threadId));
}

export function createThread(machineId: string, agentId: string, title: string) {
  return http<CreateThreadResponse>("/threads", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ machineId, agentId, title }),
  });
}
```

```ts
// /console/src/features/threads/hooks/use-thread-workspace.ts
import { useEffect, useState } from "react";
import { connectConsoleSocket } from "../../../common/api/ws";
import { fetchThread } from "../api/thread-api";
import { toWorkspaceMessage, toApprovalCard } from "../model/thread-view-model";

export function useThreadWorkspace(threadId: string) {
  const [messages, setMessages] = useState([]);
  const [pendingApprovals, setPendingApprovals] = useState([]);

  useEffect(() => {
    if (!threadId) return;
    void fetchThread(threadId).then((response) => {
      setPendingApprovals(response.pendingApprovals.map(toApprovalCard));
    });

    return connectConsoleSocket(threadId, (event) => {
      const envelope = JSON.parse(event.data);
      if (envelope.name === "turn.delta") {
        setMessages((current) => [...current, toWorkspaceMessage(envelope.payload)]);
      }
    });
  }, [threadId]);

  return { messages, pendingApprovals };
}
```

```tsx
// /console/src/features/threads/pages/threads-page.tsx
import { ThreadShell } from "../components/thread-shell";
import { useThreadHub } from "../hooks/use-thread-hub";

export function ThreadsPage() {
  const vm = useThreadHub();
  return <ThreadShell hub={vm} />;
}
```

```tsx
// /console/src/app/router/index.tsx
import { ThreadsPage } from "../../features/threads/pages/threads-page";
import { ThreadWorkspacePage } from "../../features/threads/pages/thread-workspace-page";
```

- [ ] **Step 4: Run the threads feature tests, the existing route tests, and the full visual suite**

Run: `cd console && corepack pnpm test -- src/features/threads/pages/threads-page.test.tsx src/features/threads/pages/thread-workspace-page.test.tsx src/app/shell.test.tsx`
Expected: PASS

Run: `./testing/environments/console-visual-regression/run.sh`
Expected: PASS with the thread hub and workspace screenshots matching the baseline.

- [ ] **Step 5: Commit**

```bash
git add console/src/features/threads console/src/app/router/index.tsx
git commit -m "refactor: move console threads into feature module"
```

### Task 4: Migrate Gateway Connection And Settings Into `features/settings`

**Files:**
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/model/gateway-connection-store.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/hooks/use-console-preferences.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/hooks/use-settings-page.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/components/settings-screen.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/pages/settings-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/settings/pages/settings-page.test.tsx`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/layout/connection-gate.tsx`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/router/index.tsx`

- [ ] **Step 1: Write the failing settings feature tests for local connection persistence and remote config save**

```tsx
// /console/src/features/settings/pages/settings-page.test.tsx
import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { SettingsPage } from "./settings-page";

test("keeps settings reachable while unconfigured and saves gateway connection locally before remote settings fetches", async () => {
  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <Routes>
        <Route path="/settings" element={<SettingsPage />} />
      </Routes>
    </MemoryRouter>,
  );

  fireEvent.change(await screen.findByLabelText("Gateway URL"), {
    target: { value: "http://localhost:18080" },
  });
  fireEvent.change(screen.getByLabelText("Gateway API Key"), {
    target: { value: "test-key" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Save Gateway Connection" }));

  await waitFor(() => {
    expect(document.cookie).toContain("cag_gateway_url=http%3A%2F%2Flocalhost%3A18080");
    expect(document.cookie).toContain("cag_gateway_api_key=test-key");
  });
});
```

- [ ] **Step 2: Run the settings feature test to confirm the new feature does not exist yet**

Run: `cd console && corepack pnpm test -- src/features/settings/pages/settings-page.test.tsx`
Expected: FAIL because the settings feature files do not exist yet.

- [ ] **Step 3: Move the gateway connection store and settings screen into `features/settings` and update the app-level gate to consume them**

```ts
// /console/src/features/settings/model/gateway-connection-store.ts
export interface GatewayConnectionConfig {
  gatewayUrl: string;
  apiKey: string;
}

const GATEWAY_URL_COOKIE = "cag_gateway_url";
const GATEWAY_API_KEY_COOKIE = "cag_gateway_api_key";

export function saveGatewayConnectionToCookies(config: GatewayConnectionConfig) {
  document.cookie = `${GATEWAY_URL_COOKIE}=${encodeURIComponent(config.gatewayUrl)}; Path=/`;
  document.cookie = `${GATEWAY_API_KEY_COOKIE}=${encodeURIComponent(config.apiKey)}; Path=/`;
}

export function clearGatewayConnectionCookies() {
  document.cookie = `${GATEWAY_URL_COOKIE}=; Max-Age=0; Path=/`;
  document.cookie = `${GATEWAY_API_KEY_COOKIE}=; Max-Age=0; Path=/`;
}
```

```tsx
// /console/src/app/layout/connection-gate.tsx
import { useLocation, useNavigate } from "react-router-dom";
import { useGatewayConnectionState } from "../../features/settings/hooks/use-settings-page";

export function ConnectionGate() {
  const connection = useGatewayConnectionState();
  const location = useLocation();
  const navigate = useNavigate();
  const isSettingsRoute = location.pathname === "/settings" || location.pathname.startsWith("/settings/");
  if (connection.status === "ready" || isSettingsRoute) return null;

  return (
    <div role="dialog" aria-modal="true" aria-label="Gateway 连接未配置">
      <p>{connection.message}</p>
      <button type="button" onClick={() => navigate("/settings")}>前往设置</button>
    </div>
  );
}
```

```tsx
// /console/src/features/settings/pages/settings-page.tsx
import { SettingsScreen } from "../components/settings-screen";
import { useSettingsPage } from "../hooks/use-settings-page";

export function SettingsPage() {
  const vm = useSettingsPage();
  return <SettingsScreen {...vm} />;
}
```

- [ ] **Step 4: Run the settings tests, route tests, build, and visual suite**

Run: `cd console && corepack pnpm test -- src/features/settings/pages/settings-page.test.tsx src/app/shell.test.tsx`
Expected: PASS

Run: `cd console && corepack pnpm build`
Expected: PASS

Run: `./testing/environments/settings-e2e/run.sh`
Expected: PASS

Run: `./testing/environments/console-visual-regression/run.sh`
Expected: PASS with the settings screenshot matching the baseline.

- [ ] **Step 5: Commit**

```bash
git add console/src/features/settings console/src/app/layout/connection-gate.tsx console/src/app/router/index.tsx
git commit -m "refactor: move console settings into feature module"
```

### Task 5: Migrate Environment Resource Management Into `features/environment`

**Files:**
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/environment/hooks/use-environment-page.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/environment/components/environment-screen.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/environment/pages/environment-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/environment/pages/environment-page.test.tsx`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/router/index.tsx`

- [ ] **Step 1: Write the failing environment feature test for resource actions and form visibility**

```tsx
// /console/src/features/environment/pages/environment-page.test.tsx
import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { EnvironmentPage } from "./environment-page";

test("renders gateway-backed resources and keeps the add-skill form behavior unchanged", async () => {
  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <Routes>
        <Route path="/environment" element={<EnvironmentPage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("Debugger")).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Add skill" }));
  expect(screen.getByLabelText("Skill name")).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Sync catalog" }));
  await waitFor(() => {
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/environment/sync"),
      expect.objectContaining({ method: "POST" }),
    );
  });
});
```

- [ ] **Step 2: Run the new environment feature test and confirm it fails before the feature-local page exists**

Run: `cd console && corepack pnpm test -- src/features/environment/pages/environment-page.test.tsx`
Expected: FAIL because the feature files do not exist yet.

- [ ] **Step 3: Move the current environment hook and visual markup under `features/environment` without changing resource action paths**

```ts
// /console/src/features/environment/hooks/use-environment-page.ts
import { useState, useEffect } from "react";
import { http } from "../../../common/api/http";

export function useEnvironmentPage() {
  const [sections, setSections] = useState({ skills: [], mcps: [], plugins: [] });
  const [expandedResourceKey, setExpandedResourceKey] = useState<string | null>(null);

  useEffect(() => {
    void Promise.all([
      http("/environment/skills"),
      http("/environment/mcps"),
      http("/environment/plugins"),
    ]).then(([skills, mcps, plugins]) => setSections({
      skills: skills.items,
      mcps: mcps.items,
      plugins: plugins.items,
    }));
  }, []);

  return { sections, expandedResourceKey, setExpandedResourceKey };
}
```

```tsx
// /console/src/features/environment/pages/environment-page.tsx
import { EnvironmentScreen } from "../components/environment-screen";
import { useEnvironmentPage } from "../hooks/use-environment-page";

export function EnvironmentPage() {
  const vm = useEnvironmentPage();
  return <EnvironmentScreen {...vm} />;
}
```

```tsx
// /console/src/app/router/index.tsx
import { EnvironmentPage } from "../../features/environment/pages/environment-page";
```

- [ ] **Step 4: Run environment tests and the full visual suite**

Run: `cd console && corepack pnpm test -- src/features/environment/pages/environment-page.test.tsx`
Expected: PASS

Run: `./testing/environments/console-visual-regression/run.sh`
Expected: PASS with the environment screenshot matching the baseline, including dialog-open snapshots if the spec records them.

- [ ] **Step 5: Commit**

```bash
git add console/src/features/environment console/src/app/router/index.tsx
git commit -m "refactor: move console environment into feature module"
```

### Task 6: Migrate Machines And Agent Config Editing Into `features/machines`

**Files:**
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/machines/hooks/use-machines-page.ts`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/machines/components/machines-screen.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/machines/pages/machines-page.tsx`
- Create: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/features/machines/pages/machines-page.test.tsx`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/router/index.tsx`

- [ ] **Step 1: Write the failing machines feature test for agent config fetch and save**

```tsx
// /console/src/features/machines/pages/machines-page.test.tsx
import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { MachinesPage } from "./machines-page";

test("reads and saves per-agent config from the feature-local machines page", async () => {
  render(
    <MemoryRouter initialEntries={["/machines"]}>
      <Routes>
        <Route path="/machines" element={<MachinesPage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("Machine 01")).toBeInTheDocument();
  fireEvent.click(screen.getAllByTitle("编辑配置")[0]);

  await waitFor(() => {
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/machines/machine-01/agents/agent-01/config"),
      expect.anything(),
    );
  });

  fireEvent.click(screen.getByRole("button", { name: "保存" }));

  await waitFor(() => {
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/machines/machine-01/agents/agent-01/config"),
      expect.objectContaining({ method: "PUT" }),
    );
  });
});
```

- [ ] **Step 2: Run the machines feature test to confirm the feature does not exist yet**

Run: `cd console && corepack pnpm test -- src/features/machines/pages/machines-page.test.tsx`
Expected: FAIL because the machines feature files do not exist yet.

- [ ] **Step 3: Move machine list and agent config logic into `features/machines` and remove inline HTTP calls from presentation components**

```ts
// /console/src/features/machines/hooks/use-machines-page.ts
import { useEffect, useState } from "react";
import { http } from "../../../common/api/http";

export function useMachinesPage() {
  const [machines, setMachines] = useState([]);

  useEffect(() => {
    void http("/machines").then((response) => setMachines(response.items));
  }, []);

  async function saveAgentConfig(machineId: string, agentId: string, content: string) {
    await http(`/machines/${encodeURIComponent(machineId)}/agents/${encodeURIComponent(agentId)}/config`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ document: { format: "toml", content } }),
    });
  }

  return { machines, saveAgentConfig };
}
```

```tsx
// /console/src/features/machines/pages/machines-page.tsx
import { MachinesScreen } from "../components/machines-screen";
import { useMachinesPage } from "../hooks/use-machines-page";

export function MachinesPage() {
  const vm = useMachinesPage();
  return <MachinesScreen {...vm} />;
}
```

```tsx
// /console/src/app/router/index.tsx
import { MachinesPage } from "../../features/machines/pages/machines-page";
```

- [ ] **Step 4: Run the machines tests, build, and the visual suite**

Run: `cd console && corepack pnpm test -- src/features/machines/pages/machines-page.test.tsx`
Expected: PASS

Run: `cd console && corepack pnpm build`
Expected: PASS

Run: `./testing/environments/console-visual-regression/run.sh`
Expected: PASS with the machines screenshot matching the baseline.

- [ ] **Step 5: Commit**

```bash
git add console/src/features/machines console/src/app/router/index.tsx
git commit -m "refactor: move console machines into feature module"
```

### Task 7: Delete Legacy Layers And Make The Feature-Oriented Structure The Only Runtime

**Files:**
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/app/shell.test.tsx`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/design-source`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/design-host`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/design-bridge`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/gateway`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/design`
- Delete: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/src/pages`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/console/README.md`
- Modify: `/Users/zfcode/.codex/worktrees/c505/CodingAgentGateway/docs/superpowers/specs/2026-04-18-console-feature-oriented-frontend-architecture-design.md`

- [ ] **Step 1: Write the failing legacy-guard test that proves the runtime still imports old layers**

```tsx
// /console/src/app/shell.test.tsx
import { existsSync } from "node:fs";
import { join } from "node:path";

test("no runtime entry imports legacy console layers after the migration", () => {
  const srcRoot = join(process.cwd(), "src");
  expect(existsSync(join(srcRoot, "design-source"))).toBe(false);
  expect(existsSync(join(srcRoot, "design-host"))).toBe(false);
  expect(existsSync(join(srcRoot, "design-bridge"))).toBe(false);
  expect(existsSync(join(srcRoot, "gateway"))).toBe(false);
  expect(existsSync(join(srcRoot, "design"))).toBe(false);
  expect(existsSync(join(srcRoot, "pages"))).toBe(false);
});
```

- [ ] **Step 2: Run the legacy-guard test to confirm it fails before the old directories are deleted**

Run: `cd console && corepack pnpm test -- src/app/shell.test.tsx`
Expected: FAIL because the current runtime still references legacy layers.

- [ ] **Step 3: Delete the legacy directories, update docs, and remove all remaining imports that reference them**

```tsx
// /console/src/main.tsx
import "./common/ui/console.css";
import { renderApp } from "./app/entry/main";
```

```md
<!-- /console/README.md -->
## Architecture

The console now follows a feature-oriented frontend structure:

- `src/app/` for entry, router, providers, and layout
- `src/features/` for product domains
- `src/common/` for true cross-feature utilities

Legacy `design-source`, `design-host`, `design-bridge`, and `gateway` directories are no longer part of the runtime and must not be reintroduced.
```

```bash
cp console/src/design-source/styles/index.css console/src/common/ui/console.css
rm -rf console/src/design-source console/src/design-host console/src/design-bridge \
  console/src/gateway console/src/design console/src/pages
```

- [ ] **Step 4: Run the full regression matrix to prove the new structure is now the only runtime**

Run: `cd console && corepack pnpm test`
Expected: PASS

Run: `cd console && corepack pnpm build`
Expected: PASS

Run: `cd console && corepack pnpm e2e`
Expected: PASS

Run: `./testing/environments/settings-e2e/run.sh`
Expected: PASS

Run: `./testing/environments/dev-integration/run.sh up`
Expected: PASS, with the feature-oriented Console rendering successfully against the live dev-integration environment.

Run: `./testing/environments/console-visual-regression/run.sh`
Expected: PASS with zero screenshot diffs.

- [ ] **Step 5: Commit**

```bash
git add console/src console/README.md docs/superpowers/specs/2026-04-18-console-feature-oriented-frontend-architecture-design.md
git commit -m "refactor: finalize feature-oriented console architecture"
```
