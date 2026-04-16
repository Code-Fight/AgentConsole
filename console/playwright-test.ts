// Shim for centralized specs under /testing/playwright to import Playwright
// from the console package without hardcoding a node_modules path.
export { expect, test } from "@playwright/test";
