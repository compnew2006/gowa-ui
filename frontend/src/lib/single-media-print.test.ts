import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("./media-actions", async () => {
  const actual = await vi.importActual<typeof import("./media-actions")>(
    "./media-actions",
  );
  return {
    ...actual,
    openPrintDialogForMedia: vi.fn(),
  };
});

vi.mock("./media-merge-print", () => ({
  mergePhotosAndPdfsAndOpenPrintDialog: vi.fn(),
}));

vi.mock("./chat-bubble-merge-print", async () => {
  const actual = await vi.importActual<typeof import("./chat-bubble-merge-print")>(
    "./chat-bubble-merge-print",
  );
  return {
    ...actual,
    toMergePrintableFile: vi.fn(),
  };
});

import { toMergePrintableFile } from "./chat-bubble-merge-print";
import { openPrintDialogForMedia } from "./media-actions";
import { mergePhotosAndPdfsAndOpenPrintDialog } from "./media-merge-print";
import {
  isMessagePrintSupported,
  isTiffLikeMessage,
  openPrintDialogForSingleMessage,
} from "./single-media-print";

describe("isTiffLikeMessage", () => {
  it("detects TIFF by MIME type", () => {
    expect(
      isTiffLikeMessage({
        id: "m-1",
        message_type: "document",
        media_mime_type: "image/tiff",
      }),
    ).toBe(true);
  });

  it("detects TIFF by file extension", () => {
    expect(
      isTiffLikeMessage({
        id: "m-2",
        message_type: "document",
        media_filename: "scan_01.TIF",
      }),
    ).toBe(true);
  });

  it("returns false for non-TIFF media", () => {
    expect(
      isTiffLikeMessage({
        id: "m-3",
        message_type: "document",
        media_filename: "invoice.pdf",
        media_mime_type: "application/pdf",
      }),
    ).toBe(false);
  });
});

describe("isMessagePrintSupported", () => {
  it("supports PDF and images only", () => {
    expect(
      isMessagePrintSupported({
        id: "p-1",
        message_type: "document",
        media_filename: "invoice.pdf",
      }),
    ).toBe(true);

    expect(
      isMessagePrintSupported({
        id: "p-2",
        message_type: "document",
        media_filename: "report.docx",
        media_mime_type:
          "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
      }),
    ).toBe(false);
  });
});

describe("openPrintDialogForSingleMessage", () => {
  const rawPrintMock = vi.mocked(openPrintDialogForMedia);
  const mergePrintMock = vi.mocked(mergePhotosAndPdfsAndOpenPrintDialog);
  const toPrintableFileMock = vi.mocked(toMergePrintableFile);

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("uses raw print flow for non-TIFF files", async () => {
    rawPrintMock.mockReturnValue(true);
    const resolveBlob = vi.fn(async () => new Blob(["x"], { type: "image/png" }));

    const opened = await openPrintDialogForSingleMessage({
      mediaUrl: "blob:raw-image",
      message: {
        id: "m-10",
        message_type: "image",
        media_filename: "photo.png",
        media_mime_type: "image/png",
      },
      resolveBlob,
    });

    expect(opened).toBe(true);
    expect(rawPrintMock).toHaveBeenCalledWith("blob:raw-image");
    expect(resolveBlob).not.toHaveBeenCalled();
    expect(mergePrintMock).not.toHaveBeenCalled();
  });

  it("converts TIFF through merge print flow", async () => {
    const file = { name: "scan.tiff", type: "image/tiff" } as File;
    const resolveBlob = vi.fn(async () => new Blob(["x"], { type: "image/tiff" }));
    toPrintableFileMock.mockReturnValue(file);
    mergePrintMock.mockResolvedValue(true);

    const opened = await openPrintDialogForSingleMessage({
      mediaUrl: "blob:tiff-image",
      message: {
        id: "m-11",
        message_type: "document",
        media_filename: "scan.tiff",
        media_mime_type: "image/tiff",
      },
      resolveBlob,
    });

    expect(opened).toBe(true);
    expect(resolveBlob).toHaveBeenCalledTimes(1);
    expect(toPrintableFileMock).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "m-11",
      }),
      expect.any(Blob),
      0,
    );
    expect(mergePrintMock).toHaveBeenCalledWith([file]);
    expect(rawPrintMock).not.toHaveBeenCalled();
  });

  it("fails cleanly when TIFF conversion throws", async () => {
    const resolveBlob = vi.fn(async () => {
      throw new Error("decode failed");
    });

    const opened = await openPrintDialogForSingleMessage({
      mediaUrl: "blob:tiff-image",
      message: {
        id: "m-12",
        message_type: "document",
        media_filename: "scan.tif",
      },
      resolveBlob,
    });

    expect(opened).toBe(false);
    expect(rawPrintMock).not.toHaveBeenCalled();
    expect(mergePrintMock).not.toHaveBeenCalled();
  });

  it("returns false for non-printable document types", async () => {
    const resolveBlob = vi.fn(
      async () =>
        new Blob(["x"], {
          type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        }),
    );

    const opened = await openPrintDialogForSingleMessage({
      mediaUrl: "blob:docx-file",
      message: {
        id: "m-13",
        message_type: "document",
        media_filename: "contract.docx",
        media_mime_type:
          "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
      },
      resolveBlob,
    });

    expect(opened).toBe(false);
    expect(resolveBlob).not.toHaveBeenCalled();
    expect(rawPrintMock).not.toHaveBeenCalled();
    expect(mergePrintMock).not.toHaveBeenCalled();
  });
});
