import type { FacebookComment, FacebookCommentReply } from "@/types/facebookComments";

export type FacebookCommentEvent = FacebookComment;

export interface MergeResult {
  comments: FacebookComment[];
  appended: boolean;
  replaced: boolean;
  prependIndex: number;
}

export function applyCommentCreated(
  current: FacebookComment[],
  payload: FacebookCommentEvent,
): MergeResult {
  if (!payload || !payload.id) {
    return { comments: current, appended: false, replaced: false, prependIndex: -1 };
  }
  if (current.some((c) => c.id === payload.id)) {
    return { comments: current, appended: false, replaced: false, prependIndex: -1 };
  }
  return {
    comments: [payload, ...current],
    appended: true,
    replaced: false,
    prependIndex: 0,
  };
}

export function applyCommentUpdated(
  current: FacebookComment[],
  payload: FacebookCommentEvent,
): MergeResult {
  if (!payload || !payload.id) {
    return { comments: current, appended: false, replaced: false, prependIndex: -1 };
  }
  const idx = current.findIndex((c) => c.id === payload.id);
  if (idx === -1) {
    return {
      comments: [payload, ...current],
      appended: true,
      replaced: false,
      prependIndex: 0,
    };
  }
  const payloadReplies = (payload as { replies?: FacebookCommentReply[] }).replies;
  const merged: FacebookComment = {
    ...current[idx],
    ...payload,
    replies: payloadReplies === undefined ? current[idx].replies : payloadReplies,
  };
  return {
    comments: [...current.slice(0, idx), merged, ...current.slice(idx + 1)],
    appended: false,
    replaced: true,
    prependIndex: idx,
  };
}

export function isReplySkipped(
  reply: { status: string; metadata?: Record<string, unknown> | null } | null | undefined,
): boolean {
  if (!reply) return false;
  if (reply.status === "skipped") return true;
  return reply.metadata?.dm_skipped === true;
}
