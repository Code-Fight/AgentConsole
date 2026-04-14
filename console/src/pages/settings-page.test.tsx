import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ConsoleHostRouter } from "../design-host/console-host-router";
import { resetConsolePreferencesStoreForTests } from "../gateway/use-console-preferences";

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

afterEach(() => {
  vi.unstubAllGlobals();
  resetConsolePreferencesStoreForTests();
});

test("renders global default and machine override settings", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

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
  expect(scope.getByRole("button", { name: "Edit gateway endpoint" })).toBeDisabled();
});

test("saving global default sends put request", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

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
    expect(fetchMock).toHaveBeenCalledWith(
      "/settings/agents/codex/global",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ content: "model = \"gpt-5.4\"\n" }),
      }),
    );
  });
});

test("saving machine override and applying settings use the machine endpoint", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

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
    expect(fetchMock).toHaveBeenCalledWith(
      "/settings/machines/machine-01/agents/codex",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ content: "model = \"gpt-5.2\"\n" }),
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/settings/machines/machine-01/agents/codex/apply",
      expect.objectContaining({
        method: "POST",
      }),
    );
  });
});

test("invalid toml blocks saving", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }
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
  expect(await scope.findByText("Using global default")).toBeInTheDocument();

  fireEvent.change(await scope.findByLabelText("Global Default TOML"), {
    target: { value: "model = [" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save Global Default" }));

  expect(await scope.findByText("Invalid TOML content.")).toBeInTheDocument();
  expect(fetchMock).not.toHaveBeenCalledWith(
    "/settings/agents/codex/global",
    expect.objectContaining({ method: "PUT" }),
  );
});

test("empty toml blocks saving", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }
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
  expect(await scope.findByText("Using global default")).toBeInTheDocument();

  fireEvent.change(await scope.findByLabelText("Global Default TOML"), {
    target: { value: "   " },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save Global Default" }));

  expect(await scope.findByText("Invalid TOML content.")).toBeInTheDocument();
});

test("deleting machine override falls back to global default", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }
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
        machineOverride: { agentType: "codex", format: "toml", content: "model = \"gpt-5.2\"\n" },
        usesGlobalDefault: false,
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex" && init?.method === "DELETE") {
      return new Response(null, { status: 204 });
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
  fireEvent.click(await scope.findByRole("button", { name: "Delete Machine Override" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/settings/machines/machine-01/agents/codex",
      expect.objectContaining({
        method: "DELETE",
      }),
    );
  });
  expect(await scope.findByText("Machine override deleted.")).toBeInTheDocument();
  expect(scope.getByText("Using global default")).toBeInTheDocument();
});

test("renders console preferences from gateway settings", async () => {
  const storedPreferences = {
    consoleUrl: "http://localhost:3100",
    apiKey: "test-key",
    profile: "dev",
    safetyPolicy: "strict",
    lastThreadId: "",
  };
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    if (path === "/settings/console") {
      return jsonResponse({ preferences: storedPreferences });
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
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect(await scope.findByLabelText("Console URL")).toHaveValue("http://localhost:3100");
  const apiKeyInput = scope.getByLabelText("API Key");
  expect(apiKeyInput).toHaveValue("test-key");
  expect(apiKeyInput).toHaveAttribute("type", "password");
  expect(scope.getByLabelText("Console Profile")).toHaveValue("dev");
  expect(scope.getByLabelText("Safety Policy")).toHaveValue("strict");
});

test("saving console preferences uses the settings endpoint", async () => {
  let storedPreferences = {
    consoleUrl: "",
    apiKey: "",
    profile: "",
    safetyPolicy: "",
    lastThreadId: "",
  };
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    if (path === "/settings/console" && (!init?.method || init.method === "GET")) {
      return jsonResponse({ preferences: storedPreferences });
    }
    if (path === "/settings/console" && init?.method === "PUT") {
      const body = init.body ? JSON.parse(init.body.toString()) : null;
      storedPreferences = body?.preferences ?? storedPreferences;
      return jsonResponse({ preferences: storedPreferences });
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
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.change(await scope.findByLabelText("Console URL"), {
    target: { value: "http://localhost:3100" },
  });
  fireEvent.change(scope.getByLabelText("API Key"), {
    target: { value: "test-key" },
  });
  fireEvent.change(scope.getByLabelText("Console Profile"), {
    target: { value: "dev" },
  });
  fireEvent.change(scope.getByLabelText("Safety Policy"), {
    target: { value: "strict" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save Console Settings" }));

  await waitFor(() => {
    const putCall = fetchMock.mock.calls.find(
      (call) =>
        call[0] === "/settings/console" &&
        call[1] &&
        typeof call[1] === "object" &&
        (call[1] as RequestInit).method === "PUT",
    );
    expect(putCall).toBeTruthy();
    const body = JSON.parse((putCall?.[1] as RequestInit).body as string);
    expect(body).toEqual({
      preferences: {
        consoleUrl: "http://localhost:3100",
        apiKey: "test-key",
        profile: "dev",
        safetyPolicy: "strict",
        lastThreadId: "",
      },
    });
  });
});

test("reloads console preferences from gateway settings", async () => {
  let storedPreferences = {
    consoleUrl: "",
    apiKey: "",
    profile: "",
    safetyPolicy: "",
    lastThreadId: "",
  };
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    if (path === "/settings/console" && (!init?.method || init.method === "GET")) {
      return jsonResponse({ preferences: storedPreferences });
    }
    if (path === "/settings/console" && init?.method === "PUT") {
      const body = init.body ? JSON.parse(init.body.toString()) : null;
      storedPreferences = body?.preferences ?? storedPreferences;
      return jsonResponse({ preferences: storedPreferences });
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
  vi.stubGlobal("fetch", fetchMock);

  const { unmount } = render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  let scope = await getMainScope();
  fireEvent.change(await scope.findByLabelText("Console URL"), {
    target: { value: "http://localhost:3100" },
  });
  fireEvent.change(scope.getByLabelText("API Key"), {
    target: { value: "test-key" },
  });
  fireEvent.change(scope.getByLabelText("Console Profile"), {
    target: { value: "dev" },
  });
  fireEvent.change(scope.getByLabelText("Safety Policy"), {
    target: { value: "strict" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save Console Settings" }));

  await waitFor(() => {
    expect(storedPreferences.consoleUrl).toBe("http://localhost:3100");
  });

  unmount();

  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  scope = await getMainScope();
  expect(await scope.findByLabelText("Console URL")).toHaveValue("http://localhost:3100");
  expect(scope.getByLabelText("API Key")).toHaveValue("test-key");
  expect(scope.getByLabelText("Console Profile")).toHaveValue("dev");
  expect(scope.getByLabelText("Safety Policy")).toHaveValue("strict");
});

test("shows load error when settings bootstrap fails", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }
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

test("shows machine settings error when machine fetch fails", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }
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
  expect(await scope.findByText("Unable to load machine settings.")).toBeInTheDocument();
});

async function getMainScope() {
  const mains = await screen.findAllByRole("main");
  return within(mains[0]);
}

function bootstrapResponse(path: string): Response | null {
  if (path === "/settings/console") {
    return jsonResponse({ preferences: null });
  }
  return null;
}

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
    },
  });
}
