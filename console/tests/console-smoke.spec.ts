import { expect, test } from "@playwright/test";

test("navigates to thread workspace", async ({ page }) => {
  await page.goto("/");
  await page.getByText("Threads").click();
  await expect(page.getByRole("heading", { name: "Threads" })).toBeVisible();
  await page.getByRole("link", { name: "Open thread-1" }).click();
  await expect(page.getByText("Thread Workspace")).toBeVisible();
});
