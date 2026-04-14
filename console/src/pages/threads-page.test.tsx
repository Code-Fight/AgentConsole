import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ConsoleHostRouter } from "../design-host/console-host-router";

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
  settingsEditGatewayEndpoint: false,
  settingsEditConsoleProfile: false,
  settingsEditSafetyPolicy: false,
  settingsGlobalDefault: true,
  settingsMachineOverride: true,
  settingsApplyMachine: true,
  dashboardMetrics: false,
  agentLifecycle: false,
};

afterEach(() => {
  FakeWebSocket.instances = [];
  vi.unstubAllGlobals();
});

test("renders the active console thread list from /threads and machines from /machines", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);

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
