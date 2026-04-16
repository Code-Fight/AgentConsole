import { afterEach, expect, test, vi } from "vitest";
import {
  clearGatewayConnectionCookies,
  markGatewayAuthFailed,
} from "../../gateway/gateway-connection-store";
import { buildThreadApiPath, http } from "./http";

afterEach(() => {
  clearGatewayConnectionCookies();
  document.cookie = "";
  vi.unstubAllGlobals();
});

test("blocks HTTP when gateway cookies are missing", async () => {
  document.cookie = "";
  const fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);

  await expect(http("/threads")).rejects.toThrow(
    "Gateway connection is not configured.",
  );
  expect(fetchMock).not.toHaveBeenCalled();
});

test("adds Bearer auth when gateway cookies exist", async () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  const fetchMock = vi.fn<
    (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  >(async () => new Response(JSON.stringify({ items: [] }), { status: 200 }));
  vi.stubGlobal("fetch", fetchMock);

  await http("/threads");

  const init = fetchMock.mock.calls[0]?.[1];
  expect(new Headers(init?.headers).get("Authorization")).toBe(
    "Bearer test-key",
  );
});

test("preserves default accept header when caller provides custom headers", async () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  const fetchMock = vi.fn<
    (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  >(async () => new Response(JSON.stringify({ ok: true }), { status: 200 }));
  vi.stubGlobal("fetch", fetchMock);

  await http<{ ok: boolean }>("/status", {
    headers: {
      Authorization: "Bearer token"
    }
  });

  expect(fetchMock).toHaveBeenCalledOnce();
  const init = fetchMock.mock.calls[0]?.[1];
  const headers = new Headers(init?.headers);

  expect(headers.get("Accept")).toBe("application/json");
  expect(headers.get("Authorization")).toBe("Bearer test-key");
  expect(fetchMock).toHaveBeenCalledWith(
    "http://localhost:18080/status",
    expect.objectContaining({
      headers: expect.any(Headers)
    }),
  );
});

test("preserves caller provided Accept header", async () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  const fetchMock = vi.fn<
    (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  >(async () => new Response(JSON.stringify({ ok: true }), { status: 200 }));
  vi.stubGlobal("fetch", fetchMock);

  await http<{ ok: boolean }>("/status", {
    headers: {
      Accept: "text/plain",
    },
  });

  const init = fetchMock.mock.calls[0]?.[1];
  const headers = new Headers(init?.headers);
  expect(headers.get("Accept")).toBe("text/plain");
});

test("returns auth failed error when gateway state is authFailed", async () => {
  document.cookie = "cag_gateway_url=http://localhost:18080";
  document.cookie = "cag_gateway_api_key=test-key";
  markGatewayAuthFailed();

  await expect(http("/threads")).rejects.toThrow("Gateway authentication failed.");
});

test("builds thread api path for workspace placeholders", () => {
  expect(buildThreadApiPath("thread-1")).toBe("/threads/thread-1");
  expect(buildThreadApiPath("thread 1", "messages")).toBe(
    "/threads/thread%201/messages",
  );
});
