import { expect, test } from "../../console/playwright-test";
import type { Page } from "@playwright/test";
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

async function clickEnabledButton(page: Page, ariaLabel: string): Promise<void> {
  await page
    .locator(`button[aria-label="${ariaLabel}"]:visible:not([disabled])`)
    .first()
    .click();
}

async function ensureGatewayConnection(page: Page) {
  const gatewayUrl = process.env.SETTINGS_E2E_GATEWAY_URL ?? "http://127.0.0.1:18081";
  const apiKey = process.env.SETTINGS_E2E_GATEWAY_API_KEY ?? "settings-e2e-key";
  await page.goto("/settings", { waitUntil: "networkidle" });
  await page.evaluate(
    ({ nextGatewayUrl, nextApiKey }) => {
      document.cookie = `cag_gateway_url=${encodeURIComponent(nextGatewayUrl)}; Path=/; SameSite=Lax`;
      document.cookie = `cag_gateway_api_key=${encodeURIComponent(nextApiKey)}; Path=/; SameSite=Lax`;
    },
    { nextGatewayUrl: gatewayUrl, nextApiKey: apiKey },
  );
  await page.goto("/settings", { waitUntil: "networkidle" });
}

test("prompts for gateway connection before remote settings load", async ({ page }) => {
  await page.goto("/", { waitUntil: "networkidle" });
  await expect(page.getByText(/Gateway 连接未配置/)).toBeVisible();
  await page.goto("/settings", { waitUntil: "networkidle" });
  await page
    .locator('input[aria-label="Gateway URL"]:visible')
    .first()
    .fill(process.env.SETTINGS_E2E_GATEWAY_URL ?? "http://127.0.0.1:18081");
  await page
    .locator('input[aria-label="Gateway API Key"]:visible')
    .first()
    .fill(process.env.SETTINGS_E2E_GATEWAY_API_KEY ?? "settings-e2e-key");
  await clickEnabledButton(page, "Save Gateway Connection");
  await expect(page.getByText(/Gateway connection saved/).first()).toBeVisible();
});

test("applies global default config to machine", async ({ page }) => {
  await ensureGatewayConnection(page);

  await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
  await page
    .locator('textarea[aria-label="Global Default TOML"]:visible')
    .first()
    .fill("model = \"gpt-5.4\"\napproval_policy = \"never\"\n");
  await clickEnabledButton(page, "Save Global Default");
  await expect(page.getByText("Global default saved.").first()).toBeVisible();
  await clickEnabledButton(page, "Apply To Machine");
  await expect(page.getByText(/Applied global default to /).first()).toBeVisible();

  await expect.poll(() => readConfig()).toContain("model = \"gpt-5.4\"");
  await expect.poll(() => readConfig()).toContain("approval_policy = \"never\"");
});

test("applies machine override config to machine", async ({ page }) => {
  await ensureGatewayConnection(page);
  await expect(page.getByText("Using global default").first()).toBeVisible();

  await page
    .locator('textarea[aria-label="Machine Override TOML"]:visible')
    .first()
    .fill("model = \"gpt-5.2\"\n");
  await expect(page.locator('textarea[aria-label="Machine Override TOML"]:visible').first()).toHaveValue(
    "model = \"gpt-5.2\"\n",
  );
  await clickEnabledButton(page, "Save Machine Override");
  await expect(page.getByText("Machine override saved.").first()).toBeVisible();
  await clickEnabledButton(page, "Apply To Machine");
  await expect(page.getByText(/Applied machine override to /).first()).toBeVisible();

  await expect.poll(() => readConfig()).toBe("model = \"gpt-5.2\"\n");
});

test("deleting machine override falls back to global default", async ({ page }) => {
  await ensureGatewayConnection(page);
  await expect(page.locator('textarea[aria-label="Global Default TOML"]:visible').first()).not.toHaveValue("");

  await page
    .locator('textarea[aria-label="Global Default TOML"]:visible')
    .first()
    .fill("model = \"gpt-5.4\"\n");
  await clickEnabledButton(page, "Save Global Default");
  await expect(page.getByText("Global default saved.").first()).toBeVisible();
  await page
    .locator('textarea[aria-label="Machine Override TOML"]:visible')
    .first()
    .fill("model = \"gpt-5.2\"\n");
  await clickEnabledButton(page, "Save Machine Override");
  await expect(page.getByText("Machine override saved.").first()).toBeVisible();
  await clickEnabledButton(page, "Delete Machine Override");
  await expect(page.getByText("Machine override deleted.").first()).toBeVisible();
  await clickEnabledButton(page, "Apply To Machine");
  await expect(page.getByText(/Applied global default to /).first()).toBeVisible();

  await expect.poll(() => readConfig()).toBe("model = \"gpt-5.4\"\n");
});

test("invalid toml blocks save", async ({ page }) => {
  await ensureGatewayConnection(page);
  await expect(page.locator('textarea[aria-label="Global Default TOML"]:visible').first()).not.toHaveValue("");

  await page.locator('textarea[aria-label="Global Default TOML"]:visible').first().fill("model = [");
  await expect(page.locator('textarea[aria-label="Global Default TOML"]:visible').first()).toHaveValue(
    "model = [",
  );
  await clickEnabledButton(page, "Save Global Default");

  await expect(page.getByText("Invalid TOML content.").first()).toBeVisible();
});
