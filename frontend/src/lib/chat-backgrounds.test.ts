// @vitest-environment happy-dom

import { describe, expect, it } from "vitest";
import {
  CHAT_BACKGROUND_UPLOAD_MAX_BYTES,
  normalizeChatBackgroundPreference,
  resolveChatBackgroundEditorMode,
  resolveChatBackgroundStyle,
  validateChatBackgroundFile,
} from "./chat-backgrounds";

describe("chat-backgrounds", () => {
  it("resolves preset image backgrounds with cover sizing", () => {
    const style = resolveChatBackgroundStyle(
      { kind: "preset", preset_id: "aurora-veil" },
      { theme: "dark" },
    );

    expect(style.backgroundImage).toContain('url("data:image/svg+xml');
    expect(style.backgroundSize).toBe("cover, cover, auto, auto");
    expect(style.backgroundRepeat).toBe(
      "no-repeat, no-repeat, no-repeat, no-repeat",
    );
  });

  it("resolves preset pattern backgrounds with repeating pattern sizing", () => {
    const style = resolveChatBackgroundStyle(
      { kind: "preset", preset_id: "linen-grid" },
      { theme: "light" },
    );

    expect(style.backgroundImage).toContain('url("data:image/svg+xml');
    expect(style.backgroundSize).toBe("cover, 360px 360px, auto, auto");
    expect(style.backgroundRepeat).toBe(
      "no-repeat, repeat, no-repeat, no-repeat",
    );
  });

  it("resolves custom backgrounds through the authenticated me endpoint", () => {
    const style = resolveChatBackgroundStyle({
      kind: "custom",
      custom_asset_id: "asset-123",
      custom_filename: "photo.png",
      custom_mime_type: "image/png",
    });

    expect(style.backgroundImage).toContain(
      "/api/me/chat-background?asset=asset-123",
    );
    expect(style.backgroundSize).toBe("cover, cover, auto, auto");
  });

  it("falls back to the default background when the selection is invalid", () => {
    const style = resolveChatBackgroundStyle({
      kind: "preset",
      preset_id: "missing",
    });

    expect(style.backgroundImage).not.toContain("url(");
    expect(style.backgroundRepeat).toBe("no-repeat, no-repeat");
    expect(resolveChatBackgroundStyle(null).backgroundRepeat).toBe(
      "no-repeat, no-repeat",
    );
  });

  it("rejects unsupported upload types and oversize files", () => {
    const badType = new File(["text"], "notes.txt", { type: "text/plain" });
    const tooLarge = new File(
      [new Uint8Array(CHAT_BACKGROUND_UPLOAD_MAX_BYTES + 1)],
      "photo.png",
      { type: "image/png" },
    );
    const valid = new File([new Uint8Array([1, 2, 3])], "photo.webp", {
      type: "image/webp",
    });

    expect(validateChatBackgroundFile(badType)).toEqual({
      valid: false,
      errorKey: "settings.chatBackgroundUploadInvalidType",
    });
    expect(validateChatBackgroundFile(tooLarge)).toEqual({
      valid: false,
      errorKey: "settings.chatBackgroundUploadTooLarge",
    });
    expect(validateChatBackgroundFile(valid)).toEqual({ valid: true });
  });

  it("normalizes saved metadata and derives the correct editor mode", () => {
    expect(
      normalizeChatBackgroundPreference({
        kind: "custom",
        custom_asset_id: "asset-abc",
        custom_filename: "vacation.webp",
        custom_mime_type: "image/webp",
      }),
    ).toEqual({
      kind: "custom",
      custom_asset_id: "asset-abc",
      custom_filename: "vacation.webp",
      custom_mime_type: "image/webp",
    });

    expect(
      normalizeChatBackgroundPreference({
        kind: "preset",
        preset_id: "dot-orbit",
      }),
    ).toEqual({
      kind: "preset",
      preset_id: "dot-orbit",
    });
    expect(
      resolveChatBackgroundEditorMode({
        kind: "preset",
        preset_id: "dot-orbit",
      }),
    ).toBe("patterns");
    expect(
      resolveChatBackgroundEditorMode({
        kind: "custom",
        custom_asset_id: "asset-abc",
        custom_filename: "vacation.webp",
        custom_mime_type: "image/webp",
      }),
    ).toBe("upload");
    expect(normalizeChatBackgroundPreference(null)).toBeNull();
    expect(resolveChatBackgroundEditorMode(null)).toBe("default");
  });
});
