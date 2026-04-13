import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import { DesignSourceAppRoot } from "../design-host/app-root";

class FakeWebSocket {
  readonly close = vi.fn();

  addEventListener() {}

  removeEventListener() {}
}

beforeEach(() => {
  window.history.pushState({}, "", "/");
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

  const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
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
    });
  vi.stubGlobal("fetch", fetchSpy);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(<DesignSourceAppRoot />);

  const [promptInput] = await screen.findAllByPlaceholderText(/发送指令/);
  fireEvent.change(promptInput, { target: { value: "Ping from test" } });
  fireEvent.keyDown(promptInput, { key: "Enter", code: "Enter", charCode: 13 });

  await waitFor(() => {
    expect(fetchSpy).toHaveBeenCalledWith(
      "/threads/thread-1/turns",
      expect.objectContaining({ method: "POST" }),
    );
  });
  expect(screen.queryByText(/收到你的指令/)).not.toBeInTheDocument();
});

test("keeps thread list when machine load fails", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);

      if (path === "/threads") {
        return new Response(
          JSON.stringify({
            items: [
              {
                threadId: "thread-2",
                machineId: "machine-2",
                status: "idle",
                title: "Partial load thread",
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
        throw new Error("network down");
      }

      throw new Error(`unexpected fetch: ${path}`);
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(<DesignSourceAppRoot />);

  expect(await screen.findByText("Partial load thread")).toBeInTheDocument();
});

test("surfaces system error thread status instead of treating it as completed", async () => {
  window.history.pushState({}, "", "/threads/thread-3");

  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);

      if (path === "/threads") {
        return new Response(
          JSON.stringify({
            items: [
              {
                threadId: "thread-3",
                machineId: "machine-3",
                status: "systemError",
                title: "System error thread",
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
                id: "machine-3",
                name: "Machine Three",
                status: "reconnecting",
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

      if (path === "/threads/thread-3") {
        return new Response(
          JSON.stringify({
            thread: {
              threadId: "thread-3",
              machineId: "machine-3",
              status: "systemError",
              title: "System error thread",
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

      if (path === "/machines/machine-3") {
        return new Response(
          JSON.stringify({
            machine: {
              id: "machine-3",
              name: "Machine Three",
              status: "reconnecting",
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

  render(<DesignSourceAppRoot />);

  expect((await screen.findAllByText("异常")).length).toBeGreaterThan(0);
});
