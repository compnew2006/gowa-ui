import { describe, expect, it } from "vitest";

import ar from "./locales/ar.json";
import en from "./locales/en.json";
import es from "./locales/es.json";

const requiredKeys = [
  "softDeleteChat",
  "softDeleteConfirm",
  "softDeleteSuccess",
  "softDeleteFailed",
  "chatDeletedByUserNotification",
] as const;

describe("chat soft-delete locale coverage", () => {
  it("keeps the required chat soft-delete strings in en, ar, and es", () => {
    for (const messages of [en, ar, es]) {
      for (const key of requiredKeys) {
        expect(messages.chat[key]).toEqual(expect.any(String));
        expect(messages.chat[key].trim().length).toBeGreaterThan(0);
      }
    }
  });
});
