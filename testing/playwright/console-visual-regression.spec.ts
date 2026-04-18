import { expect, test } from "../../console/playwright-test";
import {
  primeConsoleVisualRoutes,
  visualScenarios,
} from "./console-visual-regression.helpers";

test.beforeEach(async ({ page }) => {
  await primeConsoleVisualRoutes(page);
});

for (const scenario of visualScenarios) {
  test(`visual baseline: ${scenario.name}`, async ({ page }) => {
    await page.goto(scenario.path);
    if (scenario.ready.kind === "text") {
      await expect(page.getByText(scenario.ready.value).first()).toBeVisible();
    } else if (scenario.ready.kind === "label") {
      await expect(page.getByLabel(scenario.ready.value).first()).toBeVisible();
    } else {
      await expect(page.getByPlaceholder(scenario.ready.value).first()).toBeVisible();
    }
    const hasIbmPlexSans = await page.evaluate(async () => {
      await document.fonts.ready;
      return document.fonts.check('16px "IBM Plex Sans"');
    });
    if (!hasIbmPlexSans) {
      throw new Error('Font prerequisite failed: "IBM Plex Sans" is not available.');
    }
    await expect(page).toHaveScreenshot(`${scenario.snapshot}.png`, {
      fullPage: true,
      animations: "disabled",
    });
  });
}
