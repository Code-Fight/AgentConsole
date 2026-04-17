import { afterEach, expect, test, vi } from "vitest";
import {
  clearGatewayConnectionCookies,
  getGatewayConnectionState,
} from "../../gateway/gateway-connection-store";
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
  clearGatewayConnectionCookies();
  document.cookie = "";
  FakeWebSocket.instances = [];
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

test("uses configured gateway websocket url", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const onMessage = vi.fn();
  const disconnect = connectConsoleSocket("thread-1", onMessage);

  const socket = FakeWebSocket.instances[0];
  expect(socket.url).toBe(
    "ws://localhost:18080/ws?threadId=thread-1&apiKey=test-key",
  );

  disconnect();

  expect(socket.addEventListener).toHaveBeenCalledWith("message", onMessage);
  expect(socket.removeEventListener).toHaveBeenCalledWith("message", onMessage);
  expect(socket.close).toHaveBeenCalledOnce();
});

test("builds thread websocket url for workspace placeholders", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  const expectedUrl =
    "ws://localhost:18080/ws?threadId=thread+1&apiKey=test-key";

  expect(buildThreadSocketUrl("thread 1")).toBe(expectedUrl);
});

test("uses gateway origin for websocket url when gateway path is configured", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080/api";
  document.cookie = "cag_gateway_api_key=test-key";
  const expectedUrl =
    "ws://localhost:18080/ws?threadId=thread+1&apiKey=test-key";

  expect(buildThreadSocketUrl("thread 1")).toBe(expectedUrl);
});

test("connects without a thread filter when no thread id is provided", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const onMessage = vi.fn();
  const disconnect = connectConsoleSocket(undefined, onMessage);

  const socket = FakeWebSocket.instances[0];
  expect(socket.url).toBe("ws://localhost:18080/ws?apiKey=test-key");

  disconnect();
  expect(socket.close).toHaveBeenCalledOnce();
});

test("reconnects with bounded backoff after socket close", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
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
  FakeWebSocket.instances[1].emit("close", new CloseEvent("close"));
  vi.advanceTimersByTime(1999);
  expect(FakeWebSocket.instances).toHaveLength(2);

  vi.advanceTimersByTime(1);
  expect(FakeWebSocket.instances).toHaveLength(3);

  disconnect();
  FakeWebSocket.instances[2].emit("close", new Event("close"));
  vi.advanceTimersByTime(5_000);
  expect(FakeWebSocket.instances).toHaveLength(3);
});

test("recovers when error occurs without a matching close event", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  vi.useFakeTimers();
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  connectConsoleSocket("thread-1", vi.fn());
  expect(FakeWebSocket.instances).toHaveLength(1);

  const firstSocket = FakeWebSocket.instances[0];
  FakeWebSocket.instances[0].emit("error", new Event("error"));
  expect(firstSocket.close).toHaveBeenCalledOnce();
  vi.advanceTimersByTime(999);
  expect(FakeWebSocket.instances).toHaveLength(1);
  vi.advanceTimersByTime(1);
  expect(FakeWebSocket.instances).toHaveLength(2);
});

test("fails closed when gateway cookies are missing", () => {
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const disconnect = connectConsoleSocket("thread-1", vi.fn());
  expect(FakeWebSocket.instances).toHaveLength(0);

  disconnect();
});

test("marks auth failed and stops reconnecting on auth close", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  vi.useFakeTimers();
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  connectConsoleSocket("thread-1", vi.fn());
  expect(FakeWebSocket.instances).toHaveLength(1);

  FakeWebSocket.instances[0].emit("close", new CloseEvent("close", { code: 1008 }));
  vi.advanceTimersByTime(5_000);

  expect(FakeWebSocket.instances).toHaveLength(1);
  expect(getGatewayConnectionState()).toBe("authFailed");
});
