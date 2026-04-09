import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

const connectConsoleSocketMock = vi.fn();

vi.mock("../common/api/ws", () => ({
  connectConsoleSocket: (
    threadId: string | undefined,
    onMessage: (event: MessageEvent<string>) => void,
  ) => connectConsoleSocketMock(threadId, onMessage)
}));

import { EnvironmentPage } from "./environment-page";

afterEach(() => {
  connectConsoleSocketMock.mockReset();
  vi.unstubAllGlobals();
});

test("renders fetched skills and plugins from environment endpoints", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path.endsWith("/environment/skills")) {
      return new Response(
        JSON.stringify({
          items: [
            {
              resourceId: "skill-1",
              machineId: "machine-1",
              kind: "skill",
              displayName: "Debugger",
              status: "enabled",
              restartRequired: false,
              lastObservedAt: "2026-04-08T13:00:03Z"
            }
          ]
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    if (path.endsWith("/environment/mcps")) {
      return new Response(
        JSON.stringify({
          items: []
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    if (path.endsWith("/environment/plugins")) {
      return new Response(
        JSON.stringify({
          items: [
            {
              resourceId: "plugin-1",
              machineId: "machine-1",
              kind: "plugin",
              displayName: "Marketplace A",
              status: "enabled",
              restartRequired: true,
              lastObservedAt: "2026-04-08T13:10:00Z"
            }
          ]
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(<EnvironmentPage />);

  expect(await screen.findByText("Debugger")).toBeInTheDocument();
  expect(await screen.findByText("Marketplace A")).toBeInTheDocument();
  expect(screen.getByText("Skills")).toBeInTheDocument();
  expect(screen.getByText("Plugins")).toBeInTheDocument();
});

test("clicking a skill action sends the path-based resource id and machineId", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const skillPath = "/tmp/project/.codex/skills/skill-1/SKILL.md";
  const encodedSkillPath = encodeURIComponent(skillPath);

  const fetchMock = vi.fn<
    (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  >(async (input, init) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path.endsWith("/environment/skills")) {
      return new Response(
        JSON.stringify({
          items: [
            {
              resourceId: skillPath,
              machineId: "machine-9",
              kind: "skill",
              displayName: "Debugger",
              status: "enabled",
              restartRequired: false,
              lastObservedAt: "2026-04-08T13:00:03Z"
            }
          ]
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    if (path.endsWith("/environment/mcps") || path.endsWith("/environment/plugins")) {
      return new Response(
        JSON.stringify({ items: [] }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    if (path.endsWith(`/environment/skills/${encodedSkillPath}/disable`)) {
      const body = typeof init?.body === "string" ? init.body : "";
      return new Response(body, {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<EnvironmentPage />);

  fireEvent.click(await screen.findByRole("button", { name: "Disable" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      `/environment/skills/${encodedSkillPath}/disable`,
      expect.objectContaining({
        body: JSON.stringify({ machineId: "machine-9" }),
        method: "POST"
      }),
    );
  });
});

test("does not render uninstall for plugins that are not installed", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/mcps")) {
      return new Response(
        JSON.stringify({ items: [] }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    if (path.endsWith("/environment/plugins")) {
      return new Response(
        JSON.stringify({
          items: [
            {
              resourceId: "plugin-unknown",
              machineId: "machine-1",
              kind: "plugin",
              displayName: "Marketplace B",
              status: "unknown",
              restartRequired: false,
              lastObservedAt: "2026-04-08T13:10:00Z"
            }
          ]
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(<EnvironmentPage />);

  expect(await screen.findByText("Marketplace B")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Uninstall" })).not.toBeInTheDocument();
});

test("renders plugin detail contents and install action for marketplace plugins", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/mcps")) {
      return new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      });
    }

    if (path.endsWith("/environment/plugins")) {
      return new Response(
        JSON.stringify({
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
                bundledMcpServers: ["gmail"]
              }
            }
          ]
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" }
        },
      );
    }

    if (path.endsWith("/environment/plugins/gmail%40openai-curated/install")) {
      return new Response(typeof init?.body === "string" ? init.body : "", {
        status: 200,
        headers: { "Content-Type": "application/json" }
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(<EnvironmentPage />);

  expect(await screen.findByText("Gmail")).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "View details" }));
  expect(await screen.findByText("Read and draft Gmail messages")).toBeInTheDocument();
  expect(screen.getByText("gmail_triage")).toBeInTheDocument();
  expect(screen.getByText("gmail")).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Install" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/plugins/gmail%40openai-curated/install",
      expect.objectContaining({
        body: JSON.stringify({ machineId: "machine-1" }),
        method: "POST"
      }),
    );
  });
});

test("submits MCP config through the create form", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path.endsWith("/environment/skills") || path.endsWith("/environment/plugins")) {
      return new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      });
    }

    if (path.endsWith("/environment/mcps")) {
      if (init?.method === "POST") {
        return new Response(typeof init?.body === "string" ? init.body : "", {
          status: 200,
          headers: { "Content-Type": "application/json" }
        });
      }
      return new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });

  vi.stubGlobal("fetch", fetchMock);

  render(<EnvironmentPage />);

  fireEvent.click(await screen.findByRole("button", { name: "Add MCP" }));
  fireEvent.change(screen.getByLabelText("Machine ID"), {
    target: { value: "machine-1" }
  });
  fireEvent.change(screen.getByLabelText("Server ID"), {
    target: { value: "github" }
  });
  fireEvent.change(screen.getByLabelText("Config JSON"), {
    target: { value: "{\"command\":\"npx\",\"args\":[\"-y\",\"@modelcontextprotocol/server-github\"]}" }
  });
  fireEvent.click(screen.getByRole("button", { name: "Save MCP" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/environment/mcps",
      expect.objectContaining({
        body: JSON.stringify({
          machineId: "machine-1",
          resourceId: "github",
          config: {
            command: "npx",
            args: ["-y", "@modelcontextprotocol/server-github"]
          }
        }),
        method: "POST"
      }),
    );
  });
});
