import { test } from "@e2e-dev/web";
import { expect } from "e2e";

test("SSO callback redirects to sign-in when no provider session exists", async ({
  app,
  browser,
  screen,
  agent,
}) => {
  await app.open("/auth/sso/callback");

  await expect(browser).toHaveURL(/\/login\?redirect=%2Fauth%2Fsso%2Fcallback/);
  await expect(screen.getByRole("heading", "Welcome to Gowa-UI")).toBeVisible();

  await agent.act(
    "Verify an SSO callback without a provider session returns the visitor to sign-in and preserves the callback destination. Do not enter credentials.",
  );

  await expect(browser).toHaveURL(/\/login\?redirect=%2Fauth%2Fsso%2Fcallback/);
});

test("SSO callback shows a recovery link when session lookup has a server error", async ({
  app,
  browser,
  screen,
  agent,
}) => {
  await browser.route(/\/api\/me(?:\?|$)/, async (route) => {
    await route.fulfill({
      status: 503,
      json: { message: "Temporary session lookup failure" },
    });
  });

  await app.open("/auth/sso/callback");

  await expect(screen.getByRole("heading", "SSO Login Failed")).toBeVisible({
    timeout: 20_000,
  });
  await expect(screen.getByRole("link", "Return to login")).toBeVisible();

  await agent.act(
    "Review the SSO recovery error and confirm the Return to login link is clear and visible. Do not activate it or enter credentials.",
  );

  await screen.getByRole("link", "Return to login").tap();
  await expect(browser).toHaveURL("/login");
});
