import { describe, expect, it } from "vitest";
import { can, denyMessage } from "@/lib/permissions";

describe("permissions", () => {
  it("defaults to deny when no current-user permission contract exists", () => {
    expect(can("can_delete")).toBe(false);
  });

  it("uses explicit permission maps when provided", () => {
    expect(can("can_delete", { can_delete: false })).toBe(false);
    expect(can("can_export", { can_export: true })).toBe(true);
  });

  it("returns localized deny messages", () => {
    expect(denyMessage("can_manage_team")).toContain("الفريق");
  });
});
