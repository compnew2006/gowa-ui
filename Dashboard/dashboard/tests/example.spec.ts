import { expect, test } from "@playwright/test";

test("login shell is RTL and available without a session", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("dir", "rtl");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "تسجيل الدخول" })).toBeVisible();
});

test("unknown private routes do not expose app chrome to anonymous visitors", async ({ page }) => {
  await page.goto("/console");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "تسجيل الدخول" })).toBeVisible();
});

test("public registration is closed by default", async ({ page }) => {
  await page.goto("/register");
  await expect(page.getByRole("heading", { name: "التسجيل مغلق" })).toBeVisible();
});
