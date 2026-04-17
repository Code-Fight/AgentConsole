import { expect, test } from "../../console/playwright-test";
import type { Page, Route } from "@playwright/test";

async function seedGatewayCookies(page: Page) {
  await page.context().addCookies([
    {
      name: "cag_gateway_url",
      value: "http://127.0.0.1:4173",
      url: "http://127.0.0.1:4173",
    },
    {
      name: "cag_gateway_api_key",
      value: "test-key",
      url: "http://127.0.0.1:4173",
    },
  ]);
}

function expectAuth(route: Route) {
  expect(route.request().headers()["authorization"]).toBe("Bearer test-key");
}

test("navigates to thread workspace", async ({ page }) => {
  await page.route("**/overview/metrics", async (route) => {
    expectAuth(route);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        onlineMachines: 1,
        activeThreads: 1,
        pendingApprovals: 0,
        runningAgents: 1,
        environmentItems: 3,
      })
    });
  });

  await page.route("**/threads", async (route) => {
    expectAuth(route);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        items: [
          {
            threadId: "thread-1",
            machineId: "machine-1",
            status: "idle",
            title: "Gateway Thread 1"
          }
        ]
      })
    });
  });

  await page.route("**/machines", async (route) => {
    expectAuth(route);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        items: [
          {
            id: "machine-1",
            name: "Machine One",
            status: "online",
            runtimeStatus: "running"
          }
        ]
      })
    });
  });

  await page.route("**/threads/thread-1", async (route) => {
    expectAuth(route);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        thread: {
          threadId: "thread-1",
          machineId: "machine-1",
          title: "Gateway Thread 1",
          status: "idle"
        },
        activeTurnId: null,
        pendingApprovals: []
      })
    });
  });

  await page.route("**/machines/machine-1", async (route) => {
    expectAuth(route);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        machine: {
          id: "machine-1",
          name: "Machine One",
          status: "online",
          runtimeStatus: "running"
        }
      })
    });
  });

  await seedGatewayCookies(page);
  await page.goto("/");
  await expect(page).toHaveURL(/\/$/);
  await page.getByRole("button", { name: "概览" }).click();
  await expect(page).toHaveURL(/\/overview$/);
  await expect(page.getByText("Online Machines").first()).toBeVisible();
  await expect(page.getByText("Running Agents").first()).toBeVisible();
  await expect(page.getByText("Active Threads").first()).toBeVisible();

  await seedGatewayCookies(page);
  await page.goto("/");
  await expect(page.getByRole("button", { name: "机器管理" })).toBeVisible();
  await expect(page.getByRole("button", { name: "环境资源" })).toBeVisible();
  await expect(page.getByRole("button", { name: "设置" })).toBeVisible();
  await expect(page.getByText("Gateway Thread 1", { exact: true })).toBeVisible();
  await expect(page.getByText("实现 CodexClient 线程缓存逻辑")).toHaveCount(0);

  await page.getByText("Gateway Thread 1", { exact: true }).click();
  await expect(page).toHaveURL(/\/threads\/thread-1$/);
  await expect(page.getByPlaceholder(/发送指令/).first()).toBeVisible();
});
