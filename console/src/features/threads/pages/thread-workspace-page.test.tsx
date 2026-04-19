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
