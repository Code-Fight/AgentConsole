import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  clearGatewayConnectionCookies,
  saveGatewayConnectionToCookies,
} from "../../../gateway/gateway-connection-store";
import { EnvironmentPage } from "./environment-page";

function getPath(input: RequestInfo | URL | string): string {
  const raw = typeof input === "string" ? input : input.toString();
  if (raw.startsWith("http://") || raw.startsWith("https://")) {
    return new URL(raw).pathname;
  }
  return raw;
}

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

async function getMainScope() {
  const mains = await screen.findAllByRole("main");
  return within(mains[0]);
}

const connectConsoleSocketMock = vi.fn();
const capabilitySnapshot = vi.hoisted(() => ({
  environmentSyncCatalog: true,
  environmentRestartBridge: true,
  environmentOpenMarketplace: true,
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
const useCapabilitiesMock = vi.hoisted(() => vi.fn(() => capabilitySnapshot));

vi.mock("../../../gateway/capabilities", () => ({
  useCapabilities: (enabled?: boolean) => useCapabilitiesMock(enabled),
  supportsCapability: (capability: string) =>
    Boolean(capabilitySnapshot[capability as keyof typeof capabilitySnapshot]),
}));

vi.mock("../../../common/api/ws", () => ({
  connectConsoleSocket: (
    threadId: string | undefined,
    onMessage: (event: MessageEvent<string>) => void,
  ) => connectConsoleSocketMock(threadId, onMessage),
}));

beforeEach(() => {
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "environment-feature-test-key",
  });
});

afterEach(() => {
  connectConsoleSocketMock.mockReset();
  useCapabilitiesMock.mockReset();
  clearGatewayConnectionCookies();
  vi.unstubAllGlobals();
  capabilitySnapshot.environmentMutateResources = true;
});

test("preserves environment shell chrome, machine-aware skill dialog, and sync action", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path.endsWith("/machines")) {
      return jsonResponse({
        items: [
          {
            id: "machine-1",
            name: "Machine One",
            status: "reconnecting",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-1",
                agentType: "codex",
                displayName: "Design Agent",
                name: "Design Agent",
                model: "gpt-5.4",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path.endsWith("/environment/skills")) {
      if (init?.method === "POST") {
        return jsonResponse({});
      }
      return jsonResponse({
        items: [
          {
            resourceId: "skill-1",
            machineId: "machine-1",
            agentId: "agent-1",
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
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/plugins")) {
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/sync") && init?.method === "POST") {
      return jsonResponse({ targetedMachines: 1 });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter>
      <EnvironmentPage />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect(screen.getAllByRole("button", { name: "返回线程" })).toHaveLength(2);
  expect(screen.getByTitle("返回线程")).toBeInTheDocument();
  expect(await scope.findByText("Debugger")).toBeInTheDocument();
  expect(await scope.findByText("Machine One (重连中)")).toBeInTheDocument();
  expect(await scope.findByText("Design Agent (gpt-5.4)")).toBeInTheDocument();

  fireEvent.click(scope.getByRole("button", { name: "Add skill" }));
  expect(scope.getByLabelText("Skill name")).toBeInTheDocument();
  expect(scope.getByLabelText("目标 Agent")).toBeInTheDocument();
  expect(screen.getByRole("option", { name: "Design Agent (gpt-5.4)" })).toBeInTheDocument();

  fireEvent.click(scope.getByRole("button", { name: "Sync catalog" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/environment/sync"),
      expect.objectContaining({ method: "POST" }),
    );
  });
});

test("plugin add remains available and agent-aware without mutateResources capability", async () => {
  capabilitySnapshot.environmentMutateResources = false;
  connectConsoleSocketMock.mockReturnValue(() => {});
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = getPath(input);

      if (path.endsWith("/machines")) {
        return jsonResponse({
          items: [
            {
              id: "machine-1",
              name: "Machine One",
              status: "unknown",
              runtimeStatus: "running",
              agents: [
                {
                  agentId: "agent-1",
                  agentType: "codex",
                  displayName: "Design Agent",
                  name: "Design Agent",
                  model: "gpt-5.4",
                  status: "running",
                },
              ],
            },
          ],
        });
      }

      if (
        path.endsWith("/environment/skills") ||
        path.endsWith("/environment/mcps") ||
        path.endsWith("/environment/plugins")
      ) {
        return jsonResponse({ items: [] });
      }

      throw new Error(`Unexpected request: ${path}`);
    }),
  );

  render(
    <MemoryRouter>
      <EnvironmentPage />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect(await scope.findByRole("button", { name: "Add plugin record" })).toBeEnabled();

  fireEvent.click(scope.getByRole("button", { name: "Add plugin record" }));
  expect(scope.getByLabelText("Plugin ID")).toBeInTheDocument();
  fireEvent.change(scope.getByLabelText("Machine ID"), {
    target: { value: "machine-1" },
  });
  expect(scope.getByLabelText("目标 Agent")).toBeInTheDocument();
  expect(scope.getByRole("option", { name: "Design Agent (gpt-5.4)" })).toBeInTheDocument();
  expect(scope.getByRole("button", { name: "Install plugin" })).toBeEnabled();
});

test("falls back to agentType in labels and refreshes machine data on machine.updated", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  let machineFetchCount = 0;
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = getPath(input);

    if (path.endsWith("/machines")) {
      machineFetchCount += 1;
      if (machineFetchCount === 1) {
        return jsonResponse({
          items: [
            {
              id: "machine-2",
              name: "Machine Two",
              status: "unknown",
              runtimeStatus: "running",
              agents: [
                {
                  agentId: "agent-2",
                  agentType: "codex",
                  displayName: "Review Agent",
                  name: "Review Agent",
                  status: "running",
                },
              ],
            },
          ],
        });
      }

      return jsonResponse({
        items: [
          {
            id: "machine-2",
            name: "Machine Two",
            status: "reconnecting",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-2",
                agentType: "codex",
                displayName: "Review Agent",
                name: "Review Agent",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path.endsWith("/environment/skills")) {
      return jsonResponse({
        items: [
          {
            resourceId: "skill-2",
            machineId: "machine-2",
            agentId: "agent-2",
            kind: "skill",
            displayName: "Linter",
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

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter>
      <EnvironmentPage />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  expect(await scope.findByText("Machine Two (未知)")).toBeInTheDocument();
  expect((await scope.findAllByText("Review Agent (codex)")).length).toBeGreaterThan(0);

  fireEvent.click(scope.getByRole("button", { name: "Add skill" }));
  expect(scope.getByRole("option", { name: "Review Agent (codex)" })).toBeInTheDocument();

  const socketMessage = connectConsoleSocketMock.mock.calls[0]?.[1] as
    | ((event: MessageEvent<string>) => void)
    | undefined;
  expect(socketMessage).toBeDefined();

  socketMessage?.(
    new MessageEvent("message", {
      data: JSON.stringify({
        version: "1",
        category: "event",
        name: "machine.updated",
        timestamp: "2026-04-19T00:00:00Z",
        payload: {
          machine: {
            id: "machine-2",
          },
        },
      }),
    }),
  );

  expect(await scope.findByText("Machine Two (重连中)")).toBeInTheDocument();
});

test("direct environment visits proactively load capabilities when enabled", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = getPath(input);

      if (path.endsWith("/machines")) {
        return jsonResponse({ items: [] });
      }

      if (
        path.endsWith("/environment/skills") ||
        path.endsWith("/environment/mcps") ||
        path.endsWith("/environment/plugins")
      ) {
        return jsonResponse({ items: [] });
      }

      throw new Error(`Unexpected request: ${path}`);
    }),
  );

  render(
    <MemoryRouter>
      <EnvironmentPage />
    </MemoryRouter>,
  );

  await getMainScope();

  expect(useCapabilitiesMock).toHaveBeenCalledWith(true);
});

test("submits skill create requests with the selected agentId", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path.endsWith("/machines")) {
      return jsonResponse({
        items: [
          {
            id: "machine-10",
            name: "Machine Ten",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-10a",
                agentType: "codex",
                displayName: "Planner",
                name: "Planner",
                model: "gpt-5.4",
                status: "running",
              },
              {
                agentId: "agent-10b",
                agentType: "codex",
                displayName: "Reviewer",
                name: "Reviewer",
                model: "gpt-5.4-mini",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path.endsWith("/environment/skills")) {
      if (init?.method === "POST") {
        return jsonResponse({});
      }
      return jsonResponse({ items: [] });
    }

    if (path.endsWith("/environment/mcps") || path.endsWith("/environment/plugins")) {
      return jsonResponse({ items: [] });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter>
      <EnvironmentPage />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.click(await scope.findByRole("button", { name: "Add skill" }));
  fireEvent.change(scope.getByLabelText("Machine ID"), {
    target: { value: "machine-10" },
  });
  fireEvent.change(scope.getByLabelText("目标 Agent"), {
    target: { value: "agent-10b" },
  });
  await waitFor(() => {
    expect((scope.getByLabelText("目标 Agent") as HTMLSelectElement).value).toBe("agent-10b");
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
      expect.stringContaining("/environment/skills"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          machineId: "machine-10",
          agentId: "agent-10b",
          name: "Debug Helper",
          description: "Describe what the skill does.",
        }),
      }),
    );
  });
});

test("changing machine resets the visible and submitted agent to the new machine default", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path.endsWith("/machines")) {
      return jsonResponse({
        items: [
          {
            id: "machine-a",
            name: "Machine A",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-a1",
                agentType: "codex",
                displayName: "Agent A1",
                name: "Agent A1",
                model: "gpt-5.4",
                status: "running",
              },
              {
                agentId: "agent-a2",
                agentType: "codex",
                displayName: "Agent A2",
                name: "Agent A2",
                model: "gpt-5.4-mini",
                status: "running",
              },
            ],
          },
          {
            id: "machine-b",
            name: "Machine B",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-b1",
                agentType: "codex",
                displayName: "Agent B1",
                name: "Agent B1",
                model: "gpt-5.4",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path.endsWith("/environment/skills")) {
      if (init?.method === "POST") {
        return jsonResponse({});
      }
      return jsonResponse({
        items: [
          {
            resourceId: "skill-a",
            machineId: "machine-a",
            agentId: "agent-a1",
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

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter>
      <EnvironmentPage />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.click(await scope.findByRole("button", { name: "Add skill" }));
  fireEvent.change(scope.getByLabelText("目标 Agent"), {
    target: { value: "agent-a2" },
  });
  fireEvent.change(scope.getByLabelText("Machine ID"), {
    target: { value: "machine-b" },
  });

  await waitFor(() => {
    expect((scope.getByLabelText("目标 Agent") as HTMLSelectElement).value).toBe("agent-b1");
  });

  fireEvent.change(scope.getByLabelText("Skill name"), {
    target: { value: "Fresh Skill" },
  });
  fireEvent.change(scope.getByLabelText("Description"), {
    target: { value: "After machine switch" },
  });
  fireEvent.click(scope.getByRole("button", { name: "Create skill" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/environment/skills"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          machineId: "machine-b",
          agentId: "agent-b1",
          name: "Fresh Skill",
          description: "After machine switch",
        }),
      }),
    );
  });
});

test("resource mutations include the resource agentId", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = getPath(input);

    if (path.endsWith("/machines")) {
      return jsonResponse({
        items: [
          {
            id: "machine-20",
            name: "Machine Twenty",
            status: "online",
            runtimeStatus: "running",
            agents: [
              {
                agentId: "agent-20a",
                agentType: "codex",
                displayName: "Worker",
                name: "Worker",
                model: "gpt-5.4",
                status: "running",
              },
            ],
          },
        ],
      });
    }

    if (path.endsWith("/environment/skills")) {
      return jsonResponse({
        items: [
          {
            resourceId: "skill-20",
            machineId: "machine-20",
            agentId: "agent-20a",
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

    if (path.endsWith("/environment/skills/skill-20/disable") && init?.method === "POST") {
      return jsonResponse({});
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(
    <MemoryRouter>
      <EnvironmentPage />
    </MemoryRouter>,
  );

  const scope = await getMainScope();
  fireEvent.click(await scope.findByRole("button", { name: "Disable" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/environment/skills/skill-20/disable"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          machineId: "machine-20",
          agentId: "agent-20a",
        }),
      }),
    );
  });
});
