// @vitest-environment happy-dom

import { describe, expect, it } from "vitest";
import {
  applyCommentCreated,
  applyCommentUpdated,
  isReplySkipped,
} from "./facebookCommentsMerge";
import type { FacebookComment, FacebookCommentReply } from "@/types/facebookComments";

function makeReply(overrides: Partial<FacebookCommentReply> = {}): FacebookCommentReply {
  return {
    id: "r-1",
    user_id: "u-1",
    reply_text: "hi",
    private_message_text: "",
    status: "sent",
    is_auto: false,
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function makeComment(overrides: Partial<FacebookComment> = {}): FacebookComment {
  return {
    id: "c-1",
    account_id: "acc-1",
    page_id: "page-1",
    page_name: "Page 1",
    post_id: "post-1",
    external_id: "ext-1",
    from_id: "psid-1",
    from_name: "Alice",
    message: "hi",
    status: "open",
    direction: "incoming",
    is_admin_reply: false,
    commented_at: "2026-01-01T00:00:00Z",
    replies: [],
    ...overrides,
  };
}

describe("applyCommentCreated", () => {
  it("returns the same list for null payload", () => {
    const list = [makeComment({ id: "c-1" })];
    const result = applyCommentCreated(list, null as unknown as FacebookComment);
    expect(result.appended).toBe(false);
    expect(result.replaced).toBe(false);
    expect(result.comments).toBe(list);
  });

  it("returns the same list for payload with no id", () => {
    const list = [makeComment({ id: "c-1" })];
    const result = applyCommentCreated(list, makeComment({ id: "" }));
    expect(result.appended).toBe(false);
    expect(result.replaced).toBe(false);
    expect(result.comments).toBe(list);
  });

  it("ignores duplicate id (idempotent prepend)", () => {
    const list = [makeComment({ id: "c-1" })];
    const result = applyCommentCreated(list, makeComment({ id: "c-1" }));
    expect(result.appended).toBe(false);
    expect(result.replaced).toBe(false);
    expect(result.comments).toHaveLength(1);
  });

  it("prepends a new comment to the head of the list", () => {
    const list = [makeComment({ id: "c-1" })];
    const fresh = makeComment({ id: "c-2", message: "new" });
    const result = applyCommentCreated(list, fresh);
    expect(result.appended).toBe(true);
    expect(result.replaced).toBe(false);
    expect(result.prependIndex).toBe(0);
    expect(result.comments).toHaveLength(2);
    expect(result.comments[0].id).toBe("c-2");
    expect(result.comments[1].id).toBe("c-1");
  });
});

describe("applyCommentUpdated", () => {
  it("returns the same list for null payload", () => {
    const list = [makeComment({ id: "c-1" })];
    const result = applyCommentUpdated(list, null as unknown as FacebookComment);
    expect(result.replaced).toBe(false);
    expect(result.appended).toBe(false);
    expect(result.comments).toBe(list);
  });

  it("prepends when the id is not present in the list", () => {
    const list = [makeComment({ id: "c-1" })];
    const fresh = makeComment({ id: "c-2" });
    const result = applyCommentUpdated(list, fresh);
    expect(result.appended).toBe(true);
    expect(result.replaced).toBe(false);
    expect(result.comments).toHaveLength(2);
    expect(result.comments[0].id).toBe("c-2");
  });

  it("replaces in place and preserves the original replies when payload omits them", () => {
    const original = makeComment({
      id: "c-1",
      message: "old",
      status: "open",
      replies: [makeReply({ id: "r-1", reply_text: "hi" })],
    });
    const list = [makeComment({ id: "c-2" }), original];
    const updated = makeComment({ id: "c-1", message: "new", status: "replied" });
    const updatedWithoutReplies = { ...updated, replies: undefined } as unknown as FacebookComment;
    const result = applyCommentUpdated(list, updatedWithoutReplies);
    expect(result.replaced).toBe(true);
    expect(result.appended).toBe(false);
    expect(result.prependIndex).toBe(1);
    expect(result.comments).toHaveLength(2);
    expect(result.comments[1].id).toBe("c-1");
    expect(result.comments[1].message).toBe("new");
    expect(result.comments[1].status).toBe("replied");
    expect(result.comments[1].replies).toHaveLength(1);
    expect(result.comments[1].replies[0].id).toBe("r-1");
  });

  it("uses payload replies when the payload includes them", () => {
    const original = makeComment({ id: "c-1", message: "old" });
    const list = [original];
    const updated = makeComment({
      id: "c-1",
      message: "new",
      replies: [makeReply({ id: "r-2", reply_text: "fresh" })],
    });
    const result = applyCommentUpdated(list, updated);
    expect(result.replaced).toBe(true);
    expect(result.comments[0].replies).toHaveLength(1);
    expect(result.comments[0].replies[0].id).toBe("r-2");
  });
});

describe("isReplySkipped", () => {
  it("returns true for status === skipped", () => {
    expect(isReplySkipped({ status: "skipped" })).toBe(true);
  });

  it("returns true for metadata.dm_skipped === true", () => {
    expect(isReplySkipped({ status: "sent", metadata: { dm_skipped: true } })).toBe(true);
  });

  it("returns false for a normal sent reply", () => {
    expect(isReplySkipped({ status: "sent", metadata: null })).toBe(false);
  });

  it("returns false for a failed reply without dm_skipped", () => {
    expect(isReplySkipped({ status: "failed" })).toBe(false);
  });

  it("returns false for null/undefined", () => {
    expect(isReplySkipped(null)).toBe(false);
    expect(isReplySkipped(undefined)).toBe(false);
  });
});
