import "@testing-library/jest-dom/vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { useConsolePreferences } from "./use-console-preferences";

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
