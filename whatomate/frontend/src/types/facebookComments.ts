export type FacebookCommentStatus = "open" | "replied" | "closed" | "archived";

export interface FacebookCommentReply {
  id: string;
  user_id: string;
  reply_text: string;
  private_message_text: string;
  graph_comment_reply_id?: string;
  graph_private_reply_id?: string;
  status: "sent" | "partial" | "failed" | string;
  error_message?: string;
  is_auto: boolean;
  created_at: string;
}

export interface FacebookComment {
  id: string;
  account_id: string;
  page_id: string;
  page_name: string;
  post_id: string;
  post_permalink?: string;
  post_message?: string;
  external_id: string;
  parent_id?: string;
  from_id: string;
  from_name: string;
  message: string;
  permalink?: string;
  status: FacebookCommentStatus;
  direction: "incoming" | "outgoing";
  is_admin_reply: boolean;
  commented_at: string;
  last_synced_at?: string;
  last_replied_at?: string;
  auto_replied_at?: string;
  metadata?: Record<string, unknown>;
  replies: FacebookCommentReply[];
}

export interface FacebookCommentSettings {
  id?: string;
  organization_id?: string;
  enabled: boolean;
  sync_enabled: boolean;
  auto_reply_enabled: boolean;
  auto_comment_reply_enabled: boolean;
  auto_private_reply_enabled: boolean;
  auto_comment_reply_text: string;
  auto_private_message_text: string;
  only_auto_reply_unanswered: boolean;
  ignore_page_admin_comments: boolean;
  default_sync_post_limit: number;
  default_sync_comments_per_post: number;
}

export interface FacebookCommentsListResponse {
  comments: FacebookComment[];
  total: number;
  page: number;
  limit: number;
}
