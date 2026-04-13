import { expect, test } from "@playwright/test";

test("navigates to thread workspace", async ({ page }) => {
  await page.route("**/threads", async (route) => {
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

  await page.goto("/");
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole("button", { name: "机器管理" })).toBeVisible();
  await expect(page.getByRole("button", { name: "环境资源" })).toBeVisible();
  await expect(page.getByRole("button", { name: "设置" })).toBeVisible();
  await expect(page.getByText("Gateway Thread 1", { exact: true })).toBeVisible();
  await expect(page.getByText("实现 CodexClient 线程缓存逻辑")).toHaveCount(0);

  await page.getByText("Gateway Thread 1", { exact: true }).click();
  await expect(page).toHaveURL(/\/threads\/thread-1$/);
  await expect(page.getByPlaceholder(/发送指令/).first()).toBeVisible();
});
