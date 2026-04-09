import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  testMatch: ["console-smoke.spec.ts"],
  outputDir: "./playwright-report/test-results",
  use: {
    baseURL: "http://127.0.0.1:4173"
  },
  webServer: {
    command: "corepack pnpm --ignore-workspace dev --host 127.0.0.1 --port 4173",
    url: "http://127.0.0.1:4173",
    reuseExistingServer: true
  }
});
