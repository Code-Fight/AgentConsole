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

test("submits a prompt and renders turn deltas for the current thread", async () => {
  const fetchMock = vi.fn(async () =>
    new Response(
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
