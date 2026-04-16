import { afterEach, expect, test, vi } from "vitest";
import {
  clearGatewayConnectionCookies,
  getGatewayConnectionConfig,
  getGatewayConnectionIdentity,
  getGatewayConnectionState,
  markGatewayAuthFailed,
  readGatewayConnectionFromCookies,
  saveGatewayConnectionToCookies,
  subscribeGatewayConnection,
} from "./gateway-connection-store";

afterEach(() => {
  clearGatewayConnectionCookies();
  document.cookie = "";
  vi.unstubAllGlobals();
});

test("reads gateway connection from cookies", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";

  expect(readGatewayConnectionFromCookies()).toEqual({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });
});

test("treats blank or invalid gateway url as missing", () => {
  document.cookie = "cag_gateway_url=   ";
  document.cookie = "cag_gateway_api_key=test-key";
  expect(readGatewayConnectionFromCookies()).toBeNull();

  document.cookie = "cag_gateway_url=not-a-url";
  document.cookie = "cag_gateway_api_key=test-key";
  expect(readGatewayConnectionFromCookies()).toBeNull();
});

test("treats malformed cookie encoding as missing", () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=%E0%A4%A";

  expect(readGatewayConnectionFromCookies()).toBeNull();
});

test("saves and clears gateway cookies", () => {
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });
  expect(document.cookie).toContain("cag_gateway_url=http%3A%2F%2Flocalhost%3A18080");
  expect(document.cookie).toContain("cag_gateway_api_key=test-key");

  clearGatewayConnectionCookies();
  expect(readGatewayConnectionFromCookies()).toBeNull();
});

test("adds Secure to cookie writes on https", () => {
  const setCookieSpy = vi.spyOn(Document.prototype, "cookie", "set");
  vi.stubGlobal("location", { protocol: "https:" });

  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });

  const writes = setCookieSpy.mock.calls.map(([value]) => value);
  expect(writes).toHaveLength(2);
  expect(writes[0]).toContain("; Secure");
  expect(writes[1]).toContain("; Secure");
});

test("keeps cookie writes unchanged on http", () => {
  const setCookieSpy = vi.spyOn(Document.prototype, "cookie", "set");
  vi.stubGlobal("location", { protocol: "http:" });

  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });

  const writes = setCookieSpy.mock.calls.map(([value]) => value);
  expect(writes).toHaveLength(2);
  expect(writes.every((value) => !value.includes("; Secure"))).toBe(true);
});

test("tracks missing, ready, and authFailed states", () => {
  clearGatewayConnectionCookies();
  expect(getGatewayConnectionState()).toBe("missing");
  expect(getGatewayConnectionConfig()).toBeNull();

  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });
  expect(getGatewayConnectionState()).toBe("ready");
  expect(getGatewayConnectionConfig()).toEqual({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });

  markGatewayAuthFailed();
  expect(getGatewayConnectionState()).toBe("authFailed");
});

test("notifies subscribers on state changes", () => {
  const listener = vi.fn();
  const unsubscribe = subscribeGatewayConnection(listener);

  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });
  markGatewayAuthFailed();
  clearGatewayConnectionCookies();

  expect(listener).toHaveBeenCalledTimes(3);

  unsubscribe();
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });
  expect(listener).toHaveBeenCalledTimes(3);
});

test("gateway identity changes with config and does not expose plaintext api key", () => {
  clearGatewayConnectionCookies();
  expect(getGatewayConnectionIdentity()).toBe("missing");

  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "plain-secret-key",
  });
  const firstIdentity = getGatewayConnectionIdentity();
  expect(firstIdentity).toContain("ready:");
  expect(firstIdentity).not.toContain("plain-secret-key");
  expect(firstIdentity).not.toContain("localhost:18080");

  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "another-key",
  });
  const secondIdentity = getGatewayConnectionIdentity();
  expect(secondIdentity).not.toBe(firstIdentity);
});
