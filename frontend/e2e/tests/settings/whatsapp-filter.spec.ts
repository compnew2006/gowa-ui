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
      "wa_filter:read",
      "wa_filter:write",
    ],
  },
  settings: {},
};

const MOCK_CONFIG = {
  status: "success",
  data: {
    whatsapp_provider: "meta",
    features: {
      templates: true,
      flows: true,
      catalog: true,
      business_profile: true,
      campaigns: true,
      meta_insights: true,
    },
  },
};

const MOCK_ACCOUNTS = {
  data: {
    accounts: [
      { id: "conn-123", name: "Meta Account Alpha", phone_id: "111", business_id: "222", status: "active" },
    ],
  },
};

// Generate 100 mock results for detailed testing of pagination and selections
const mockResults = Array.from({ length: 100 }, (_, index) => {
  const idNum = index + 1;
  const isValid = idNum % 3 !== 0; // 67 valid, 33 invalid
  return {
    id: `res-${idNum}`,
    batch_id: "batch-12345678",
    phone_number: `+1555000${String(idNum).padStart(4, "0")}`,
    contact_name: `Contact Name ${idNum}`,
    is_valid: isValid,
    error_message: isValid ? "" : "Not registered on WhatsApp",
    checked_at: "2026-05-29T21:05:00Z",
    created_at: "2026-05-29T21:05:00Z",
  };
});

test.describe("WhatsApp Filter Results Table E2E tests (wa_filter)", () => {
  let contactsCreatedCount = 0;

  test.beforeEach(async ({ page }) => {
    contactsCreatedCount = 0;

    // 1. Mock standard bootstrap and auth APIs with wildcard matching
    await page.route("**/api/license/bootstrap*", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            enabled: true,
            status: "active",
            locked: false,
            show_quota_overage: false,
          },
        }),
      });
    });

    await page.route("**/api/me", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: adminUser }),
      });
    });

    await page.route("**/api/users/me*", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { settings: {} } }),
      });
    });

    await page.route("**/api/me/organizations*", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: [{ id: "org-1", name: "Whatomate", slug: "whatomate" }],
        }),
      });
    });

    await page.route("**/api/config*", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CONFIG),
      });
    });

    await page.route("**/api/auth/ws-token*", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { token: "test-token" } }),
      });
    });

    await page.route("**/api/org/settings*", async (route) => {
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
            },
          },
        }),
      });
    });

    await page.route("**/api/accounts*", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_ACCOUNTS),
      });
    });

    // 2. Mock WhatsApp filter campaign batch and results endpoints using robust RegExp matching
    await page.route(/\/api\/whatsapp-filter\/batches/, async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes("/results")) {
        // Detailed paginated search/filter results - Return directly as flat object
        const urlObj = new URL(url);
        const pageParam = parseInt(urlObj.searchParams.get("page") || "1", 10);
        const limitParam = parseInt(urlObj.searchParams.get("limit") || "25", 10);
        const statusParam = urlObj.searchParams.get("status") || "all";
        const searchParam = (urlObj.searchParams.get("q") || "").toLowerCase().trim();

        let filtered = [...mockResults];

        if (statusParam === "valid") {
          filtered = filtered.filter(r => r.is_valid);
        } else if (statusParam === "invalid") {
          filtered = filtered.filter(r => !r.is_valid);
        }

        if (searchParam) {
          filtered = filtered.filter(
            r => r.phone_number.includes(searchParam) || r.contact_name.toLowerCase().includes(searchParam)
          );
        }

        const total = filtered.length;
        const start = (pageParam - 1) * limitParam;
        const end = start + limitParam;
        const items = filtered.slice(start, end);

        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            data: items,
            total: total,
            page: pageParam,
            limit: limitParam,
          }),
        });
        return;
      }

      if (url.includes("/batches/batch-12345678")) {
        // Detail fetch - Return batch directly at the root as expected by activeBatch.value = res.data
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            id: "batch-12345678",
            whatsapp_account: "Meta Account Alpha",
            instance_id: "conn-123",
            total_numbers: 100,
            valid_numbers: 67,
            invalid_numbers: 33,
            status: "completed",
            created_at: "2026-05-29T21:00:00Z",
          }),
        });
        return;
      }

      // Index batches list fetch - Return flat object with direct .data and .total as expected by batches.value = res.data.data
      if (method === "GET") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            data: [
              {
                id: "batch-12345678",
                whatsapp_account: "Meta Account Alpha",
                instance_id: "conn-123",
                total_numbers: 100,
                valid_numbers: 67,
                invalid_numbers: 33,
                status: "completed",
                created_at: "2026-05-29T21:00:00Z",
              },
            ],
            total: 1,
            page: 1,
            limit: 10,
          }),
        });
        return;
      }

      await route.continue();
    });

    // 3. Mock contact creation endpoint for importing selected contacts
    await page.route("**/api/contacts*", async (route) => {
      if (route.request().method() === "POST") {
        contactsCreatedCount++;
        await route.fulfill({
          status: 201,
          contentType: "application/json",
          body: JSON.stringify({ data: { id: `contact-${contactsCreatedCount}` } }),
        });
        return;
      }
      await route.continue();
    });
  });

  test("should display campaign log, view details, and operate selections/pagination correctly", async ({ page }) => {
    // Navigate directly to the WhatsApp number filter settings page
    await page.goto("/settings/whatsapp-filter");
    await page.waitForLoadState("networkidle");

    // Verify view is correctly initialized
    await expect(page.locator("h1")).toContainText("WhatsApp Number Filter");
    await expect(page.locator("table")).toBeVisible();
    await expect(page.locator("text=Meta Account Alpha")).toBeVisible();

    // Click "View Logs" to navigate to the detailed campaign results view
    await page.getByRole("button", { name: "View Logs" }).click();
    await page.waitForLoadState("networkidle");

    // Assert that we are in the detailed results view
    await expect(page.locator("text=Verification Results - Batch")).toBeVisible();
    await expect(page.locator("text=Verification Log")).toBeVisible();
    
    // Check initial bulk operation buttons are disabled (no selection)
    const exportButton = page.getByRole("button", { name: /Export Selected/i });
    const importButton = page.getByRole("button", { name: /Import Selected/i });
    await expect(exportButton).toBeDisabled();
    await expect(importButton).toBeDisabled();

    // Verify row 1 contact exists in the table (exact matching)
    await expect(page.getByRole("cell", { name: "Contact Name 1", exact: true })).toBeVisible();

    // --------------------------------------------------------
    // Scenario 1: Row-level checkbox selection
    // --------------------------------------------------------
    // Select the first row checkbox
    const row1Checkbox = page.getByRole("checkbox", { name: "Select row 1", exact: true });
    await row1Checkbox.click();

    // Assert selection banner updates to "Selected: 1 items"
    await expect(page.locator("text=Selected: 1 items")).toBeVisible();
    await expect(exportButton).toBeEnabled();
    await expect(importButton).toBeEnabled();
    await expect(exportButton).toContainText("Export Selected (1)");

    // Unselect first row
    await row1Checkbox.click();
    await expect(page.locator("text=Selected:")).not.toBeVisible();
    await expect(exportButton).toBeDisabled();

    // --------------------------------------------------------
    // Scenario 2: Select all on page via header checkbox
    // --------------------------------------------------------
    const headerCheckbox = page.getByRole("checkbox", { name: "Select all current page" });
    await headerCheckbox.click();

    // 25 rows per page default -> should show 25 records selected
    await expect(page.locator("text=Selected: 25 items")).toBeVisible();
    await expect(exportButton).toContainText("Export Selected (25)");

    // Assert "Select all matching records" prompt appears using correct locale
    const selectAllMatchingBtn = page.getByRole("button", { name: "Select all 100 matching items across all pages" });
    await expect(selectAllMatchingBtn).toBeVisible();

    // --------------------------------------------------------
    // Scenario 3: Select all matching records across all pages
    // --------------------------------------------------------
    await selectAllMatchingBtn.click();

    // Assert selection message updates to all matching selected using correct locale
    await expect(page.locator("text=All 100 matching items across all pages are selected.")).toBeVisible();
    await expect(exportButton).toContainText("Export Selected (100)");

    // Clear selection using the Clear button in the banner
    await page.getByRole("button", { name: "Clear Selection" }).click();
    await expect(page.locator("text=Selected:")).not.toBeVisible();
    await expect(exportButton).toBeDisabled();

    // --------------------------------------------------------
    // Scenario 4: Searching clears selection states
    // --------------------------------------------------------
    // Reselect header
    await headerCheckbox.click();
    await expect(page.locator("text=Selected: 25 items")).toBeVisible();

    // Enter a search query in the search field
    const searchInput = page.getByPlaceholder("Search by phone number or contact name...");
    await searchInput.fill("Contact Name 5");
    
    // Playwright search is debounced -> wait for search results to load
    await page.waitForTimeout(500);

    // Verify search results are displayed (exact matching)
    await expect(page.getByRole("cell", { name: "Contact Name 5", exact: true })).toBeVisible();
    // Selection state should be cleared automatically after filter parameters change
    await expect(page.locator("text=Selected:")).not.toBeVisible();

    // Clear search query
    await searchInput.clear();
    await page.waitForTimeout(500);

    // --------------------------------------------------------
    // Scenario 5: Contact Import operation execution
    // --------------------------------------------------------
    // Select first row (which is Contact Name 1, marked valid in our mock index % 3 !== 0)
    await page.getByRole("checkbox", { name: "Select row 1", exact: true }).click();
    await expect(page.locator("text=Selected: 1 items")).toBeVisible();

    // Click Import button
    await importButton.click();

    // Wait for the contacts create POST mock to trigger and show success toast using correct locale
    await expect(page.locator("text=Successfully imported 1 valid numbers into your contacts list.")).toBeVisible();
    // Verify our POST mock was indeed called once
    expect(contactsCreatedCount).toBe(1);

    // Selection should be cleared after a successful import
    await expect(page.locator("text=Selected:")).not.toBeVisible();
  });
});
