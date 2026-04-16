import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { ConsoleHostRouter } from "../design-host/console-host-router";
import { resetConsolePreferencesStoreForTests } from "../gateway/use-console-preferences";

const GATEWAY_URL = "http://localhost:18080";
const GATEWAY_API_KEY = "test-key";

const capabilitySnapshot = vi.hoisted(() => ({
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
}));

vi.mock("../gateway/capabilities", () => ({
  useCapabilities: () => capabilitySnapshot,
  supportsCapability: (capability: string) =>
    Boolean(capabilitySnapshot[capability as keyof typeof capabilitySnapshot]),
}));

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
  setGatewayCookies();
});

afterEach(() => {
  vi.unstubAllGlobals();
  resetConsolePreferencesStoreForTests();
});

test("renders global default and machine override settings", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = getPath(input);
    if (path === "/settings/agents") {
      return jsonResponse({
        items: [{ agentType: "codex", displayName: "Codex" }],
      });
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
        machineOverride: null,
        usesGlobalDefault: true,
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect((await scope.findAllByRole("heading", { name: "Settings" })).length).toBeGreaterThan(0);
  await waitFor(() => {
    expect(scope.getByLabelText("Global Default TOML")).toHaveValue("model = \"gpt-5.4\"\n");
  });
  expect(await scope.findByText("Using global default")).toBeInTheDocument();
  expect(scope.getByText("Codex")).toBeInTheDocument();
  expect(scope.getByText("Machine 01")).toBeInTheDocument();
  expect(scope.getByLabelText("Gateway URL")).toBeInTheDocument();
});

test("saving global default sends put request", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);
    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
    }
    if (path === "/settings/agents/codex/global" && (!init?.method || init.method === "GET")) {
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
    if (path === "/settings/agents/codex/global" && init?.method === "PUT") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" },
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.change(await scope.findByLabelText("Global Default TOML"), {
    target: { value: "model = \"gpt-5.4\"\n" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save Global Default" }));

  await waitFor(() => {
    const putCall = fetchMock.mock.calls.find(
      ([input, init]) => getPath(input) === "/settings/agents/codex/global" && init?.method === "PUT",
    );
    expect(putCall).toBeTruthy();
  });
});

test("saving machine override and applying settings use the machine endpoint", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);
    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
    }
    if (path === "/settings/agents/codex/global") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" },
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex" && (!init?.method || init.method === "GET")) {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" },
        machineOverride: null,
        usesGlobalDefault: true,
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex" && init?.method === "PUT") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.2\"\n" },
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex/apply" && init?.method === "POST") {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        source: "machine",
        filePath: "/tmp/.codex/config.toml",
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.change(await scope.findByLabelText("Machine Override TOML"), {
    target: { value: "model = \"gpt-5.2\"\n" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save Machine Override" }));
  fireEvent.click(await scope.findByRole("button", { name: "Apply To Machine" }));

  await waitFor(() => {
    const overrideCall = fetchMock.mock.calls.find(
      ([input, init]) => getPath(input) === "/settings/machines/machine-01/agents/codex" && init?.method === "PUT",
    );
    const applyCall = fetchMock.mock.calls.find(
      ([input, init]) => getPath(input) === "/settings/machines/machine-01/agents/codex/apply" && init?.method === "POST",
    );
    expect(overrideCall).toBeTruthy();
    expect(applyCall).toBeTruthy();
  });
});

test("invalid toml blocks saving", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = getPath(input);
    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
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
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );
  const scope = await getMainScope();

  fireEvent.change(await scope.findByLabelText("Global Default TOML"), {
    target: { value: "model = [" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save Global Default" }));

  expect(await scope.findByText("Invalid TOML content.")).toBeInTheDocument();
});

test("shows a load error when settings bootstrap fails", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = getPath(input);
    if (path === "/settings/agents") {
      throw new Error("boom");
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect(await scope.findByText("Unable to load settings.")).toBeInTheDocument();
});

test("saving local gateway connection writes cookies before any remote settings fetch", async () => {
  clearGatewayCookies();

  const events: string[] = [];
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    events.push(getPath(input));
    expect(document.cookie).toContain("cag_gateway_url=http%3A%2F%2Flocalhost%3A3100");
    expect(document.cookie).toContain("cag_gateway_api_key=test-key");
    return jsonResponse({ items: [] });
  });
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect(fetchMock).not.toHaveBeenCalled();

  fireEvent.change(await scope.findByLabelText("Gateway URL"), {
    target: { value: "http://localhost:3100" },
  });
  fireEvent.change(scope.getByLabelText("API Key"), {
    target: { value: "test-key" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save Gateway Connection" }));

  await waitFor(() => {
    expect(document.cookie).toContain("cag_gateway_url=http%3A%2F%2Flocalhost%3A3100");
    expect(document.cookie).toContain("cag_gateway_api_key=test-key");
  });

  await waitFor(() => {
    expect(events.length).toBeGreaterThan(0);
  });
});

async function getMainScope() {
  const mains = await screen.findAllByRole("main");
  return within(mains[0]);
}

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
    },
  });
}
