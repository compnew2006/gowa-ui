import { describe, expect, it } from "vitest";
import {
  canAccessInstance,
  canUserAccessInstance,
  getAllowedInstanceIDsFromSettings,
  getSendRestrictionsEnabled,
  getSendRestrictionsSettings,
  normalizeAllowedInstanceIDs,
} from "./instance-access";

describe("instance-access", () => {
  it("normalizes configured instance ids", () => {
    expect(
      normalizeAllowedInstanceIDs([
        " instance-a ",
        "instance-b",
        "instance-a",
        "",
        null,
      ]),
    ).toEqual(["instance-a", "instance-b"]);
  });

  it("prefers allowed_instance_ids over the legacy single value", () => {
    expect(
      getAllowedInstanceIDsFromSettings({
        send_restrictions: {
          allowed_instance_ids: ["instance-a", " instance-b "],
          allowed_instance_id: "legacy-instance",
        },
      }),
    ).toEqual(["instance-a", "instance-b"]);
  });

  it("falls back to the legacy allowed_instance_id value", () => {
    expect(
      getAllowedInstanceIDsFromSettings({
        send_restrictions: {
          allowed_instance_id: "legacy-instance",
        },
      }),
    ).toEqual(["legacy-instance"]);
  });

  it("allows unrestricted access when no instance restriction exists", () => {
    expect(canAccessInstance({}, "instance-a")).toBe(true);
  });

  it("allows access when the instance is present in the allowlist", () => {
    expect(
      canAccessInstance(
        {
          send_restrictions: {
            allowed_instance_ids: ["instance-a", "instance-b"],
          },
        },
        "instance-b",
      ),
    ).toBe(true);
  });

  it("rejects access when the instance is absent from the allowlist", () => {
    expect(
      canAccessInstance(
        {
          send_restrictions: {
            allowed_instance_ids: ["instance-a"],
          },
        },
        "instance-b",
      ),
    ).toBe(false);
  });

  it("reads settings from user objects", () => {
    expect(
      canUserAccessInstance(
        {
          settings: {
            send_restrictions: {
              allowed_instance_ids: ["instance-a"],
            },
          },
        },
        "instance-a",
      ),
    ).toBe(true);
  });

  it("detects enabled send restrictions", () => {
    expect(
      getSendRestrictionsEnabled({
        send_restrictions: { enabled: true },
      }),
    ).toBe(true);
    expect(
      getSendRestrictionsEnabled({
        send_restrictions: { enabled: false },
      }),
    ).toBe(false);
    expect(getSendRestrictionsEnabled({})).toBe(false);
  });

  it("detects configured send restriction settings even when disabled", () => {
    expect(
      getSendRestrictionsSettings({
        send_restrictions: {
          enabled: false,
          allowed_instance_ids: [],
        },
      }),
    ).not.toBeNull();
    expect(getSendRestrictionsSettings({})).toBeNull();
  });

  it("blocks access when restrictions are enabled without selected instances", () => {
    expect(
      canAccessInstance(
        {
          send_restrictions: {
            enabled: true,
            allowed_instance_ids: [],
          },
        },
        "instance-a",
      ),
    ).toBe(false);
  });

  it("blocks access when restrictions are configured without selected instances", () => {
    expect(
      canAccessInstance(
        {
          send_restrictions: {
            enabled: false,
            allowed_instance_ids: [],
          },
        },
        "instance-a",
      ),
    ).toBe(false);
  });
});
