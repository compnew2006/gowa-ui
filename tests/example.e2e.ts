import { test } from "@e2e-dev/web";
import { expect } from "e2e";

test("app opens", async ({ app, browser, agent }) => {
  await app.open("/");
  await expect(browser.locator("body")).toBeVisible();
  await agent.act(
    "Check that the app has loaded and identify the current page without changing any data.",
  );
  await expect(browser.locator("body")).toBeVisible();
});
