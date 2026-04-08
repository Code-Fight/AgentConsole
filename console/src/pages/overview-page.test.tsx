import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { OverviewPage } from "./overview-page";

afterEach(() => {
  vi.unstubAllGlobals();
});

test("renders derived machine counts from the machines response", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () =>
      new Response(
        JSON.stringify({
          items: [
            { id: "machine-1", name: "Alpha", status: "online" },
            { id: "machine-2", name: "Bravo", status: "offline" },
            { id: "machine-3", name: "Charlie", status: "reconnecting" },
            { id: "machine-4", name: "Delta", status: "online" }
          ]
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      ),
    ),
  );

  render(<OverviewPage />);

  expect(await screen.findByText("4")).toBeInTheDocument();
  expect(screen.getByText("Total machines")).toBeInTheDocument();
  expect(screen.getByText("Online")).toBeInTheDocument();
  expect(screen.getByText("Offline")).toBeInTheDocument();
  expect(screen.getByText("Reconnecting")).toBeInTheDocument();
  expect(screen.getByText("2")).toBeInTheDocument();
  expect(screen.getAllByText("1")).toHaveLength(2);
});
