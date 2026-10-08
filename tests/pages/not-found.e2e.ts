import { test } from "@e2e-dev/web";
import { expect } from "e2e";
import { prepareFreshBrowserSession } from "../helpers/auth-session";

test("unauthenticated unknown routes preserve their destination and ask the user to sign in", async ({
  app,
  browser,
  screen,
  agent,
}) => {
  await app.open("/e2e-route-that-does-not-exist");

  await expect(browser).toHaveURL(/\/login\?redirect=/);
  await expect(screen.getByRole("heading", "Welcome to Gowa-UI")).toBeVisible();

  await agent.act(
    "Verify an unauthenticated visit to an unknown route is redirected to sign-in and retains a redirect destination. Do not enter credentials.",
  );

  await expect(browser).toHaveURL(/\/login\?redirect=/);
});

for (const session of ["super-admin", "user"] as const) {
  test(
    `authenticated unknown routes show a clear not-found state (${session})`,
    { session, tags: ["page", "authenticated", session] },
    async ({ app, browser, screen, agent }) => {
      await prepareFreshBrowserSession(
        app,
        browser,
        session === "user" ? "user" : "super-admin",
      );
      await app.open("/e2e-route-that-does-not-exist");

      await expect(screen.getByRole("heading", "404")).toBeVisible();
      await expect(screen.getByRole("heading", "Page Not Found")).toBeVisible();
      await expect(screen.getByRole("link", "Go Home")).toBeVisible();

      await agent.act(
        `Review the not-found explanation as a ${session} and verify the home recovery link is clear. Do not leave the page.`,
      );

      await expect(browser).toHaveURL("/e2e-route-that-does-not-exist");
    },
  );
}
