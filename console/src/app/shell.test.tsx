import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, vi } from "vitest";
import { AppShell } from "./shell";
import { DesignAppShell } from "../design/shell/design-app-shell";
import { ThreadWorkspacePage } from "../pages/thread-workspace-page";

class FakeWebSocket {
  readonly close = vi.fn();

  addEventListener() {}

  removeEventListener() {}
}

afterEach(() => {
  vi.unstubAllGlobals();
});

test("renders the design-driven thread hub shell", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ items: [] }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  })));
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/"]}>
      <AppShell />
    </MemoryRouter>,
  );

  expect(screen.getAllByText("Thread Hub").length).toBeGreaterThan(0);
  expect(screen.queryByText("上下文面板")).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Machines" })).toHaveAttribute("href", "/machines");
  expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/settings");
});

test("renders live gateway threads in the design shell left rail for workspace routes", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);

      if (path === "/threads") {
        return new Response(
          JSON.stringify({
            items: [
              {
                threadId: "thread-1",
                machineId: "machine-1",
                status: "active",
                title: "Investigate flaky test",
              },
            ],
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        );
      }

      if (path === "/machines") {
        return new Response(
          JSON.stringify({
            items: [
              {
                id: "machine-1",
                name: "machine-1",
                status: "online",
                runtimeStatus: "running",
              },
            ],
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        );
      }

      if (path === "/threads/thread-1") {
        return new Response(
          JSON.stringify({
            thread: {
              threadId: "thread-1",
              machineId: "machine-1",
              status: "active",
              title: "Investigate flaky test",
            },
            pendingApprovals: [],
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        );
      }

      if (path === "/machines/machine-1") {
        return new Response(
          JSON.stringify({
            machine: {
              id: "machine-1",
              name: "machine-1",
              status: "online",
              runtimeStatus: "running",
            },
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        );
      }

      throw new Error(`unexpected fetch: ${path}`);
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route element={<DesignAppShell />}>
          <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );

  expect((await screen.findAllByText("Investigate flaky test")).length).toBeGreaterThan(1);
  expect(screen.queryByText("Gateway rollout checks")).not.toBeInTheDocument();
});
