import { test } from "@e2e-dev/web";
import { expect, secrets } from "e2e";

test("login validates required fields and keeps the password masked", async ({
  app,
  browser,
  screen,
  agent,
}) => {
  await app.open("/login");

  await expect(screen.getByRole("heading", "Welcome to Gowa-UI")).toBeVisible();
  await expect(screen.getByLabel("Email")).toBeVisible();
  await expect(screen.getByLabel("Password")).toBeVisible();

  await agent.act(
    "Submit the empty sign-in form once and verify the app asks for the required email and password.",
  );

  await expect(
    screen.getByText("Please enter email and password"),
  ).toBeVisible();
  await expect(browser.locator('input[type="password"]')).toHaveCount(1);
  await expect(browser).toHaveURL("/login");
});

test("login reports invalid credentials without exposing the password", async ({
  app,
  browser,
  screen,
  agent,
}) => {
  await app.open("/login");
  await screen.getByLabel("Email").fill("qa.user@example.invalid");
  await screen
    .getByLabel("Password")
    .fill(secrets.get("invalid-login-password"));

  await agent.act(
    "Submit these invalid sign-in details once. Confirm the app shows a useful authentication error and stays on the sign-in page. Do not inspect or repeat the password.",
  );

  await expect(screen.getByText("Invalid credentials")).toBeVisible();
  await expect(browser.locator('input[type="password"]')).toHaveCount(1);
  await expect(browser).toHaveURL("/login");
});
