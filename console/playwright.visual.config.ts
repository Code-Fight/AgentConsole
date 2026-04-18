import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "../testing/playwright",
  testMatch: ["console-visual-regression.spec.ts"],
  snapshotPathTemplate: "../testing/playwright/{testFilePath}-snapshots/{arg}{ext}",
  outputDir: "./playwright-report/visual-results",
  workers: 1,
  use: {
    ...devices["Desktop Chrome"],
    baseURL: "http://127.0.0.1:4174",
    viewport: {
      width: 1440,
      height: 1024,
    },
    locale: "zh-CN",
    timezoneId: "Asia/Shanghai",
    colorScheme: "dark",
  },
  webServer: {
    command: "corepack pnpm --ignore-workspace dev --host 127.0.0.1 --port 4174",
    url: "http://127.0.0.1:4174",
    reuseExistingServer: false,
  },
});
