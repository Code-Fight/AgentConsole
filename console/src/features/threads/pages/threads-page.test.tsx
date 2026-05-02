import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { AppProviders } from "../../../app/providers/index";
import { createAppRouter } from "../../../app/router/index";
import { resetCapabilitiesForTests } from "../../../common/config/capabilities";
import { clearGatewayConnectionCookies } from "../../../common/config/gateway-connection-store";

const GATEWAY_URL = "http://localhost:18080";
const GATEWAY_API_KEY = "feature-test-key";

class FakeWebSocket {
  close() {}

  addEventListener() {}

  removeEventListener() {}
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
  clearGatewayConnectionCookies();
  resetCapabilitiesForTests();
  vi.unstubAllGlobals();
});

test("renders / with Gateway Thread 1 and Machine One visible", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
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
              status: "active",
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
            },
          ],
        });
      }

      throw new Error(`unexpected fetch path: ${path}`);
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const router = createAppRouter({ initialEntries: ["/"] });
  render(<AppProviders router={router} />);

  expect(await screen.findByText("Gateway Thread 1", { exact: true })).toBeInTheDocument();
  expect(await screen.findByText("Machine One", { exact: true })).toBeInTheDocument();
});
