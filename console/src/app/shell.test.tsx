import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { beforeEach } from "vitest";
import { clearGatewayConnectionCookies } from "../gateway/gateway-connection-store";
import { resetConsolePreferencesStoreForTests } from "../gateway/use-console-preferences";
import { DesignSourceAppRoot } from "../design-host/app-root";

beforeEach(() => {
  window.history.pushState({}, "", "/");
  resetConsolePreferencesStoreForTests();
});

test("settings stays reachable when gateway cookies are missing", async () => {
  window.history.pushState({}, "", "/settings");
  clearGatewayConnectionCookies();

  render(<DesignSourceAppRoot />);

  expect((await screen.findAllByLabelText("Gateway URL")).length).toBeGreaterThan(0);
  expect(screen.queryByText(/Gateway 连接未配置/)).not.toBeInTheDocument();
});
