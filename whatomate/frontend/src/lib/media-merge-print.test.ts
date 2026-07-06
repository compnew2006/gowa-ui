import { describe, expect, it } from "vitest";

import {
  isMergePrintableFile,
  isMergePrintableMimeType,
} from "./media-merge-print";

describe("isMergePrintableMimeType", () => {
  it.each([
    { mimeType: "application/pdf", expected: true },
    { mimeType: "image/jpeg", expected: true },
    { mimeType: "image/png", expected: true },
    { mimeType: "image/tiff", expected: true },
    { mimeType: "IMAGE/WEBP", expected: true },
    { mimeType: "application/msword", expected: false },
    { mimeType: "text/plain", expected: false },
    { mimeType: "", expected: false },
  ])("returns $expected for $mimeType", ({ mimeType, expected }) => {
    expect(isMergePrintableMimeType(mimeType)).toBe(expected);
  });
});

describe("isMergePrintableFile", () => {
  it("accepts image and pdf files only", () => {
    const files = [
      { type: "application/pdf" },
      { type: "image/jpeg" },
      { type: "image/png" },
      { type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document" },
    ];

    expect(isMergePrintableFile(files[0] as File)).toBe(true);
    expect(isMergePrintableFile(files[1] as File)).toBe(true);
    expect(isMergePrintableFile(files[2] as File)).toBe(true);
    expect(isMergePrintableFile(files[3] as File)).toBe(false);
  });
});
