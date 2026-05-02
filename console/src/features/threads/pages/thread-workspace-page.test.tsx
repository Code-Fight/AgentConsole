import "@testing-library/jest-dom/vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { AppProviders } from "../../../app/providers/index";
import { createAppRouter } from "../../../app/router/index";
import { resetCapabilitiesForTests } from "../../../common/config/capabilities";
import { clearGatewayConnectionCookies } from "../../../common/config/gateway-connection-store";

const GATEWAY_URL = "http://localhost:18080";
const GATEWAY_API_KEY = "feature-test-key";

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  private readonly listeners = new Map<string, Set<(event: MessageEvent<string>) => void>>();
  readonly close = vi.fn();

  constructor(_url: string) {
    FakeWebSocket.instances.push(this);
  }

  addEventListener(type: string, listener: (event: MessageEvent<string>) => void) {
    const listeners = this.listeners.get(type) ?? new Set();
    listeners.add(listener);
    this.listeners.set(type, listeners);
  }

  removeEventListener(type: string, listener: (event: MessageEvent<string>) => void) {
    this.listeners.get(type)?.delete(listener);
  }

  emitMessage(data: string) {
    const event = { data } as MessageEvent<string>;
    for (const listener of this.listeners.get("message") ?? []) {
      listener(event);
    }
  }
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function getPath(input: RequestInfo | URL): string {
  const raw = String(input);
  return new URL(raw).pathname;
}

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, resolve, reject };
}

beforeEach(() => {
  resetCapabilitiesForTests();
  document.cookie = `cag_gateway_url=${encodeURIComponent(GATEWAY_URL)}; Path=/`;
  document.cookie = `cag_gateway_api_key=${encodeURIComponent(GATEWAY_API_KEY)}; Path=/`;
});

afterEach(() => {
  FakeWebSocket.instances = [];
  clearGatewayConnectionCookies();
  resetCapabilitiesForTests();
  vi.unstubAllGlobals();
});

test("renders /threads/thread-1, streams gateway output, and posts turns", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path === "/capabilities") {
      return jsonResponse({
        threadHub: true,
        threadWorkspace: true,
        approvals: true,
        startTurn: true,
        steerTurn: true,
        interruptTurn: true,
        machineInstallAgent: false,
        machineRemoveAgent: false,
        environmentSyncCatalog: false,
        environmentRestartBridge: false,
        environmentOpenMarketplace: false,
        environmentMutateResources: false,
        environmentWriteMcp: false,
        environmentWriteSkills: false,
        settingsEditGatewayEndpoint: false,
        settingsEditConsoleProfile: false,
        settingsEditSafetyPolicy: false,
        settingsGlobalDefault: true,
        settingsMachineOverride: true,
        settingsApplyMachine: true,
        dashboardMetrics: false,
        agentLifecycle: false,
        agentTimelineEvents: false,
      });
    }

    if (path === "/settings/console") {
      return jsonResponse({ preferences: null });
    }

    if (path === "/threads") {
      return jsonResponse({
        items: [
          {
            threadId: "thread-1",
            machineId: "machine-1",
            agentId: "agent-1",
            status: "idle",
            title: "Gateway Thread 1",
          },
        ],
      });
    }

    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-1",
            name: "Machine One",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-1",
                agentType: "codex",
                displayName: "Primary Codex",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path === "/threads/thread-1") {
      return jsonResponse({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          agentId: "agent-1",
          status: "idle",
          title: "Gateway Thread 1",
        },
        pendingApprovals: [],
      });
    }

    if (path === "/machines/machine-1") {
      return jsonResponse({
        machine: {
          id: "machine-1",
          name: "Machine One",
          status: "online",
          runtimeStatus: "running",
        },
      });
    }

    if (path === "/threads/thread-1/turns") {
      return jsonResponse(
        {
          turn: {
            turnId: "turn-1",
            threadId: "thread-1",
          },
        },
        202,
      );
    }

    throw new Error(`unexpected fetch path: ${path} (${init?.method ?? "GET"})`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const router = createAppRouter({ initialEntries: ["/threads/thread-1"] });
  render(<AppProviders router={router} />);

  expect((await screen.findAllByText("Gateway Thread 1", { exact: true })).length).toBeGreaterThan(0);

  await waitFor(() => {
    expect(FakeWebSocket.instances).toHaveLength(2);
  });

  await act(async () => {
    FakeWebSocket.instances[1]?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.delta",
        timestamp: "2026-04-19T10:00:00Z",
        payload: {
          threadId: "thread-1",
          turnId: "turn-1",
          sequence: 1,
          delta: "hello from gateway",
        },
      }),
    );
  });

  expect((await screen.findAllByText("hello from gateway", { exact: true })).length).toBeGreaterThan(0);

  await act(async () => {
    FakeWebSocket.instances[1]?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.completed",
        timestamp: "2026-04-19T10:00:01Z",
        payload: {
          turn: {
            threadId: "thread-1",
            turnId: "turn-1",
            status: "completed",
          },
        },
      }),
    );
  });

  const [promptInput] = await screen.findAllByLabelText("Prompt");
  fireEvent.change(promptInput, {
    target: { value: "run tests" },
  });
  const [sendButton] = screen.getAllByRole("button", { name: "Send prompt" });
  fireEvent.click(sendButton);

  await waitFor(() => {
    expect(
      fetchMock.mock.calls.some(
        ([input, init]) =>
          getPath(input as RequestInfo | URL) === "/threads/thread-1/turns" &&
          (init as RequestInit | undefined)?.method === "POST",
      ),
    ).toBe(true);
  });
});

test("shows execution state immediately, streams deltas, and hides turn control messages", async () => {
  const startTurn = createDeferred<Response>();
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path === "/capabilities") {
      return jsonResponse({
        threadHub: true,
        threadWorkspace: true,
        approvals: true,
        startTurn: true,
        steerTurn: true,
        interruptTurn: true,
        machineInstallAgent: false,
        machineRemoveAgent: false,
        environmentSyncCatalog: false,
        environmentRestartBridge: false,
        environmentOpenMarketplace: false,
        environmentMutateResources: false,
        environmentWriteMcp: false,
        environmentWriteSkills: false,
        settingsEditGatewayEndpoint: false,
        settingsEditConsoleProfile: false,
        settingsEditSafetyPolicy: false,
        settingsGlobalDefault: true,
        settingsMachineOverride: true,
        settingsApplyMachine: true,
        dashboardMetrics: false,
        agentLifecycle: false,
        agentTimelineEvents: false,
      });
    }

    if (path === "/settings/console") {
      return jsonResponse({ preferences: null });
    }

    if (path === "/threads") {
      return jsonResponse({
        items: [
          {
            threadId: "thread-1",
            machineId: "machine-1",
            agentId: "agent-1",
            status: "idle",
            title: "Gateway Thread 1",
          },
        ],
      });
    }

    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-1",
            name: "Machine One",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-1",
                agentType: "codex",
                displayName: "Primary Codex",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path === "/threads/thread-1") {
      return jsonResponse({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          agentId: "agent-1",
          status: "idle",
          title: "Gateway Thread 1",
        },
        pendingApprovals: [],
      });
    }

    if (path === "/machines/machine-1") {
      return jsonResponse({
        machine: {
          id: "machine-1",
          name: "Machine One",
          status: "online",
          runtimeStatus: "running",
        },
      });
    }

    if (path === "/threads/thread-1/turns") {
      return startTurn.promise;
    }

    throw new Error(`unexpected fetch path: ${path} (${init?.method ?? "GET"})`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const router = createAppRouter({ initialEntries: ["/threads/thread-1"] });
  render(<AppProviders router={router} />);

  expect((await screen.findAllByText("Gateway Thread 1", { exact: true })).length).toBeGreaterThan(0);

  await waitFor(() => {
    expect(FakeWebSocket.instances).toHaveLength(2);
  });

  const [promptInput] = await screen.findAllByLabelText("Prompt");
  fireEvent.change(promptInput, {
    target: { value: "run focused tests" },
  });
  const [sendButton] = screen.getAllByRole("button", { name: "Send prompt" });
  fireEvent.click(sendButton);

  expect(
    (await screen.findAllByText("run focused tests", { exact: true, selector: "p" })).length,
  ).toBeGreaterThan(0);
  expect(screen.getAllByText("正在执行...", { exact: true }).length).toBeGreaterThan(0);
  expect(promptInput).toHaveValue("");

  await act(async () => {
    FakeWebSocket.instances[1]?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.started",
        timestamp: "2026-04-19T10:00:00Z",
        payload: {
          threadId: "thread-1",
          turnId: "turn-1",
        },
      }),
    );
    FakeWebSocket.instances[1]?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.delta",
        timestamp: "2026-04-19T10:00:01Z",
        payload: {
          threadId: "thread-1",
          turnId: "turn-1",
          sequence: 1,
          delta: "hello",
        },
      }),
    );
    FakeWebSocket.instances[1]?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.delta",
        timestamp: "2026-04-19T10:00:02Z",
        payload: {
          threadId: "thread-1",
          turnId: "turn-1",
          sequence: 2,
          delta: " world",
        },
      }),
    );
  });

  expect((await screen.findAllByText("hello world", { exact: true })).length).toBeGreaterThan(0);
  expect(screen.queryByText(/Turn started:/)).not.toBeInTheDocument();

  await act(async () => {
    FakeWebSocket.instances[1]?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.completed",
        timestamp: "2026-04-19T10:00:03Z",
        payload: {
          turn: {
            threadId: "thread-1",
            turnId: "turn-1",
            status: "completed",
          },
        },
      }),
    );
  });

  expect(screen.queryByText("正在执行...", { exact: true })).not.toBeInTheDocument();
  expect(screen.queryByText(/Turn .* completed/)).not.toBeInTheDocument();

  await act(async () => {
    startTurn.resolve(
      jsonResponse(
        {
          turn: {
            turnId: "turn-1",
            threadId: "thread-1",
          },
        },
        202,
      ),
    );
  });
});

test("renders timeline history as one markdown answer with collapsible process and terminal output", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path === "/capabilities") {
      return jsonResponse({
        threadHub: true,
        threadWorkspace: true,
        approvals: true,
        startTurn: true,
        steerTurn: true,
        interruptTurn: true,
        machineInstallAgent: false,
        machineRemoveAgent: false,
        environmentSyncCatalog: false,
        environmentRestartBridge: false,
        environmentOpenMarketplace: false,
        environmentMutateResources: false,
        environmentWriteMcp: false,
        environmentWriteSkills: false,
        settingsEditGatewayEndpoint: false,
        settingsEditConsoleProfile: false,
        settingsEditSafetyPolicy: false,
        settingsGlobalDefault: true,
        settingsMachineOverride: true,
        settingsApplyMachine: true,
        dashboardMetrics: false,
        agentLifecycle: false,
        agentTimelineEvents: false,
      });
    }

    if (path === "/settings/console") {
      return jsonResponse({ preferences: null });
    }

    if (path === "/threads") {
      return jsonResponse({
        items: [
          {
            threadId: "thread-1",
            machineId: "machine-1",
            agentId: "agent-1",
            status: "idle",
            title: "Timeline Thread",
          },
        ],
      });
    }

    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-1",
            name: "Machine One",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-1",
                agentType: "codex",
                displayName: "Primary Codex",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path === "/threads/thread-1") {
      return jsonResponse({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          agentId: "agent-1",
          status: "idle",
          title: "Timeline Thread",
        },
        pendingApprovals: [],
        events: [
          {
            schemaVersion: "agent-timeline.v1",
            eventId: "event-1",
            sequence: 1,
            threadId: "thread-1",
            turnId: "turn-1",
            itemId: "reasoning-1",
            eventType: "item.delta",
            itemType: "reasoning",
            phase: "analysis",
            content: { contentType: "markdown", delta: "我先分析范围", appendMode: "append" },
          },
          {
            schemaVersion: "agent-timeline.v1",
            eventId: "event-2",
            sequence: 2,
            threadId: "thread-1",
            turnId: "turn-1",
            itemId: "command-1",
            eventType: "item.delta",
            itemType: "command",
            phase: "progress",
            content: { contentType: "terminal", delta: "PASS\n", appendMode: "append" },
          },
          {
            schemaVersion: "agent-timeline.v1",
            eventId: "event-3",
            sequence: 3,
            threadId: "thread-1",
            turnId: "turn-1",
            itemId: "message-1",
            eventType: "item.delta",
            itemType: "message",
            role: "assistant",
            phase: "final",
            content: {
              contentType: "markdown",
              delta: "**总结报告**\n\n| Agent | 状态 |\n| --- | --- |\n| Codex | 可用 |",
              appendMode: "append",
            },
          },
          {
            schemaVersion: "agent-timeline.v1",
            eventId: "event-4",
            sequence: 4,
            threadId: "thread-1",
            turnId: "turn-1",
            eventType: "turn.completed",
            status: "completed",
          },
        ],
      });
    }

    if (path === "/machines/machine-1") {
      return jsonResponse({
        machine: {
          id: "machine-1",
          name: "Machine One",
          status: "online",
          runtimeStatus: "running",
        },
      });
    }

    throw new Error(`unexpected fetch path: ${path} (${init?.method ?? "GET"})`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const router = createAppRouter({ initialEntries: ["/threads/thread-1"] });
  render(<AppProviders router={router} />);

  expect((await screen.findAllByText("总结报告")).length).toBeGreaterThan(0);
  expect(screen.getAllByRole("table").length).toBeGreaterThan(0);
  expect(screen.getAllByTestId("agent-message-bubble").length).toBeGreaterThan(0);
  expect(screen.getAllByText("已处理").length).toBeGreaterThan(0);
  expect(screen.queryByText("我先分析范围")).not.toBeInTheDocument();
  expect(screen.getAllByText("终端输出").length).toBeGreaterThan(0);
  expect(screen.getAllByText("PASS").length).toBeGreaterThan(0);

  fireEvent.click(screen.getAllByText("已处理")[0]);

  expect(screen.getAllByText("我先分析范围").length).toBeGreaterThan(0);
});

test("uses selected session title when thread detail title is empty", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path === "/capabilities") {
      return jsonResponse({
        threadHub: true,
        threadWorkspace: true,
        approvals: true,
        startTurn: true,
        steerTurn: true,
        interruptTurn: true,
        machineInstallAgent: false,
        machineRemoveAgent: false,
        environmentSyncCatalog: false,
        environmentRestartBridge: false,
        environmentOpenMarketplace: false,
        environmentMutateResources: false,
        environmentWriteMcp: false,
        environmentWriteSkills: false,
        settingsEditGatewayEndpoint: false,
        settingsEditConsoleProfile: false,
        settingsEditSafetyPolicy: false,
        settingsGlobalDefault: true,
        settingsMachineOverride: true,
        settingsApplyMachine: true,
        dashboardMetrics: false,
        agentLifecycle: false,
        agentTimelineEvents: false,
      });
    }

    if (path === "/settings/console") {
      return jsonResponse({ preferences: null });
    }

    if (path === "/threads") {
      return jsonResponse({
        items: [
          {
            threadId: "thread-1",
            machineId: "machine-1",
            agentId: "agent-1",
            status: "idle",
            title: "服务测试",
          },
        ],
      });
    }

    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-1",
            name: "Machine One",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-1",
                agentType: "codex",
                displayName: "Primary Codex",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path === "/threads/thread-1") {
      return jsonResponse({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          agentId: "agent-1",
          status: "idle",
          title: "",
        },
        pendingApprovals: [],
      });
    }

    if (path === "/machines/machine-1") {
      return jsonResponse({
        machine: {
          id: "machine-1",
          name: "Machine One",
          status: "online",
          runtimeStatus: "running",
        },
      });
    }

    throw new Error(`unexpected fetch path: ${path} (${init?.method ?? "GET"})`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const router = createAppRouter({ initialEntries: ["/threads/thread-1"] });
  render(<AppProviders router={router} />);

  expect((await screen.findAllByText("服务测试", { exact: true })).length).toBeGreaterThan(0);
  expect(screen.queryAllByText("线程工作区", { exact: true })).toHaveLength(0);
});
