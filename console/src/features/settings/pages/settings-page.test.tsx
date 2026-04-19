import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { AppProviders } from "../../../app/providers/index";
import { createAppRouter } from "../../../app/router/index";
import { resetCapabilitiesForTests } from "../../../common/config/capabilities";
import {
  clearGatewayConnectionCookies,
  resetGatewayConnectionStoreForTests,
} from "../model/gateway-connection-store";
import { resetConsolePreferencesStoreForTests } from "../hooks/use-console-preferences";

const GATEWAY_URL = "http://localhost:18080";
const GATEWAY_API_KEY = "test-key";

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
    },
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

function setGatewayCookies() {
  document.cookie = `cag_gateway_url=${encodeURIComponent(GATEWAY_URL)}; Path=/`;
  document.cookie = `cag_gateway_api_key=${encodeURIComponent(GATEWAY_API_KEY)}; Path=/`;
}

async function getMainScope() {
  const mains = await screen.findAllByRole("main");
  return within(mains[0]);
}

beforeEach(() => {
  resetGatewayConnectionStoreForTests();
  resetCapabilitiesForTests();
  resetConsolePreferencesStoreForTests();
  setGatewayCookies();
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

test("/settings remains reachable while unconfigured", async () => {
  clearGatewayConnectionCookies();
  const fetchSpy = vi.fn();
  vi.stubGlobal("fetch", fetchSpy);

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  const scope = await getMainScope();
  expect(await scope.findByLabelText("Gateway URL")).toBeInTheDocument();
  expect(scope.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
  expect(fetchSpy).not.toHaveBeenCalled();
});

test("save gateway connection stays disabled until the current draft passes connection test", async () => {
  clearGatewayConnectionCookies();

  const requestSnapshots: Array<{ path: string; cookie: string }> = [];
  const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);
    requestSnapshots.push({ path, cookie: document.cookie });

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
      });
    }

    if (path === "/settings/console" && (!init?.method || init.method === "GET")) {
      return jsonResponse({
        preferences: {
          profile: "",
          safetyPolicy: "",
          lastThreadId: "",
        },
      });
    }

    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }

    if (path === "/machines") {
      return jsonResponse({
        items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }],
      });
    }

    if (path === "/settings/agents/codex/global") {
      return jsonResponse({ document: null });
    }

    if (path === "/settings/machines/machine-01/agents/codex") {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: null,
        machineOverride: null,
        usesGlobalDefault: true,
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchSpy);

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  const scope = await getMainScope();
  expect(fetchSpy).not.toHaveBeenCalled();
  const saveButton = scope.getByRole("button", { name: "Save Gateway Connection" });
  expect(saveButton).toBeDisabled();

  fireEvent.change(await scope.findByLabelText("Gateway URL"), {
    target: { value: "http://localhost:3100" },
  });
  fireEvent.change(scope.getByLabelText("Gateway API Key"), {
    target: { value: "test-key" },
  });
  expect(saveButton).toBeDisabled();
  expect(fetchSpy).not.toHaveBeenCalled();

  fireEvent.click(scope.getByRole("button", { name: "Test Gateway Connection" }));

  await waitFor(() => {
    expect(
      fetchSpy.mock.calls.some(([input]) => getPath(input as RequestInfo | URL) === "/capabilities"),
    ).toBe(true);
  });

  expect(await scope.findByText("Gateway 连接测试成功。")).toBeInTheDocument();
  expect(saveButton).toBeEnabled();

  fireEvent.change(scope.getByLabelText("Gateway URL"), {
    target: { value: "http://localhost:3200" },
  });
  expect(saveButton).toBeDisabled();

  fireEvent.click(scope.getByRole("button", { name: "Test Gateway Connection" }));

  await waitFor(() => {
    expect(fetchSpy.mock.calls.filter(([input]) => getPath(input as RequestInfo | URL) === "/capabilities")).toHaveLength(2);
  });

  expect(saveButton).toBeEnabled();
  fireEvent.click(saveButton);

  await waitFor(() => {
    expect(requestSnapshots.some(({ path }) => path === "/settings/agents")).toBe(true);
  });

  const savedFetchSnapshots = requestSnapshots.filter(({ path }) =>
    path !== "/capabilities" ? true : false,
  );
  expect(
    savedFetchSnapshots.every(({ cookie }) =>
      cookie.includes("cag_gateway_url=http%3A%2F%2Flocalhost%3A3200"),
    ),
  ).toBe(true);
  expect(
    savedFetchSnapshots.every(({ cookie }) => cookie.includes("cag_gateway_api_key=test-key")),
  ).toBe(true);
});

test("remote settings config load path still works for global default and machine override", async () => {
  const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
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
      });
    }

    if (path === "/settings/console" && (!init?.method || init.method === "GET")) {
      return jsonResponse({
        preferences: {
          profile: "dev",
          safetyPolicy: "strict",
          lastThreadId: "",
        },
      });
    }

    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }

    if (path === "/machines") {
      return jsonResponse({
        items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }],
      });
    }

    if (path === "/settings/agents/codex/global") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" },
      });
    }

    if (path === "/settings/machines/machine-01/agents/codex") {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" },
        machineOverride: { agentType: "codex", format: "toml", content: "model = \"gpt-5.2\"\n" },
        usesGlobalDefault: false,
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchSpy);

  const router = createAppRouter({ initialEntries: ["/settings/advanced"] });
  render(<AppProviders router={router} />);

  const scope = await getMainScope();
  await waitFor(() => {
    expect(scope.getByLabelText("Global Default TOML")).toHaveValue("model = \"gpt-5.4\"\n");
  });
  expect(scope.getByLabelText("Machine Override TOML")).toHaveValue("model = \"gpt-5.2\"\n");
  expect(scope.getByLabelText("Console Profile")).toHaveValue("dev");
  expect(scope.getByLabelText("Safety Policy")).toHaveValue("strict");
  expect(scope.getByText("Machine 01")).toBeInTheDocument();
  expect(scope.queryByText("Using global default")).not.toBeInTheDocument();
});
