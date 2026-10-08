import { test } from "@e2e-dev/web";
import { expect } from "e2e";

test("registration explains how to get an invitation link", async ({
  app,
  browser,
  screen,
  agent,
}) => {
  await app.open("/register");

  await expect(screen.getByRole("heading", "Create an account")).toBeVisible();
  await expect(
    screen.getByText(/Registration is invitation-only/i),
  ).toBeVisible();
  await expect(screen.getByRole("link", "Sign in")).toBeVisible();
  await expect(browser.locator("input")).toHaveCount(0);

  await agent.act(
    "Review the invitation-only sign-up state. Verify it explains who can provide the registration link and how the user returns to sign-in.",
  );

  await expect(screen.getByText(/organization admin/i)).toBeVisible();
});

test("invited registration form validates required fields", async ({
  app,
  screen,
  agent,
}) => {
  await app.open("/register?org=e2e-example-organization");

  await expect(screen.getByRole("heading", "Create an account")).toBeVisible();
  await expect(screen.getByLabel("Full Name")).toBeVisible();
  await expect(screen.getByLabel("Email")).toBeVisible();
  await expect(screen.getByLabel("Password")).toBeVisible();
  await expect(screen.getByLabel("Confirm Password")).toBeVisible();

  await agent.act(
    "Submit the empty invitation-backed registration form once and verify required-field feedback. Do not enter or submit account data.",
  );

  await expect(screen.getByText("Please fill in all fields")).toBeVisible();
});
