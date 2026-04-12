import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, vi } from "vitest";
import { AppShell } from "./shell";

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
