import "@testing-library/jest-dom/vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ThreadsPage } from "./threads-page";

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  private readonly listeners = new Map<string, Set<(event: MessageEvent<string>) => void>>();
  readonly close = vi.fn();

  constructor(_url: string) {
    FakeWebSocket.instances.push(this);
  }

  addEventListener(type: string, listener: (event: MessageEvent<string>) => void) {
    const listeners = this.listeners.get(type) ?? new Set();
    listeners.add(listener);
    this.listeners.set(type, listeners);
  }

  removeEventListener(type: string, listener: (event: MessageEvent<string>) => void) {
    this.listeners.get(type)?.delete(listener);
  }

  emitMessage(data: string) {
    const event = { data } as MessageEvent<string>;
    for (const listener of this.listeners.get("message") ?? []) {
      listener(event);
    }
  }
}

afterEach(() => {
  FakeWebSocket.instances = [];
  vi.unstubAllGlobals();
});

test("shows the live load error without inventing a fallback thread", async () => {
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      throw new Error("network failure");
    }),
  );

  render(
    <MemoryRouter>
      <ThreadsPage />
    </MemoryRouter>,
  );

  expect(await screen.findByText("Unable to load live threads.")).toBeInTheDocument();

  await waitFor(() => {
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  expect(screen.getByText("当前没有可用线程。")).toBeInTheDocument();
});

test("renders the design thread hub with live gateway thread data", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      if (String(input) === "/machines") {
        return new Response(
          JSON.stringify({
            items: [
              {
                id: "machine-1",
                name: "machine-1",
                status: "online",
                runtimeStatus: "running"
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

      return new Response(
        JSON.stringify({
          items: [
            {
              threadId: "thread-1",
              machineId: "machine-1",
              status: "unknown",
              title: "Investigate flaky test"
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
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter>
      <ThreadsPage />
    </MemoryRouter>,
  );

  expect(await screen.findByRole("heading", { name: "Thread Hub" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Create thread" })).toBeInTheDocument();
  expect(await screen.findByRole("link", { name: "Investigate flaky test" })).toBeInTheDocument();
  expect(screen.getByText("未知")).toBeInTheDocument();
});

test("refreshes threads when websocket thread.updated arrives", async () => {
  let threadFetchCount = 0;
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    if (String(input) === "/machines") {
      return new Response(
        JSON.stringify({
          items: [
            {
              id: "machine-1",
              name: "machine-1",
              status: "online",
              runtimeStatus: "running"
            }
          ]
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" }
        },
      );
    }

    threadFetchCount += 1;
    const items =
      threadFetchCount === 1
        ? [
            {
              threadId: "thread-1",
              machineId: "machine-1",
              status: "idle",
              title: "Initial thread"
            }
          ]
        : [
            {
              threadId: "thread-2",
              machineId: "machine-1",
              status: "active",
              title: "Refreshed thread"
            }
          ];

    return new Response(JSON.stringify({ items }), {
      status: 200,
      headers: { "Content-Type": "application/json" }
    });
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter>
      <ThreadsPage />
    </MemoryRouter>,
  );

  expect(await screen.findByRole("link", { name: "Initial thread" })).toBeInTheDocument();

  await act(async () => {
    FakeWebSocket.instances[0].emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "thread.updated",
        timestamp: "2026-04-09T10:00:00Z",
        payload: {
          machineId: "machine-1",
          threadId: "thread-2"
        }
      }),
    );
  });

  expect(await screen.findByRole("link", { name: "Refreshed thread" })).toBeInTheDocument();
  expect(threadFetchCount).toBe(2);
});

test("create archive and delete actions call the expected thread endpoints", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);

    if (url === "/machines") {
      return new Response(
        JSON.stringify({
          items: [
            {
              id: "machine-1",
              name: "machine-1",
              status: "online",
              runtimeStatus: "running"
            }
          ]
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" }
        },
      );
    }

    if (url === "/threads" && (!init?.method || init.method === "GET")) {
      return new Response(
        JSON.stringify({
          items: [
            {
              threadId: "thread-1",
              machineId: "machine-1",
              status: "idle",
              title: "Keep me"
            },
            {
              threadId: "thread-2",
              machineId: "machine-1",
              status: "idle",
              title: "Delete me"
            }
          ]
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" }
        },
      );
    }

    if (url === "/threads" && init?.method === "POST") {
      return new Response(
        JSON.stringify({
          thread: {
            threadId: "thread-3",
            machineId: "machine-1",
            status: "idle",
            title: "Created thread"
          }
        }),
        {
          status: 202,
          headers: { "Content-Type": "application/json" }
        },
      );
    }

    if (url === "/threads/thread-1/archive") {
      return new Response(JSON.stringify({ threadId: "thread-1" }), {
        status: 202,
        headers: { "Content-Type": "application/json" }
      });
    }

    if (url === "/threads/thread-2") {
      return new Response(JSON.stringify({ threadId: "thread-2", deleted: true, archived: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      });
    }

    throw new Error(`unexpected fetch: ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter>
      <ThreadsPage />
    </MemoryRouter>,
  );

  expect(await screen.findByRole("link", { name: "Keep me" })).toBeInTheDocument();

  fireEvent.change(screen.getByLabelText("Machine ID"), {
    target: { value: "machine-1" }
  });
  fireEvent.change(screen.getByLabelText("Title"), {
    target: { value: "Created thread" }
  });
  fireEvent.click(screen.getByRole("button", { name: "Create thread" }));

  fireEvent.click(screen.getByRole("button", { name: "Archive Keep me" }));
  fireEvent.click(screen.getByRole("button", { name: "Delete Delete me" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/threads",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ machineId: "machine-1", title: "Created thread" })
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/threads/thread-1/archive",
      expect.objectContaining({
        method: "POST"
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/threads/thread-2",
      expect.objectContaining({
        method: "DELETE"
      }),
    );
  });
});
