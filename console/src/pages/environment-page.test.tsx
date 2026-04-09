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

test("clicking a skill action sends machineId", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});

  const fetchMock = vi.fn<
    (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  >(async (input, init) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path.endsWith("/environment/skills")) {
      return new Response(
        JSON.stringify({
          items: [
            {
              resourceId: "skill-1",
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

    if (path.endsWith("/environment/skills/skill-1/disable")) {
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
      "/environment/skills/skill-1/disable",
      expect.objectContaining({
        body: JSON.stringify({ machineId: "machine-9" }),
        method: "POST"
      }),
    );
  });
});
