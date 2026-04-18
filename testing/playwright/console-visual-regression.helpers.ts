import type { Page, Route } from "@playwright/test";

export interface VisualScenario {
  name: string;
  path: string;
  snapshot: string;
  ready: { kind: "text" | "label" | "placeholder"; value: string };
}

export const visualScenarios: VisualScenario[] = [
  {
    name: "Thread Hub",
    path: "/",
    snapshot: "thread-hub",
    ready: { kind: "text", value: "Gateway Thread 1" },
  },
  {
    name: "Thread Workspace",
    path: "/threads/thread-1",
    snapshot: "thread-workspace",
    ready: { kind: "text", value: "Visual baseline seeded." },
  },
  {
    name: "Settings",
    path: "/settings",
    snapshot: "settings",
    ready: { kind: "text", value: "Using global default" },
  },
  {
    name: "Environment",
    path: "/environment",
    snapshot: "environment",
    ready: { kind: "text", value: "Code Review" },
  },
  {
    name: "Machines",
    path: "/machines",
    snapshot: "machines",
    ready: { kind: "text", value: "Machine One" },
  },
];

const GATEWAY_URL = "http://127.0.0.1:4174";
const GATEWAY_API_KEY = "visual-test-key";

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
  return {
    status: 200,
    contentType: "application/json",
    body: JSON.stringify(payload),
  };
}

function assertAuthHeader(route: Route): void {
  const authorization = route.request().headers()["authorization"];
  if (authorization !== `Bearer ${GATEWAY_API_KEY}`) {
    throw new Error(`unexpected authorization header: ${authorization ?? "(missing)"}`);
  }
}

function shouldFulfillApiRequest(route: Route): boolean {
  const request = route.request();
  const requestType = request.resourceType();
  return requestType === "fetch" || requestType === "xhr";
}

export async function primeConsoleVisualRoutes(page: Page): Promise<void> {
  await page.context().addCookies([
    {
      name: "cag_gateway_url",
      value: GATEWAY_URL,
      url: GATEWAY_URL,
    },
    {
      name: "cag_gateway_api_key",
      value: GATEWAY_API_KEY,
      url: GATEWAY_URL,
    },
  ]);

  await page.route("**/capabilities", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(jsonResponse(capabilitySnapshot));
  });

  await page.route("**/settings/console", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        preferences: {
          profile: "",
          safetyPolicy: "",
          lastThreadId: "",
          threadTitles: {},
        },
      }),
    );
  });

  await page.route("**/settings/agents", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        items: [
          {
            agentType: "codex",
            displayName: "Codex",
          },
        ],
      }),
    );
  });

  await page.route("**/settings/agents/codex/global", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        document: {
          agentType: "codex",
          format: "toml",
          content: "model = \"gpt-5.4\"\napproval_policy = \"never\"\n",
        },
      }),
    );
  });

  await page.route("**/settings/machines/machine-1/agents/codex", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        machineId: "machine-1",
        agentType: "codex",
        machineOverride: null,
        usesGlobalDefault: true,
      }),
    );
  });

  await page.route("**/threads", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        items: [
          {
            threadId: "thread-1",
            machineId: "machine-1",
            status: "active",
            title: "Gateway Thread 1",
          },
        ],
      }),
    );
  });

  await page.route("**/threads/thread-1", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "active",
          title: "Gateway Thread 1",
        },
        activeTurnId: null,
        pendingApprovals: [],
        messages: [
          {
            id: "seed-agent-message",
            kind: "agent",
            text: "Visual baseline seeded.",
            turnId: "turn-seeded",
          },
        ],
      }),
    );
  });

  await page.route("**/machines", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        items: [
          {
            id: "machine-1",
            name: "Machine One",
            status: "online",
            runtimeStatus: "running",
          },
        ],
      }),
    );
  });

  await page.route("**/machines/machine-1", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        machine: {
          id: "machine-1",
          name: "Machine One",
          status: "online",
          runtimeStatus: "running",
        },
      }),
    );
  });

  await page.route("**/environment/skills", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        items: [
          {
            resourceId: "code-review",
            machineId: "machine-1",
            kind: "skill",
            displayName: "Code Review",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-18T00:00:00Z",
            details: {
              description: "Static analysis and review workflow",
            },
          },
        ],
      }),
    );
  });

  await page.route("**/environment/mcps", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        items: [
          {
            resourceId: "figma",
            machineId: "machine-1",
            kind: "mcp",
            displayName: "Figma",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-18T00:00:00Z",
            details: {
              config: {
                endpoint: "https://mcp.figma.com",
              },
            },
          },
        ],
      }),
    );
  });

  await page.route("**/environment/plugins", async (route) => {
    if (!shouldFulfillApiRequest(route)) {
      await route.fallback();
      return;
    }
    assertAuthHeader(route);
    await route.fulfill(
      jsonResponse({
        items: [
          {
            resourceId: "slack",
            machineId: "machine-1",
            kind: "plugin",
            displayName: "Slack",
            status: "enabled",
            restartRequired: false,
            lastObservedAt: "2026-04-18T00:00:00Z",
            details: {
              source: "marketplace",
            },
          },
        ],
      }),
    );
  });
}
