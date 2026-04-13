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
  await expect(page.getByRole("link", { name: "Machines" })).toHaveAttribute("href", "/machines");
  await expect(page.getByRole("link", { name: "Environment" })).toHaveAttribute("href", "/environment");
  await expect(page.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/settings");
  await page.getByRole("link", { name: "Gateway Thread 1", exact: true }).click();
  await expect(page).toHaveURL(/\/threads\/thread-1$/);
  await expect(page.getByRole("textbox", { name: "Prompt" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Send prompt" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Gateway Thread 1Machine One" })).toHaveAttribute("href", "/threads/thread-1");
});
