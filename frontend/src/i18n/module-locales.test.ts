import { describe, expect, it } from "vitest";

import ar from "./locales/ar.json";
import en from "./locales/en.json";
import es from "./locales/es.json";

const requiredKeys = [
  "title",
  "description",
  "empty",
  "enabled",
  "disabled",
  "dependencies",
  "enableForOrganization",
  "disableForOrganization",
  "enableGlobally",
  "disableGlobally",
  "loadFailed",
  "updateFailed",
  "updated",
] as const;

describe("module administration locale coverage", () => {
  it("keeps module strings complete in en, ar, and es", () => {
    for (const messages of [en, ar, es]) {
      expect(messages.nav.modules).toEqual(expect.any(String));
      for (const key of requiredKeys) {
        expect(messages.modules[key]).toEqual(expect.any(String));
        expect(messages.modules[key].trim().length).toBeGreaterThan(0);
      }
    }
  });
});
