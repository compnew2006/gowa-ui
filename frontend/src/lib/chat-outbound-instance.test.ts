import { describe, expect, it } from "vitest";

import { resolvePreferredOutboundInstanceID } from "./chat-outbound-instance";
import type { Contact, Message } from "@/stores/contacts";

function buildContact(overrides: Partial<Contact> = {}): Contact {
  return {
    id: "contact-1",
    phone_number: "201234567890",
    instance_id: "instance-current",
    name: "Test Contact",
    status: "open",
    tags: [],
    metadata: {},
    unread_count: 0,
    created_at: "2026-01-01T00:00:00.000Z",
    updated_at: "2026-01-01T00:00:00.000Z",
    ...overrides,
  };
}

function buildMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: "message-1",
    contact_id: "contact-1",
    direction: "incoming",
    message_type: "text",
    content: { body: "hello" },
    status: "received",
    created_at: "2026-01-01T00:00:00.000Z",
    updated_at: "2026-01-01T00:00:00.000Z",
    ...overrides,
  };
}

describe("chat-outbound-instance", () => {
  it("prefers an explicitly selected source contact instance", () => {
    expect(
      resolvePreferredOutboundInstanceID({
        messages: [
          buildMessage({ instance_id: "instance-inbound" }),
        ],
        selectedSourceContact: buildContact({ instance_id: "instance-source" }),
        currentContact: buildContact(),
      }),
    ).toBe("instance-source");
  });

  it("falls back to the latest inbound message instance before the current contact", () => {
    expect(
      resolvePreferredOutboundInstanceID({
        messages: [
          buildMessage({
            id: "message-older",
            instance_id: "instance-old",
          }),
          buildMessage({
            id: "message-latest",
            instance_id: "instance-inbound",
            created_at: "2026-01-02T00:00:00.000Z",
            updated_at: "2026-01-02T00:00:00.000Z",
          }),
        ],
        currentContact: buildContact({ instance_id: "instance-current" }),
      }),
    ).toBe("instance-inbound");
  });

  it("falls back to the current contact instance when no inbound message instance is present", () => {
    expect(
      resolvePreferredOutboundInstanceID({
        messages: [
          buildMessage({
            direction: "outgoing",
            instance_id: "",
          }),
        ],
        currentContact: buildContact({ instance_id: " instance-current " }),
      }),
    ).toBe("instance-current");
  });

  it("falls back to the latest thread instance and then the sidebar instance filter", () => {
    expect(
      resolvePreferredOutboundInstanceID({
        messages: [
          buildMessage({
            direction: "outgoing",
            instance_id: "instance-thread",
          }),
        ],
        currentContact: buildContact({ instance_id: "" }),
        selectedInstanceID: "instance-filter",
      }),
    ).toBe("instance-thread");

    expect(
      resolvePreferredOutboundInstanceID({
        messages: [],
        currentContact: buildContact({ instance_id: "" }),
        selectedInstanceID: " instance-filter ",
      }),
    ).toBe("instance-filter");
  });
});
