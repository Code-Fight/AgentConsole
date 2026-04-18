import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import { clearGatewayConnectionCookies } from "../gateway/gateway-connection-store";
import { resetConsolePreferencesStoreForTests } from "../gateway/use-console-preferences";
import { DesignSourceAppRoot } from "../design-host/app-root";
import { AppProviders } from "./providers/index";
import { createAppRouter } from "./router/index";

beforeEach(() => {
  window.history.pushState({}, "", "/");
  resetConsolePreferencesStoreForTests();
});

afterEach(() => {
  vi.unstubAllGlobals();
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

test("settings stays reachable when gateway cookies are missing", async () => {
  window.history.pushState({}, "", "/settings");
  clearGatewayConnectionCookies();

  render(<DesignSourceAppRoot />);

  expect((await screen.findAllByLabelText("Gateway URL")).length).toBeGreaterThan(0);
  expect(screen.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
});

test("new app router keeps settings reachable when gateway cookies are missing", async () => {
  clearGatewayConnectionCookies();

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  expect((await screen.findAllByLabelText("Gateway URL")).length).toBeGreaterThan(0);
  expect(screen.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
});
