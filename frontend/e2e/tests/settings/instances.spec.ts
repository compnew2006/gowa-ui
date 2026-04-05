import { test, expect, type Page } from "@playwright/test";
import { loginAsAdmin } from "../../helpers";

interface MockInstance {
  id: string;
  name: string;
  status: string;
  is_default: boolean;
  auto_read_receipt: boolean;
  organization_id: string;
  settings?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

const HEALTH_PAYLOAD = {
  uptime_seconds: 0,
  messages_sent_today: 0,
  messages_received_today: 0,
  messages_failed_today: 0,
  error_rate_percent: 0,
  queue_depth: 0,
};

type MockHooks = {
  onCreate?: (payload: Record<string, unknown>) => void;
  onUpdate?: (id: string, payload: Record<string, unknown>) => void;
  onDelete?: (id: string) => void;
  onDeleteRequest?: (id: string, deleteChats: boolean) => void;
  onConnect?: (id: string) => void;
  onDisconnect?: (id: string) => void;
  onReconnect?: (id: string) => void;
  onPairPhone?: (id: string, payload: Record<string, unknown>) => void;
  onGetInstance?: (id: string) => MockInstance | null;
  onGetQRCode?: (id: string) => Record<string, unknown> | null;
  onHealth?: (
    id: string,
    requestCount: number,
  ) => Record<string, unknown> | null;
  onOrganizationSettings?: () => Record<string, unknown> | null;
};

async function mockInstancesApi(
  page: Page,
  instances: MockInstance[],
  hooks: MockHooks = {},
) {
  const healthRequests = new Map<string, number>();
  await page.route("**/api/org/settings", async (route) => {
    const payload = hooks.onOrganizationSettings?.() ?? {
      campaign_draft_only: false,
    };
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "success",
        data: {
          id: "e2e-org-id",
          name: "E2E Org",
          settings: payload,
        },
      }),
    });
  });

  await page.route("**/api/instances**", async (route) => {
    const request = route.request();
    const method = request.method();
    const url = new URL(request.url());
    const pathname = url.pathname;

    if (method === "GET" && pathname.endsWith("/api/instances")) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: instances,
        }),
      });
      return;
    }

    if (method === "POST" && pathname.endsWith("/api/instances")) {
      const payload = (request.postDataJSON() || {}) as Record<string, unknown>;
      hooks.onCreate?.(payload);

      const now = new Date().toISOString();
      const createdInstance: MockInstance = {
        id: `e2e-instance-created-${Date.now()}`,
        name:
          typeof payload.name === "string" && payload.name.trim()
            ? payload.name
            : `E2E Instance ${instances.length + 1}`,
        status: "disconnected",
        is_default: false,
        auto_read_receipt: true,
        organization_id: "e2e-org-id",
        created_at: now,
        updated_at: now,
      };
      instances.push(createdInstance);

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: createdInstance,
        }),
      });
      return;
    }

    const qrMatch = pathname.match(/\/api\/instances\/([^/]+)\/qr$/);
    if (qrMatch && method === "GET") {
      const instanceID = qrMatch[1];
      const payload = hooks.onGetQRCode?.(instanceID) ?? { available: false };
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: {
            instance_id: instanceID,
            timeout_seconds: 20,
            ...payload,
          },
        }),
      });
      return;
    }

    const instanceRootMatch = pathname.match(/\/api\/instances\/([^/]+)$/);
    if (instanceRootMatch && method === "GET") {
      const instanceID = instanceRootMatch[1];
      const hooked = hooks.onGetInstance?.(instanceID);
      const instance =
        hooked ?? instances.find((item) => item.id === instanceID) ?? null;

      if (!instance) {
        await route.fulfill({
          status: 404,
          contentType: "application/json",
          body: JSON.stringify({
            status: "error",
            message: "Instance not found",
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: instance,
        }),
      });
      return;
    }

    if (instanceRootMatch && method === "PUT") {
      const instanceID = instanceRootMatch[1];
      const payload = (request.postDataJSON() || {}) as Record<string, unknown>;
      hooks.onUpdate?.(instanceID, payload);

      const index = instances.findIndex(
        (instance) => instance.id === instanceID,
      );
      const current = index >= 0 ? instances[index] : null;
      const nextInstance: MockInstance = {
        ...(current || {
          id: instanceID,
          name: "Unnamed Instance",
          status: "disconnected",
          is_default: false,
          auto_read_receipt: true,
          organization_id: "e2e-org-id",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }),
        ...(typeof payload.name === "string" ? { name: payload.name } : {}),
        ...(payload.settings && typeof payload.settings === "object"
          ? { settings: payload.settings as Record<string, unknown> }
          : {}),
        updated_at: new Date().toISOString(),
      };
      if (index >= 0) {
        instances[index] = nextInstance;
      } else {
        instances.push(nextInstance);
      }

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: nextInstance,
        }),
      });
      return;
    }

    if (instanceRootMatch && method === "DELETE") {
      const instanceID = instanceRootMatch[1];
      hooks.onDelete?.(instanceID);
      const deleteChats = url.searchParams.get("delete_chats") === "true";
      hooks.onDeleteRequest?.(instanceID, deleteChats);
      const index = instances.findIndex(
        (instance) => instance.id === instanceID,
      );
      if (index >= 0) {
        instances.splice(index, 1);
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: null,
        }),
      });
      return;
    }

    const actionMatch = pathname.match(
      /\/api\/instances\/([^/]+)\/(connect|disconnect|reconnect|pair-phone)$/,
    );
    if (actionMatch && method === "POST") {
      const [, instanceID, action] = actionMatch;
      if (action === "connect") {
        hooks.onConnect?.(instanceID);
      } else if (action === "disconnect") {
        hooks.onDisconnect?.(instanceID);
      } else if (action === "reconnect") {
        hooks.onReconnect?.(instanceID);
      } else if (action === "pair-phone") {
        const payload = (request.postDataJSON() || {}) as Record<
          string,
          unknown
        >;
        hooks.onPairPhone?.(instanceID, payload);
      }

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data:
            action === "pair-phone"
              ? {
                  pairing_code: "123-456",
                  phone_number: "+15551234567",
                  timeout_seconds: 30,
                }
              : {},
        }),
      });
      return;
    }

    if (method === "GET" && /\/api\/instances\/[^/]+\/health$/.test(pathname)) {
      const match = pathname.match(/\/api\/instances\/([^/]+)\/health$/);
      const instanceID = match?.[1] || "";
      const requestCount = (healthRequests.get(instanceID) || 0) + 1;
      healthRequests.set(instanceID, requestCount);
      const healthPayload =
        hooks.onHealth?.(instanceID, requestCount) ?? HEALTH_PAYLOAD;

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: healthPayload,
        }),
      });
      return;
    }

    await route.fallback();
  });
}

async function emitWS(
  page: Page,
  eventType: string,
  payload: Record<string, unknown>,
) {
  await page.evaluate(
    ({ type, body }) => {
      (window as any).__WHM_WS_TEST_EMIT__?.(type, body);
    },
    { type: eventType, body: payload },
  );
}

test.describe("WhatsApp Instances", () => {
  test("should create a new instance from Add Account dialog", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const instances: MockInstance[] = [];
    const newInstanceName = `E2E Instance ${Date.now()}`;
    let receivedCreatePayload: { name?: string } | null = null;

    await mockInstancesApi(page, instances, {
      onCreate: (payload) => {
        receivedCreatePayload = payload as { name?: string };
      },
    });

    await page.goto("/settings/instances");
    await expect(
      page.getByRole("heading", { name: /WhatsApp Instances/i }),
    ).toBeVisible();

    await page.getByRole("button", { name: /Add Account/i }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Add WhatsApp Account")).toBeVisible();
    await dialog.locator("input#name").fill(newInstanceName);
    await dialog.getByRole("button", { name: "Create" }).click();

    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Instance created successfully" }),
    ).toBeVisible();
    await expect(
      page.locator("h3").filter({ hasText: newInstanceName }),
    ).toBeVisible();
    expect(receivedCreatePayload).toEqual({ name: newInstanceName });
  });

  test("should edit an instance name from the instance card", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const originalInstance: MockInstance = {
      id: "e2e-instance-edit-id",
      name: `Original Instance ${Date.now()}`,
      status: "disconnected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      created_at: now,
      updated_at: now,
    };
    const updatedInstanceName = `${originalInstance.name} Updated`;
    const instances: MockInstance[] = [originalInstance];
    let receivedUpdatePayload: { name?: string } | null = null;

    await mockInstancesApi(page, instances, {
      onUpdate: (_, payload) => {
        receivedUpdatePayload = payload as { name?: string };
      },
    });

    await page.goto("/settings/instances");
    await expect(
      page.locator("h3").filter({ hasText: originalInstance.name }),
    ).toBeVisible();

    await page
      .getByRole("button", { name: /Edit instance name/i })
      .first()
      .click();

    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Edit Account Name")).toBeVisible();
    await dialog.locator("input#edit-name").fill(updatedInstanceName);
    await dialog.getByRole("button", { name: "Update" }).click();

    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Instance updated successfully" }),
    ).toBeVisible();
    await expect(
      page.locator("h3").filter({ hasText: updatedInstanceName }),
    ).toBeVisible();
    expect(receivedUpdatePayload).toEqual({ name: updatedInstanceName });
  });

  test("should delete an instance from the instance card", async ({ page }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const deletingInstance: MockInstance = {
      id: "e2e-instance-delete-id",
      name: `Delete Me ${Date.now()}`,
      status: "disconnected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [deletingInstance];
    let deletedInstanceID = "";

    await mockInstancesApi(page, instances, {
      onDelete: (id) => {
        deletedInstanceID = id;
      },
    });

    await page.goto("/settings/instances");
    await expect(
      page.locator("h3").filter({ hasText: deletingInstance.name }),
    ).toBeVisible();

    await page
      .getByRole("button", { name: /Delete instance/i })
      .first()
      .click();

    const deleteDialog = page.getByRole("alertdialog");
    await expect(
      deleteDialog.getByText("Delete WhatsApp Account"),
    ).toBeVisible();
    await deleteDialog.getByRole("button", { name: "Delete Account" }).click();

    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Instance deleted successfully" }),
    ).toBeVisible();
    await expect(
      page.locator("h3").filter({ hasText: deletingInstance.name }),
    ).toHaveCount(0);
    expect(deletedInstanceID).toBe(deletingInstance.id);
  });

  test("should send delete_chats=true when delete related chats is checked", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const deletingInstance: MockInstance = {
      id: "e2e-instance-delete-chats-id",
      name: `Delete Chats ${Date.now()}`,
      status: "disconnected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [deletingInstance];
    let deleteChats = false;

    await mockInstancesApi(page, instances, {
      onDeleteRequest: (_id, withChats) => {
        deleteChats = withChats;
      },
    });

    await page.goto("/settings/instances");
    await expect(
      page.locator("h3").filter({ hasText: deletingInstance.name }),
    ).toBeVisible();

    await page
      .getByRole("button", { name: /Delete instance/i })
      .first()
      .click();
    const deleteDialog = page.getByRole("alertdialog");
    await expect(
      deleteDialog.getByText("Delete WhatsApp Account"),
    ).toBeVisible();
    await deleteDialog.getByLabel("Also delete chats for this account").click();
    await deleteDialog.getByRole("button", { name: "Delete Account" }).click();

    expect(deleteChats).toBe(true);
  });

  test("should initiate connect flow and request QR regeneration", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-connect-id",
      name: `Connect Flow ${Date.now()}`,
      status: "disconnected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];
    const connectCalls: string[] = [];
    const reconnectCalls: string[] = [];

    await mockInstancesApi(page, instances, {
      onConnect: (id) => {
        connectCalls.push(id);
      },
      onReconnect: (id) => {
        reconnectCalls.push(id);
      },
    });

    await page.goto("/settings/instances");
    await expect(
      page.locator("h3").filter({ hasText: instance.name }),
    ).toBeVisible();

    await page
      .getByRole("button", { name: /Connect \/ Scan QR/i })
      .first()
      .click();
    await expect(
      page.locator("[data-sonner-toast]").filter({
        hasText: /Connection initiated\. Waiting for QR code\.\.\./i,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("dialog").getByText("Link WhatsApp Device"),
    ).toBeVisible();
    expect(connectCalls).toEqual([instance.id]);

    await page.getByRole("button", { name: /Regenerate QR/i }).click();
    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Requesting a new QR code..." }),
    ).toBeVisible();
    expect(reconnectCalls).toEqual([instance.id]);
  });

  test("should update status on banned and logged_out websocket events", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-ws-status-id",
      name: `WS Status ${Date.now()}`,
      status: "connected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];

    await mockInstancesApi(page, instances);

    await page.goto("/settings/instances");
    await expect(
      page.getByRole("button", { name: /^Disconnect$/i }).first(),
    ).toBeVisible();

    await emitWS(page, "instance_banned", { instance_id: instance.id });
    await expect(page.getByText("Banned")).toBeVisible();
    await expect(
      page.getByRole("button", { name: /Connect \/ Scan QR/i }).first(),
    ).toBeVisible();

    await emitWS(page, "instance_logged_out", { instance_id: instance.id });
    await expect(page.getByText("Logged out")).toBeVisible();
    await expect(
      page.getByRole("button", { name: /Connect \/ Scan QR/i }).first(),
    ).toBeVisible();
  });

  test("should show connect watchdog timeout when QR is not received", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-watchdog-id",
      name: `Watchdog ${Date.now()}`,
      status: "disconnected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];

    await mockInstancesApi(page, instances, {
      onGetQRCode: () => ({
        available: false,
      }),
      onGetInstance: (id) => {
        const current = instances.find((item) => item.id === id);
        return current ? { ...current, status: "disconnected" } : null;
      },
    });

    await page.goto("/settings/instances");
    await page
      .getByRole("button", { name: /Connect \/ Scan QR/i })
      .first()
      .click();

    await expect(
      page.locator("[data-sonner-toast]").filter({
        hasText:
          "No QR code or connection confirmation was received. Please regenerate QR code.",
      }),
    ).toBeVisible({ timeout: 22000 });
  });

  test("should disconnect a connected instance from the instance card", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-disconnect-id",
      name: `Disconnect Flow ${Date.now()}`,
      status: "connected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];
    const disconnectCalls: string[] = [];

    await mockInstancesApi(page, instances, {
      onDisconnect: (id) => {
        disconnectCalls.push(id);
      },
    });

    await page.goto("/settings/instances");
    await expect(
      page.locator("h3").filter({ hasText: instance.name }),
    ).toBeVisible();

    await page
      .getByRole("button", { name: /^Disconnect$/i })
      .first()
      .click();

    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Instance logged out" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: /Connect \/ Scan QR/i }).first(),
    ).toBeVisible();
    expect(disconnectCalls).toEqual([instance.id]);
  });

  test("should save auto campaign settings payload", async ({ page }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-auto-campaign-id",
      name: `Auto Campaign ${Date.now()}`,
      status: "connected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      settings: {},
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];
    let receivedUpdatePayload: any = null;

    await mockInstancesApi(page, instances, {
      onUpdate: (_id, payload) => {
        receivedUpdatePayload = payload as Record<string, any>;
      },
    });

    await page.goto("/settings/instances");
    await page
      .getByRole("button", { name: "Configure auto campaign" })
      .first()
      .click();

    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Auto Campaign Settings")).toBeVisible();
    const enableLabel = dialog.getByText("Enable auto campaign");
    await enableLabel
      .locator('xpath=ancestor::div[contains(@class,"rounded-lg")][1]')
      .getByRole("switch")
      .click();
    await dialog.getByPlaceholder("e.g. promo-").fill("promo-");
    await dialog.locator('input[type="number"]').nth(0).fill("9");
    await dialog.locator('input[type="number"]').nth(1).fill("1");
    await dialog.locator('input[type="number"]').nth(2).fill("3");
    await dialog.getByRole("combobox").click();
    await page.getByRole("option", { name: "Run immediately" }).click();
    await dialog.locator("textarea").fill("Hello {contact_name}");
    await dialog.getByRole("button", { name: "Save settings" }).click();

    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Instance updated successfully" }),
    ).toBeVisible();
    expect(receivedUpdatePayload).not.toBeNull();
    expect(receivedUpdatePayload?.settings?.auto_campaign?.enabled).toBe(true);
    expect(receivedUpdatePayload?.settings?.auto_campaign?.name_prefix).toBe(
      "promo-",
    );
    expect(receivedUpdatePayload?.settings?.auto_campaign?.interval_days).toBe(
      9,
    );
    expect(
      receivedUpdatePayload?.settings?.auto_campaign?.min_delay_minutes,
    ).toBe(1);
    expect(
      receivedUpdatePayload?.settings?.auto_campaign?.max_delay_minutes,
    ).toBe(3);
    expect(receivedUpdatePayload?.settings?.auto_campaign?.target_status).toBe(
      "run",
    );
    expect(receivedUpdatePayload?.settings?.auto_campaign?.message).toBe(
      "Hello {contact_name}",
    );
  });

  test("should show auto campaign evaluation status and eligibility guidance", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-auto-campaign-status-id",
      name: `Auto Campaign Status ${Date.now()}`,
      status: "connected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      settings: {
        auto_campaign: {
          enabled: true,
          interval_days: 7,
          target_status: "draft",
          message: "Hello {contact_name}",
          last_generated_at: "2026-03-01T12:00:00Z",
        },
      },
      created_at: now,
      updated_at: now,
    };

    await mockInstancesApi(page, [instance]);

    await page.goto("/settings/instances");
    await page
      .getByRole("button", { name: "Configure auto campaign" })
      .first()
      .click();

    const dialog = page.getByRole("dialog");
    await expect(
      dialog.getByText(
        "Only contacts with inbound messages on this instance during the evaluation window are included.",
      ),
    ).toBeVisible();
    await expect(dialog.getByText("Last evaluation:")).toBeVisible();
    await expect(dialog.getByText("Next evaluation:")).toBeVisible();
    await expect(dialog.getByText("2026")).toBeVisible();
  });

  test("should force target_status=draft when campaign_draft_only is enabled", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-auto-campaign-policy-id",
      name: `Auto Campaign Policy ${Date.now()}`,
      status: "connected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      settings: {},
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];
    let receivedUpdatePayload: any = null;

    await mockInstancesApi(page, instances, {
      onOrganizationSettings: () => ({ campaign_draft_only: true }),
      onUpdate: (_id, payload) => {
        receivedUpdatePayload = payload as Record<string, any>;
      },
    });

    await page.goto("/settings/instances");
    await page
      .getByRole("button", { name: "Configure auto campaign" })
      .first()
      .click();

    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Auto Campaign Settings")).toBeVisible();
    const enableLabel = dialog.getByText("Enable auto campaign");
    await enableLabel
      .locator('xpath=ancestor::div[contains(@class,"rounded-lg")][1]')
      .getByRole("switch")
      .click();
    await dialog.locator("textarea").fill("Draft-only message");
    await dialog.getByRole("combobox").click();
    await page.getByRole("option", { name: "Run immediately" }).click();
    await dialog.getByRole("button", { name: "Save settings" }).click();

    await expect(
      page.locator("[data-sonner-toast]").filter({
        hasText:
          "Campaign draft-only policy is active. Target status was set to draft.",
      }),
    ).toBeVisible();
    expect(receivedUpdatePayload).not.toBeNull();
    expect(receivedUpdatePayload?.settings?.auto_campaign?.target_status).toBe(
      "draft",
    );
  });

  test("should refresh queue depth after websocket health-triggering events", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-health-refresh-id",
      name: `Health Refresh ${Date.now()}`,
      status: "disconnected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];

    await mockInstancesApi(page, instances, {
      onHealth: (_id, count) => ({
        ...HEALTH_PAYLOAD,
        queue_depth: count === 1 ? 777 : 888,
      }),
    });

    await page.goto("/settings/instances");
    await expect(page.getByText("777")).toBeVisible();

    await emitWS(page, "instance_connected", {
      instance_id: instance.id,
      phone_number: "+15550001111",
    });

    await expect(page.getByText("888")).toBeVisible();
  });

  test("should send auto_download_incoming_media=true and preserve existing settings", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-auto-download-on-id",
      name: `Auto Download ON ${Date.now()}`,
      status: "connected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      settings: {
        auto_sync_history: true,
        custom_existing_setting: "keep-me",
      },
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];
    let receivedUpdatePayload: any = null;

    await mockInstancesApi(page, instances, {
      onUpdate: (_, payload) => {
        receivedUpdatePayload = payload as Record<string, any>;
      },
    });

    await page.goto("/settings/instances");
    await expect(
      page.locator("h3").filter({ hasText: instance.name }),
    ).toBeVisible();

    const autoDownloadLabel = page.getByText("Auto-download incoming media");
    await expect(autoDownloadLabel).toBeVisible();
    const autoDownloadSwitch = autoDownloadLabel
      .locator('xpath=ancestor::div[contains(@class,"rounded-md")][1]')
      .getByRole("switch");
    await autoDownloadSwitch.click();

    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Instance updated successfully" }),
    ).toBeVisible();
    expect(receivedUpdatePayload).not.toBeNull();
    expect(receivedUpdatePayload?.settings?.auto_download_incoming_media).toBe(
      true,
    );
    expect(receivedUpdatePayload?.settings?.auto_sync_history).toBe(true);
    expect(receivedUpdatePayload?.settings?.custom_existing_setting).toBe(
      "keep-me",
    );
  });

  test("should send auto_download_incoming_media=false when toggled off", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-auto-download-off-id",
      name: `Auto Download OFF ${Date.now()}`,
      status: "connected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      settings: {
        auto_download_incoming_media: true,
      },
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];
    let receivedUpdatePayload: any = null;

    await mockInstancesApi(page, instances, {
      onUpdate: (_, payload) => {
        receivedUpdatePayload = payload as Record<string, any>;
      },
    });

    await page.goto("/settings/instances");
    await expect(
      page.locator("h3").filter({ hasText: instance.name }),
    ).toBeVisible();

    const autoDownloadLabel = page.getByText("Auto-download incoming media");
    await expect(autoDownloadLabel).toBeVisible();
    const autoDownloadSwitch = autoDownloadLabel
      .locator('xpath=ancestor::div[contains(@class,"rounded-md")][1]')
      .getByRole("switch");
    await autoDownloadSwitch.click();

    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Instance updated successfully" }),
    ).toBeVisible();
    expect(receivedUpdatePayload).not.toBeNull();
    expect(receivedUpdatePayload?.settings?.auto_download_incoming_media).toBe(
      false,
    );
  });

  test("should save instance specific chat close rating settings", async ({
    page,
  }) => {
    await loginAsAdmin(page);

    const now = new Date().toISOString();
    const instance: MockInstance = {
      id: "e2e-instance-chat-close-rating-id",
      name: `Chat Close Rating ${Date.now()}`,
      status: "connected",
      is_default: false,
      auto_read_receipt: true,
      organization_id: "e2e-org-id",
      settings: {},
      created_at: now,
      updated_at: now,
    };
    const instances: MockInstance[] = [instance];
    let receivedUpdatePayload: any = null;

    await mockInstancesApi(page, instances, {
      onUpdate: (_id, payload) => {
        receivedUpdatePayload = payload as Record<string, any>;
      },
    });

    await page.goto("/settings/instances");

    await page
      .getByRole("button", { name: "Configure Rating Message" })
      .first()
      .click();

    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Chat Close Rating Settings")).toBeVisible();
    await dialog.locator('input[type="number"]').fill("30");
    await dialog.locator("textarea").nth(1).fill("Rate us!");
    await dialog.getByRole("button", { name: "Save" }).click();

    await expect(
      page
        .locator("[data-sonner-toast]")
        .filter({ hasText: "Instance updated successfully" }),
    ).toBeVisible();
    expect(receivedUpdatePayload).not.toBeNull();
    expect(receivedUpdatePayload?.settings?.chat_close_rating_enabled).toBe(
      true,
    );
    expect(
      receivedUpdatePayload?.settings
        ?.chat_close_rating_followup_window_minutes,
    ).toBe(30);
    expect(
      receivedUpdatePayload?.settings?.chat_close_rating_templates?.en,
    ).toBe("Rate us!");
    await expect(page.getByText("Reply window: 30 min")).toBeVisible();

    // Reopen dialog to verify persistence
    await page
      .getByRole("button", { name: "Configure Rating Message" })
      .first()
      .click();
    const reopenDialog = page.getByRole("dialog");
    await expect(
      reopenDialog.getByText("Chat Close Rating Settings"),
    ).toBeVisible();
    await expect(
      reopenDialog.getByText("Override Organization Settings"),
    ).toHaveCount(0);
    await expect(reopenDialog.locator('input[type="number"]')).toHaveValue(
      "30",
    );
    await expect(reopenDialog.locator("textarea").nth(1)).toHaveValue(
      "Rate us!",
    );
  });
});
