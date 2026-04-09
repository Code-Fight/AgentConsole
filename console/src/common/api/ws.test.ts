import { afterEach, expect, test, vi } from "vitest";
import { buildThreadSocketUrl, connectConsoleSocket } from "./ws";

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  private readonly listeners = new Map<string, Set<EventListener>>();
  readonly addEventListener = vi.fn((type: string, listener: EventListener) => {
    const listeners = this.listeners.get(type) ?? new Set<EventListener>();
    listeners.add(listener);
    this.listeners.set(type, listeners);
  });
  readonly removeEventListener = vi.fn((type: string, listener: EventListener) => {
    this.listeners.get(type)?.delete(listener);
  });
  readonly close = vi.fn();
  readonly url: string;

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  emit(type: string, event: Event) {
    for (const listener of this.listeners.get(type) ?? []) {
      listener(event);
    }
  }
}

afterEach(() => {
  FakeWebSocket.instances = [];
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

test("uses an origin-aware websocket default url", () => {
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const onMessage = vi.fn();
  const disconnect = connectConsoleSocket("thread-1", onMessage);

  const socket = FakeWebSocket.instances[0];
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const expectedUrl = `${protocol}://${window.location.host}/ws?threadId=thread-1`;

  expect(socket.url).toBe(expectedUrl);

  disconnect();

  expect(socket.addEventListener).toHaveBeenCalledWith("message", onMessage);
  expect(socket.removeEventListener).toHaveBeenCalledWith("message", onMessage);
  expect(socket.close).toHaveBeenCalledOnce();
});

test("builds thread websocket url for workspace placeholders", () => {
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const expectedUrl = `${protocol}://${window.location.host}/ws?threadId=thread%201`;

  expect(buildThreadSocketUrl("thread 1")).toBe(expectedUrl);
});

test("connects without a thread filter when no thread id is provided", () => {
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const onMessage = vi.fn();
  const disconnect = connectConsoleSocket(undefined, onMessage);

  const socket = FakeWebSocket.instances[0];
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";

  expect(socket.url).toBe(`${protocol}://${window.location.host}/ws`);

  disconnect();
  expect(socket.close).toHaveBeenCalledOnce();
});

test("reconnects with bounded backoff after socket close", () => {
  vi.useFakeTimers();
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const disconnect = connectConsoleSocket("thread-1", vi.fn());
  expect(FakeWebSocket.instances).toHaveLength(1);

  FakeWebSocket.instances[0].emit("close", new Event("close"));
  vi.advanceTimersByTime(999);
  expect(FakeWebSocket.instances).toHaveLength(1);

  vi.advanceTimersByTime(1);
  expect(FakeWebSocket.instances).toHaveLength(2);

  FakeWebSocket.instances[1].emit("error", new Event("error"));
  vi.advanceTimersByTime(1999);
  expect(FakeWebSocket.instances).toHaveLength(2);

  vi.advanceTimersByTime(1);
  expect(FakeWebSocket.instances).toHaveLength(3);

  disconnect();
  FakeWebSocket.instances[2].emit("close", new Event("close"));
  vi.advanceTimersByTime(5_000);
  expect(FakeWebSocket.instances).toHaveLength(3);
});
