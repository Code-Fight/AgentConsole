import { expect, test } from "../../console/playwright-test";
import type { Locator, Page } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

const clientHome = process.env.SETTINGS_E2E_CLIENT_HOME ?? "";

function configPath(): string {
  const machineStatePath = path.join(clientHome, ".code-agent-gateway", "machine.json");
  let machineId = "";
  if (fs.existsSync(machineStatePath)) {
    try {
      const machineState = JSON.parse(fs.readFileSync(machineStatePath, "utf8")) as {
        machineId?: string;
      };
      machineId = machineState.machineId ?? "";
    } catch {
      machineId = "";
    }
  }

  if (machineId !== "") {
    return path.join(
      clientHome,
      ".code-agent-gateway",
      "machines",
      machineId,
      "agents",
      "agent-01",
      "home",
      ".codex",
      "config.toml",
    );
  }

  return path.join(clientHome, ".codex", "config.toml");
}

function readConfig(): string {
  const target = configPath();
  if (!fs.existsSync(target)) {
    return "";
  }
  return fs.readFileSync(target, "utf8");
}

function settingsScope(page: Page): Locator {
  return page.getByRole("main").first();
}

function apiConfigurationCard(scope: Locator): Locator {
  return scope
    .getByRole("heading", { name: "API Configuration" })
    .first()
    .locator(
      "xpath=ancestor::div[.//*[@aria-label='Gateway URL'] and .//*[@aria-label='Gateway API Key'] and .//*[@aria-label='Save Gateway Connection']][1]",
    );
}

function globalDefaultCard(scope: Locator): Locator {
  return scope
    .getByRole("heading", { name: "Global Default" })
    .first()
    .locator(
      "xpath=ancestor::div[.//*[@aria-label='Global Default TOML'] and .//*[@aria-label='Save Global Default']][1]",
    );
}

function machineOverrideCard(scope: Locator): Locator {
  return scope
    .getByRole("heading", { name: "Machine Override" })
    .first()
    .locator(
      "xpath=ancestor::div[.//*[@aria-label='Machine Override TOML'] and .//*[@aria-label='Save Machine Override']][1]",
    );
}

async function clickEnabledButton(scope: Locator, ariaLabel: string): Promise<void> {
  const button = scope.getByRole("button", { name: ariaLabel });
  await expect(button).toBeEnabled();
  await button.click();
}

async function ensureGatewayConnection(page: Page) {
  const gatewayUrl = process.env.SETTINGS_E2E_GATEWAY_URL ?? "http://127.0.0.1:14174/api";
  const apiKey = process.env.SETTINGS_E2E_GATEWAY_API_KEY ?? "settings-e2e-key";
  await page.goto("/settings", { waitUntil: "networkidle" });
  const scope = settingsScope(page);
  const apiConfig = apiConfigurationCard(scope);
  await apiConfig.getByLabel("Gateway URL").fill(gatewayUrl);
  await apiConfig.getByLabel("Gateway API Key").fill(apiKey);
  await clickEnabledButton(apiConfig, "Save Gateway Connection");
  await expect(scope.getByText(/Gateway connection saved/).first()).toBeVisible();
  await page.goto("/settings", { waitUntil: "networkidle" });
}

test("prompts for gateway connection before remote settings load", async ({ page }) => {
  await page.goto("/", { waitUntil: "networkidle" });
  await expect(page.getByText(/Gateway 连接未配置/)).toBeVisible();
  await page.goto("/settings", { waitUntil: "networkidle" });
  const scope = settingsScope(page);
  const apiConfig = apiConfigurationCard(scope);
  await apiConfig
    .getByLabel("Gateway URL")
    .fill(process.env.SETTINGS_E2E_GATEWAY_URL ?? "http://127.0.0.1:14174/api");
  await apiConfig
    .getByLabel("Gateway API Key")
    .fill(process.env.SETTINGS_E2E_GATEWAY_API_KEY ?? "settings-e2e-key");
  await clickEnabledButton(apiConfig, "Save Gateway Connection");
  await expect(scope.getByText(/Gateway connection saved/).first()).toBeVisible();
});

test("applies global default config to machine", async ({ page }) => {
  await ensureGatewayConnection(page);
  const scope = settingsScope(page);
  const globalDefault = globalDefaultCard(scope);

  await expect(scope.getByRole("heading", { name: "Settings" })).toBeVisible();
  await globalDefault.getByLabel("Global Default TOML").fill("model = \"gpt-5.4\"\napproval_policy = \"never\"\n");
  await clickEnabledButton(globalDefault, "Save Global Default");
  await expect(scope.getByText("Global default saved.").first()).toBeVisible();
  await clickEnabledButton(machineOverrideCard(scope), "Apply To Machine");
  await expect(scope.getByText(/Applied global default to /).first()).toBeVisible();

  await expect.poll(() => readConfig()).toContain("model = \"gpt-5.4\"");
  await expect.poll(() => readConfig()).toContain("approval_policy = \"never\"");
});

test("applies machine override config to machine", async ({ page }) => {
  await ensureGatewayConnection(page);
  const scope = settingsScope(page);
  const machineOverride = machineOverrideCard(scope);
  await expect(scope.getByText("Using global default").first()).toBeVisible();

  await machineOverride.getByLabel("Machine Override TOML").fill("model = \"gpt-5.2\"\n");
  await expect(machineOverride.getByLabel("Machine Override TOML")).toHaveValue("model = \"gpt-5.2\"\n");
  await clickEnabledButton(machineOverride, "Save Machine Override");
  await expect(scope.getByText("Machine override saved.").first()).toBeVisible();
  await clickEnabledButton(machineOverride, "Apply To Machine");
  await expect(scope.getByText(/Applied machine override to /).first()).toBeVisible();

  await expect.poll(() => readConfig()).toBe("model = \"gpt-5.2\"\n");
});

test("deleting machine override falls back to global default", async ({ page }) => {
  await ensureGatewayConnection(page);
  const scope = settingsScope(page);
  const globalDefault = globalDefaultCard(scope);
  const machineOverride = machineOverrideCard(scope);
  await expect(globalDefault.getByLabel("Global Default TOML")).not.toHaveValue("");

  await globalDefault.getByLabel("Global Default TOML").fill("model = \"gpt-5.4\"\n");
  await clickEnabledButton(globalDefault, "Save Global Default");
  await expect(scope.getByText("Global default saved.").first()).toBeVisible();
  await machineOverride.getByLabel("Machine Override TOML").fill("model = \"gpt-5.2\"\n");
  await clickEnabledButton(machineOverride, "Save Machine Override");
  await expect(scope.getByText("Machine override saved.").first()).toBeVisible();
  await clickEnabledButton(machineOverride, "Delete Machine Override");
  await expect(scope.getByText("Machine override deleted.").first()).toBeVisible();
  await clickEnabledButton(machineOverride, "Apply To Machine");
  await expect(scope.getByText(/Applied global default to /).first()).toBeVisible();

  await expect.poll(() => readConfig()).toBe("model = \"gpt-5.4\"\n");
});

test("invalid toml blocks save", async ({ page }) => {
  await ensureGatewayConnection(page);
  const scope = settingsScope(page);
  const globalDefault = globalDefaultCard(scope);
  await expect(globalDefault.getByLabel("Global Default TOML")).not.toHaveValue("");

  await globalDefault.getByLabel("Global Default TOML").fill("model = [");
  await expect(globalDefault.getByLabel("Global Default TOML")).toHaveValue("model = [");
  await clickEnabledButton(globalDefault, "Save Global Default");

  await expect(scope.getByText("Invalid TOML content.").first()).toBeVisible();
});
