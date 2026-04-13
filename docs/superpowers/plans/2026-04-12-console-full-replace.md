# Console Full Replace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current `console/` application with the new design-source-driven React UI while wiring every existing Gateway capability into the new surfaces and marking unsupported actions as disabled or not connected.

**Architecture:** The replacement is split into three layers: a `design` layer that holds imported upstream UI code, a `gateway` adapter layer that converts Gateway HTTP/WebSocket data into page view-models, and an `app` layer that owns routing, composition, and capability policy wiring. The imported design code remains as untouched as possible so later design updates mostly replace files under `console/src/design/`, while Gateway integration remains isolated in adapters and hooks.

**Tech Stack:** React 19, TypeScript, Vite, React Router 7, Vitest, Playwright, Gateway HTTP/WebSocket APIs, imported design-source components with Radix/Lucide/utility dependencies.

---

### Task 1: Snapshot The Existing Console And Prepare The New Dependency Surface

**Files:**
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/package.json`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/app/shell.test.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/main.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/index.ts`

- [ ] **Step 1: Write the failing shell test for the replacement baseline**

```tsx
test("renders the design-driven thread hub shell", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ items: [] }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  })));
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/"]}>
      <AppShell />
    </MemoryRouter>,
  );

  expect(screen.getAllByText("Thread Hub").length).toBeGreaterThan(0);
  expect(screen.queryByText("上下文面板")).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails if the old shell is still active**

Run: `corepack pnpm test src/app/shell.test.tsx`
Expected: FAIL with the legacy shell still rendering old layout text such as `上下文面板`, or with the new `Thread Hub` expectation missing.

- [ ] **Step 3: Add the imported UI dependency set and a single design entrypoint**

```json
{
  "dependencies": {
    "@radix-ui/react-dialog": "^1.1.6",
    "@radix-ui/react-select": "^2.1.6",
    "@radix-ui/react-scroll-area": "^1.2.3",
    "class-variance-authority": "^0.7.1",
    "clsx": "^2.1.1",
    "lucide-react": "^0.487.0",
    "tailwind-merge": "^3.2.0"
  }
}
```

```ts
// /console/src/design/index.ts
export { DesignAppShell } from "./shell/design-app-shell";
export { ThreadHubPage } from "./pages/thread-hub-page";
export { ThreadWorkspacePageView } from "./pages/thread-workspace-page-view";
```

```tsx
// /console/src/main.tsx
import "./styles.css";
import "./design/styles/index.css";
```

- [ ] **Step 4: Run the targeted shell test again**

Run: `corepack pnpm test src/app/shell.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add console/package.json console/src/app/shell.test.tsx console/src/main.tsx console/src/design/index.ts
git commit -m "feat: prepare console for design-source replacement"
```

### Task 2: Import The Design Source Into An Isolated `design/` Layer

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/shell/design-app-shell.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/components/thread-hub-panel.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/components/session-chat-view.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/pages/thread-hub-page.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/pages/thread-workspace-page-view.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/pages/machines-page-view.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/pages/environment-page-view.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/pages/settings-page-view.tsx`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/design/styles/index.css`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/app/router.tsx`

- [ ] **Step 1: Write the failing route-level test for the design-first default route**

```tsx
test("defaults to the thread hub route and renders management links in the hub footer", async () => {
  render(<AppProviders router={appRouter} />);

  expect(await screen.findByRole("link", { name: "Machines" })).toHaveAttribute("href", "/machines");
  expect(screen.getByRole("link", { name: "Environment" })).toHaveAttribute("href", "/environment");
  expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/settings");
});
```

- [ ] **Step 2: Run the route test to verify it fails before the imported design pages are wired**

Run: `corepack pnpm test src/app/shell.test.tsx`
Expected: FAIL because the default route still points at the old page tree or the imported hub footer is missing.

- [ ] **Step 3: Copy the design-source shell, pages, and styles into `console/src/design/` with only minimal compatibility edits**

```tsx
// /console/src/design/shell/design-app-shell.tsx
export function DesignAppShell(props: {
  sidebar: React.ReactNode;
  main: React.ReactNode;
  mobileSidebar: React.ReactNode;
  title: string;
  subtitle: string;
  managementBackHref?: string;
}) {
  return (
    <div className="thread-shell">
      {/* imported design-source layout, with composition props replacing mock state */}
      {props.main}
    </div>
  );
}
```

```tsx
// /console/src/app/router.tsx
{
  path: "/",
  element: <AppShell />,
  children: [
    { index: true, element: <ThreadsPage /> },
    { path: "threads", element: <ThreadsPage /> },
    { path: "threads/:threadId", element: <ThreadWorkspacePage /> },
    { path: "machines", element: <MachinesPage /> },
    { path: "environment", element: <EnvironmentPage /> },
    { path: "settings", element: <SettingsPage /> }
  ]
}
```

- [ ] **Step 4: Run the route test again**

Run: `corepack pnpm test src/app/shell.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add console/src/design console/src/app/router.tsx
git commit -m "feat: import design-source console surfaces"
```

### Task 3: Build The Gateway Adapter Layer For Thread Hub And Workspace

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/capabilities.ts`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/thread-view-model.ts`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-thread-hub.ts`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-thread-workspace.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/threads-page.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/thread-workspace-page.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/thread-workspace-page.test.tsx`

- [ ] **Step 1: Write the failing workspace adapter test for live deltas, approvals, and turn actions**

```tsx
test("maps gateway thread events into the design workspace view-model", async () => {
  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("等待实时消息")).toBeInTheDocument();

  await act(async () => {
    FakeWebSocket.instances[0].emitMessage(JSON.stringify({
      version: "v1",
      category: "event",
      name: "turn.delta",
      timestamp: "2026-04-08T14:00:01Z",
      payload: {
        threadId: "thread-1",
        turnId: "turn-1",
        sequence: 1,
        delta: "hello from gateway"
      }
    }));
  });

  expect(await screen.findByText("hello from gateway")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the workspace test to verify it fails before the adapters are introduced**

Run: `corepack pnpm test src/pages/thread-workspace-page.test.tsx`
Expected: FAIL because the page still renders direct legacy structure instead of the imported design view-model.

- [ ] **Step 3: Introduce adapter hooks that isolate HTTP, websocket, and capability wiring**

```ts
// /console/src/gateway/capabilities.ts
export const consoleCapabilities = {
  threadHub: true,
  threadWorkspace: true,
  approvals: true,
  steerTurn: true,
  interruptTurn: true,
  dashboardMetrics: false,
  agentLifecycle: false,
} as const;
```

```ts
// /console/src/gateway/thread-view-model.ts
export function toWorkspaceMessage(delta: TurnDeltaPayload) {
  return {
    id: `${delta.turnId}:${delta.sequence}`,
    kind: "agent" as const,
    text: delta.delta,
    turnId: delta.turnId,
  };
}
```

```ts
// /console/src/gateway/use-thread-workspace.ts
export function useThreadWorkspace(threadId: string) {
  const [messages, setMessages] = useState<WorkspaceMessageViewModel[]>([]);
  const [pendingApprovals, setPendingApprovals] = useState<ApprovalCardViewModel[]>([]);

  useEffect(() => connectConsoleSocket(threadId, (event) => {
    const envelope = parseEnvelope(event.data);
    if (!envelope) return;
    // normalize turn.delta, turn.started, turn.completed, approval.required, approval.resolved
  }), [threadId]);

  return { messages, pendingApprovals };
}
```

- [ ] **Step 4: Run thread and workspace tests**

Run: `corepack pnpm test src/pages/threads-page.test.tsx src/pages/thread-workspace-page.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add console/src/gateway console/src/pages/threads-page.tsx console/src/pages/thread-workspace-page.tsx console/src/pages/thread-workspace-page.test.tsx
git commit -m "feat: connect thread hub and workspace to gateway adapters"
```

### Task 4: Replace Machines, Environment, And Settings With Design Views Plus Capability Policy

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-machines-page.ts`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-environment-page.ts`
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/use-settings-page.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/machines-page.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/environment-page.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/settings-page.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/environment-page.test.tsx`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/pages/settings-page.test.tsx`

- [ ] **Step 1: Write failing tests for disabled unsupported actions in management pages**

```tsx
test("shows unsupported agent lifecycle actions as disabled in the machines page", async () => {
  render(<MachinesPage />);

  expect(await screen.findByText("未接入")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Install agent" })).toBeDisabled();
});
```

```tsx
test("keeps supported environment mutations enabled while showing unsupported design actions as disabled", async () => {
  render(<EnvironmentPage />);

  expect(await screen.findByRole("button", { name: "Disable" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "Edit plugin marketplace" })).toBeDisabled();
});
```

- [ ] **Step 2: Run the management page tests to verify they fail**

Run: `corepack pnpm test src/pages/environment-page.test.tsx src/pages/settings-page.test.tsx`
Expected: FAIL because capability-policy-driven disabled states are not implemented yet.

- [ ] **Step 3: Convert management pages to view-model-driven design pages**

```ts
// /console/src/gateway/use-machines-page.ts
export function useMachinesPage() {
  return {
    machineCards,
    actions: {
      installAgent: { supported: false, reason: "Gateway capability gap" },
      deleteAgent: { supported: false, reason: "Gateway capability gap" },
    },
  };
}
```

```tsx
// /console/src/pages/machines-page.tsx
export function MachinesPage() {
  const vm = useMachinesPage();
  return <MachinesPageView {...vm} />;
}
```

```tsx
// /console/src/pages/environment-page.tsx
export function EnvironmentPage() {
  const vm = useEnvironmentPage();
  return <EnvironmentPageView {...vm} />;
}
```

- [ ] **Step 4: Run the full console unit test suite**

Run: `corepack pnpm test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add console/src/pages/machines-page.tsx console/src/pages/environment-page.tsx console/src/pages/settings-page.tsx console/src/gateway/use-machines-page.ts console/src/gateway/use-environment-page.ts console/src/gateway/use-settings-page.ts console/src/pages/environment-page.test.tsx console/src/pages/settings-page.test.tsx
git commit -m "feat: wire management pages through capability-aware adapters"
```

### Task 5: Document The Design-Driven Console And Remove Legacy Console Paths

**Files:**
- Create: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/README.md`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/tests/console-smoke.spec.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/tests/settings-e2e.spec.ts`
- Modify: `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/styles.css`

- [ ] **Step 1: Write the failing smoke test for the new default landing route**

```ts
test("opens the thread hub as the default console landing page", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText("Thread Hub")).toBeVisible();
  await expect(page.getByText("上下文面板")).toHaveCount(0);
});
```

- [ ] **Step 2: Run smoke and e2e tests to verify they fail with stale expectations**

Run: `corepack pnpm e2e`
Expected: FAIL because existing smoke/e2e assertions still point at the legacy console structure.

- [ ] **Step 3: Add the README section that explains the replacement architecture and future design update flow**

```md
# Console

## Design-Driven Architecture

- `src/design/`: imported upstream design-source pages, components, styles, and assets
- `src/gateway/`: Gateway adapters, hooks, capability policy, and view-model mapping
- `src/app/`: application shell, route composition, and page mounting

## Updating The Design Source

1. Export the latest design-source React code
2. Replace files under `src/design/`
3. Review capability policy changes
4. Update Gateway adapters only where data shapes changed
5. Run `corepack pnpm test && corepack pnpm build && corepack pnpm e2e`
```

- [ ] **Step 4: Run verification**

Run: `corepack pnpm test && corepack pnpm build && corepack pnpm e2e`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add console/README.md console/tests/console-smoke.spec.ts console/tests/settings-e2e.spec.ts console/src/styles.css
git commit -m "docs: record design-driven console maintenance workflow"
```
