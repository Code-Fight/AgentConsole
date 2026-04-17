import { afterEach, expect, test, vi } from "vitest";
import {
  clearGatewayConnectionCookies,
  saveGatewayConnectionToCookies,
} from "./gateway-connection-store";

function seedGatewayConnection() {
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "test-key",
  });
}

afterEach(() => {
  clearGatewayConnectionCookies();
  vi.unstubAllGlobals();
  vi.resetModules();
});

test("falls back to disabled capabilities when fetch fails", async () => {
  seedGatewayConnection();
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      throw new Error("network down");
    }),
  );

  const { refreshCapabilities, supportsCapability } = await import("./capabilities");

  await refreshCapabilities();

  expect(supportsCapability("threadHub")).toBe(false);
  expect(supportsCapability("startTurn")).toBe(false);
});

test("merges fetched snapshot with defaults", async () => {
  seedGatewayConnection();
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      return new Response(JSON.stringify({ threadHub: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }),
  );

  const { refreshCapabilities, supportsCapability } = await import("./capabilities");

  await refreshCapabilities();

  expect(supportsCapability("threadHub")).toBe(true);
  expect(supportsCapability("startTurn")).toBe(false);
});
