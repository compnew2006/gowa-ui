import { describe, expect, it } from "vitest";
import {
  canAccessInstance,
  canUserAccessInstance,
  getAllowedInstanceIDsFromSettings,
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
});
