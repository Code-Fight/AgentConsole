import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { AppProviders } from "../../../app/providers/index";
import { createAppRouter } from "../../../app/router/index";
import { resetCapabilitiesForTests } from "../../../common/config/capabilities";
import {
  clearGatewayConnectionCookies,
  resetGatewayConnectionStoreForTests,
} from "../model/gateway-connection-store";

const GATEWAY_URL = "http://localhost:18080";
const GATEWAY_API_KEY = "test-key";

function setGatewayCookies() {
  document.cookie = `cag_gateway_url=${encodeURIComponent(GATEWAY_URL)}; Path=/`;
  document.cookie = `cag_gateway_api_key=${encodeURIComponent(GATEWAY_API_KEY)}; Path=/`;
}

async function getMainScope() {
  const mains = await screen.findAllByRole("main");
  return within(mains[0]);
}

beforeEach(() => {
  resetGatewayConnectionStoreForTests();
  resetCapabilitiesForTests();
  setGatewayCookies();
  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
    value: vi.fn(),
    writable: true,
  });
});

afterEach(() => {
  clearGatewayConnectionCookies();
  resetCapabilitiesForTests();
  vi.unstubAllGlobals();
});

test("/settings remains reachable while unconfigured", async () => {
  clearGatewayConnectionCookies();
  const fetchSpy = vi.fn();
  vi.stubGlobal("fetch", fetchSpy);

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  const scope = await getMainScope();
  expect(await scope.findByLabelText("Gateway URL")).toBeInTheDocument();
  expect(scope.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
  expect(fetchSpy).not.toHaveBeenCalled();
});

test("saving gateway connection requires a successful test for current values", async () => {
  clearGatewayConnectionCookies();

  const fetchSpy = vi.fn(async () =>
    new Response(JSON.stringify({ items: [] }), {
      status: 200,
      headers: {
        "Content-Type": "application/json",
      },
    }),
  );
  vi.stubGlobal("fetch", fetchSpy);

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  const scope = await getMainScope();
  expect(fetchSpy).not.toHaveBeenCalled();

  fireEvent.change(await scope.findByLabelText("Gateway URL"), {
    target: { value: "http://localhost:3100" },
  });
  fireEvent.change(scope.getByLabelText("Gateway API Key"), {
    target: { value: "test-key" },
  });

  const saveButton = scope.getByRole("button", { name: "Save Gateway Connection" });
  expect(saveButton).toBeDisabled();

  fireEvent.click(scope.getByRole("button", { name: "Test Gateway Connection" }));

  await waitFor(() => {
    expect(scope.getByText("Gateway connection test succeeded.")).toBeInTheDocument();
  });
  expect(saveButton).toBeEnabled();

  fireEvent.click(saveButton);

  await waitFor(() => {
    expect(document.cookie).toContain("cag_gateway_url=http%3A%2F%2Flocalhost%3A3100");
    expect(document.cookie).toContain("cag_gateway_api_key=test-key");
  });

  fireEvent.change(scope.getByLabelText("Gateway URL"), {
    target: { value: "http://localhost:3200" },
  });
  expect(saveButton).toBeDisabled();
  expect(fetchSpy).toHaveBeenCalledTimes(1);
});

test("settings only exposes gateway connection controls", async () => {
  const fetchSpy = vi.fn();
  vi.stubGlobal("fetch", fetchSpy);

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  const scope = await getMainScope();
  expect(await scope.findByText("API Configuration")).toBeInTheDocument();
  expect(scope.getByLabelText("Gateway URL")).toBeInTheDocument();
  expect(scope.getByLabelText("Gateway API Key")).toBeInTheDocument();
  expect(scope.getByRole("button", { name: "Test Gateway Connection" })).toBeInTheDocument();
  expect(scope.queryByText("Console Preferences")).not.toBeInTheDocument();
  expect(scope.queryByText("Agent 默认配置")).not.toBeInTheDocument();
  expect(scope.queryByText("Machine Override")).not.toBeInTheDocument();
  expect(scope.queryByLabelText("Console Profile")).not.toBeInTheDocument();
  expect(scope.queryByLabelText("Safety Policy")).not.toBeInTheDocument();
  expect(scope.queryByLabelText("Global Default TOML")).not.toBeInTheDocument();
  expect(scope.queryByLabelText("Machine Override TOML")).not.toBeInTheDocument();
  expect(scope.queryByRole("button", { name: "Apply To Machine" })).not.toBeInTheDocument();
  expect(fetchSpy).not.toHaveBeenCalled();
});

test("testing gateway connection validates the current URL and API key", async () => {
  const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    expect(String(input)).toBe("http://localhost:3100/machines");
    expect(init?.headers).toBeInstanceOf(Headers);
    const headers = init?.headers as Headers;
    expect(headers.get("Accept")).toBe("application/json");
    expect(headers.get("Authorization")).toBe("Bearer test-key");

    return new Response(JSON.stringify({ items: [] }), {
      status: 200,
      headers: {
        "Content-Type": "application/json",
      },
    });
  });
  vi.stubGlobal("fetch", fetchSpy);

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  const scope = await getMainScope();
  fireEvent.change(await scope.findByLabelText("Gateway URL"), {
    target: { value: "http://localhost:3100" },
  });
  fireEvent.change(scope.getByLabelText("Gateway API Key"), {
    target: { value: "test-key" },
  });

  fireEvent.click(scope.getByRole("button", { name: "Test Gateway Connection" }));

  await waitFor(() => {
    expect(scope.getByText("Gateway connection test succeeded.")).toBeInTheDocument();
  });
  expect(scope.getByRole("button", { name: "Save Gateway Connection" })).toBeEnabled();
  expect(fetchSpy).toHaveBeenCalledTimes(1);
});

test("testing gateway connection surfaces authentication failures", async () => {
  const fetchSpy = vi.fn(async () => new Response(null, { status: 401 }));
  vi.stubGlobal("fetch", fetchSpy);

  const router = createAppRouter({ initialEntries: ["/settings"] });
  render(<AppProviders router={router} />);

  const scope = await getMainScope();
  fireEvent.change(await scope.findByLabelText("Gateway URL"), {
    target: { value: "http://localhost:3100" },
  });
  fireEvent.change(scope.getByLabelText("Gateway API Key"), {
    target: { value: "bad-key" },
  });

  fireEvent.click(scope.getByRole("button", { name: "Test Gateway Connection" }));

  await waitFor(() => {
    expect(scope.getByText("Gateway authentication failed.")).toBeInTheDocument();
  });
  expect(fetchSpy).toHaveBeenCalledTimes(1);
});
