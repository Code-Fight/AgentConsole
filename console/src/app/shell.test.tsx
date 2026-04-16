import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import mainEntrySource from "../main.tsx?raw";
import environmentSource from "../design-source/components/Environment.tsx?raw";
import settingsSource from "../design-source/components/Settings.tsx?raw";
import { resetConsolePreferencesStoreForTests } from "../gateway/use-console-preferences";
import { DesignSourceAppRoot } from "../design-host/app-root";

const GATEWAY_URL = "http://localhost:18080";
const GATEWAY_API_KEY = "test-key";

class FakeWebSocket {
  readonly close = vi.fn();

  addEventListener() {}

  removeEventListener() {}
}

const capabilitySnapshot = {
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
};

function jsonResponse(payload: unknown) {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function handleConsoleSettings(init?: RequestInit) {
  if (init?.method === "PUT") {
    const body = init.body ? JSON.parse(String(init.body)) : { preferences: null };
    return jsonResponse(body);
  }
  return jsonResponse({ preferences: null });
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

function clearGatewayCookies() {
  document.cookie = "cag_gateway_url=; Max-Age=0; Path=/";
  document.cookie = "cag_gateway_api_key=; Max-Age=0; Path=/";
}

beforeEach(() => {
  window.history.pushState({}, "", "/");
  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
    value: vi.fn(),
    writable: true,
  });
  resetConsolePreferencesStoreForTests();
  setGatewayCookies();
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

test("main entry only loads the design-source styles for the active figma-driven console", () => {
  expect(mainEntrySource).not.toContain('import "./styles.css";');
  expect(mainEntrySource).not.toContain('import "./design/styles/index.css";');
  expect(mainEntrySource).toContain('import "./design-source/styles/index.css";');
});

test("environment and settings stay inside design-source instead of wrapping custom design views", () => {
  expect(environmentSource).not.toContain('from "../../design";');
  expect(settingsSource).not.toContain('from "../../design";');
  expect(environmentSource).toContain("useEnvironmentPage({ enabled: connection.remoteEnabled })");
});

test("shows a blocking connection dialog when gateway cookies are missing", async () => {
  clearGatewayCookies();
  const fetchSpy = vi.fn();
  vi.stubGlobal("fetch", fetchSpy);

  render(<DesignSourceAppRoot />);

  expect(await screen.findByText(/Gateway 连接未配置/)).toBeInTheDocument();
  expect(fetchSpy).not.toHaveBeenCalled();
});

test("settings stays local when gateway cookies are missing", async () => {
  window.history.pushState({}, "", "/settings");
  clearGatewayCookies();
  const fetchSpy = vi.fn();
  vi.stubGlobal("fetch", fetchSpy);

  render(<DesignSourceAppRoot />);

  expect((await screen.findAllByLabelText("Gateway URL")).length).toBeGreaterThan(0);
  expect(screen.getAllByLabelText("API Key").length).toBeGreaterThan(0);
  expect(fetchSpy).not.toHaveBeenCalled();
});

test("nested settings routes stay reachable when gateway cookies are missing", async () => {
  window.history.pushState({}, "", "/settings/advanced");
  clearGatewayCookies();
  const fetchSpy = vi.fn();
  vi.stubGlobal("fetch", fetchSpy);

  render(<DesignSourceAppRoot />);

  expect((await screen.findAllByLabelText("Gateway URL")).length).toBeGreaterThan(0);
  expect(screen.getAllByLabelText("API Key").length).toBeGreaterThan(0);
  expect(screen.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
  expect(fetchSpy).not.toHaveBeenCalled();
});

test("loads gateway thread and machine lists for the active console shell", async () => {
  const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
    const path = getPath(input);

    if (path === "/capabilities") {
      return jsonResponse(capabilitySnapshot);
    }

    if (path === "/settings/console") {
      return handleConsoleSettings();
    }

    if (path === "/threads") {
      return jsonResponse({ items: [] });
    }

    if (path === "/machines") {
      return jsonResponse({ items: [] });
    }

    throw new Error(`unexpected fetch: ${path}`);
  });

  vi.stubGlobal("fetch", fetchSpy);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(<DesignSourceAppRoot />);

  await waitFor(() => {
    const paths = fetchSpy.mock.calls.map(([input]) => getPath(input));
    expect(paths).toContain("/threads");
    expect(paths).toContain("/machines");
  });
});

test("does not rely on local mock assistant replies in the active workspace", async () => {
  window.history.pushState({}, "", "/threads/thread-1");

  const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = getPath(input);

      if (path === "/capabilities") {
        return jsonResponse(capabilitySnapshot);
      }

      if (path === "/settings/console") {
        return handleConsoleSettings(init);
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

      if (path === "/threads/thread-1") {
        return jsonResponse({
          thread: {
            threadId: "thread-1",
            machineId: "machine-1",
            status: "active",
            title: "Gateway Thread 1",
          },
          activeTurnId: null,
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

      if (path === "/threads/thread-1/turns" && init?.method === "POST") {
        return jsonResponse({
          turn: {
            turnId: "turn-1",
            threadId: "thread-1",
          },
        });
      }

      throw new Error(`unexpected fetch: ${path}`);
    });
  vi.stubGlobal("fetch", fetchSpy);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(<DesignSourceAppRoot />);

  await waitFor(() => {
    const paths = fetchSpy.mock.calls.map(([input]) => getPath(input));
    expect(paths).toContain("/capabilities");
  });

  const [promptInput] = await screen.findAllByPlaceholderText(/发送指令/);
  fireEvent.change(promptInput, { target: { value: "Ping from test" } });
  fireEvent.keyDown(promptInput, { key: "Enter", code: "Enter", charCode: 13 });

  await waitFor(() => {
    const postCall = fetchSpy.mock.calls.find(
      ([input, init]) =>
        getPath(input) === "/threads/thread-1/turns" &&
        typeof init === "object" &&
        (init as RequestInit)?.method === "POST",
    );
    expect(postCall).toBeTruthy();
  });
  expect(screen.queryByText(/收到你的指令/)).not.toBeInTheDocument();
});

test("keeps thread list when machine load fails", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = getPath(input);

      if (path === "/capabilities") {
        return jsonResponse(capabilitySnapshot);
      }

      if (path === "/settings/console") {
        return handleConsoleSettings();
      }

      if (path === "/threads") {
        return jsonResponse({
          items: [
            {
              threadId: "thread-2",
              machineId: "machine-2",
              status: "idle",
              title: "Partial load thread",
            },
          ],
        });
      }

      if (path === "/machines") {
        throw new Error("network down");
      }

      throw new Error(`unexpected fetch: ${path}`);
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(<DesignSourceAppRoot />);

  expect(await screen.findByText("Partial load thread")).toBeInTheDocument();
});

test("surfaces system error thread status instead of treating it as completed", async () => {
  window.history.pushState({}, "", "/threads/thread-3");

  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = getPath(input);

      if (path === "/capabilities") {
        return jsonResponse(capabilitySnapshot);
      }

      if (path === "/settings/console") {
        return handleConsoleSettings();
      }

      if (path === "/threads") {
        return jsonResponse({
          items: [
            {
              threadId: "thread-3",
              machineId: "machine-3",
              status: "systemError",
              title: "System error thread",
            },
          ],
        });
      }

      if (path === "/machines") {
        return jsonResponse({
          items: [
            {
              id: "machine-3",
              name: "Machine Three",
              status: "reconnecting",
              runtimeStatus: "running",
            },
          ],
        });
      }

      if (path === "/threads/thread-3") {
        return jsonResponse({
          thread: {
            threadId: "thread-3",
            machineId: "machine-3",
            status: "systemError",
            title: "System error thread",
          },
          activeTurnId: null,
          pendingApprovals: [],
        });
      }

      if (path === "/machines/machine-3") {
        return jsonResponse({
          machine: {
            id: "machine-3",
            name: "Machine Three",
            status: "reconnecting",
            runtimeStatus: "running",
          },
        });
      }

      throw new Error(`unexpected fetch: ${path}`);
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(<DesignSourceAppRoot />);

  expect((await screen.findAllByText("异常")).length).toBeGreaterThan(0);
});

test("clears persisted last thread when restore fails", async () => {
  const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path === "/capabilities") {
      return jsonResponse(capabilitySnapshot);
    }

    if (path === "/settings/console" && init?.method === "PUT") {
      const body = init.body ? JSON.parse(String(init.body)) : { preferences: null };
      return jsonResponse(body);
    }

    if (path === "/settings/console") {
      return jsonResponse({
        preferences: {
          consoleUrl: "",
          apiKey: "",
          profile: "",
          safetyPolicy: "",
          lastThreadId: "missing-thread",
        },
      });
    }

    if (path === "/threads") {
      return jsonResponse({ items: [] });
    }

    if (path === "/machines") {
      return jsonResponse({ items: [] });
    }

    if (path === "/threads/missing-thread") {
      return new Response("not found", { status: 404 });
    }

    throw new Error(`unexpected fetch: ${path}`);
  });

  vi.stubGlobal("fetch", fetchSpy);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(<DesignSourceAppRoot />);

  await waitFor(() => {
    const putCall = fetchSpy.mock.calls.find(
      ([input, init]) => getPath(input) === "/settings/console" && init?.method === "PUT",
    );
    expect(putCall).toBeTruthy();
    const body = putCall?.[1]?.body ? JSON.parse(String(putCall[1]?.body)) : null;
    expect(body?.preferences?.lastThreadId).toBe("");
  });

  await waitFor(() => {
    expect(window.location.pathname).toBe("/");
  });
});
