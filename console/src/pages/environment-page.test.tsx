import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { EnvironmentPage } from "./environment-page";

afterEach(() => {
  vi.unstubAllGlobals();
});

test("renders fetched skills and plugins from environment endpoints", async () => {
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
