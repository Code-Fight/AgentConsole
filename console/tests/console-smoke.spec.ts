import { expect, test } from "@playwright/test";

test("navigates to thread workspace", async ({ page }) => {
  await page.goto("/");
  await page.getByText("Threads").click();
  await expect(page.getByText("Thread Workspace")).toBeVisible();
});
