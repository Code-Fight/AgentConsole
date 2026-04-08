import { afterEach, expect, test, vi } from "vitest";
import { connectConsoleSocket } from "./ws";

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  readonly addEventListener = vi.fn();
  readonly removeEventListener = vi.fn();
  readonly close = vi.fn();
  readonly url: string;

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }
}

afterEach(() => {
  FakeWebSocket.instances = [];
  vi.unstubAllGlobals();
});

test("uses an origin-aware websocket default url", () => {
  vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);

  const onMessage = vi.fn();
  const disconnect = connectConsoleSocket(onMessage);

  const socket = FakeWebSocket.instances[0];
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const expectedUrl = `${protocol}://${window.location.host}/ws`;

  expect(socket.url).toBe(expectedUrl);

  disconnect();

  expect(socket.addEventListener).toHaveBeenCalledWith("message", onMessage);
  expect(socket.removeEventListener).toHaveBeenCalledWith("message", onMessage);
  expect(socket.close).toHaveBeenCalledOnce();
});
