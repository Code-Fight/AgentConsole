import { afterEach, expect, test, vi } from "vitest";
import { http } from "./http";

afterEach(() => {
  vi.unstubAllGlobals();
});

test("preserves default accept header when caller provides custom headers", async () => {
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
  expect(headers.get("Authorization")).toBe("Bearer token");
});
