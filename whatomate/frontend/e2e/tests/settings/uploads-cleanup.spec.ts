import { test, expect } from "@playwright/test";

const adminUser = {
  id: "user-1",
  email: "admin@test.com",
  full_name: "Test Admin",
  organization_id: "org-1",
  is_super_admin: false,
  role: {
    id: "role-1",
    name: "admin",
    permissions: [
      "settings.general:read",
      "settings.general:write",
      "settings.uploads_cleanup:read",
      "settings.uploads_cleanup:write",
      "settings.uploads_cleanup:execute",
    ],
  },
  settings: {},
};

test.describe("Uploads cleanup settings", () => {
  test("uses a time input and persists a whole-hour schedule", async ({
    page,
  }) => {
    let savedPayload: Record<string, unknown> | null = null;

    await page.route("**/api/me", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: adminUser }),
      });
    });

    await page.route("**/api/users/me", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { settings: {} } }),
      });
    });

    await page.route("**/api/me/organizations", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: [{ id: "org-1", name: "Whatomate", slug: "whatomate" }],
        }),
      });
    });

    await page.route("**/api/config", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { whatsapp_provider: "meta" } }),
      });
    });

    await page.route("**/api/auth/ws-token", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { token: "test-token" } }),
      });
    });

    await page.route("**/api/org/settings", async (route) => {
      if (route.request().method() === "PUT") {
        savedPayload = route.request().postDataJSON() as Record<
          string,
          unknown
        >;
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            data: { message: "Settings updated successfully" },
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            name: "Whatomate",
            slug: "whatomate",
            settings: {
              timezone: "UTC",
              date_format: "YYYY-MM-DD",
              mask_phone_numbers: false,
              uploads_cleanup_retention_days: 5,
              uploads_cleanup_schedule_hour: 3,
            },
          },
        }),
      });
    });

    await page.goto("/settings");

    const scheduleInput = page.getByTestId(
      "uploads-cleanup-schedule-hour-input",
    );
    await expect(scheduleInput).toBeVisible();
    await expect(scheduleInput).toHaveAttribute("type", "time");
    await expect(scheduleInput).toHaveValue("03:00");

    await page.getByTestId("uploads-cleanup-retention-days-input").fill("5");
    await scheduleInput.fill("05:00");
    await page.getByTestId("uploads-cleanup-save").click();

    await expect
      .poll(() => savedPayload)
      .toMatchObject({
        uploads_cleanup_retention_days: 5,
        uploads_cleanup_schedule_hour: 5,
      });
  });
});
