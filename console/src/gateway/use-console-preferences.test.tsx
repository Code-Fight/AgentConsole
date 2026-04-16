import "@testing-library/jest-dom/vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

const httpMock = vi.hoisted(() => vi.fn());

vi.mock("../common/api/http", () => ({
  http: httpMock,
}));

import {
  resetConsolePreferencesStoreForTests,
  useConsolePreferences,
} from "./use-console-preferences";

beforeEach(() => {
  resetConsolePreferencesStoreForTests();
  httpMock.mockReset();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test("loads and saves console preferences through shared http transport", async () => {
  httpMock.mockResolvedValueOnce({
    preferences: {
      consoleUrl: "http://localhost:3200",
      apiKey: "key-1",
      profile: "dev",
      safetyPolicy: "strict",
      lastThreadId: "",
    },
  });
  httpMock.mockResolvedValueOnce({
    preferences: {
      consoleUrl: "http://localhost:3200",
      apiKey: "key-1",
      profile: "prod",
      safetyPolicy: "strict",
      lastThreadId: "",
    },
  });

  const { result } = renderHook(() => useConsolePreferences());

  await waitFor(() => {
    expect(result.current.hasLoadedSuccessfully).toBe(true);
  });
  expect(httpMock).toHaveBeenCalledWith("/settings/console");

  await act(async () => {
    await result.current.updatePreferences({ profile: "prod" });
  });

  expect(httpMock).toHaveBeenCalledWith(
    "/settings/console",
    expect.objectContaining({
      method: "PUT",
      headers: { "Content-Type": "application/json" },
    }),
  );
});

test("keeps hook inert when disabled", () => {
  const { result } = renderHook(() => useConsolePreferences({ enabled: false }));

  expect(result.current.preferences).toBeNull();
  expect(result.current.hasAttempted).toBe(false);
  expect(httpMock).not.toHaveBeenCalled();
});
