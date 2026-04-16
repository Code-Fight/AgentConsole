import "@testing-library/jest-dom/vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { ConsoleHostRouter } from "../design-host/console-host-router";
import {
  clearGatewayConnectionCookies,
  saveGatewayConnectionToCookies,
} from "../gateway/gateway-connection-store";

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  private readonly listeners = new Map<string, Set<(event: MessageEvent<string>) => void>>();
  readonly close = vi.fn();
  readonly url: string;

  constructor(url: string) {
    this.url = url;
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

function getPath(input: RequestInfo | URL): string {
  const raw = String(input);
  if (raw.startsWith("http://") || raw.startsWith("https://")) {
    return new URL(raw).pathname;
  }
  return raw;
}

const capabilities = {
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
};

beforeEach(() => {
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "workspace-test-key",
  });
});

afterEach(() => {
  FakeWebSocket.instances = [];
  clearGatewayConnectionCookies();
  vi.unstubAllGlobals();
});

test("selecting a thread loads /threads/{threadId} and /machines/{machineId}", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = getPath(input);

    if (url === "/capabilities") {
      return new Response(JSON.stringify(capabilities), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (url === "/settings/console") {
      return new Response(JSON.stringify({ preferences: null }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (url === "/threads") {
      return new Response(
        JSON.stringify({
          items: [
            {
              threadId: "thread-1",
              machineId: "machine-1",
              status: "idle",
              title: "Investigate flaky test",
            },
          ],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/machines") {
      return new Response(
        JSON.stringify({
          items: [
            {
              id: "machine-1",
              name: "Primary Node",
              status: "online",
              runtimeStatus: "running",
            },
          ],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/threads/thread-1") {
      return new Response(
        JSON.stringify({
          thread: {
            threadId: "thread-1",
            machineId: "machine-1",
            status: "idle",
            title: "Investigate flaky test",
          },
          pendingApprovals: [],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/machines/machine-1") {
      return new Response(
        JSON.stringify({
          machine: {
            id: "machine-1",
            name: "Primary Node",
            status: "online",
            runtimeStatus: "running",
          },
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    throw new Error(`Unhandled request: ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  await screen.findByText("Investigate flaky test");
  expect(FakeWebSocket.instances).toHaveLength(1);

  fireEvent.click(await screen.findByText("Investigate flaky test"));

  await waitFor(() => {
    expect(
      fetchMock.mock.calls.some(([input]) => getPath(input as RequestInfo | URL) === "/threads/thread-1"),
    ).toBe(true);
    expect(
      fetchMock.mock.calls.some(([input]) => getPath(input as RequestInfo | URL) === "/machines/machine-1"),
    ).toBe(true);
  });

  await waitFor(() => {
    expect(FakeWebSocket.instances).toHaveLength(2);
  });
});

test("sending a prompt starts a real turn request", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = getPath(input);

    if (url === "/capabilities") {
      return new Response(JSON.stringify(capabilities), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (url === "/settings/console") {
      return new Response(JSON.stringify({ preferences: null }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (url === "/threads") {
      return new Response(
        JSON.stringify({
          items: [
            {
              threadId: "thread-1",
              machineId: "machine-1",
              status: "idle",
              title: "Investigate flaky test",
            },
          ],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/machines") {
      return new Response(
        JSON.stringify({
          items: [
            {
              id: "machine-1",
              name: "Primary Node",
              status: "online",
              runtimeStatus: "running",
            },
          ],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/threads/thread-1") {
      return new Response(
        JSON.stringify({
          thread: {
            threadId: "thread-1",
            machineId: "machine-1",
            status: "idle",
            title: "Investigate flaky test",
          },
          pendingApprovals: [],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/machines/machine-1") {
      return new Response(
        JSON.stringify({
          machine: {
            id: "machine-1",
            name: "Primary Node",
            status: "online",
            runtimeStatus: "running",
          },
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/threads/thread-1/turns") {
      return new Response(
        JSON.stringify({
          turn: {
            turnId: "turn-1",
            threadId: "thread-1",
          },
        }),
        {
          status: 202,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    throw new Error(`Unhandled request: ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const promptInputs = await screen.findAllByLabelText("Prompt");
  expect(FakeWebSocket.instances).toHaveLength(2);
  const [promptInput] = promptInputs;
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

test("websocket workspace events update the active console timeline", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = getPath(input);

    if (url === "/capabilities") {
      return new Response(JSON.stringify(capabilities), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (url === "/settings/console") {
      return new Response(JSON.stringify({ preferences: null }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (url === "/threads") {
      return new Response(
        JSON.stringify({
          items: [
            {
              threadId: "thread-1",
              machineId: "machine-1",
              status: "idle",
              title: "Investigate flaky test",
            },
          ],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/machines") {
      return new Response(
        JSON.stringify({
          items: [
            {
              id: "machine-1",
              name: "Primary Node",
              status: "online",
              runtimeStatus: "running",
            },
          ],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/threads/thread-1") {
      return new Response(
        JSON.stringify({
          thread: {
            threadId: "thread-1",
            machineId: "machine-1",
            status: "idle",
            title: "Investigate flaky test",
          },
          pendingApprovals: [],
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    if (url === "/machines/machine-1") {
      return new Response(
        JSON.stringify({
          machine: {
            id: "machine-1",
            name: "Primary Node",
            status: "online",
            runtimeStatus: "running",
          },
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    throw new Error(`Unhandled request: ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  await screen.findAllByLabelText("Prompt");
  await waitFor(() => {
    expect(FakeWebSocket.instances).toHaveLength(2);
  });
  const workspaceSocket = FakeWebSocket.instances[1];
  expect(workspaceSocket).toBeDefined();

  await act(async () => {
    workspaceSocket?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.delta",
        timestamp: "2026-04-08T14:00:01Z",
        payload: {
          threadId: "thread-1",
          turnId: "turn-1",
          sequence: 1,
          delta: "hello from gateway",
        },
      }),
    );
  });

  expect(await screen.findAllByText("hello from gateway")).toHaveLength(2);

  await act(async () => {
    workspaceSocket?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "approval.required",
        requestId: "approval-1",
        timestamp: "2026-04-08T14:00:02Z",
        payload: {
          requestId: "approval-1",
          threadId: "thread-1",
          turnId: "turn-1",
          itemId: "item-1",
          kind: "command",
          command: "go test ./...",
        },
      }),
    );
  });

  expect(await screen.findAllByText("待处理审批")).toHaveLength(2);
  expect(screen.getAllByText("go test ./...")).toHaveLength(2);

  await act(async () => {
    workspaceSocket?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "approval.resolved",
        requestId: "approval-1",
        timestamp: "2026-04-08T14:00:03Z",
        payload: {
          requestId: "approval-1",
          threadId: "thread-1",
          decision: "accept",
        },
      }),
    );
  });

  await waitFor(() => {
    expect(screen.queryAllByText("go test ./...")).toHaveLength(0);
  });

  await act(async () => {
    workspaceSocket?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.completed",
        timestamp: "2026-04-08T14:00:04Z",
        payload: {
          turn: {
            turnId: "turn-1",
            threadId: "thread-1",
            status: "completed",
          },
        },
      }),
    );
    workspaceSocket?.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.failed",
        timestamp: "2026-04-08T14:00:05Z",
        payload: {
          turn: {
            turnId: "turn-2",
            threadId: "thread-1",
            status: "failed",
          },
        },
      }),
    );
  });

  expect(await screen.findAllByText("Turn turn-1 completed")).toHaveLength(2);
  expect(await screen.findAllByText("Turn turn-2 failed")).toHaveLength(2);
});
