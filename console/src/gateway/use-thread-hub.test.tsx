import "@testing-library/jest-dom/vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { useThreadHub } from "./use-thread-hub";

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

test("enabled false prevents initial load and socket connection", async () => {
  const fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const { result } = renderHook(() => useThreadHub({ enabled: false }));

  await waitFor(() => {
    expect(result.current.machineCount).toBe(0);
  });

  expect(fetchMock).not.toHaveBeenCalled();
  expect(FakeWebSocket.instances).toHaveLength(0);
});

test("initial load success maps gateway data into the thread hub view-model", async () => {
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

      return new Response(
        JSON.stringify({
          items: [
            {
              id: "machine-1",
              name: "Runner 01",
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
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const { result } = renderHook(() => useThreadHub());

  await waitFor(() => {
    expect(result.current.threads).toHaveLength(1);
  });

  expect(result.current.error).toBeNull();
  expect(result.current.machineCount).toBe(1);
  expect(result.current.machineSuggestions).toEqual([
    { id: "machine-1", label: "Runner 01" },
  ]);
  expect(result.current.threads[0]).toMatchObject({
    id: "thread-1",
    title: "Investigate flaky test",
    machineLabel: "Runner 01",
    status: "active",
    statusLabel: "进行中",
    machineRuntimeLabel: "在线 / running",
  });
  expect(FakeWebSocket.instances).toHaveLength(1);
});

test("initial load failure surfaces the hub error", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      throw new Error("network failure");
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const { result } = renderHook(() => useThreadHub());

  await waitFor(() => {
    expect(result.current.error).toBe("Unable to load live threads.");
  });

  expect(result.current.threads).toEqual([]);
  expect(result.current.machineCount).toBe(0);
});

test("create thread toggles submitting state and refreshes the hub data", async () => {
  let createRequestResolved = false;
  let listVersion = 0;
  let resolveCreate: (() => void) | null = null;
  const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const path = String(input);

    if (path === "/threads" && init?.method === "POST") {
      return new Promise<Response>((resolve) => {
        resolveCreate = () => {
          createRequestResolved = true;
          resolve(
            new Response(
              JSON.stringify({
                thread: {
                  threadId: "thread-2",
                  machineId: "machine-1",
                  status: "idle",
                  title: "Created thread",
                },
              }),
              {
                status: 200,
                headers: { "Content-Type": "application/json" },
              },
            ),
          );
        };
      });
    }

    if (path === "/threads") {
      listVersion += 1;
      const items =
        createRequestResolved
          ? [
              {
                threadId: "thread-2",
                machineId: "machine-1",
                status: "idle",
                title: "Created thread",
              },
            ]
          : [
              {
                threadId: "thread-1",
                machineId: "machine-1",
                status: "active",
                title: "Initial thread",
              },
            ];

      return Promise.resolve(
        new Response(JSON.stringify({ items }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }

    return Promise.resolve(
      new Response(
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
      ),
    );
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const { result } = renderHook(() => useThreadHub());

  await waitFor(() => {
    expect(result.current.threads[0]?.title).toBe("Initial thread");
  });

  act(() => {
    result.current.setMachineId("machine-1");
    result.current.setTitle("Created thread");
  });

  let createPromise: Promise<void> | undefined;
  await act(async () => {
    createPromise = result.current.handleCreateThread();
  });

  expect(result.current.isSubmitting).toBe(true);
  expect(result.current.error).toBeNull();

  await act(async () => {
    resolveCreate?.();
    await createPromise;
  });

  await waitFor(() => {
    expect(result.current.isSubmitting).toBe(false);
    expect(result.current.threads[0]?.title).toBe("Created thread");
  });

  expect(result.current.title).toBe("");
  expect(listVersion).toBe(2);
  expect(fetchMock).toHaveBeenCalledWith(
    "/threads",
    expect.objectContaining({
      method: "POST",
      body: JSON.stringify({
        machineId: "machine-1",
        title: "Created thread",
      }),
    }),
  );
});

test("socket-triggered refresh updates the thread hub data", async () => {
  let listVersion = 0;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);

      if (path === "/threads") {
        listVersion += 1;
        const items =
          listVersion === 1
            ? [
                {
                  threadId: "thread-1",
                  machineId: "machine-1",
                  status: "idle",
                  title: "Initial thread",
                },
              ]
            : [
                {
                  threadId: "thread-2",
                  machineId: "machine-1",
                  status: "active",
                  title: "Refreshed thread",
                },
              ];

        return new Response(JSON.stringify({ items }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }

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
    }),
  );
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const { result } = renderHook(() => useThreadHub());

  await waitFor(() => {
    expect(result.current.threads[0]?.title).toBe("Initial thread");
  });

  await act(async () => {
    FakeWebSocket.instances[0].emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "thread.updated",
        timestamp: "2026-04-09T10:00:00Z",
        payload: {
          machineId: "machine-1",
          threadId: "thread-2",
        },
      }),
    );
  });

  await waitFor(() => {
    expect(result.current.threads[0]?.title).toBe("Refreshed thread");
  });

  expect(listVersion).toBe(2);
});
