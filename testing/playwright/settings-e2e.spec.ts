import { expect, test } from "../../console/playwright-test";
import fs from "node:fs";
import path from "node:path";

const clientHome = process.env.SETTINGS_E2E_CLIENT_HOME ?? "";

function configPath(): string {
  return path.join(clientHome, ".codex", "config.toml");
}

function readConfig(): string {
  const target = configPath();
  if (!fs.existsSync(target)) {
    return "";
  }
  return fs.readFileSync(target, "utf8");
}

test("applies global default config to machine", async ({ page }) => {
  await page.goto("/settings", { waitUntil: "networkidle" });

  await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
  await page.getByLabel("Global Default TOML").fill("model = \"gpt-5.4\"\napproval_policy = \"never\"\n");
  await page.getByRole("button", { name: "Save Global Default" }).click();
  await expect(page.getByText("Global default saved.")).toBeVisible();
  await page.getByRole("button", { name: "Apply To Machine" }).click();
  await expect(page.getByText(/Applied global default to /)).toBeVisible();

  await expect.poll(() => readConfig()).toContain("model = \"gpt-5.4\"");
  await expect.poll(() => readConfig()).toContain("approval_policy = \"never\"");
});

test("applies machine override config to machine", async ({ page }) => {
  await page.goto("/settings", { waitUntil: "networkidle" });
  await expect(page.getByText("Using global default")).toBeVisible();

  await page.getByLabel("Machine Override TOML").fill("model = \"gpt-5.2\"\n");
  await expect(page.getByLabel("Machine Override TOML")).toHaveValue("model = \"gpt-5.2\"\n");
  await page.getByRole("button", { name: "Save Machine Override" }).click();
  await expect(page.getByText("Machine override saved.")).toBeVisible();
  await page.getByRole("button", { name: "Apply To Machine" }).click();
  await expect(page.getByText(/Applied machine override to /)).toBeVisible();

  await expect.poll(() => readConfig()).toBe("model = \"gpt-5.2\"\n");
});

test("deleting machine override falls back to global default", async ({ page }) => {
  await page.goto("/settings", { waitUntil: "networkidle" });
  await expect(page.getByLabel("Global Default TOML")).not.toHaveValue("");

  await page.getByLabel("Global Default TOML").fill("model = \"gpt-5.4\"\n");
  await page.getByRole("button", { name: "Save Global Default" }).click();
  await expect(page.getByText("Global default saved.")).toBeVisible();
  await page.getByLabel("Machine Override TOML").fill("model = \"gpt-5.2\"\n");
  await page.getByRole("button", { name: "Save Machine Override" }).click();
  await expect(page.getByText("Machine override saved.")).toBeVisible();
  await page.getByRole("button", { name: "Delete Machine Override" }).click();
  await expect(page.getByText("Machine override deleted.")).toBeVisible();
  await page.getByRole("button", { name: "Apply To Machine" }).click();
  await expect(page.getByText(/Applied global default to /)).toBeVisible();

  await expect.poll(() => readConfig()).toBe("model = \"gpt-5.4\"\n");
});

test("invalid toml blocks save", async ({ page }) => {
  await page.goto("/settings", { waitUntil: "networkidle" });
  await expect(page.getByLabel("Global Default TOML")).not.toHaveValue("");

  await page.getByLabel("Global Default TOML").fill("model = [");
  await expect(page.getByLabel("Global Default TOML")).toHaveValue("model = [");
  await page.getByRole("button", { name: "Save Global Default" }).click();

  await expect(page.getByText("Invalid TOML content.")).toBeVisible();
});
