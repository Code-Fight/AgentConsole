import { defineConfig } from "@playwright/test";

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? "http://127.0.0.1:14174";

export default defineConfig({
  testDir: "./tests",
  testMatch: ["settings-e2e.spec.ts"],
  outputDir: "./playwright-report/settings-e2e-results",
  workers: 1,
  use: {
    baseURL
  }
});
