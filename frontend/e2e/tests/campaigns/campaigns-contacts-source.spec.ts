import { expect, test, type Page } from "@playwright/test";
import { loginAsAdmin } from "../../helpers";
import { CampaignsPage } from "../../pages";

interface MockCampaignContact {
  id: string;
  phone_number: string;
  profile_name: string;
  instance_id: string;
  created_at: string;
}

const MOCK_CONFIG = {
  status: "success",
  data: {
    whatsapp_provider: "whatsmeow",
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

const MOCK_INSTANCES = {
  status: "success",
  data: [
    { id: "instance-alpha", name: "Instance Alpha", status: "connected" },
    { id: "instance-bravo", name: "Instance Bravo", status: "connected" },
  ],
};

const MOCK_CAMPAIGNS = {
  status: "success",
  data: {
    campaigns: [
      {
        id: "campaign-instance-alpha",
        name: "Instance Alpha Campaign",
        template_name: "",
        whatsapp_account: "instance-alpha",
        status: "draft",
        total_recipients: 0,
        sent_count: 0,
        delivered_count: 0,
        read_count: 0,
        failed_count: 0,
        created_at: "2026-03-01T10:00:00Z",
      },
    ],
    total: 1,
    page: 1,
    limit: 20,
  },
};

const CONTACTS: MockCampaignContact[] = [
  {
    id: "contact-alpha-created",
    phone_number: "+15550002001",
    profile_name: "Alpha Created",
    instance_id: "instance-alpha",
    created_at: "2026-03-03T10:00:00Z",
  },
  {
    id: "contact-alpha-inbound",
    phone_number: "+15550002002",
    profile_name: "Inbound Alpha",
    instance_id: "instance-alpha",
    created_at: "2026-01-15T10:00:00Z",
  },
  {
    id: "contact-bravo-created",
    phone_number: "+15550002003",
    profile_name: "Bravo Created",
    instance_id: "instance-bravo",
    created_at: "2026-03-04T10:00:00Z",
  },
];

const INBOUND_ACTIVITY_BY_CONTACT_ID: Record<string, string[]> = {
  "contact-alpha-created": ["2026-02-01"],
  "contact-alpha-inbound": ["2026-03-14"],
  "contact-bravo-created": ["2026-03-14"],
};

async function mockCampaignContactsSourceRoutes(
  page: Page,
  contactsRequests: string[],
) {
  await page.route("**/api/config*", async (route) => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_CONFIG),
    });
  });

  await page.route("**/api/instances*", async (route) => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_INSTANCES),
    });
  });

  await page.route("**/api/campaigns*", async (route) => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_CAMPAIGNS),
    });
  });

  await page.route("**/api/contacts*", async (route) => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }

    const url = new URL(route.request().url());
    contactsRequests.push(url.search);

    const instanceId = (url.searchParams.get("instance_id") || "").trim();
    const dateBasis = (url.searchParams.get("date_basis") || "created").trim();
    const dateFrom = (url.searchParams.get("date_from") || "").trim();
    const dateTo = (url.searchParams.get("date_to") || "").trim();
    const search = (url.searchParams.get("search") || "").trim().toLowerCase();
    const pageNumber = Math.max(1, Number(url.searchParams.get("page") || "1"));
    const limit = Math.max(1, Number(url.searchParams.get("limit") || "50"));

    let filtered = CONTACTS.filter((contact) =>
      instanceId ? contact.instance_id === instanceId : true,
    );

    if (dateBasis === "incoming_any") {
      filtered = filtered.filter((contact) => {
        const inboundDates = INBOUND_ACTIVITY_BY_CONTACT_ID[contact.id] || [];
        return inboundDates.some((date) => {
          if (dateFrom && date < dateFrom) return false;
          if (dateTo && date > dateTo) return false;
          return true;
        });
      });
    } else {
      filtered = filtered.filter((contact) => {
        const createdDate = contact.created_at.slice(0, 10);
        if (dateFrom && createdDate < dateFrom) return false;
        if (dateTo && createdDate > dateTo) return false;
        return true;
      });
    }

    if (search) {
      filtered = filtered.filter(
        (contact) =>
          contact.profile_name.toLowerCase().includes(search) ||
          contact.phone_number.toLowerCase().includes(search),
      );
    }

    const offset = (pageNumber - 1) * limit;
    const pageContacts = filtered.slice(offset, offset + limit);

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "success",
        data: {
          contacts: pageContacts,
          total: filtered.length,
          page: pageNumber,
          limit,
        },
      }),
    });
  });
}

test.describe("Campaign Contacts Source Filters", () => {
  test("filters contacts by selected instance and switches date basis to inbound activity", async ({
    page,
  }) => {
    const campaignsPage = new CampaignsPage(page);
    const contactsRequests: string[] = [];

    await loginAsAdmin(page);
    await mockCampaignContactsSourceRoutes(page, contactsRequests);
    await campaignsPage.goto();

    await campaignsPage.clickAddRecipientsButton();
    await campaignsPage.createDialog
      .getByRole("tab", { name: /From Contacts/i })
      .click();

    await expect(
      page.locator('[data-testid="campaign-contacts-scope-banner"]'),
    ).toContainText("Instance Alpha");
    await expect(campaignsPage.createDialog).toContainText("Alpha Created");
    await expect(campaignsPage.createDialog).not.toContainText("Bravo Created");
    expect(
      contactsRequests.some(
        (query) =>
          query.includes("instance_id=instance-alpha") &&
          query.includes("date_basis=created"),
      ),
    ).toBeTruthy();

    const alphaCreatedRow = campaignsPage.createDialog
      .locator("tr")
      .filter({ hasText: "Alpha Created" })
      .first();
    await alphaCreatedRow.getByRole("checkbox").click();
    await expect(campaignsPage.createDialog).toContainText("1 selected");

    await campaignsPage.createDialog
      .locator('[data-testid="campaign-contacts-date-basis-trigger"]')
      .click();
    await page
      .getByRole("option", { name: /Contacts that messaged us/i })
      .click();

    const dateInputs = campaignsPage.createDialog.locator('input[type="date"]');
    await dateInputs.nth(0).fill("2026-03-10");
    await dateInputs.nth(1).fill("2026-03-20");

    await expect(campaignsPage.createDialog).toContainText("Inbound Alpha");
    await expect(campaignsPage.createDialog).not.toContainText("Alpha Created");
    await expect(campaignsPage.createDialog).toContainText("0 selected");
    expect(
      contactsRequests.some(
        (query) =>
          query.includes("instance_id=instance-alpha") &&
          query.includes("date_basis=incoming_any") &&
          query.includes("date_from=2026-03-10") &&
          query.includes("date_to=2026-03-20"),
      ),
    ).toBeTruthy();
  });
});
