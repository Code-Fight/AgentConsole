import "@testing-library/jest-dom/vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ThreadWorkspacePage } from "./thread-workspace-page";

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  private readonly listeners = new Map<string, Set<(event: MessageEvent<string>) => void>>();
  readonly close = vi.fn();
  readonly url: string;

  constructor(url: string) {
    this.url = url;
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

test("maps gateway thread events into the design workspace view-model", async () => {
  const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
    if (init?.method === "POST") {
      return new Response(
        JSON.stringify({
          turn: {
            turnId: "turn-1",
            threadId: "thread-1"
          }
        }),
        {
          status: 202,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    return new Response(
      JSON.stringify({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "idle",
          title: "Investigate flaky test"
        },
        pendingApprovals: []
      }),
      {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      },
    );
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("Waiting for live Gateway events.")).toBeInTheDocument();

  fireEvent.change(screen.getByRole("textbox", { name: "Prompt" }), {
    target: { value: "run tests" }
  });
  fireEvent.click(screen.getByRole("button", { name: "Send prompt" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/threads/thread-1/turns",
      expect.objectContaining({
        method: "POST"
      }),
    );
  });

  const socket = FakeWebSocket.instances[0];
  await act(async () => {
    socket.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.delta",
        timestamp: "2026-04-08T14:00:01Z",
        payload: {
          threadId: "thread-1",
          turnId: "turn-1",
          sequence: 1,
          delta: "hello from gateway"
        }
      }),
    );
  });

  expect(await screen.findByText("hello from gateway")).toBeInTheDocument();
});

test("renders a turn-started message for the current thread", async () => {
  vi.stubGlobal("fetch", vi.fn(async () =>
    new Response(
      JSON.stringify({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "idle",
          title: "Investigate flaky test"
        },
        pendingApprovals: []
      }),
      {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      },
    ),
  ));
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  const socket = FakeWebSocket.instances[0];
  await act(async () => {
    socket.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.started",
        timestamp: "2026-04-08T14:00:01Z",
        payload: {
          threadId: "thread-1",
          turnId: "turn-1"
        }
      }),
    );
  });

  expect(await screen.findByText("Turn started: turn-1")).toBeInTheDocument();
});

test("renders tool-user-input approval controls for approval.required events on the current thread", async () => {
  vi.stubGlobal("fetch", vi.fn(async () =>
    new Response(
      JSON.stringify({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "idle",
          title: "Investigate flaky test"
        },
        pendingApprovals: []
      }),
      {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      },
    ),
  ));
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  const socket = FakeWebSocket.instances[0];
  await act(async () => {
    socket.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "approval.required",
        requestId: "approval-1",
        timestamp: "2026-04-08T14:00:01Z",
        payload: {
          requestId: "approval-1",
          threadId: "thread-1",
          turnId: "turn-1",
          itemId: "item-1",
          kind: "tool_user_input",
          reason: "Pick an option"
        }
      }),
    );
  });

  expect(await screen.findByText("待处理审批")).toBeInTheDocument();
  expect(screen.getByText("Pick an option")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Accept" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Decline" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
});

test("renders tool-user-input questions and posts explicit answers on accept", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path === "/approvals/approval-1/respond" && init?.method === "POST") {
      return new Response(null, { status: 204 });
    }

    if (path === "/machines/machine-1") {
      return new Response(
        JSON.stringify({
          machine: {
            id: "machine-1",
            name: "machine-1",
            status: "online",
            runtimeStatus: "running"
          }
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
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "idle",
          title: "Investigate flaky test"
        },
        activeTurnId: null,
        pendingApprovals: [
          {
            requestId: "approval-1",
            threadId: "thread-1",
            turnId: "turn-1",
            itemId: "item-1",
            kind: "tool_user_input",
            reason: "Need operator input",
            questions: [
              {
                id: "question-1",
                text: "Pick a branch",
                options: ["main", "release"]
              },
              {
                id: "question-2",
                text: "Why are you overriding it?"
              }
            ]
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
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("Pick a branch")).toBeInTheDocument();
  expect(screen.getByRole("option", { name: "main" })).toBeInTheDocument();
  expect(screen.getByRole("option", { name: "release" })).toBeInTheDocument();

  fireEvent.change(screen.getByRole("combobox", { name: "Pick a branch" }), {
    target: { value: "release" }
  });
  fireEvent.change(screen.getByRole("textbox", { name: "Why are you overriding it?" }), {
    target: { value: "Need the release branch for validation." }
  });
  fireEvent.click(screen.getByRole("button", { name: "Accept" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/approvals/approval-1/respond",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          decision: "accept",
          answers: {
            "question-1": "release",
            "question-2": "Need the release branch for validation."
          }
        })
      }),
    );
  });
});

test("clicking accept posts the approval decision to the approval endpoint", async () => {
  const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
    if (init?.method === "POST") {
      return new Response(null, { status: 204 });
    }

    return new Response(
      JSON.stringify({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "idle",
          title: "Investigate flaky test"
        },
        pendingApprovals: []
      }),
      {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      },
    );
  });
  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  const socket = FakeWebSocket.instances[0];
  await act(async () => {
    socket.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "approval.required",
        requestId: "approval-1",
        timestamp: "2026-04-08T14:00:01Z",
        payload: {
          requestId: "approval-1",
          threadId: "thread-1",
          turnId: "turn-1",
          itemId: "item-1",
          kind: "command",
          command: "go test ./..."
        }
      }),
    );
  });

  fireEvent.click(await screen.findByRole("button", { name: "Accept" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/approvals/approval-1/respond",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ decision: "accept" })
      }),
    );
  });
});

test("hydrates the active turn from thread detail so steer and interrupt remain available after reload", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path === "/machines/machine-1") {
      return new Response(
        JSON.stringify({
          machine: {
            id: "machine-1",
            name: "machine-1",
            status: "online",
            runtimeStatus: "running"
          }
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
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "active",
          title: "Investigate flaky test"
        },
        activeTurnId: "turn-active-1",
        pendingApprovals: []
      }),
      {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      },
    );
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("当前 Turn：turn-active-1")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Send steer" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Interrupt turn" })).toBeInTheDocument();
});

test("removes a pending approval when approval.resolved arrives for the current thread", async () => {
  vi.stubGlobal("fetch", vi.fn(async () =>
    new Response(
      JSON.stringify({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "idle",
          title: "Investigate flaky test"
        },
        pendingApprovals: []
      }),
      {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      },
    ),
  ));
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  const socket = FakeWebSocket.instances[0];
  await act(async () => {
    socket.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "approval.required",
        requestId: "approval-1",
        timestamp: "2026-04-08T14:00:01Z",
        payload: {
          requestId: "approval-1",
          threadId: "thread-1",
          turnId: "turn-1",
          itemId: "item-1",
          kind: "command",
          command: "go test ./..."
        }
      }),
    );
  });

  expect(await screen.findByText("待处理审批")).toBeInTheDocument();

  await act(async () => {
    socket.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "approval.resolved",
        requestId: "approval-1",
        timestamp: "2026-04-08T14:00:02Z",
        payload: {
          requestId: "approval-1",
          threadId: "thread-1",
          decision: "accept"
        }
      }),
    );
  });

  await waitFor(() => {
    expect(screen.queryByText("go test ./...")).not.toBeInTheDocument();
  });
});

test("hydrates pending approvals from the initial thread detail fetch", async () => {
  const fetchMock = vi.fn(async () =>
    new Response(
      JSON.stringify({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          status: "unknown",
          title: "Investigate flaky test"
        },
        pendingApprovals: [
          {
            requestId: "approval-1",
            threadId: "thread-1",
            turnId: "turn-1",
            itemId: "item-1",
            kind: "command",
            command: "go test ./..."
          }
        ]
      }),
      {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      },
    ),
  );

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByText("待处理审批")).toBeInTheDocument();
  expect(screen.getByText("go test ./...")).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledWith(
    "/threads/thread-1",
    expect.objectContaining({
      headers: expect.any(Headers)
    }),
  );
});

test("renders steer and interrupt controls for an active turn and posts to the expected endpoints", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);

    if (url === "/threads/thread-1" && (!init?.method || init.method === "GET")) {
      return new Response(
        JSON.stringify({
          thread: {
            threadId: "thread-1",
            machineId: "machine-1",
            status: "active",
            title: "Investigate flaky test"
          },
          pendingApprovals: []
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    if (url === "/threads/thread-1/turns/turn-1/steer") {
      return new Response(
        JSON.stringify({
          turn: {
            turnId: "turn-1",
            threadId: "thread-1"
          }
        }),
        {
          status: 202,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    if (url === "/threads/thread-1/turns/turn-1/interrupt") {
      return new Response(
        JSON.stringify({
          turn: {
            turnId: "turn-1",
            threadId: "thread-1",
            status: "interrupted"
          }
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    throw new Error(`unexpected fetch: ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  const socket = FakeWebSocket.instances[0];
  await act(async () => {
    socket.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.started",
        timestamp: "2026-04-08T14:00:01Z",
        payload: {
          threadId: "thread-1",
          turnId: "turn-1"
        }
      }),
    );
  });

  expect(await screen.findByRole("button", { name: "Send steer" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Interrupt turn" })).toBeInTheDocument();

  fireEvent.change(screen.getByRole("textbox", { name: "Steer prompt" }), {
    target: { value: "try a smaller patch" }
  });
  fireEvent.click(screen.getByRole("button", { name: "Send steer" }));
  fireEvent.click(screen.getByRole("button", { name: "Interrupt turn" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/threads/thread-1/turns/turn-1/steer",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ input: "try a smaller patch" })
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/threads/thread-1/turns/turn-1/interrupt",
      expect.objectContaining({
        method: "POST"
      }),
    );
  });
});

test("turn.failed clears the active turn controls for the current thread", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);

    if (url === "/threads/thread-1" && (!init?.method || init.method === "GET")) {
      return new Response(
        JSON.stringify({
          thread: {
            threadId: "thread-1",
            machineId: "machine-1",
            status: "active",
            title: "Investigate flaky test"
          },
          activeTurnId: "turn-1",
          pendingApprovals: []
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      );
    }

    throw new Error(`unexpected fetch: ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  render(
    <MemoryRouter initialEntries={["/threads/thread-1"]}>
      <Routes>
        <Route path="/threads/:threadId" element={<ThreadWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
  );

  expect(await screen.findByRole("button", { name: "Send steer" })).toBeInTheDocument();

  const socket = FakeWebSocket.instances[0];
  await act(async () => {
    socket.emitMessage(
      JSON.stringify({
        version: "v1",
        category: "event",
        name: "turn.failed",
        timestamp: "2026-04-08T14:00:05Z",
        payload: {
          turn: {
            turnId: "turn-1",
            threadId: "thread-1",
            status: "failed"
          }
        }
      }),
    );
  });

  await waitFor(() => {
    expect(screen.queryByRole("button", { name: "Send steer" })).not.toBeInTheDocument();
  });
  expect(screen.getByText("Turn turn-1 failed")).toBeInTheDocument();
});
