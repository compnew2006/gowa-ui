import { z } from "zod";
import { clearSession } from "@/lib/session";

const BASE = "/api";
const DEFAULT_TIMEOUT_MS = 20_000;

const ApiErrorSchema = z.object({
  detail: z.unknown().optional(),
  message: z.string().optional(),
  error: z.string().optional(),
});

export class ApiError extends Error {
  status: number;
  detail?: unknown;

  constructor(status: number, message: string, detail?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.detail = detail;
  }
}

export type ApiRequestInit = RequestInit & { timeoutMs?: number };

export async function apiFetch<T>(path: string, init?: ApiRequestInit): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), init?.timeoutMs ?? DEFAULT_TIMEOUT_MS);
  const headers = new Headers(init?.headers);
  const isFormData = init?.body instanceof FormData;
  if (!isFormData && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");

  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers,
    credentials: init?.credentials ?? "include",
    signal: init?.signal ?? controller.signal,
  }).finally(() => clearTimeout(timeout));

  if (!res.ok) {
    if (res.status === 401 && typeof window !== "undefined") {
      clearSession();
      // Prevent infinite redirect loops on login page itself
      if (!window.location.pathname.startsWith("/login")) {
        window.location.href = "/login";
      }
    }
    const payload = await res.json().catch(() => ({ detail: `HTTP ${res.status}` }));
    const parsed = ApiErrorSchema.safeParse(payload);
    const detail = parsed.success ? parsed.data.detail : payload;
    const message =
      (parsed.success && (parsed.data.message || parsed.data.error)) ||
      (typeof detail === "string" ? detail : undefined) ||
      `HTTP ${res.status}`;
    throw new ApiError(res.status, message, detail);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export type PagePayload = Pick<Page, "platform" | "page_id" | "name" | "is_active" | "auto_reply_enabled" | "shadow_mode" | "auto_reply_end_date"> & {
  avatar_url?: string;
  access_token_encrypted?: string;
};
export type SettingsPayload = Partial<Omit<AppSettings, "id" | "created_at" | "updated_at">>;
export type TeamMemberPayload = Pick<TeamMember, "name" | "email" | "role"> & { permissions?: Record<string, boolean>; is_active?: boolean };
export type RulePayload = Pick<AutomationRule, "name" | "conditions" | "condition_logic" | "action" | "action_config" | "priority" | "is_active"> & { description?: string };
export type IntegrationPayload = Pick<Integration, "type" | "name" | "config" | "trigger_events" | "is_active">;
export type CampaignPayload = Omit<Campaign, "id" | "status" | "total_recipients" | "sent_count" | "failed_count" | "created_by" | "created_at" | "updated_at">;
export type KnowledgeBasePayload = Pick<KnowledgeBaseEntry, "category" | "question" | "answer" | "language" | "is_active" | "intent_tags"> & { page_id?: string | null };
export type AgencyProfilePayload = Partial<Pick<AgencyProfile, "agency_name" | "logo_url" | "primary_color" | "support_email" | "dashboard_title" | "custom_domain">>;
export interface PostPayload { page_id: string; platform: string; message: string; scheduled_at: string; }
export interface PostContentRequest { page_id?: string; platform: string; prompt?: string; tone?: string; language?: string; }
export interface PostContentResponse { message: string; }

export const api = {
  // Pages
  getPages: () => apiFetch<Page[]>("/pages"),
  createPage: (data: PagePayload) => apiFetch<Page>("/pages", { method: "POST", body: JSON.stringify(data) }),
  updatePage: (id: string, data: Partial<PagePayload>) => apiFetch<Page>(`/pages/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deletePage: (id: string) => apiFetch<void>(`/pages/${id}`, { method: "DELETE" }),

  // Conversations
  getConversations: (params?: Record<string, string>) => {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return apiFetch<ConversationList>(`/conversations${q}`);
  },
  getConversation: (id: string) => apiFetch<Conversation>(`/conversations/${id}`),
  manualReply: (id: string, reply: string) =>
    apiFetch<Conversation>(`/conversations/${id}/reply`, { method: "POST", body: JSON.stringify({ reply }) }),
  resolveConversation: (id: string) =>
    apiFetch<Conversation>(`/conversations/${id}/resolve`, { method: "POST" }),
  approveReply: (id: string, reply?: string) =>
    apiFetch<Conversation>(`/conversations/${id}/approve`, { method: "POST", ...(reply ? { body: JSON.stringify({ reply }) } : {}) }),

  // Shadow mode
  approveShadow: (id: string) =>
    apiFetch<any>(`/shadow-mode/${id}/approve`, { method: "POST" }),
  rejectShadow: (id: string, reason?: string, correctIntent?: string, correctSentiment?: string) =>
    apiFetch<any>(`/shadow-mode/${id}/reject`, { method: "POST", body: JSON.stringify({ reason, correct_intent: correctIntent, correct_sentiment: correctSentiment }) }),
  undoShadow: (id: string) =>
    apiFetch<any>(`/shadow-mode/${id}/undo`, { method: "POST" }),
  correctShadow: (id: string, correctIntent?: string, correctSentiment?: string) =>
    apiFetch<any>(`/shadow-mode/${id}/correct`, { method: "PATCH", body: JSON.stringify({ correct_intent: correctIntent, correct_sentiment: correctSentiment }) }),

  // Customers
  getCustomers: (params?: Record<string, string>) => {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return apiFetch<CustomerList>(`/customers${q}`);
  },
  getCustomer: (id: string) => apiFetch<Customer>(`/customers/${id}`),
  updateCustomer: (id: string, data: Partial<Customer>) =>
    apiFetch<Customer>(`/customers/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  addNote: (id: string, content: string) =>
    apiFetch<Customer>(`/customers/${id}/notes`, { method: "POST", body: JSON.stringify({ content }) }),
  refreshCustomerPredictions: (id: string) =>
    apiFetch<Customer>(`/customers/${id}/refresh-predictions`, { method: "POST" }),
  getCustomerConversations: (id: string) =>
    apiFetch<Conversation[]>(`/customers/${id}/conversations`),
  deleteCustomer: (id: string) =>
    apiFetch<void>(`/customers/${id}`, { method: "DELETE" }),
  bulkUpdateCustomers: (ids: string[], update: Partial<Customer>) =>
    apiFetch<Customer[]>(`/customers/bulk`, { method: "PATCH", body: JSON.stringify({ ids, update }) }),
  bulkDeleteCustomers: (ids: string[]) =>
    apiFetch<void>(`/customers/bulk-delete`, { method: "POST", body: JSON.stringify({ ids }) }),

  // Escalations
  getEscalations: (params?: Record<string, string>) => {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return apiFetch<EscalationList>(`/escalations${q}`);
  },
  resolveEscalation: (id: string, notes?: string) =>
    apiFetch<Escalation>(`/escalations/${id}/resolve`, {
      method: "POST",
      body: JSON.stringify({ admin_notes: notes, resolved_by: "admin" }),
    }),

  // Knowledge base
  getKnowledgeBase: (params?: Record<string, string>) => {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return apiFetch<KnowledgeBaseEntry[]>(`/knowledge-base${q}`);
  },
  createKBEntry: (data: KnowledgeBasePayload) => apiFetch<KnowledgeBaseEntry>("/knowledge-base", { method: "POST", body: JSON.stringify(data) }),
  updateKBEntry: (id: string, data: Partial<KnowledgeBasePayload>) =>
    apiFetch<KnowledgeBaseEntry>(`/knowledge-base/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deleteKBEntry: (id: string) => apiFetch<void>(`/knowledge-base/${id}`, { method: "DELETE" }),

  // Settings (per-page)
  getSettings: (pageId?: string | null) => apiFetch<AppSettings>(`/settings${pageId ? `?page_id=${pageId}` : ""}`),
  updateSettings: (data: SettingsPayload, pageId?: string | null) => apiFetch<AppSettings>(`/settings${pageId ? `?page_id=${pageId}` : ""}`, { method: "PATCH", body: JSON.stringify(data) }),

  // Tokens
  getTokens: () => apiFetch<TokenStatus[]>("/tokens"),
  refreshToken: (id: string) => apiFetch<Page>(`/tokens/${id}/refresh`, { method: "POST" }),

  // Analytics (per-page)
  getDashboardStats: (pageId?: string | null) => apiFetch<DashboardStats>(`/analytics/dashboard${pageId ? `?page_id=${pageId}` : ""}`),
  getConversationAnalytics: (period = "7d", pageId?: string | null) => apiFetch<ConversationPoint[]>(`/analytics/conversations?period=${period}${pageId ? `&page_id=${pageId}` : ""}`),
  getIntentBreakdown: (pageId?: string | null) => apiFetch<IntentItem[]>(`/analytics/intents${pageId ? `?page_id=${pageId}` : ""}`),
  getSentimentBreakdown: (pageId?: string | null) => apiFetch<SentimentItem[]>(`/analytics/sentiment${pageId ? `?page_id=${pageId}` : ""}`),

  // Advanced Analytics (new)
  getAdvancedAnalyticsSummary: (period = "7d", pageId?: string | null) =>
    apiFetch<AdvancedAnalyticsSummary>(`/analytics/advanced-summary?period=${period}${pageId ? `&page_id=${pageId}` : ""}`),
  getROIMetrics: (pageId?: string | null) => apiFetch<ROIMetrics>(`/analytics/roi${pageId ? `?page_id=${pageId}` : ""}`),
  getConversionFunnel: (pageId?: string | null) => apiFetch<ConversionFunnel>(`/analytics/funnel${pageId ? `?page_id=${pageId}` : ""}`),
  getAIPerformanceTrend: (period = "7d", pageId?: string | null) => apiFetch<PerformancePoint[]>(`/analytics/performance?period=${period}${pageId ? `&page_id=${pageId}` : ""}`),
  getLanguageBreakdown: (pageId?: string | null) => apiFetch<LanguageItem[]>(`/analytics/language-breakdown${pageId ? `?page_id=${pageId}` : ""}`),
  getChurnRiskDistribution: (pageId?: string | null) => apiFetch<ChurnRiskData>(`/analytics/churn-risk${pageId ? `?page_id=${pageId}` : ""}`),
  getResponseTimeTrend: (period = "7d", pageId?: string | null) => apiFetch<ResponseTimeTrend[]>(`/analytics/response-time-trend?period=${period}${pageId ? `&page_id=${pageId}` : ""}`),

  // Team
  getTeamMembers: () => apiFetch<TeamMember[]>("/teams"),
  createTeamMember: (data: TeamMemberPayload) => apiFetch<TeamMember>("/teams", { method: "POST", body: JSON.stringify(data) }),
  updateTeamMember: (id: string, data: Partial<TeamMemberPayload>) => apiFetch<TeamMember>(`/teams/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deleteTeamMember: (id: string) => apiFetch<void>(`/teams/${id}`, { method: "DELETE" }),
  getAuditLog: (params?: Record<string, any>) => {
    const q = params ? "?" + new URLSearchParams(
      Object.fromEntries(Object.entries(params).map(([k, v]) => [k, String(v)]))
    ).toString() : "";
    return apiFetch<AuditLogData>(`/teams/audit-log${q}`);
  },

  // Automation Rules (per-page)
  getRules: (pageId?: string | null) => apiFetch<AutomationRule[]>(`/rules${pageId ? `?page_id=${pageId}` : ""}`),
  createRule: (data: RulePayload) => apiFetch<AutomationRule>("/rules", { method: "POST", body: JSON.stringify(data) }),
  updateRule: (id: string, data: Partial<RulePayload>) => apiFetch<AutomationRule>(`/rules/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deleteRule: (id: string) => apiFetch<void>(`/rules/${id}`, { method: "DELETE" }),
  testRule: (id: string, context: any) =>
    apiFetch<any>(`/rules/${id}/test`, { method: "POST", body: JSON.stringify(context) }),

  // Bulk Operations
  bulkConversationAction: (ids: string[], action: string, value?: string) =>
    apiFetch<any>("/bulk/conversations/action", { method: "POST", body: JSON.stringify({ ids, action, value }) }),
  bulkTagCustomers: (ids: string[], tag: string) =>
    apiFetch<any>("/bulk/customers/tag", { method: "POST", body: JSON.stringify({ ids, tag }) }),
  bulkReEngage: (filters: Record<string, string>) =>
    apiFetch<any>("/bulk/customers/re-engage", { method: "POST", body: JSON.stringify(filters) }),

  // Integrations
  getIntegrations: () => apiFetch<Integration[]>("/integrations"),
  createIntegration: (data: IntegrationPayload) => apiFetch<Integration>("/integrations", { method: "POST", body: JSON.stringify(data) }),
  updateIntegration: (id: string, data: Partial<IntegrationPayload>) => apiFetch<Integration>(`/integrations/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deleteIntegration: (id: string) => apiFetch<void>(`/integrations/${id}`, { method: "DELETE" }),
  testIntegration: (id: string) =>
    apiFetch<{ success: boolean; error?: string }>(`/integrations/${id}/test`, { method: "POST" }),

  // Audit Logs
  getAuditLogs: (params?: Record<string, string>) => {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return apiFetch<AuditLogData>(`/audit-logs${q}`);
  },
  getAuditStats: (params?: Record<string, string>) => {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return apiFetch<Record<string, unknown>>(`/audit-logs/stats${q}`);
  },

  // Compliance
  scanPII: (conversationId: string) => apiFetch<any>(`/compliance/pii-scan/${conversationId}`),
  gdprExport: (customerId: string) =>
    apiFetch<any>(`/compliance/gdpr-export/${customerId}`, { method: "POST" }),
  gdprDelete: (customerId: string) =>
    apiFetch<any>(`/compliance/gdpr-delete/${customerId}`, { method: "POST" }),
  getDataRetention: () => apiFetch<DataRetentionStats>("/compliance/data-retention"),
  getAuditSummary: () => apiFetch<AuditSummary>("/compliance/audit-summary"),

  // Campaigns
  getCampaigns: (params?: Record<string, string>) => {
    const q = params ? "?" + new URLSearchParams(params).toString() : "";
    return apiFetch<CampaignList>(`/campaigns${q}`);
  },
  getCampaign: (id: string) => apiFetch<Campaign>(`/campaigns/${id}`),
  createCampaign: (data: CampaignPayload & { page_id?: string | null }) => apiFetch<Campaign>("/campaigns", { method: "POST", body: JSON.stringify(data) }),
  updateCampaign: (id: string, data: Partial<CampaignPayload>) =>
    apiFetch<Campaign>(`/campaigns/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deleteCampaign: (id: string) => apiFetch<void>(`/campaigns/${id}`, { method: "DELETE" }),
  activateCampaign: (id: string) => apiFetch<Campaign>(`/campaigns/${id}/activate`, { method: "POST" }),
  pauseCampaign: (id: string) => apiFetch<Campaign>(`/campaigns/${id}/pause`, { method: "POST" }),
  previewCampaignRecipients: (id: string) => apiFetch<CampaignPreview>(`/campaigns/${id}/preview-recipients`),
  previewCampaignAudience: (data: Pick<CampaignPayload, "target_filter" | "customer_ids"> & { page_id?: string | null }) =>
    apiFetch<CampaignPreview>("/campaigns/preview-recipients", { method: "POST", body: JSON.stringify(data) }),
  uploadCampaignMedia: (file: File) => {
    const form = new FormData();
    form.append("file", file);
    return apiFetch<{ url: string; media_type: string; filename: string }>("/campaigns/upload-media", {
      method: "POST",
      body: form,
    });
  },

  // Agency profile
  getAgencyProfile: () => apiFetch<AgencyProfile>("/settings/agency-profile"),
  updateAgencyProfile: (data: AgencyProfilePayload) => apiFetch<AgencyProfile>("/settings/agency-profile", { method: "PATCH", body: JSON.stringify(data) }),
  uploadAgencyLogo: (file: File) => {
    const form = new FormData();
    form.append("file", file);
    return apiFetch<{ url: string; filename: string }>("/settings/agency-profile/logo", { method: "POST", body: form });
  },

  // Posts (Hermes Parity)
  getPosts: (pageId?: string) => {
    const q = pageId ? `?page_id=${pageId}` : "";
    return apiFetch<any[]>(`/posts${q}`);
  },
  createPost: (data: PostPayload) => apiFetch<Post>("/posts", { method: "POST", body: JSON.stringify(data) }),
  deletePost: (id: string) => apiFetch<void>(`/posts/${id}`, { method: "DELETE" }),
  generatePostContent: (data: PostContentRequest) =>
    apiFetch<PostContentResponse>("/posts/generate-content", { method: "POST", body: JSON.stringify(data) }),

  // Custom AI Models
  getAIModels: () => apiFetch<CustomAIModel[]>("/ai-models"),
  createAIModel: (data: Partial<CustomAIModel> & { api_key: string }) => apiFetch<CustomAIModel>("/ai-models", { method: "POST", body: JSON.stringify(data) }),
  updateAIModel: (id: string, data: Partial<CustomAIModel> & { api_key?: string }) => apiFetch<CustomAIModel>(`/ai-models/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deleteAIModel: (id: string) => apiFetch<void>(`/ai-models/${id}`, { method: "DELETE" }),
  testAIModel: (id: string) => apiFetch<{ status: "success" | "error"; response?: string; message?: string }>(`/ai-models/${id}/test`, { method: "POST" }),

  // Authentication
  login: (data: any) => apiFetch<any>("/auth/login", { method: "POST", body: JSON.stringify(data) }),
  register: (data: any) => apiFetch<any>("/auth/register", { method: "POST", body: JSON.stringify(data) }),
  logout: () => apiFetch<{ ok: boolean }>("/auth/logout", { method: "POST" }),
  getMe: () => apiFetch<any>("/auth/me"),
};

// ─── TypeScript Interfaces ────────────────────────────────────────────────

export interface CustomAIModel {
  id: string;
  name: string;
  provider: string;
  model_name: string;
  api_key_masked: string;
  api_base?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface AgencyProfile {
  id: string; agency_name: string; logo_url?: string;
  primary_color: string; support_email?: string;
  dashboard_title: string; is_whitelabeled: boolean;
  custom_domain?: string; created_at: string; updated_at: string;
}

export interface Page {
  id: string; platform: string; page_id: string; name: string;
  avatar_url?: string; is_active: boolean; auto_reply_enabled: boolean;
  shadow_mode: boolean; token_status?: string; token_expires_at?: string;
  token_last_refreshed_at?: string; token_last_error?: string;
  auto_reply_end_date?: string;
  created_at: string; updated_at: string;
}

export interface Conversation {
  id: string; page_id: string; page_name: string; platform: string;
  comment_id: string; post_id: string; customer_id?: string; customer_name: string;
  customer_avatar_url?: string; original_comment: string; ai_reply?: string;
  admin_reply?: string; status: string; intent?: string; sentiment?: string;
  confidence_score?: number; language?: string; is_shadow_mode: boolean;
  sentiment_history: string[]; escalation_reason?: string; processing_time?: number;
  guardrail_triggered?: boolean; guardrail_reason?: string;
  pii_detected?: boolean; matched_rule_id?: string;
  replied_at?: string; created_at: string; updated_at: string;
}
export interface ConversationList { data: Conversation[]; total: number; page: number; limit: number; }

export interface Customer {
  id: string; facebook_id?: string; instagram_id?: string; whatsapp_id?: string; username?: string;
  full_name?: string; profile_url?: string; avatar_url?: string;
  page_name?: string; platform?: string;
  first_contact_date?: string; last_interaction?: string; interaction_count: number;
  lead_score: number; purchase_intent: string; conversion_status: string;
  assigned_admin?: string; tags: string[]; notes: any[]; escalation_history: string[];
  churn_risk: string; churn_risk_score: number; next_best_action?: string;
  re_engage_sent_at?: string; gdpr_deleted?: boolean;
  created_at: string; updated_at: string;
}
export interface CustomerList { data: Customer[]; total: number; page: number; limit: number; }

export interface Escalation {
  id: string; conversation_id: string; page_id: string; page_name: string;
  customer_id?: string; customer_name: string; original_comment: string;
  reason: string; priority: string; status: string; assigned_to?: string;
  admin_notes?: string; resolved_by?: string; resolved_at?: string;
  created_at: string; updated_at: string;
}
export interface EscalationList { data: Escalation[]; total: number; page: number; limit: number; }

export interface KnowledgeBaseEntry {
  id: string; category: string; question: string; answer: string;
  intent_tags: string[]; language: string; is_active: boolean;
  usage_count: number; quality_score?: number; created_at: string; updated_at: string;
}

export interface AppSettings {
  id: string; confidence_threshold: number; auto_escalate_angry: boolean;
  telegram_bot_token?: string; telegram_chat_id?: string;
  primary_llm_model: string; fallback_llm_model: string;
  whatsapp_notification_phone?: string;
  whatsapp_notification_api_key?: string;
  enable_private_replies: boolean;
  whatsapp_business_phone_number_id?: string;
  whatsapp_cloud_api_token?: string;
  webhook_verify_token: string; max_retries: number;
  rate_limit_warning_threshold: number; default_language: string;
  warmup_mode: boolean; safe_reply_ar?: string; safe_reply_en?: string;
  public_reply_message_ar?: string; public_reply_message_en?: string;
  reply_mode?: string; auto_reply_start_date?: string; auto_reply_end_date?: string;
  brand_description?: string;
  brand_industry?: string;
  brand_target_audience?: string;
  brand_tone_of_voice?: string;
  brand_preferred_hashtags?: string;
  brand_restricted_words?: string;
  brand_sample_posts?: string;
  created_at: string; updated_at: string;
}

export interface TokenStatus {
  id: string; name: string; platform: string; token_status?: string;
  token_expires_at?: string; token_last_refreshed_at?: string; token_last_error?: string;
}

export interface DashboardStats {
  total_conversations: number; pending_conversations: number; open_escalations: number;
  total_customers: number; high_intent_leads: number; avg_confidence_score: number;
  auto_reply_rate: number; avg_response_time_seconds: number; shadow_mode_reviews: number;
  token_healthy: number; token_expiring_soon: number; token_expired: number;
}

export interface ConversationPoint { date: string; total: number; replied: number; escalated: number; }
export interface IntentItem { intent: string; count: number; percentage: number; }
export interface SentimentItem { sentiment: string; count: number; percentage: number; }

// Advanced Analytics
export interface ROIMetrics {
  total_comments_processed: number; auto_replied: number; auto_reply_rate_pct: number;
  avg_ai_response_time_sec: number; estimated_time_saved_hours: number;
  estimated_cost_saved_usd: number; high_intent_leads_generated: number;
  converted_customers: number; conversion_rate_pct: number;
}
export interface ConversionFunnel {
  stages: { stage: string; label: string; count: number; pct: number }[];
}
export interface PerformancePoint {
  date: string; total: number; avg_confidence: number;
  auto_reply_rate: number; escalation_rate: number;
}
export interface LanguageItem { language: string; count: number; percentage: number; }
export interface ChurnRiskData {
  total_customers: number;
  distribution: { low: number; medium: number; high: number };
  high_risk_customers: {
    id: string; name: string; churn_risk_score: number;
    next_best_action?: string; last_interaction?: string;
  }[];
}
export interface ResponseTimeTrend { date: string; avg_response_time_sec: number; }
export interface AdvancedAnalyticsSummary {
  roi: ROIMetrics;
  funnel: ConversionFunnel;
  performance: PerformancePoint[];
  language_breakdown: LanguageItem[];
  churn_risk: ChurnRiskData;
  response_time_trend: ResponseTimeTrend[];
}

// Team
export interface TeamMember {
  id: string; email: string; name: string; role: string;
  is_active: boolean; avatar_url?: string; telegram_user_id?: string;
  permissions: Record<string, boolean>; last_active_at?: string; created_at: string;
}
export interface AuditLogData {
  total: number; page: number;
  data: {
    id: string; admin_name: string; action: string;
    entity_type: string; entity_id?: string; details: any; created_at: string;
  }[];
}

// Rules
export interface AutomationRule {
  id: string; name: string; description?: string;
  conditions: { field: string; op: string; value: string }[];
  condition_logic: string; action: string; action_config: any;
  priority: number; is_active: boolean; trigger_count: number;
  last_triggered_at?: string; created_at: string;
}

// Integrations
export interface Integration {
  id: string; type: string; name: string; config: any;
  is_active: boolean; trigger_events: string[];
  trigger_count: number; last_triggered_at?: string;
  last_error?: string; created_at: string;
}

// Compliance
export interface DataRetentionStats {
  retention_policy_days: number; resolved_older_than_90d: number;
  all_conversations_older_than_365d: number; total_customers: number;
  gdpr_deleted_customers: number; gdpr_deletion_rate: number; recommendation: string;
}
export interface AuditSummary {
  total_audit_events: number;
  by_action: { action: string; count: number }[];
}

// Campaigns
export interface Campaign {
  id: string;
  name: string;
  description?: string;
  status: string;
  target_filter: Record<string, string>;
  customer_ids: string[];
  message_ar: string;
  message_en: string;
  media_urls: string[];
  media_type?: string;
  send_at?: string;
  interval_hours?: number;
  max_sends?: number;
  total_recipients: number;
  sent_count: number;
  failed_count: number;
  created_by?: string;
  created_at: string;
  updated_at: string;
}
export interface CampaignList { data: Campaign[]; total: number; page: number; limit: number; }
export interface CampaignPreview { count: number; sample: { id: string; name: string }[]; }

export interface Post {
  id: string;
  page_id: string;
  platform: string;
  message: string;
  scheduled_at: string;
  status: string;
  error?: string;
  created_at?: string;
  updated_at?: string;
}
