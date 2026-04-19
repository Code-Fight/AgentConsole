import "@testing-library/jest-dom/vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  clearGatewayConnectionCookies,
  markGatewayAuthFailed,
  saveGatewayConnectionToCookies,
} from "../model/gateway-connection-store";

const httpMock = vi.hoisted(() => vi.fn());

vi.mock("../../../common/api/http", () => ({
  http: httpMock,
}));

import {
  resetConsolePreferencesStoreForTests,
  useConsolePreferences,
} from "./use-console-preferences";

beforeEach(() => {
  resetConsolePreferencesStoreForTests();
  httpMock.mockReset();
  clearGatewayConnectionCookies();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test("loads and saves console preferences through shared http transport", async () => {
  httpMock.mockResolvedValueOnce({
    preferences: {
      profile: "dev",
      safetyPolicy: "strict",
      lastThreadId: "",
    },
  });
  httpMock.mockResolvedValueOnce({
    preferences: {
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

test("reloads after gateway connection identity changes while enabled", async () => {
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "key-1",
  });

  httpMock.mockResolvedValueOnce({
    preferences: {
      profile: "dev",
      safetyPolicy: "strict",
      lastThreadId: "",
    },
  });
  httpMock.mockResolvedValueOnce({
    preferences: {
      profile: "prod",
      safetyPolicy: "strict",
      lastThreadId: "",
    },
  });

  const { result } = renderHook(() => useConsolePreferences({ enabled: true }));
  await waitFor(() => {
    expect(result.current.preferences?.profile).toBe("dev");
  });

  await act(async () => {
    saveGatewayConnectionToCookies({
      gatewayUrl: "http://localhost:19090",
      apiKey: "key-2",
    });
  });

  await waitFor(() => {
    expect(result.current.preferences?.profile).toBe("prod");
  });
});

test("recovers after auth-failed -> new key -> re-enable", async () => {
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "old-key",
  });

  httpMock.mockResolvedValueOnce({
    preferences: {
      profile: "dev",
      safetyPolicy: "strict",
      lastThreadId: "",
    },
  });
  httpMock.mockResolvedValueOnce({
    preferences: {
      profile: "dev",
      safetyPolicy: "strict",
      lastThreadId: "",
    },
  });

  let enabled = true;
  const { result, rerender } = renderHook(() => useConsolePreferences({ enabled }));
  await waitFor(() => {
    expect(result.current.hasLoadedSuccessfully).toBe(true);
  });

  await act(async () => {
    markGatewayAuthFailed();
    enabled = false;
    rerender();
  });

  expect(result.current.preferences).toBeNull();

  await act(async () => {
    saveGatewayConnectionToCookies({
      gatewayUrl: "http://localhost:18080",
      apiKey: "new-key",
    });
    enabled = true;
    rerender();
  });

  await waitFor(() => {
    expect(result.current.preferences?.profile).toBe("dev");
  });
});
