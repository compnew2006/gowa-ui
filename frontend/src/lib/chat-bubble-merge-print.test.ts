import { describe, expect, it } from "vitest";

import {
  inferMergePrintableMimeType,
  isMergePrintableBubbleMessage,
  resolveMergePrintableFilename,
  toMergePrintableFile,
  type MergePrintableChatMessage,
} from "./chat-bubble-merge-print";

function buildMessage(
  overrides: Partial<MergePrintableChatMessage>,
): MergePrintableChatMessage {
  return {
    id: "msg-1",
    media_url: "/api/media/msg-1",
    message_type: "document",
    media_mime_type: "application/pdf",
    ...overrides,
  };
}

describe("isMergePrintableBubbleMessage", () => {
  it("accepts image bubbles even when mime type is missing", () => {
    expect(
      isMergePrintableBubbleMessage(
        buildMessage({ message_type: "image", media_mime_type: "" }),
      ),
    ).toBe(true);
  });

  it("accepts document bubbles when filename indicates pdf", () => {
    expect(
      isMergePrintableBubbleMessage(
        buildMessage({
          message_type: "document",
          media_mime_type: "",
          media_filename: "invoice",
        }),
      ),
    ).toBe(false);

    expect(
      isMergePrintableBubbleMessage(
        buildMessage({
          message_type: "document",
          media_mime_type: "",
          media_filename: "invoice.pdf",
        }),
      ),
    ).toBe(true);
  });

  it("rejects bubbles without media url", () => {
    expect(
      isMergePrintableBubbleMessage(buildMessage({ media_url: "" })),
    ).toBe(false);
  });
});

describe("inferMergePrintableMimeType", () => {
  it("prefers blob mime type when available", () => {
    const message = buildMessage({
      message_type: "document",
      media_mime_type: "",
      media_filename: "scan.pdf",
    });
    const blob = new Blob(["x"], { type: "application/pdf" });
    expect(inferMergePrintableMimeType(message, blob)).toBe("application/pdf");
  });

  it("falls back to image mime for image bubbles", () => {
    const message = buildMessage({
      message_type: "image",
      media_mime_type: "",
      media_filename: "camera_upload",
    });
    const blob = new Blob(["x"], { type: "" });
    expect(inferMergePrintableMimeType(message, blob)).toBe("image/jpeg");
  });
});

describe("toMergePrintableFile", () => {
  it("creates file with inferred extension when missing", () => {
    const message = buildMessage({
      message_type: "document",
      media_mime_type: "",
      media_filename: "receipt",
    });
    const blob = new Blob(["x"], { type: "application/pdf" });
    const file = toMergePrintableFile(message, blob, 0);
    expect(file).not.toBeNull();
    expect(file?.name).toBe("receipt.pdf");
    expect(file?.type).toBe("application/pdf");
  });

  it("returns null for unsupported message payloads", () => {
    const message = buildMessage({
      message_type: "document",
      media_mime_type: "",
      media_filename: "notes.docx",
    });
    const blob = new Blob(["x"], {
      type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    });
    expect(toMergePrintableFile(message, blob, 0)).toBeNull();
  });
});

describe("resolveMergePrintableFilename", () => {
  it("keeps existing extension", () => {
    const message = buildMessage({
      media_filename: "photo.png",
      message_type: "image",
      media_mime_type: "image/png",
    });
    expect(resolveMergePrintableFilename(message, "image/png", 3)).toBe(
      "photo.png",
    );
  });
});
