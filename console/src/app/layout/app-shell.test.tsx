import "@testing-library/jest-dom/vitest";
import { render } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import {
  clearGatewayConnectionCookies,
  saveGatewayConnectionToCookies,
} from "../../common/config/gateway-connection-store";
import { AppShell } from "./app-shell";

afterEach(() => {
  clearGatewayConnectionCookies();
});

test("keeps the app shell sized to the viewport so feature pages can fill the screen", () => {
  saveGatewayConnectionToCookies({
    gatewayUrl: "http://localhost:18080",
    apiKey: "shell-test-key",
  });

  const { container } = render(
    <MemoryRouter initialEntries={["/settings"]}>
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/settings" element={<div>settings</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );

  expect(container.firstElementChild).toHaveClass("size-full");
  expect(container.firstElementChild).toHaveClass("relative");
  expect(container.firstElementChild).toHaveClass("overflow-hidden");
});
