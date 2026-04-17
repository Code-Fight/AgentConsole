import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { ConsoleHostRouter } from "../design-host/console-host-router";

const GATEWAY_URL = "http://localhost:18080";
const GATEWAY_API_KEY = "test-key";

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

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function getPath(input: RequestInfo | URL): string {
  const raw = String(input);
  try {
    return new URL(raw).pathname;
  } catch {
    return raw;
  }
}

beforeEach(() => {
  document.cookie = `cag_gateway_url=${encodeURIComponent(GATEWAY_URL)}; Path=/`;
  document.cookie = `cag_gateway_api_key=${encodeURIComponent(GATEWAY_API_KEY)}; Path=/`;
});

afterEach(() => {
  FakeWebSocket.instances = [];
  vi.unstubAllGlobals();
});

test("renders the active console thread list from /threads and machines from /machines", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = getPath(input);

    if (url === "/capabilities") {
      return jsonResponse(capabilities);
    }

    if (url === "/settings/console") {
      return jsonResponse({ preferences: null });
    }

    if (url === "/threads") {
      return jsonResponse({
        items: [
          {
            threadId: "thread-1",
            machineId: "machine-1",
            agentId: "agent-01",
            status: "idle",
            title: "Investigate flaky test",
          },
        ],
      });
    }

    if (url === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-1",
            name: "Primary Node",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-01",
                agentType: "codex",
                displayName: "Primary Codex",
                status: "running",
              },
            ],
          },
        ],
      });
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

  expect(await screen.findByText("Investigate flaky test")).toBeInTheDocument();
  expect(screen.getByText("Primary Node")).toBeInTheDocument();
  expect(screen.getByText("ID: machine-1")).toBeInTheDocument();
  expect(FakeWebSocket.instances).toHaveLength(1);
});

test("active create-thread flow submits both machineId and agentId", async () => {
  let threads = [
    {
      threadId: "thread-1",
      machineId: "machine-1",
      agentId: "agent-01",
      status: "idle",
      title: "Investigate flaky test",
    },
  ];

  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = getPath(input);
    const method = init?.method ?? "GET";

    if (url === "/capabilities") {
      return jsonResponse(capabilities);
    }
    if (url === "/settings/console") {
      return jsonResponse({ preferences: null });
    }
    if (url === "/threads" && method === "GET") {
      return jsonResponse({ items: threads });
    }
    if (url === "/threads" && method === "POST") {
      threads = [
        ...threads,
        {
          threadId: "thread-2",
          machineId: "machine-1",
          agentId: "agent-02",
          status: "idle",
          title: "Ship managed agents",
        },
      ];
      return jsonResponse(
        {
          thread: {
            threadId: "thread-2",
            machineId: "machine-1",
            agentId: "agent-02",
            status: "idle",
            title: "Ship managed agents",
          },
        },
        201,
      );
    }
    if (url === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-1",
            name: "Primary Node",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-01",
                agentType: "codex",
                displayName: "Primary Codex",
                status: "running",
              },
              {
                agentId: "agent-02",
                agentType: "codex",
                displayName: "Secondary Codex",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    throw new Error(`Unhandled request: ${method} ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  expect(await screen.findByText("Investigate flaky test")).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "新建" }));
  fireEvent.change(screen.getByRole("combobox"), {
    target: { value: "machine-1" },
  });
  fireEvent.change(screen.getAllByRole("combobox")[1], {
    target: { value: "agent-02" },
  });
  fireEvent.change(screen.getByPlaceholderText("例如: 实现用户认证功能"), {
    target: { value: "Ship managed agents" },
  });
  fireEvent.click(screen.getByRole("button", { name: "创建" }));

  await waitFor(() => {
    const postCall = fetchMock.mock.calls.find(
      ([input, init]) => getPath(input) === "/threads" && (init as RequestInit | undefined)?.method === "POST",
    );
    expect(postCall).toBeTruthy();
    expect(postCall?.[1]).toEqual(
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          machineId: "machine-1",
          agentId: "agent-02",
          title: "Ship managed agents",
        }),
      }),
    );
  });
});
