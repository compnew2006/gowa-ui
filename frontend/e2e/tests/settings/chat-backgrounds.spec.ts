import { test, expect, type Page } from "@playwright/test";
import { promises as fs } from "node:fs";

type ChatBackgroundState = {
  currentUserSettings: Record<string, unknown>;
  customAsset: {
    id: string;
    filename: string;
    mimeType: string;
    body: Buffer;
  } | null;
  settingsSaveCalls: number;
  uploadCalls: number;
};

const SAMPLE_CONTACT = {
  id: "contact-1",
  phone_number: "15551234567",
  name: "Taylor Example",
  profile_name: "Taylor Example",
  status: "open",
  tags: [],
  metadata: {},
  unread_count: 0,
  assigned_user_id: "user-1",
  assigned_user_name: "Agent Example",
  last_message_at: "2025-01-01T12:00:00.000Z",
  last_message_preview: "Hello from Playwright",
  created_at: "2025-01-01T12:00:00.000Z",
  updated_at: "2025-01-01T12:00:00.000Z",
};

const SAMPLE_MESSAGE = {
  id: "message-1",
  contact_id: "contact-1",
  direction: "incoming",
  message_type: "text",
  content: "Hello from Playwright",
  status: "delivered",
  created_at: "2025-01-01T12:00:00.000Z",
  updated_at: "2025-01-01T12:00:00.000Z",
};

const TINY_PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+nmZ0AAAAASUVORK5CYII=",
  "base64",
);

async function installAppMocks(page: Page): Promise<ChatBackgroundState> {
  const state: ChatBackgroundState = {
    currentUserSettings: {},
    customAsset: null,
    settingsSaveCalls: 0,
    uploadCalls: 0,
  };

  await page.addInitScript(() => {
    class MockWebSocket {
      static OPEN = 1;
      readyState = 1;
      onopen: ((event: Event) => void) | null = null;
      onclose: ((event: Event) => void) | null = null;
      onerror: ((event: Event) => void) | null = null;
      onmessage: ((event: MessageEvent) => void) | null = null;

      constructor() {
        window.setTimeout(() => this.onopen?.(new Event("open")), 0);
      }

      addEventListener() {}
      removeEventListener() {}
      send() {}
      close() {
        this.readyState = 3;
        this.onclose?.(new Event("close"));
      }
    }

    // @ts-expect-error test stub
    window.WebSocket = MockWebSocket;
  });

  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const method = request.method();
    const url = new URL(request.url());
    const pathname = url.pathname;

    const fulfillJSON = async (data: unknown, status = 200) => {
      await route.fulfill({
        status,
        contentType: "application/json",
        body: JSON.stringify({
          status: status >= 400 ? "error" : "success",
          ...(status >= 400 ? { message: data } : { data }),
        }),
      });
    };

    if (pathname.endsWith("/api/me") && method === "GET") {
      await fulfillJSON({
        id: "user-1",
        email: "agent@test.local",
        full_name: "Agent Example",
        organization_id: "org-1",
        is_available: true,
        role: {
          id: "role-1",
          name: "agent",
          permissions: [
            { resource: "chat", action: "read" },
            { resource: "settings.general", action: "read" },
          ],
        },
        settings: state.currentUserSettings,
      });
      return;
    }

    if (pathname.endsWith("/api/org/settings") && method === "GET") {
      await fulfillJSON({
        id: "org-1",
        name: "Whatomate E2E",
        settings: {
          timezone: "UTC",
          date_format: "YYYY-MM-DD",
          assigned_chat_reset_enabled: true,
          assigned_chat_reset_mode: "midnight",
          assigned_chat_reset_hour: 0,
          chat_close_rating_enabled: true,
          chat_close_rating_followup_window_minutes: 15,
          chat_close_rating_templates: {
            en: "Rate this chat from 1 to 10.",
            ar: "قيّم هذه المحادثة من 1 إلى 10.",
            es: "Califica este chat del 1 al 10.",
          },
        },
      });
      return;
    }

    if (pathname.endsWith("/api/org/settings") && method === "PUT") {
      await fulfillJSON({
        id: "org-1",
        name: "Whatomate E2E",
        settings: request.postDataJSON() || {},
      });
      return;
    }

    if (pathname.endsWith("/api/me/settings") && method === "PUT") {
      state.settingsSaveCalls += 1;
      const payload = (request.postDataJSON() || {}) as Record<string, unknown>;

      if ("email_notifications" in payload) {
        state.currentUserSettings.email_notifications =
          payload.email_notifications;
      }
      if ("new_message_alerts" in payload) {
        state.currentUserSettings.new_message_alerts =
          payload.new_message_alerts;
      }
      if ("campaign_updates" in payload) {
        state.currentUserSettings.campaign_updates = payload.campaign_updates;
      }
      if ("notification_sound" in payload) {
        state.currentUserSettings.notification_sound =
          payload.notification_sound;
      }
      if ("chat_background" in payload) {
        if (payload.chat_background === null) {
          delete state.currentUserSettings.chat_background;
          state.customAsset = null;
        } else if (
          payload.chat_background &&
          typeof payload.chat_background === "object"
        ) {
          state.currentUserSettings.chat_background = payload.chat_background;
          const chatBackground = payload.chat_background as Record<
            string,
            unknown
          >;
          if (chatBackground.kind === "preset") {
            state.customAsset = null;
          }
        }
      }

      await fulfillJSON({
        message: "Settings updated successfully",
        settings: state.currentUserSettings,
      });
      return;
    }

    if (pathname.endsWith("/api/me/chat-background") && method === "POST") {
      state.uploadCalls += 1;
      const nextAssetID = `asset-${state.uploadCalls}`;
      state.customAsset = {
        id: nextAssetID,
        filename: "custom-background.png",
        mimeType: "image/png",
        body: TINY_PNG,
      };
      state.currentUserSettings.chat_background = {
        kind: "custom",
        custom_asset_id: nextAssetID,
        custom_filename: state.customAsset.filename,
        custom_mime_type: state.customAsset.mimeType,
      };

      await fulfillJSON({
        message: "Chat background uploaded successfully",
        chat_background: state.currentUserSettings.chat_background,
      });
      return;
    }

    if (pathname.endsWith("/api/me/chat-background") && method === "GET") {
      if (!state.customAsset) {
        await route.fulfill({
          status: 404,
          contentType: "application/json",
          body: JSON.stringify({
            status: "error",
            message: "Chat background not found",
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: state.customAsset.mimeType,
        headers: {
          "Cache-Control": "private",
        },
        body: state.customAsset.body,
      });
      return;
    }

    if (pathname.endsWith("/api/chats") && method === "GET") {
      const status = url.searchParams.get("status");
      const contacts = status === "open" ? [SAMPLE_CONTACT] : [];
      await fulfillJSON({
        contacts,
        total: contacts.length,
        page: 1,
        limit: contacts.length || 1,
      });
      return;
    }

    if (pathname.endsWith("/api/contacts/contact-1") && method === "GET") {
      await fulfillJSON(SAMPLE_CONTACT);
      return;
    }

    if (
      pathname.endsWith("/api/chats/contact-1/messages") &&
      method === "GET"
    ) {
      await fulfillJSON({
        messages: [SAMPLE_MESSAGE],
        total: 1,
        page: 1,
        limit: 50,
        has_more: false,
      });
      return;
    }

    if (
      pathname.endsWith("/api/contacts/contact-1/session-data") &&
      method === "GET"
    ) {
      await fulfillJSON({});
      return;
    }

    if (
      pathname.endsWith("/api/contacts/contact-1/notes") &&
      method === "GET"
    ) {
      await fulfillJSON({
        notes: [],
        total: 0,
        has_more: false,
      });
      return;
    }

    if (pathname.endsWith("/api/instances") && method === "GET") {
      await fulfillJSON([]);
      return;
    }

    if (pathname.endsWith("/api/chatbot/transfers") && method === "GET") {
      await fulfillJSON({
        transfers: [],
        general_queue_count: 0,
        team_queue_counts: {},
        total_count: 0,
      });
      return;
    }

    if (pathname.endsWith("/api/tags") && method === "GET") {
      await fulfillJSON({
        tags: [],
        total: 0,
        page: 1,
        limit: 50,
      });
      return;
    }

    await fulfillJSON({});
  });

  return state;
}

async function openSettings(page: Page) {
  await page.goto("/settings");
  await expect(
    page.getByRole("heading", { name: "Settings", exact: true }),
  ).toBeVisible();
  await page.getByRole("tab", { name: "Chat", exact: true }).click();
  await expect(page.getByTestId("chat-background-mode-default")).toBeVisible();
}

async function saveChatSettings(page: Page) {
  await page.getByTestId("settings-chat-save").click();
  await expect(page.locator("[data-sonner-toast]")).toContainText(
    "Chat preferences saved",
  );
}

async function openChat(page: Page) {
  await page.goto("/chat/contact-1");
  await expect(page.getByTestId("chat-message-area")).toBeVisible();
}

test.describe("Chat background settings", () => {
  test("saves a preset image background and uses it in chat after reload", async ({
    page,
  }) => {
    const state = await installAppMocks(page);

    await openSettings(page);
    await page.getByTestId("chat-background-mode-images").click();
    await page.getByTestId("chat-background-preset-aurora-veil").click();
    await saveChatSettings(page);

    expect(state.currentUserSettings.chat_background).toEqual({
      kind: "preset",
      preset_id: "aurora-veil",
    });

    await openChat(page);
    const computed = await page
      .getByTestId("chat-message-area")
      .evaluate((element) => {
        const styles = window.getComputedStyle(element as HTMLElement);
        return {
          backgroundImage: styles.backgroundImage,
          backgroundRepeat: styles.backgroundRepeat,
          backgroundSize: styles.backgroundSize,
        };
      });
    expect(computed.backgroundImage).toContain("data:image/svg+xml");
    expect(computed.backgroundRepeat).toContain("no-repeat");
    expect(computed.backgroundSize).toContain("cover");

    await page.reload();
    await expect(page.getByTestId("chat-message-area")).toBeVisible();
    const reloaded = await page
      .getByTestId("chat-message-area")
      .evaluate((element) => {
        const styles = window.getComputedStyle(element as HTMLElement);
        return {
          backgroundImage: styles.backgroundImage,
          backgroundRepeat: styles.backgroundRepeat,
          backgroundSize: styles.backgroundSize,
        };
      });
    expect(reloaded.backgroundImage).toContain("data:image/svg+xml");
    expect(reloaded.backgroundRepeat).toContain("no-repeat");
    expect(reloaded.backgroundSize).toContain("cover");
  });

  test("saves a preset pattern background and uses it in chat after reload", async ({
    page,
  }) => {
    const state = await installAppMocks(page);

    await openSettings(page);
    await page.getByTestId("chat-background-mode-patterns").click();
    await page.getByTestId("chat-background-preset-linen-grid").click();
    await saveChatSettings(page);

    expect(state.currentUserSettings.chat_background).toEqual({
      kind: "preset",
      preset_id: "linen-grid",
    });

    await openChat(page);
    const computed = await page
      .getByTestId("chat-message-area")
      .evaluate((element) => {
        const styles = window.getComputedStyle(element as HTMLElement);
        return {
          backgroundImage: styles.backgroundImage,
          backgroundRepeat: styles.backgroundRepeat,
          backgroundSize: styles.backgroundSize,
        };
      });

    expect(computed.backgroundImage).toContain("data:image/svg+xml");
    expect(computed.backgroundRepeat).toContain("repeat");
    expect(computed.backgroundSize).toContain("360px 360px");
  });

  test("uploads a custom image and keeps it after reload and navigation", async ({
    page,
    browserName,
  }, testInfo) => {
    test.skip(
      browserName !== "chromium",
      "File upload coverage is only required in Chromium",
    );
    const state = await installAppMocks(page);

    const uploadPath = testInfo.outputPath("custom-chat-background.png");
    await fs.writeFile(uploadPath, TINY_PNG);

    await openSettings(page);
    await page.getByTestId("chat-background-mode-upload").click();
    await page
      .getByTestId("chat-background-upload-input")
      .setInputFiles(uploadPath);
    await saveChatSettings(page);

    expect(state.uploadCalls).toBe(1);
    expect(state.currentUserSettings.chat_background).toMatchObject({
      kind: "custom",
      custom_asset_id: "asset-1",
    });

    await openChat(page);
    const backgroundImage = await page
      .getByTestId("chat-message-area")
      .evaluate(
        (element) =>
          window.getComputedStyle(element as HTMLElement).backgroundImage,
      );
    expect(backgroundImage).toContain("/api/me/chat-background?asset=asset-1");

    await page.goto("/settings");
    await page.reload();
    await openChat(page);
    const reloadedBackgroundImage = await page
      .getByTestId("chat-message-area")
      .evaluate(
        (element) =>
          window.getComputedStyle(element as HTMLElement).backgroundImage,
      );
    expect(reloadedBackgroundImage).toContain(
      "/api/me/chat-background?asset=asset-1",
    );
  });

  test("shows visible validation errors for invalid type and oversize uploads without saving", async ({
    page,
  }, testInfo) => {
    const state = await installAppMocks(page);

    const invalidPath = testInfo.outputPath("invalid-chat-background.txt");
    const tooLargePath = testInfo.outputPath("too-large-chat-background.png");
    await fs.writeFile(invalidPath, "plain text");
    await fs.writeFile(tooLargePath, Buffer.alloc(5 * 1024 * 1024 + 1, 7));

    await openSettings(page);
    await page.getByTestId("chat-background-mode-upload").click();

    await page
      .getByTestId("chat-background-upload-input")
      .setInputFiles(invalidPath);
    await expect(
      page.getByTestId("chat-background-upload-error"),
    ).toContainText("Only JPG, PNG, and WebP images are supported.");
    await page.getByTestId("settings-chat-save").click();
    expect(state.uploadCalls).toBe(0);
    expect(state.settingsSaveCalls).toBe(0);

    await page
      .getByTestId("chat-background-upload-input")
      .setInputFiles(tooLargePath);
    await expect(
      page.getByTestId("chat-background-upload-error"),
    ).toContainText("Maximum size is 5MB");
    await page.getByTestId("settings-chat-save").click();
    expect(state.uploadCalls).toBe(0);
    expect(state.settingsSaveCalls).toBe(0);
    expect(state.currentUserSettings.chat_background).toBeUndefined();
  });

  test("clears a saved background back to the default chat surface", async ({
    page,
  }) => {
    const state = await installAppMocks(page);
    state.currentUserSettings.chat_background = {
      kind: "preset",
      preset_id: "aurora-veil",
    };

    await openSettings(page);
    await page.getByTestId("chat-background-mode-default").click();
    await saveChatSettings(page);

    expect(state.currentUserSettings.chat_background).toBeUndefined();

    await openChat(page);
    const computed = await page
      .getByTestId("chat-message-area")
      .evaluate((element) => {
        const styles = window.getComputedStyle(element as HTMLElement);
        return {
          backgroundImage: styles.backgroundImage,
          backgroundRepeat: styles.backgroundRepeat,
        };
      });

    expect(computed.backgroundImage).not.toContain("data:image/svg+xml");
    expect(computed.backgroundRepeat).toBe("no-repeat, no-repeat");
  });

  test("keeps the settings UI usable on a narrow mobile viewport", async ({
    page,
  }) => {
    await installAppMocks(page);
    await page.setViewportSize({ width: 390, height: 844 });

    await openSettings(page);
    await page.getByTestId("chat-background-mode-patterns").click();

    await expect(
      page.getByTestId("chat-background-preset-dot-orbit"),
    ).toBeVisible();
    await expect(page.getByTestId("settings-chat-save")).toBeVisible();

    const horizontalOverflow = await page.evaluate(() => {
      const root = document.documentElement;
      return root.scrollWidth - root.clientWidth;
    });
    expect(horizontalOverflow).toBeLessThan(3);
  });
});
