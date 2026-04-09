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
            title: "thread-1"
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

  await page.goto("/");
  await page.getByText("Threads").click();
  await expect(page.getByRole("heading", { name: "Threads" })).toBeVisible();
  await page.getByRole("link", { name: "thread-1" }).click();
  await expect(page.getByText("Thread Workspace")).toBeVisible();
});
