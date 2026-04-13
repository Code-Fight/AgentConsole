import "@testing-library/jest-dom/vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import { DesignSourceAppRoot } from "../design-host/app-root";

class FakeWebSocket {
  readonly close = vi.fn();

  addEventListener() {}

  removeEventListener() {}
}

beforeEach(() => {
  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
    value: vi.fn(),
    writable: true,
  });
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

test("loads gateway thread and machine lists for the active console shell", async () => {
  const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);

    if (path === "/threads") {
      return new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    if (path === "/machines") {
      return new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }

    throw new Error(`unexpected fetch: ${path}`);
  });

  vi.stubGlobal("fetch", fetchSpy);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(<DesignSourceAppRoot />);

  await waitFor(() => {
    const paths = fetchSpy.mock.calls.map(([input]) => String(input));
    expect(paths).toContain("/threads");
    expect(paths).toContain("/machines");
  });
});

test("does not rely on local mock assistant replies in the active workspace", async () => {
  window.history.pushState({}, "", "/threads/thread-1");

  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);

      if (path === "/threads") {
        return new Response(
          JSON.stringify({
            items: [
              {
                threadId: "thread-1",
                machineId: "machine-1",
                status: "active",
                title: "Gateway Thread 1",
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
                name: "Machine One",
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
              title: "Gateway Thread 1",
            },
            activeTurnId: null,
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
              name: "Machine One",
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

      if (path === "/threads/thread-1/turns" && init?.method === "POST") {
        return new Response(
          JSON.stringify({
            turn: {
              turnId: "turn-1",
              threadId: "thread-1",
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

  render(<DesignSourceAppRoot />);

  const [promptInput] = await screen.findAllByPlaceholderText(/发送指令/);
  vi.useFakeTimers();
  fireEvent.change(promptInput, { target: { value: "Ping from test" } });
  fireEvent.keyDown(promptInput, { key: "Enter", code: "Enter", charCode: 13 });

  act(() => {
    vi.advanceTimersByTime(2000);
  });

  expect(screen.queryByText(/收到你的指令/)).not.toBeInTheDocument();
});
