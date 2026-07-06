import { test, expect, type Page } from "@playwright/test";
import { loginAsAgent } from "../../helpers";

const AUTH_AGENT = {
  id: "agent-user-id",
  email: "agent@test.com",
  full_name: "Assigned Agent",
  organization_id: "e2e-org-id",
  role: {
    id: "agent-role-id",
    name: "agent",
    is_system: false,
    permissions: [{ id: "perm-chat-read", resource: "chat", action: "read" }],
  },
  is_super_admin: false,
};

function buildChat(
  id: string,
  name: string,
  status: "open" | "pending",
  assignedUserId?: string | null,
) {
  const now = new Date(Date.UTC(2026, 0, 1, 9, 0, 0)).toISOString();
  return {
    id,
    instance_id: "instance-alpha",
    phone_number: `1555${id.slice(0, 4)}`,
    name,
    profile_name: name,
    avatar_url: "",
    status,
    tags: [],
    metadata: {},
    last_message_at: now,
    last_message_preview: `${name} preview`,
    unread_count: 0,
    assigned_user_id: assignedUserId ?? null,
    assigned_user_name: assignedUserId
      ? assignedUserId === AUTH_AGENT.id
        ? AUTH_AGENT.full_name
        : "Other Agent"
      : "",
    is_public: false,
    created_at: now,
    updated_at: now,
  };
}

function contactsEnvelope(contacts: ReturnType<typeof buildChat>[]) {
  return {
    status: "success",
    data: {
      contacts,
      total: contacts.length,
      page: 1,
      limit: 50,
    },
  };
}

async function mockChatBootstrapAPIs(page: Page, openRequestQueries: string[]) {
  const myAssignedChat = buildChat(
    "open-self-1",
    "My Assigned Chat",
    "open",
    AUTH_AGENT.id,
  );
  const otherAssignedChat = buildChat(
    "open-other-1",
    "Other Agent Chat",
    "open",
    "other-agent-id",
  );
  const pendingQueueChat = buildChat("pending-1", "Pending Queue Chat", "pending");

  await page.route("**/api/contacts?*", async (route) => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(contactsEnvelope([myAssignedChat, pendingQueueChat])),
    });
  });

  await page.route("**/api/contacts", async (route) => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(contactsEnvelope([myAssignedChat, pendingQueueChat])),
    });
  });

  await page.route("**/api/instances**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (
      request.method() !== "GET" ||
      !url.pathname.endsWith("/api/instances")
    ) {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "success",
        data: [
          { id: "instance-alpha", name: "Instance Alpha", status: "connected" },
        ],
      }),
    });
  });

  await page.route("**/api/chatbot/transfers**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (
      request.method() !== "GET" ||
      !url.pathname.endsWith("/api/chatbot/transfers")
    ) {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "success",
        data: {
          transfers: [],
          general_queue_count: 0,
          team_queue_counts: {},
          total_count: 0,
        },
      }),
    });
  });

  await page.route("**/api/tags**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() !== "GET" || !url.pathname.endsWith("/api/tags")) {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "success",
        data: {
          tags: [],
          total: 0,
          page: 1,
          limit: 50,
        },
      }),
    });
  });

  await page.route("**/api/chats*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() !== "GET" || !url.pathname.endsWith("/api/chats")) {
      await route.fallback();
      return;
    }

    const status = (url.searchParams.get("status") || "").trim().toLowerCase();
    if (status === "open") {
      openRequestQueries.push(url.search);
      const assignedTo = (url.searchParams.get("assigned_to") || "")
        .trim()
        .toLowerCase();
      const contacts =
        assignedTo === "me" ? [myAssignedChat] : [myAssignedChat, otherAssignedChat];

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: {
            contacts,
            total: contacts.length,
            page: 1,
            limit: 50,
          },
        }),
      });
      return;
    }

    if (status === "pending") {
      const contacts = [pendingQueueChat];
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: {
            contacts,
            total: contacts.length,
            page: 1,
            limit: 50,
          },
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "success",
        data: {
          contacts: [],
          total: 0,
          page: 1,
          limit: 50,
        },
      }),
    });
  });
}

test.describe("Chat Assigned Default Tab", () => {
  test("opens on Assigned by default for agents and only requests my assigned chats", async ({
    page,
  }) => {
    const openRequestQueries: string[] = [];
    await mockChatBootstrapAPIs(page, openRequestQueries);
    await loginAsAgent(page);

    await page.goto("/chat");
    await page.waitForLoadState("networkidle");

    await expect(
      page.locator('[data-testid="chat-sidebar-entry"]'),
    ).toHaveCount(1);
    await expect(
      page.locator('[data-testid="chat-sidebar-entry"]'),
    ).toContainText("My Assigned Chat");
    await expect(page.locator("body")).not.toContainText("Other Agent Chat");
    expect(
      openRequestQueries.some((query) => query.includes("assigned_to=me")),
    ).toBeTruthy();

    await page.getByRole("button", { name: /Pending/i }).click();
    await expect(
      page.locator('[data-testid="chat-sidebar-entry"]'),
    ).toContainText("Pending Queue Chat");
  });
});
