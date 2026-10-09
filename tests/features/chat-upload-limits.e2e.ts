import { test } from "@e2e-dev/web";
import { expect } from "e2e";
import { prepareFreshBrowserSession } from "../helpers/auth-session";
import {
  cleanupPage,
  requireBaseUrl,
  seedPage,
  type SeededPage,
} from "../helpers/fixtures";
import {
  generatedUploads,
  type GeneratedUploads,
} from "../helpers/upload-files";

/**
 * Upload size limits in the chat composer follow the engine (GOWA), per message
 * type: image 20 MB, video 100 MB, audio and documents 50 MB. They used to be a
 * flat 16 MB for every image, video and audio file.
 *
 * The conversation is an isolated fixture whose account has no GOWA device, and
 * the media endpoint is answered by the test itself, so even a click on Send
 * cannot reach WhatsApp. These tests use exact interactions only (no agent step)
 * and need no model.
 */

const MB = 1_000_000;

interface MediaPost {
  method: string;
  contentType: string;
  /** The multipart text: part headers and form fields. The runner leaves the file bytes out. */
  body: string;
}

const chatTest = test.extend<{
  conversation: SeededPage;
  uploads: GeneratedUploads;
  mediaPosts: MediaPost[];
}>({
  conversation: async ({ app }, use) => {
    const seeded = await seedPage(
      requireBaseUrl(app.baseUrl),
      "chat-conversation",
    );
    try {
      await use(seeded.data);
    } finally {
      await cleanupPage(seeded.runId, seeded.data.resourceId);
    }
  },
  uploads: async ({}, use) => {
    const uploads = generatedUploads();
    try {
      await use(uploads);
    } finally {
      await uploads.remove();
    }
  },
  mediaPosts: async ({}, use) => {
    await use([]);
  },
});

chatTest.beforeEach(
  async ({ app, browser, screen, conversation, mediaPosts }) => {
    // The session cookies are installed on a clean context, so register the
    // route after it, then open the conversation.
    await prepareFreshBrowserSession(app, browser, "super-admin");
    await browser.route("**/api/messages/media", async (route) => {
      mediaPosts.push({
        method: route.request.method,
        contentType: route.request.headers["content-type"] ?? "",
        body: route.request.postData ?? "",
      });
      await route.fulfill({
        json: {
          status: "success",
          data: {
            id: crypto.randomUUID(),
            contact_id: conversation.resourceId ?? "",
            direction: "outgoing",
            message_type: "video",
            content: {},
            media_url: "/files/limit-test.mp4",
            media_mime_type: "video/mp4",
            media_filename: "limit-test.mp4",
            status: "sent",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        },
      });
    });

    await app.open(conversation.path);
    await expect(screen.getByPlaceholder("Type a message...")).toBeVisible({
      timeout: 20_000,
    });
  },
);

chatTest(
  "a 30 MB video, above the old 16 MB cap, is accepted and posted",
  { tags: ["chat", "upload-limits"] },
  async ({ browser, screen, uploads, mediaPosts }) => {
    const video = await uploads.create("limit-test-video-30mb.mp4", 30 * MB);

    await browser
      .locator('[data-testid="chat-file-input"]')
      .setInputFiles(video);

    const dialog = screen.getByRole("dialog", "Send Media");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText("limit-test-video-30mb.mp4")).toBeVisible();
    await expect(screen.getByText("File too large")).toHaveCount(0);

    await dialog.getByRole("button", "Send").tap();

    await expect
      .poll(() => mediaPosts.length, { timeout: 30_000 })
      .toBe(1);
    expect(mediaPosts[0].method).toBe("POST");
    expect(mediaPosts[0].contentType).toContain("multipart/form-data");
    // The file went out whole, as a video message: its part and the type field.
    expect(mediaPosts[0].body).toContain('filename="limit-test-video-30mb.mp4"');
    expect(mediaPosts[0].body).toContain("Content-Type: video/mp4");
    expect(mediaPosts[0].body).toMatch(/name="type"\s+video/);
    await expect(dialog).toBeHidden({ timeout: 30_000 });
  },
);

for (const { label, file, bytes, size, limit } of [
  {
    label: "an image over 20 MB",
    file: "limit-test-image.png",
    bytes: 20 * MB + 1,
    size: "20.1",
    limit: "20",
  },
  {
    label: "a video over 100 MB",
    file: "limit-test-video.mp4",
    bytes: 100 * MB + 1,
    size: "100.1",
    limit: "100",
  },
  {
    label: "a document over 50 MB",
    file: "limit-test-document.pdf",
    bytes: 50 * MB + 1,
    size: "50.1",
    limit: "50",
  },
]) {
  chatTest(
    `${label} is refused with its size and limit, and nothing is sent`,
    { tags: ["chat", "upload-limits"] },
    async ({ browser, screen, uploads, mediaPosts }) => {
      const path = await uploads.create(file, bytes);

      await browser.locator('[data-testid="chat-file-input"]').setInputFiles(path);

      await expect(screen.getByText("File too large")).toBeVisible();
      await expect(
        screen.getByText(`${file}: ${size} MB (limit ${limit} MB)`),
      ).toBeVisible();
      await expect(screen.getByRole("dialog")).toHaveCount(0);
      expect(mediaPosts).toHaveLength(0);
    },
  );
}
