import "@testing-library/jest-dom/vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  resetConsolePreferencesStoreForTests,
  useConsolePreferences,
} from "./use-console-preferences";

beforeEach(() => {
  resetConsolePreferencesStoreForTests();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test("keeps preferences null when load fails", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(JSON.stringify({ error: "fail" }), { status: 500 })),
  );

  const { result } = renderHook(() => useConsolePreferences());

  await waitFor(() => {
    expect(result.current.isLoading).toBe(false);
  });

  expect(result.current.error).toBeTruthy();
  expect(result.current.preferences).toBeNull();
});

test("save failure does not block a later retry", async () => {
  let putCount = 0;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);

      if (path === "/settings/console" && (!init?.method || init.method === "GET")) {
        return new Response(
          JSON.stringify({
            preferences: {
              consoleUrl: "http://localhost:3100",
              apiKey: "key-1",
              profile: "dev",
              safetyPolicy: "strict",
              lastThreadId: "",
            },
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        );
      }

      if (path === "/settings/console" && init?.method === "PUT") {
        putCount += 1;
        if (putCount === 1) {
          return new Response(JSON.stringify({ error: "save failed" }), { status: 500 });
        }

        const body = init.body ? JSON.parse(String(init.body)) : { preferences: null };
        return new Response(JSON.stringify(body), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }

      throw new Error(`unexpected fetch: ${path}`);
    }),
  );

  const { result } = renderHook(() => useConsolePreferences());

  await waitFor(() => {
    expect(result.current.hasLoadedSuccessfully).toBe(true);
  });

  const first = await result.current.updatePreferences({ lastThreadId: "thread-1" });
  expect(first).toBeNull();
  expect(result.current.saveError).toBeTruthy();

  const second = await result.current.updatePreferences({ lastThreadId: "thread-1" });
  expect(second?.lastThreadId).toBe("thread-1");
  expect(result.current.saveError).toBeNull();
});
