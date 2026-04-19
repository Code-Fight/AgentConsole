import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, test, vi } from "vitest";
import { resetCapabilitiesForTests } from "../common/config/capabilities";
import {
  clearGatewayConnectionCookies,
  resetGatewayConnectionStoreForTests,
} from "../common/config/gateway-connection-store";
import { resetConsolePreferencesStoreForTests } from "../features/settings/hooks/use-console-preferences";
import { AppProviders } from "./providers/index";
import { createAppRouter } from "./router/index";

beforeEach(() => {
  window.history.pushState({}, "", "/");
  resetGatewayConnectionStoreForTests();
  resetCapabilitiesForTests();
  resetConsolePreferencesStoreForTests();
  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
    value: vi.fn(),
    writable: true,
  });
});

afterEach(() => {
  clearGatewayConnectionCookies();
  resetCapabilitiesForTests();
  vi.unstubAllGlobals();
});

test("active runtime and cross-feature tests do not import the settings gateway shim", () => {
  const files = import.meta.glob(
    [
      "./layout/connection-gate.tsx",
      "./runtime-baseline.test.tsx",
      "./shell.test.tsx",
      "../features/overview/pages/overview-page.tsx",
      "../features/environment/pages/environment-page.tsx",
      "../features/environment/pages/environment-page.test.tsx",
      "../features/environment/hooks/use-environment-page.ts",
      "../features/machines/pages/machines-page.test.tsx",
      "../features/machines/hooks/use-machines-page.ts",
      "../features/threads/pages/threads-page.test.tsx",
      "../features/threads/pages/thread-workspace-page.test.tsx",
      "../features/threads/hooks/use-thread-hub.ts",
      "../features/threads/hooks/use-thread-workspace.ts",
      "../features/threads/components/thread-shell.tsx",
    ],
    { eager: true, query: "?raw", import: "default" },
  ) as Record<string, string>;

  const shimPath = "settings/model/gateway-connection-store";

  for (const [filePath, source] of Object.entries(files)) {
    expect(source, `${filePath} should import the common gateway connection store directly`).not.toContain(
      shimPath,
    );
  }
});

test("no runtime entry imports legacy console layers after the migration", () => {
  expect(Object.keys(import.meta.glob("../design-source/**/*"))).toHaveLength(0);
  expect(Object.keys(import.meta.glob("../design-host/**/*"))).toHaveLength(0);
  expect(Object.keys(import.meta.glob("../design-bridge/**/*"))).toHaveLength(0);
  expect(Object.keys(import.meta.glob("../gateway/**/*"))).toHaveLength(0);
  expect(Object.keys(import.meta.glob("../design/**/*"))).toHaveLength(0);
  expect(Object.keys(import.meta.glob("../pages/**/*"))).toHaveLength(0);
});

test("composes app shell routes from the new router entry", async () => {
  document.cookie = "cag_gateway_url=http%3A%2F%2Flocalhost%3A18080; Path=/";
  document.cookie = "cag_gateway_api_key=test-key; Path=/";

  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
    const path = new URL(String(input)).pathname;
    if (path === "/capabilities") {
      return new Response(JSON.stringify({
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
        environmentMutateResources: true,
        environmentWriteMcp: true,
        environmentWriteSkills: true,
        settingsEditGatewayEndpoint: false,
        settingsEditConsoleProfile: false,
        settingsEditSafetyPolicy: false,
        settingsGlobalDefault: true,
        settingsMachineOverride: true,
        settingsApplyMachine: true,
        dashboardMetrics: false,
        agentLifecycle: false,
      }), { status: 200, headers: { "Content-Type": "application/json" } });
    }

    if (path === "/settings/console") {
      return new Response(JSON.stringify({ preferences: null }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (path === "/threads") {
      return new Response(JSON.stringify({
        items: [
          { threadId: "thread-1", machineId: "machine-1", status: "active", title: "Gateway Thread 1" },
        ],
      }), { status: 200, headers: { "Content-Type": "application/json" } });
    }

    if (path === "/machines") {
      return new Response(JSON.stringify({
        items: [{ id: "machine-1", name: "Machine One", status: "online", runtimeStatus: "running" }],
      }), { status: 200, headers: { "Content-Type": "application/json" } });
    }

    throw new Error(`unexpected fetch path: ${path}`);
  }));

  class FakeWebSocket {
    close() {}

    addEventListener() {}

    removeEventListener() {}
  }

  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const router = createAppRouter({ initialEntries: ["/"] });
  render(<AppProviders router={router} />);

  expect((await screen.findAllByText("Agent Console")).length).toBeGreaterThan(0);
  expect(await screen.findByText("Gateway Thread 1", { exact: true })).toBeInTheDocument();
});

test("feature router keeps settings reachable when gateway cookies are missing", async () => {
  clearGatewayConnectionCookies();

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  expect((await screen.findAllByLabelText("Gateway URL")).length).toBeGreaterThan(0);
  expect(screen.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
});

test("feature router serves /overview from the feature runtime", async () => {
  document.cookie = "cag_gateway_url=http%3A%2F%2Flocalhost%3A18080; Path=/";
  document.cookie = "cag_gateway_api_key=test-key; Path=/";

  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = new URL(String(input)).pathname;
      if (path === "/capabilities") {
        return new Response(JSON.stringify({ threadHub: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }
      if (path === "/overview/metrics") {
        return new Response(
          JSON.stringify({
            onlineMachines: 1,
            activeThreads: 1,
            pendingApprovals: 0,
            runningAgents: 1,
            environmentItems: 3,
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        );
      }
      if (path === "/threads") {
        return new Response(JSON.stringify({ items: [] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }
      if (path === "/machines") {
        return new Response(JSON.stringify({ items: [] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }
      if (path === "/settings/console") {
        return new Response(JSON.stringify({ preferences: null }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }

      throw new Error(`unexpected fetch path: ${path}`);
    }),
  );

  class FakeWebSocket {
    close() {}

    addEventListener() {}

    removeEventListener() {}
  }

  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const router = createAppRouter({ initialEntries: ["/overview"] });
  render(<AppProviders router={router} />);

  expect(router.state.location.pathname).toBe("/overview");
  expect(await screen.findByText("Online Machines")).toBeInTheDocument();
});

test("feature router keeps nested settings routes reachable when gateway cookies are missing", async () => {
  clearGatewayConnectionCookies();

  const router = createAppRouter({ initialEntries: ["/settings/advanced"] });
  render(<AppProviders router={router} />);

  expect((await screen.findAllByLabelText("Gateway URL")).length).toBeGreaterThan(0);
  expect(screen.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
});
