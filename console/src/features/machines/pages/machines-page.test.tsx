import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

vi.mock("../../../design-host/use-console-host", () => {
  throw new Error("machines feature must not depend on design-host runtime state");
});

vi.mock("../../../gateway/capabilities", () => {
  throw new Error("machines feature must not depend on legacy gateway capabilities");
});

vi.mock("../../threads/hooks/use-thread-hub", () => {
  throw new Error("machines feature must not depend on threads runtime hub");
});

vi.mock("../../threads/model/thread-view-model", () => {
  throw new Error("machines feature must not depend on threads view-model builders");
});

import {
  clearGatewayConnectionCookies,
  saveGatewayConnectionToCookies,
} from "../../settings/model/gateway-connection-store";
import { MachinesPage } from "./machines-page";

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
}

const capabilities = {
  threadHub: true,
  threadWorkspace: true,
  approvals: true,
  startTurn: true,
  steerTurn: true,
  interruptTurn: true,
  machineInstallAgent: true,
  machineRemoveAgent: true,
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
  agentLifecycle: true,
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function getPath(input: RequestInfo | URL): string {
  const raw = String(input);
  if (raw.startsWith("http://") || raw.startsWith("https://")) {
    return new URL(raw).pathname;
  }
  return raw;
}

beforeEach(() => {
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "machines-feature-test-key",
  });
});

afterEach(() => {
  FakeWebSocket.instances = [];
  clearGatewayConnectionCookies();
  vi.unstubAllGlobals();
});

test("renders machines and saves per-agent config from the feature-local Machines page", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);
    const method = init?.method ?? "GET";

    if (path === "/capabilities") {
      return jsonResponse(capabilities);
    }
    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-01",
            name: "Machine 01",
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
    if (path === "/threads") {
      return jsonResponse({ items: [] });
    }
    if (path === "/machines/machine-01/agents/agent-01/config" && method === "GET") {
      return jsonResponse({
        document: {
          agentType: "codex",
          format: "toml",
          content: "model = \"gpt-5.4\"\n",
        },
      });
    }
    if (path === "/machines/machine-01/agents/agent-01/config" && method === "PUT") {
      return jsonResponse({
        document: {
          agentType: "codex",
          format: "toml",
          content: "model = \"gpt-5.5\"\n",
        },
      });
    }

    throw new Error(`Unexpected request: ${method} ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter>
      <MachinesPage />
    </MemoryRouter>,
  );

  expect((await screen.findAllByText("Machine 01")).length).toBeGreaterThan(0);

  fireEvent.click(screen.getAllByTitle("编辑配置")[0]);

  await waitFor(() => {
    expect(
      fetchMock.mock.calls.some(
        ([input, init]) =>
          getPath(input as RequestInfo | URL) === "/machines/machine-01/agents/agent-01/config" &&
          (!init || !("method" in init) || !init.method || init.method === "GET"),
      ),
    ).toBe(true);
  });

  fireEvent.change(screen.getByRole("textbox"), {
    target: { value: "model = \"gpt-5.5\"\n" },
  });
  fireEvent.click(screen.getByRole("button", { name: "保存" }));

  await waitFor(() => {
    expect(
      fetchMock.mock.calls.some(
        ([input, init]) =>
          getPath(input as RequestInfo | URL) ===
            "/machines/machine-01/agents/agent-01/config" &&
          (init as RequestInit | undefined)?.method === "PUT" &&
          (init as RequestInit | undefined)?.body ===
            JSON.stringify({ content: "model = \"gpt-5.5\"\n" }),
      ),
    ).toBe(true);
  });
});

test("install and delete actions call the managed agent lifecycle APIs", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);
    const method = init?.method ?? "GET";

    if (path === "/capabilities") {
      return jsonResponse(capabilities);
    }
    if (path === "/threads") {
      return jsonResponse({ items: [] });
    }
    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-01",
            name: "Machine 01",
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
    if (path === "/machines/machine-01/agents" && method === "POST") {
      return jsonResponse(
        {
          agent: {
            agentId: "agent-02",
            agentType: "codex",
            displayName: "Secondary Codex",
            status: "running",
          },
        },
        201,
      );
    }
    if (path === "/machines/machine-01/agents/agent-01" && method === "DELETE") {
      return new Response(null, { status: 204 });
    }

    throw new Error(`Unexpected request: ${method} ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter>
      <MachinesPage />
    </MemoryRouter>,
  );

  expect((await screen.findAllByText("Machine 01")).length).toBeGreaterThan(0);

  fireEvent.click(screen.getAllByRole("button", { name: "安装 Agent" })[0]);
  fireEvent.change(screen.getByRole("combobox"), {
    target: { value: "codex" },
  });
  fireEvent.change(screen.getByPlaceholderText("例如: Claude Sonnet"), {
    target: { value: "Secondary Codex" },
  });
  fireEvent.click(screen.getByRole("button", { name: "安装" }));

  await waitFor(() => {
    expect(
      fetchMock.mock.calls.some(
        ([input, init]) =>
          getPath(input as RequestInfo | URL) === "/machines/machine-01/agents" &&
          (init as RequestInit | undefined)?.method === "POST" &&
          (init as RequestInit | undefined)?.body ===
            JSON.stringify({
              agentType: "codex",
              displayName: "Secondary Codex",
            }),
      ),
    ).toBe(true);
  });

  fireEvent.click(screen.getAllByTitle("删除 Agent")[0]);
  fireEvent.click(screen.getByRole("button", { name: "删除" }));

  await waitFor(() => {
    expect(
      fetchMock.mock.calls.some(
        ([input, init]) =>
          getPath(input as RequestInfo | URL) === "/machines/machine-01/agents/agent-01" &&
          (init as RequestInit | undefined)?.method === "DELETE",
      ),
    ).toBe(true);
  });
});

test("runtime control buttons call the machine runtime endpoints", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);
    const method = init?.method ?? "GET";

    if (path === "/capabilities") {
      return jsonResponse(capabilities);
    }
    if (path === "/threads") {
      return jsonResponse({ items: [] });
    }
    if (path === "/machines") {
      return jsonResponse({
        items: [
          {
            id: "machine-01",
            name: "Machine 01",
            status: "online",
            runtimeStatus: "running",
            agents: [],
          },
          {
            id: "machine-02",
            name: "Machine 02",
            status: "offline",
            runtimeStatus: "stopped",
            agents: [],
          },
        ],
      });
    }
    if (path === "/machines/machine-01/runtime/stop" && method === "POST") {
      return jsonResponse({ machineId: "machine-01" });
    }
    if (path === "/machines/machine-02/runtime/start" && method === "POST") {
      return jsonResponse({ machineId: "machine-02" });
    }

    throw new Error(`Unexpected request: ${method} ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter>
      <MachinesPage />
    </MemoryRouter>,
  );

  expect((await screen.findAllByText("Machine 01")).length).toBeGreaterThan(0);

  fireEvent.click(screen.getAllByRole("button", { name: "Stop runtime" })[0]);
  fireEvent.click(screen.getAllByRole("button", { name: "Start runtime" })[0]);

  await waitFor(() => {
    expect(
      fetchMock.mock.calls.some(
        ([input, init]) =>
          getPath(input as RequestInfo | URL) === "/machines/machine-01/runtime/stop" &&
          (init as RequestInit | undefined)?.method === "POST",
      ),
    ).toBe(true);
    expect(
      fetchMock.mock.calls.some(
        ([input, init]) =>
          getPath(input as RequestInfo | URL) === "/machines/machine-02/runtime/start" &&
          (init as RequestInit | undefined)?.method === "POST",
      ),
    ).toBe(true);
  });
});
