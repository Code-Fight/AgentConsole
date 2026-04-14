import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ConsoleHostRouter } from "../design-host/console-host-router";

const connectConsoleSocketMock = vi.fn();
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

vi.mock("../common/api/ws", () => ({
  connectConsoleSocket: (
    threadId: string | undefined,
    onMessage: (event: MessageEvent<string>) => void,
  ) => connectConsoleSocketMock(threadId, onMessage),
}));

afterEach(() => {
  connectConsoleSocketMock.mockReset();
  vi.unstubAllGlobals();
});

test("renders the active environment surface with gateway resources and enabled environment write actions", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/skills")) {
      return jsonResponse({
        items: [
          {
            resourceId: "skill-1",
            machineId: "machine-1",
            kind: "skill",
            displayName: "Debugger",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-08T13:00:03Z",
          },
        ],
      });
    }

    if (path.endsWith("/environment/mcps")) {
      return jsonResponse({
        items: [
          {
            resourceId: "github",
            machineId: "machine-1",
            kind: "mcp",
            displayName: "GitHub MCP",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-08T13:05:03Z",
          },
        ],
      });
    }

    if (path.endsWith("/environment/plugins")) {
      return jsonResponse({
        items: [
          {
            resourceId: "plugin-1",
            machineId: "machine-1",
            kind: "plugin",
            displayName: "Marketplace A",
            status: "enabled",
            restartRequired: true,
            lastObservedAt: "2026-04-08T13:10:00Z",
          },
        ],
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect((await scope.findAllByRole("heading", { name: "Environment" })).length).toBeGreaterThan(0);
  expect(await scope.findByText("Debugger")).toBeInTheDocument();
  expect(await scope.findByText("GitHub MCP")).toBeInTheDocument();
  expect(await scope.findByText("Marketplace A")).toBeInTheDocument();
  expect(scope.getAllByText("Skills").length).toBeGreaterThan(0);
  expect(scope.getAllByText("Plugins").length).toBeGreaterThan(0);
  expect(scope.getByRole("button", { name: "Sync catalog" })).toBeDisabled();
  expect(scope.getByRole("button", { name: "Add skill" })).toBeEnabled();
  expect(scope.getByRole("button", { name: "Add plugin record" })).toBeEnabled();

  const skillCard = scope.getByText("Debugger");
  const skillActions = within(skillCard.closest("article") ?? skillCard.parentElement ?? skillCard);
  expect(skillActions.getByRole("button", { name: "Delete skill" })).toBeEnabled();
});

test("clicking a skill action sends the path-based resource id and machineId", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const skillPath = "/tmp/project/.codex/skills/skill-1/SKILL.md";
  const encodedSkillPath = encodeURIComponent(skillPath);

  const fetchMock = vi.fn<
    (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  >(async (input, init) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/skills")) {
      return jsonResponse({
        items: [
          {
            resourceId: skillPath,
            machineId: "machine-9",
            kind: "skill",
            displayName: "Debugger",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-08T13:00:03Z",
          },
        ],
      });
    }

    if (path.endsWith("/environment/mcps") || path.endsWith("/environment/plugins")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith(`/environment/skills/${encodedSkillPath}/disable`)) {
      const body = typeof init?.body === "string" ? init.body : "";
      return new Response(body, {
        status: 200,
        headers: {
          "Content-Type": "application/json",
        },
      });
    }

    if (path.endsWith(`/environment/skills/${encodedSkillPath}`) && init?.method === "DELETE") {
      const body = typeof init?.body === "string" ? init.body : "";
      return new Response(body, {
        status: 200,
        headers: {
          "Content-Type": "application/json",
        },
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.click(await scope.findByRole("button", { name: "Disable" }));
  fireEvent.click(scope.getByRole("button", { name: "Delete skill" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      `/environment/skills/${encodedSkillPath}/disable`,
      expect.objectContaining({
        body: JSON.stringify({ machineId: "machine-9" }),
        method: "POST",
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      `/environment/skills/${encodedSkillPath}`,
      expect.objectContaining({
        body: JSON.stringify({ machineId: "machine-9" }),
        method: "DELETE",
      }),
    );
  });
});

test("submits skill scaffold create requests", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/skills") && init?.method === "POST") {
      return new Response(typeof init?.body === "string" ? init.body : "", {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (path.endsWith("/environment/skills") && (!init?.method || init.method === "GET")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/mcps") || path.endsWith("/environment/plugins")) {
      return jsonResponse({ items: [] });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.click(await scope.findByRole("button", { name: "Add skill" }));
  fireEvent.change(scope.getByLabelText("Machine ID"), {
    target: { value: "machine-22" },
  });
  fireEvent.change(scope.getByLabelText("Skill name"), {
    target: { value: "Debug Helper" },
  });
  fireEvent.change(scope.getByLabelText("Description"), {
    target: { value: "Describe what the skill does." },
  });
  fireEvent.click(scope.getByRole("button", { name: "Create skill" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/skills",
      expect.objectContaining({
        body: JSON.stringify({
          machineId: "machine-22",
          name: "Debug Helper",
          description: "Describe what the skill does.",
        }),
        method: "POST",
      }),
    );
  });
});

test("submits MCP config through the create form", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/plugins")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/mcps")) {
      if (init?.method === "POST") {
        return new Response(typeof init?.body === "string" ? init.body : "", {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }
      return jsonResponse({ items: [] });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.click(await scope.findByRole("button", { name: "Add MCP" }));
  fireEvent.change(scope.getByLabelText("Machine ID"), {
    target: { value: "machine-1" },
  });
  fireEvent.change(scope.getByLabelText("Server ID"), {
    target: { value: "github" },
  });
  fireEvent.change(scope.getByLabelText("Config JSON"), {
    target: { value: "{\"command\":\"npx\",\"args\":[\"-y\",\"@modelcontextprotocol/server-github\"]}" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save MCP" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/mcps",
      expect.objectContaining({
        body: JSON.stringify({
          machineId: "machine-1",
          resourceId: "github",
          config: {
            command: "npx",
            args: ["-y", "@modelcontextprotocol/server-github"],
          },
        }),
        method: "POST",
      }),
    );
  });
});

test("edits MCP config and issues enable, disable, and delete actions", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/plugins")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/mcps") && (!init?.method || init.method === "GET")) {
      return jsonResponse({
        items: [
          {
            resourceId: "github",
            machineId: "machine-1",
            kind: "mcp",
            displayName: "GitHub MCP",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-08T13:05:03Z",
            details: {
              config: {
                command: "npx",
                args: ["-y", "@modelcontextprotocol/server-github"],
              },
            },
          },
          {
            resourceId: "slack",
            machineId: "machine-1",
            kind: "mcp",
            displayName: "Slack MCP",
            status: "disabled",
            restartRequired: false,
            lastObservedAt: "2026-04-08T13:05:03Z",
            details: {
              config: {
                command: "node",
                args: ["index.js"],
              },
            },
          },
        ],
      });
    }

    if (path.endsWith("/environment/mcps") && init?.method === "POST") {
      return new Response(typeof init?.body === "string" ? init.body : "", {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (path.endsWith("/environment/mcps/github/disable")) {
      return new Response(null, { status: 200 });
    }

    if (path.endsWith("/environment/mcps/slack/enable")) {
      return new Response(null, { status: 200 });
    }

    if (path.endsWith("/environment/mcps/github") && init?.method === "DELETE") {
      return new Response(null, { status: 204 });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  const githubCard = await scope.findByText("GitHub MCP");
  const githubActions = within(githubCard.closest("article") ?? githubCard.parentElement ?? githubCard);
  fireEvent.click(githubActions.getByRole("button", { name: "Edit" }));
  fireEvent.change(scope.getByLabelText("Config JSON"), {
    target: { value: "{\"command\":\"npx\",\"args\":[\"-y\",\"@modelcontextprotocol/server-github\",\"--verbose\"]}" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Save MCP" }));

  const slackCard = await scope.findByText("Slack MCP");
  const slackActions = within(slackCard.closest("article") ?? slackCard.parentElement ?? slackCard);
  fireEvent.click(slackActions.getByRole("button", { name: "Enable" }));
  fireEvent.click(githubActions.getByRole("button", { name: "Disable" }));
  fireEvent.click(githubActions.getByRole("button", { name: "Delete" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/mcps",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          machineId: "machine-1",
          resourceId: "github",
          config: {
            command: "npx",
            args: ["-y", "@modelcontextprotocol/server-github", "--verbose"],
          },
        }),
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/mcps/slack/enable",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ machineId: "machine-1" }),
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/mcps/github/disable",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ machineId: "machine-1" }),
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/mcps/github",
      expect.objectContaining({
        method: "DELETE",
        body: JSON.stringify({ machineId: "machine-1" }),
      }),
    );
  });
});

test("does not render uninstall for plugins that are not installed", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/mcps")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/plugins")) {
      return jsonResponse({
        items: [
          {
            resourceId: "plugin-unknown",
            machineId: "machine-1",
            kind: "plugin",
            displayName: "Marketplace B",
            status: "unknown",
            restartRequired: false,
            lastObservedAt: "2026-04-08T13:10:00Z",
          },
        ],
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect(await scope.findByText("Marketplace B")).toBeInTheDocument();
  expect(scope.queryByRole("button", { name: "Uninstall" })).not.toBeInTheDocument();
});

test("renders plugin detail contents and install action for marketplace plugins", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/mcps")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/plugins")) {
      return jsonResponse({
        items: [
          {
            resourceId: "gmail@openai-curated",
            machineId: "machine-1",
            kind: "plugin",
            displayName: "Gmail",
            status: "unknown",
            restartRequired: false,
            lastObservedAt: "2026-04-08T13:10:00Z",
            details: {
              description: "Read and draft Gmail messages",
              marketplaceName: "OpenAI Curated",
              marketplacePath: "/tmp/codex/marketplace",
              bundledSkills: ["gmail_triage"],
              bundledMcpServers: ["gmail"],
            },
          },
        ],
      });
    }

    if (path.endsWith("/environment/plugins/gmail%40openai-curated/install")) {
      return new Response(typeof init?.body === "string" ? init.body : "", {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect(await scope.findByText("Gmail")).toBeInTheDocument();
  fireEvent.click(scope.getByRole("button", { name: "View details" }));
  expect(await scope.findByText("Read and draft Gmail messages")).toBeInTheDocument();
  expect(scope.getByText("gmail_triage")).toBeInTheDocument();
  expect(scope.getByText("gmail")).toBeInTheDocument();

  fireEvent.click(scope.getByRole("button", { name: "Install" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/plugins/gmail%40openai-curated/install",
      expect.objectContaining({
        body: JSON.stringify({
          machineId: "machine-1",
          pluginId: "gmail@openai-curated",
          pluginName: "Gmail",
          marketplacePath: "/tmp/codex/marketplace",
        }),
        method: "POST",
      }),
    );
  });
});

test("plugin disable and uninstall send the machine id", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/mcps")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/plugins") && (!init?.method || init.method === "GET")) {
      return jsonResponse({
        items: [
          {
            resourceId: "plugin-installed",
            machineId: "machine-1",
            kind: "plugin",
            displayName: "Local Marketplace",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-08T13:10:00Z",
          },
        ],
      });
    }

    if (path.endsWith("/environment/plugins/plugin-installed/disable")) {
      return new Response(null, { status: 200 });
    }

    if (path.endsWith("/environment/plugins/plugin-installed") && init?.method === "DELETE") {
      return new Response(null, { status: 204 });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  const pluginCard = await scope.findByText("Local Marketplace");
  const pluginActions = within(pluginCard.closest("article") ?? pluginCard.parentElement ?? pluginCard);
  fireEvent.click(pluginActions.getByRole("button", { name: "Disable" }));
  fireEvent.click(pluginActions.getByRole("button", { name: "Uninstall" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/plugins/plugin-installed/disable",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ machineId: "machine-1" }),
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/plugins/plugin-installed",
      expect.objectContaining({
        method: "DELETE",
        body: JSON.stringify({ machineId: "machine-1" }),
      }),
    );
  });
});

test("submits plugin install requests for add plugin record", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    const bootstrap = bootstrapResponse(path);
    if (bootstrap) {
      return bootstrap;
    }

    if (path.endsWith("/environment/plugins") && (!init?.method || init.method === "GET")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/plugins/install") && init?.method === "POST") {
      return new Response(typeof init?.body === "string" ? init.body : "", {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/mcps")) {
      return jsonResponse({ items: [] });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter initialEntries={["/environment"]}>
      <ConsoleHostRouter />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.click(await scope.findByRole("button", { name: "Add plugin record" }));
  fireEvent.change(scope.getByLabelText("Machine ID"), {
    target: { value: "machine-3" },
  });
  fireEvent.change(scope.getByLabelText("Plugin name"), {
    target: { value: "calendar" },
  });
  fireEvent.change(scope.getByLabelText("Marketplace path"), {
    target: { value: "/tmp/codex/marketplace" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Install plugin" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/plugins/install",
      expect.objectContaining({
        body: JSON.stringify({
          machineId: "machine-3",
          pluginId: "calendar",
          pluginName: "calendar",
          marketplacePath: "/tmp/codex/marketplace",
        }),
        method: "POST",
      }),
    );
  });
});

async function getMainScope() {
  const mains = await screen.findAllByRole("main");
  return within(mains[0]);
}

function bootstrapResponse(path: string): Response | null {
  if (path === "/settings/console") {
    return jsonResponse({ preferences: null });
  }
  if (path === "/threads") {
    return jsonResponse({ items: [] });
  }
  if (path === "/machines") {
    return jsonResponse({ items: [] });
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
