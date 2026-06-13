import axios, {
  type AxiosInstance,
  type AxiosError,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from "axios";
import type {
  Permission,
  UserSettings,
  ChatBackgroundSettings,
  ThemeMode,
  ThemePreset,
} from "@/types/auth";

function normalizeBasePath(value: unknown): string {
  const raw = typeof value === "string" ? value.trim() : "";
  if (raw === "" || raw === "." || raw === "./" || raw === "/") {
    return "";
  }

  const trimmed = raw.replace(/\/+$/, "");
  if (trimmed === "" || trimmed === ".") {
    return "";
  }

  if (trimmed.startsWith("/")) {
    return trimmed;
  }

  return `/${trimmed.replace(/^\.?\//, "")}`;
}

function normalizeApiBaseURL(value: string): string {
  const raw = value.trim();
  if (raw === "") {
    return "/api";
  }

  if (/^https?:\/\//i.test(raw) || raw.startsWith("//")) {
    return raw.replace(/\/+$/, "");
  }

  const trimmed = raw.replace(/\/+$/, "");
  if (trimmed.startsWith("/")) {
    return trimmed;
  }

  return `/${trimmed.replace(/^\.?\//, "")}`;
}

// Get base path from server-injected config or fallback.
// Keep API base URL absolute to avoid accidental relative calls like "api/statuses".
const basePath = normalizeBasePath(
  (window as Window & { __BASE_PATH__?: string }).__BASE_PATH__,
);
const API_BASE_URL = normalizeApiBaseURL(
  import.meta.env.VITE_API_URL || `${basePath}/api`,
);

export const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  withCredentials: true,
  headers: {
    "Content-Type": "application/json",
  },
});

type LicenseLockedHandler = (error: AxiosError) => void;

let licenseLockedHandler: LicenseLockedHandler | null = null;

export function setLicenseLockedHandler(handler: LicenseLockedHandler | null) {
  licenseLockedHandler = handler;
}

type SessionExpiredHandler = (error?: AxiosError) => void;

let sessionExpiredHandler: SessionExpiredHandler | null = null;

export function setSessionExpiredHandler(handler: SessionExpiredHandler | null) {
  sessionExpiredHandler = handler;
}

type JsonRecord = Record<string, unknown>;

export interface CurrentUserSettingsResponse {
  message: string;
  settings: UserSettings;
}

export interface UploadChatBackgroundResponse {
  message: string;
  chat_background: ChatBackgroundSettings;
}

// Helper to read a cookie by name
function getCookie(name: string): string | null {
  const match = document.cookie.match(
    new RegExp("(?:^|; )" + name + "=([^;]*)"),
  );
  return match ? decodeURIComponent(match[1]) : null;
}

function shouldAttachOrganizationHeader(url?: string): boolean {
  if (!url) {
    return true;
  }

  const normalizedUrl = url.startsWith("http")
    ? new URL(url, window.location.origin).pathname
    : url;

  return !(
    normalizedUrl === "/me" ||
    normalizedUrl.startsWith("/me/") ||
    normalizedUrl.startsWith("/auth/")
  );
}

// Request interceptor to add CSRF token and organization header
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Add CSRF token on mutating requests (cookie-based auth sends cookies automatically)
    const method = (config.method || "").toUpperCase();
    if (
      method === "POST" ||
      method === "PUT" ||
      method === "DELETE" ||
      method === "PATCH"
    ) {
      const csrfToken = getCookie("whm_csrf");
      if (csrfToken) {
        config.headers["X-CSRF-Token"] = csrfToken;
      }
    }
    // Add organization override header for org switching
    const selectedOrgId = localStorage.getItem("selected_organization_id");
    if (selectedOrgId && shouldAttachOrganizationHeader(config.url)) {
      config.headers["X-Organization-ID"] = selectedOrgId;
    }
    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  },
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // Skip token refresh logic for auth endpoints
    const isAuthEndpoint = originalRequest?.url?.startsWith("/auth/");

    if (error.response?.status === 423) {
      licenseLockedHandler?.(error);
      return Promise.reject(error);
    }

    // Handle 401 errors - try to refresh token (but not for auth endpoints)
    if (
      error.response?.status === 401 &&
      !originalRequest._retry &&
      !isAuthEndpoint
    ) {
      originalRequest._retry = true;

      try {
        const csrfToken = getCookie("whm_csrf");

        // Browser sends whm_refresh cookie automatically via withCredentials
        await axios.post(
          `${API_BASE_URL}/auth/refresh`,
          {},
          {
            withCredentials: true,
            headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
          },
        );

        // Cookies are updated by the server response — retry the original request
        return api(originalRequest);
      } catch (refreshError) {
        // Refresh failed: clear any cached user/legacy tokens and notify the
        // app so it can clear Pinia state, show a "session expired" toast,
        // and soft-redirect to /login (instead of a hard page reload).
        localStorage.removeItem("user");
        localStorage.removeItem("auth_token");
        localStorage.removeItem("refresh_token");
        if (sessionExpiredHandler) {
          sessionExpiredHandler(refreshError as AxiosError);
        } else {
          // Fallback when no handler is registered (e.g. in unit tests).
          window.location.href = basePath + "/login";
        }
      }
    }

    return Promise.reject(error);
  },
);

// API service methods
export const authService = {
  createRegisterInvite: (data?: { expires_in_hours?: number }) =>
    api.post("/auth/register/invite", data || {}),
  getWSToken: () => api.get("/auth/ws-token"),
};

export const usersService = {
  list: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get("/users", { params }),
  create: (data: {
    email: string;
    password: string;
    full_name: string;
    role_id?: string;
  }) => api.post("/users", data),
  update: (
    id: string,
    data: {
      email?: string;
      password?: string;
      full_name?: string;
      role_id?: string;
      is_active?: boolean;
    },
  ) => api.put(`/users/${id}`, data),
  delete: (id: string) => api.delete(`/users/${id}`),
  getSendRestrictions: (id: string) =>
    api.get(`/users/${id}/send-restrictions`),
  updateSendRestrictions: (
    id: string,
    data: {
      enabled?: boolean;
      include_all_contacts?: boolean;
      authorized_numbers?: string[];
      allowed_instance_ids?: string[];
      allowed_instance_id?: string | null;
      prefix_agent_name?: boolean;
      allow_unclaimed_chat_view?: boolean;
      allow_unclaimed_chat_send?: boolean;
    },
  ) => api.put(`/users/${id}/send-restrictions`, data),
  me: () => api.get("/me"),
  updateSettings: (data: {
    email_notifications?: boolean;
    new_message_alerts?: boolean;
    campaign_updates?: boolean;
    notification_sound?: "notification1" | "notification2" | "notification";
    theme_mode?: ThemeMode;
    theme_preset?: ThemePreset;
    chat_background?: ChatBackgroundSettings | null;
  }) => api.put<CurrentUserSettingsResponse>("/me/settings", data),
  uploadChatBackground: (file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return api.post<UploadChatBackgroundResponse>(
      "/me/chat-background",
      formData,
      {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      },
    );
  },
  changePassword: (data: { current_password: string; new_password: string }) =>
    api.put("/me/password", data),
  updateAvailability: (isAvailable: boolean) =>
    api.put("/me/availability", { is_available: isAvailable }),
  listMyOrganizations: () => api.get("/me/organizations"),
};

export const apiKeysService = {
  list: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get<{ api_keys: JsonRecord[]; total?: number }>("/api-keys", {
      params,
    }),
  create: (data: { name: string; expires_at?: string }) =>
    api.post("/api-keys", data),
  delete: (id: string) => api.delete(`/api-keys/${id}`),
};

export const accountsService = {
  list: () => api.get("/accounts"),
  delete: (id: string) => api.delete(`/accounts/${id}`),
};

export const contactsService = {
  list: (params?: {
    search?: string;
    page?: number;
    limit?: number;
    created_from?: string;
    created_to?: string;
    date_basis?: "created" | "incoming_any";
    date_from?: string;
    date_to?: string;
    tags?: string;
    instance_id?: string;
    chat_types?: string;
    status?: "pending" | "open" | "closed";
    assigned_to?: "me" | "unassigned" | string;
  }) => api.get("/contacts", { params }),
  get: (id: string) => api.get(`/contacts/${id}`),
  create: (data: JsonRecord) => api.post("/contacts", data),
  update: (id: string, data: JsonRecord) => api.put(`/contacts/${id}`, data),
  delete: (id: string) => api.delete(`/contacts/${id}`),
  softDelete: (id: string) => api.post(`/contacts/${id}/soft-delete`),
  assign: (id: string, userId: string | null) =>
    api.put(`/contacts/${id}/assign`, { user_id: userId }),
  updateTags: (id: string, tags: string[]) =>
    api.put(`/contacts/${id}/tags`, { tags }),
  getSessionData: (id: string) => api.get(`/contacts/${id}/session-data`),
  listCollaborators: (id: string) => api.get(`/contacts/${id}/collaborators`),
  inviteCollaborator: (id: string, data: { user_id: string; role?: string }) =>
    api.post(`/contacts/${id}/collaborators`, data),
  acceptCollaborator: (id: string, userId: string) =>
    api.put(`/contacts/${id}/collaborators/${userId}/accept`, {}),
  declineCollaborator: (id: string, userId: string) =>
    api.put(`/contacts/${id}/collaborators/${userId}/decline`, {}),
  removeCollaborator: (id: string, userId: string) =>
    api.delete(`/contacts/${id}/collaborators/${userId}`),
  retryMediaDownload: (messageId: string) =>
    api.post(`/media/${encodeURIComponent(messageId)}/retry-download`),
};

export const chatsService = {
  list: (params?: {
    search?: string;
    page?: number;
    limit?: number;
    tags?: string;
    instance_id?: string;
    chat_types?: string;
    status?: "pending" | "open" | "closed";
    assigned_to?: "me" | "unassigned" | string;
    closed_by?: string;
    closed_from?: string;
    closed_to?: string;
  }) => api.get("/chats", { params }),
  claim: (id: string) => api.put(`/chats/${id}/claim`),
  close: (id: string) => api.put(`/chats/${id}/close`),
  reopen: (id: string) => api.put(`/chats/${id}/reopen`),
  setPublic: (id: string, isPublic: boolean) =>
    api.put(`/chats/${id}/public`, { is_public: isPublic }),
  listMessages: (
    id: string,
    params?: {
      page?: number;
      limit?: number;
      before_id?: string;
      account?: string;
    },
  ) => api.get(`/chats/${id}/messages`, { params }),
};

// Generic Import/Export Service
export interface ExportColumn {
  key: string;
  label: string;
}

export interface ExportConfig {
  table: string;
  columns: ExportColumn[];
  default_columns: string[];
}

export interface ImportConfig {
  table: string;
  required_columns: ExportColumn[];
  optional_columns: ExportColumn[];
  unique_column: string;
}

export interface ImportResult {
  created: number;
  updated: number;
  skipped: number;
  errors: number;
  messages: string[];
}

export const dataService = {
  // Get export configuration for a table
  getExportConfig: (table: string) =>
    api.get<ExportConfig>(`/export/${table}/config`),

  // Get import configuration for a table
  getImportConfig: (table: string) =>
    api.get<ImportConfig>(`/import/${table}/config`),

  // Export data - returns CSV blob
  exportData: async (
    table: string,
    columns?: string[],
    filters?: Record<string, string>,
  ) => {
    const response = await api.post(
      "/export",
      { table, columns, filters },
      {
        responseType: "blob",
      },
    );
    return response;
  },

  // Import data from CSV file
  importData: (
    table: string,
    file: File,
    updateOnDuplicate?: boolean,
    columnMapping?: Record<string, string>,
  ) => {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("table", table);
    if (updateOnDuplicate) {
      formData.append("update_on_duplicate", "true");
    }
    if (columnMapping) {
      formData.append("column_mapping", JSON.stringify(columnMapping));
    }
    return api.post<ImportResult>("/import", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },
};

export const messagesService = {
  list: (
    contactId: string,
    params?: {
      page?: number;
      limit?: number;
      before_id?: string;
      account?: string;
    },
  ) => api.get(`/contacts/${contactId}/messages`, { params }),
  send: (
    contactId: string,
    data: {
      type: string;
      content: unknown;
      reply_to_message_id?: string;
      instance_id?: string;
      whatsapp_account?: string;
    },
  ) => api.post(`/contacts/${contactId}/messages`, data),
  sendTyping: (
    contactId: string,
    data: { state: "composing" | "paused"; instance_id?: string },
  ) => api.post(`/contacts/${contactId}/typing`, data),
  sendMedia: (data: {
    contactId: string;
    file: File;
    type: string;
    caption?: string;
    instance_id?: string;
    whatsapp_account?: string;
  }) => {
    const formData = new FormData();
    formData.append("file", data.file);
    formData.append("contact_id", data.contactId);
    formData.append("type", data.type);
    if (data.caption?.trim()) {
      formData.append("caption", data.caption.trim());
    }
    if (data.instance_id?.trim()) {
      formData.append("instance_id", data.instance_id.trim());
    }
    if (data.whatsapp_account?.trim()) {
      formData.append("whatsapp_account", data.whatsapp_account.trim());
    }
    return api.post("/messages/media", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },
  sendReaction: (contactId: string, messageId: string, emoji: string) =>
    api.post(`/contacts/${contactId}/messages/${messageId}/reaction`, {
      emoji,
    }),
  revoke: (contactId: string, messageId: string) =>
    api.post(`/contacts/${contactId}/messages/${messageId}/revoke`),

  sendPollVote: (messageId: string, selectedOptions: string[]) =>
    api.post('/messages/poll-vote', {
      message_id: messageId,
      selected_options: selectedOptions,
    }),
};

export interface WhatsAppStatusItem {
  id: string;
  instance_id: string;
  instance_name: string;
  sender_jid: string;
  sender_name: string;
  whatsapp_message_id: string;
  status_type: "text" | "image" | "video";
  content: string;
  media_url?: string;
  media_mime_type?: string;
  media_filename?: string;
  text_argb?: number;
  background_argb?: number;
  font?: string;
  is_self: boolean;
  seen_at?: string;
  created_at: string;
  expires_at: string;
}

export interface WhatsAppStatusGroup {
  group_id: string;
  instance_id: string;
  instance_name: string;
  sender_jid: string;
  sender_name: string;
  is_self: boolean;
  statuses: WhatsAppStatusItem[];
}

export interface WhatsAppStatusesListPayload {
  groups: WhatsAppStatusGroup[];
  total: number;
}

export type WhatsAppStatusesListResponse =
  | WhatsAppStatusesListPayload
  | { status?: string; data: WhatsAppStatusesListPayload };

export const statusesService = {
  list: (params?: { instance_id?: string }) =>
    api.get<WhatsAppStatusesListResponse>("/statuses", { params }),
  sendText: (
    instanceId: string,
    data: {
      text: string;
      text_argb?: number;
      background_argb?: number;
      font?: string;
    },
  ) =>
    api.post(`/instances/${instanceId}/status/send`, {
      type: "text",
      ...data,
    }),
  sendMedia: (
    instanceId: string,
    file: File,
    options: {
      type: "image" | "video";
      caption?: string;
    },
  ) => {
    const formData = new FormData();
    formData.append("type", options.type);
    if (options.caption) {
      formData.append("caption", options.caption);
    }
    formData.append("file", file);
    return api.post(`/instances/${instanceId}/status/send`, formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },
  markSeen: (statusId: string) => api.post(`/statuses/${statusId}/mark-seen`),
  reply: (statusId: string, text: string) =>
    api.post(`/statuses/${statusId}/reply`, { text }),
};

export const templatesService = {
  list: (params?: {
    status?: string;
    category?: string;
    account?: string;
    search?: string;
    page?: number;
    limit?: number;
  }) =>
    api.get<{ templates: JsonRecord[]; total?: number }>("/templates", {
      params,
    }),
  get: (id: string) => api.get(`/templates/${id}`),
  uploadMedia: (accountName: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("account", accountName);
    return api.post("/templates/upload-media", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },
};

export const flowsService = {
  list: (params?: {
    account?: string;
    search?: string;
    page?: number;
    limit?: number;
  }) => api.get<{ flows: JsonRecord[]; total?: number }>("/flows", { params }),
  create: (data: JsonRecord) => api.post("/flows", data),
  update: (id: string, data: JsonRecord) => api.put(`/flows/${id}`, data),
  delete: (id: string) => api.delete(`/flows/${id}`),
  saveToMeta: (id: string) => api.post(`/flows/${id}/save-to-meta`),
  publish: (id: string) => api.post(`/flows/${id}/publish`),
  duplicate: (id: string) => api.post(`/flows/${id}/duplicate`),
  sync: (whatsappAccount: string) =>
    api.post("/flows/sync", { whatsapp_account: whatsappAccount }),
};

export const campaignsService = {
  list: (params?: {
    status?: string;
    from?: string;
    to?: string;
    search?: string;
    page?: number;
    limit?: number;
  }) => api.get("/campaigns", { params }),
  create: (data: JsonRecord) => api.post("/campaigns", data),
  update: (id: string, data: JsonRecord) => api.put(`/campaigns/${id}`, data),
  delete: (id: string) => api.delete(`/campaigns/${id}`),
  start: (id: string) => api.post(`/campaigns/${id}/start`),
  pause: (id: string) => api.post(`/campaigns/${id}/pause`),
  cancel: (id: string) => api.post(`/campaigns/${id}/cancel`),
  retryFailed: (id: string) => api.post(`/campaigns/${id}/retry-failed`),
  // Recipients
  getRecipients: (id: string) => api.get(`/campaigns/${id}/recipients`),
  addRecipients: (
    id: string,
    recipients: Array<{
      phone_number: string;
      recipient_name?: string;
      template_params?: Record<string, unknown>;
    }>,
  ) => api.post(`/campaigns/${id}/recipients/import`, { recipients }),
  deleteRecipient: (campaignId: string, recipientId: string) =>
    api.delete(`/campaigns/${campaignId}/recipients/${recipientId}`),
  // Media
  uploadMedia: (campaignId: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    const csrfToken = getCookie("whm_csrf");
    return axios.post(
      `${api.defaults.baseURL}/campaigns/${campaignId}/media`,
      formData,
      {
        withCredentials: true,
        headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
      },
    );
  },
  getMedia: (campaignId: string) =>
    api.get(`/campaigns/${campaignId}/media`, { responseType: "arraybuffer" }),
  // Group targeting (whatsmeow only)
  listInstanceGroups: (instanceId: string, query?: string) =>
    api.get(`/accounts/${instanceId}/groups`, { params: query ? { q: query } : {} }),
  validateGroupJIDs: (campaignId: string, data: { group_jids: string[]; campaign_id: string; instance_id: string }) =>
    api.post(`/campaigns/${campaignId}/groups/validate`, data),
  addGroups: (campaignId: string, groups: Array<{ jid: string; name: string; participant_count: number }>) =>
    api.post(`/campaigns/${campaignId}/groups`, { groups }),
  getGroups: (id: string) =>
    api.get(`/campaigns/${id}/groups`),
  deleteGroup: (campaignId: string, recipientId: string) =>
    api.delete(`/campaigns/${campaignId}/groups/${recipientId}`),
};

// Whatsmeow Instances
export interface PerInstanceRunRow {
  instance_id: string;
  instance_name: string;
  deleted_files: number;
  retention_used: number;
  source: "custom" | "default" | "disabled";
}

export const instancesService = {
  list: () => api.get("/instances"),
  get: (id: string) => api.get(`/instances/${id}`),
  health: (id: string) => api.get(`/instances/${id}/health`),
  getQRCode: (id: string) => api.get(`/instances/${id}/qr`),
  create: (data: {
    name: string;
    is_default?: boolean;
    auto_read_receipt?: boolean;
    settings?: Record<string, unknown>;
  }) => api.post("/instances", data),
  update: (
    id: string,
    data: {
      name?: string;
      is_default?: boolean;
      auto_read_receipt?: boolean;
      settings?: Record<string, unknown>;
    },
  ) => api.put(`/instances/${id}`, data),
  delete: (id: string, options?: { deleteChats?: boolean }) =>
    api.delete(`/instances/${id}`, {
      params: options?.deleteChats ? { delete_chats: true } : undefined,
    }),
  connect: (id: string) => api.post(`/instances/${id}/connect`),
  pairPhone: (
    id: string,
    data: {
      phone_number: string;
      show_push_notification?: boolean;
      client_type?: string;
      client_display_name?: string;
    },
  ) => api.post(`/instances/${id}/pair-phone`, data),
  disconnect: (id: string) => api.post(`/instances/${id}/disconnect`),
  reconnect: (id: string) => api.post(`/instances/${id}/reconnect`),
  uploadAutoCampaignMedia: (id: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    const csrfToken = getCookie("whm_csrf");
    return axios.post(
      `${api.defaults.baseURL}/instances/${id}/auto-campaign/media`,
      formData,
      {
        withCredentials: true,
        headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
      },
    );
  },
  getInstanceUploadsCleanup: (id: string) =>
    api.get(`/instances/${id}/uploads-cleanup`),
  updateInstanceUploadsCleanup: (
    id: string,
    data: { inherit: boolean; retention_days?: number; reason?: string },
  ) => api.put(`/instances/${id}/uploads-cleanup`, data),
  getInstanceUploadsCleanupHistory: (
    id: string,
    params?: { limit?: number; offset?: number },
  ) => api.get(`/instances/${id}/uploads-cleanup/history`, { params }),
  runInstanceUploadsCleanup: (id: string) =>
    api.post(`/instances/${id}/uploads-cleanup/run`),
  getOrgUploadsCleanupOverview: (params?: {
    limit?: number;
    offset?: number;
    q?: string;
    source?: string;
  }) => api.get("/org/uploads-cleanup/instances", { params }),
};

// Facebook Accounts
export const fbAccountsService = {
  list: () => api.get("/facebook/accounts"),
  get: (id: string) => api.get(`/facebook/accounts/${id}`),
  initOAuth: (params?: { action?: "connect" | "renew"; account_id?: string }) =>
    api.get("/facebook/oauth/init", { params }),
  renewOAuth: (id: string) => api.get(`/facebook/accounts/${id}/oauth/renew`),
  refreshPages: (id: string) => api.post(`/facebook/accounts/${id}/pages/refresh`),
  connectPage: (id: string, pageId: string) =>
    api.post(`/facebook/accounts/${id}/pages/${encodeURIComponent(pageId)}/connect`),
  disconnectPage: (id: string, pageId: string) =>
    api.post(`/facebook/accounts/${id}/pages/${encodeURIComponent(pageId)}/disconnect`),
  removePage: (id: string, pageId: string) =>
    api.delete(`/facebook/accounts/${id}/pages/${encodeURIComponent(pageId)}`),
  create: (data: {
    name: string;
    account_uid?: string;
    method?: "cookies" | "credentials" | "oauth";
    cookies_text?: string;
    data?: Record<string, unknown>;
  }) => api.post("/facebook/accounts", data),
  update: (
    id: string,
    data: {
      name?: string;
      account_uid?: string;
      status?: "active" | "inactive" | "closed" | "expired" | "revoked";
      method?: "cookies" | "credentials" | "oauth";
      cookies_text?: string;
      data?: Record<string, unknown>;
    },
  ) => api.put(`/facebook/accounts/${id}`, data),
  delete: (id: string) => api.delete(`/facebook/accounts/${id}`),
};

// Facebook Comments
export const facebookCommentsService = {
  list: (
    params?: {
      page?: number;
      limit?: number;
      status?: string;
      account_id?: string;
      page_id?: string;
      search?: string;
    },
    config?: AxiosRequestConfig,
  ) => api.get("/facebook/comments", { ...config, params }),
  listPages: () => api.get("/facebook/comments/pages"),
  sync: (data?: {
    account_id?: string;
    page_id?: string;
    post_limit?: number;
    comments_per_post?: number;
    post_ids?: string[];
    run_auto_reply?: boolean;
  }) => api.post("/facebook/comments/sync", data || {}, { timeout: 180000 }),
  getSettings: () => api.get("/facebook/comments/settings"),
  updateSettings: (data: Record<string, unknown>) =>
    api.put("/facebook/comments/settings", data),
  getPageSettings: (pageId: string) =>
    api.get(`/facebook/comments/pages/${pageId}/settings`),
  updatePageSettings: (pageId: string, data: Record<string, unknown>) =>
    api.put(`/facebook/comments/pages/${pageId}/settings`, data),
  reply: (
    id: string,
    data: {
      reply_text?: string;
      private_message_text?: string;
      send_comment_reply?: boolean;
      send_private_message?: boolean;
    },
  ) => api.post(`/facebook/comments/${id}/reply`, data),
  updateStatus: (id: string, status: string, config?: AxiosRequestConfig) =>
    api.put(`/facebook/comments/${id}/status`, { status }, config),
};

// Facebook People Search
export const fbPeopleSearchService = {
  search: (params: {
    campaign_id: string;
    page?: number;
    per_page?: number;
    q?: string;
  }) => api.get("/facebook/people-search", { params }),

  addContacts: (data: {
    name: string;
    data: Array<{ identifier: string; name: string }>;
  }) => api.post("/facebook/people-search/add-contacts", data),
};

export const notificationsService = {
  list: (params?: { include_dismissed?: boolean }) =>
    api.get("/notifications", { params }),
  dismiss: (id: string) => api.put(`/notifications/${id}/dismiss`),
};

export const chatbotService = {
  // Settings
  getSettings: () => api.get("/chatbot/settings"),
  updateSettings: (data: JsonRecord) => api.put("/chatbot/settings", data),

  // Keywords
  listKeywords: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get<{ rules: JsonRecord[]; total?: number }>("/chatbot/keywords", {
      params,
    }),
  createKeyword: (data: JsonRecord) => api.post("/chatbot/keywords", data),
  updateKeyword: (id: string, data: JsonRecord) =>
    api.put(`/chatbot/keywords/${id}`, data),
  deleteKeyword: (id: string) => api.delete(`/chatbot/keywords/${id}`),

  // Flows
  listFlows: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get<{ flows: JsonRecord[]; total?: number }>("/chatbot/flows", {
      params,
    }),
  getFlow: (id: string) => api.get(`/chatbot/flows/${id}`),
  createFlow: (data: JsonRecord) => api.post("/chatbot/flows", data),
  updateFlow: (id: string, data: JsonRecord) =>
    api.put(`/chatbot/flows/${id}`, data),
  deleteFlow: (id: string) => api.delete(`/chatbot/flows/${id}`),

  // AI Contexts
  listAIContexts: (params?: {
    search?: string;
    page?: number;
    limit?: number;
  }) =>
    api.get<{ contexts: JsonRecord[]; total?: number }>(
      "/chatbot/ai-contexts",
      {
        params,
      },
    ),
  createAIContext: (data: JsonRecord) => api.post("/chatbot/ai-contexts", data),
  updateAIContext: (id: string, data: JsonRecord) =>
    api.put(`/chatbot/ai-contexts/${id}`, data),
  deleteAIContext: (id: string) => api.delete(`/chatbot/ai-contexts/${id}`),

  // Agent Transfers
  listTransfers: (params?: {
    status?: string;
    agent_id?: string;
    team_id?: string;
    limit?: number;
    offset?: number;
    include?: string; // 'all' | 'contact,agent,team' etc.
  }) => api.get("/chatbot/transfers", { params }),
  createTransfer: (data: {
    contact_id: string;
    whatsapp_account: string;
    agent_id?: string;
    notes?: string;
    source?: string;
  }) => api.post("/chatbot/transfers", data),
  pickNextTransfer: () => api.post("/chatbot/transfers/pick"),
  resumeTransfer: (id: string) => api.put(`/chatbot/transfers/${id}/resume`),
  assignTransfer: (
    id: string,
    agentId: string | null,
    teamId?: string | null,
  ) =>
    api.put(`/chatbot/transfers/${id}/assign`, {
      agent_id: agentId,
      team_id: teamId,
    }),
};

export type AgentSelectionTriggerMode =
  | "first_pending_message"
  | "keyword"
  | "after_office_hours"
  | "chatbot_step"
  | "manual_test";

export type AgentSelectionCustomAction =
  | "send_only"
  | "keep_pending"
  | "close_chat"
  | "assign_to_team";

export type AgentSelectionOptionType = "agent" | "team" | "queue" | "custom";

export interface AgentSelectionSettings {
  id: string;
  organization_id: string;
  instance_id?: string | null;
  allowed_instance_ids?: string[];
  enabled: boolean;
  trigger_mode: AgentSelectionTriggerMode;
  trigger_keywords?: string[];
  prompt_delay_minutes: number;
  prompt_delay_min_minutes: number;
  prompt_delay_max_minutes: number;
  selection_timeout_minutes: number;
  max_invalid_attempts: number;
  menu_header_text: string;
  menu_footer_text: string;
  invalid_reply_text: string;
  timeout_response_text: string;
  unavailable_agent_text: string;
  custom_final_option_enabled: boolean;
  custom_final_option_text: string;
  custom_final_option_response: string;
  custom_final_option_action: AgentSelectionCustomAction;
  custom_final_option_team_id?: string | null;
  hide_unavailable_agents: boolean;
}

export interface AgentSelectionParticipant {
  id: string;
  organization_id: string;
  settings_id: string;
  user_id: string;
  display_name: string;
  description?: string;
  is_enabled: boolean;
  sort_order: number;
  show_only_when_available: boolean;
  max_open_chats?: number | null;
  user?: {
    id: string;
    email: string;
    full_name: string;
    is_active: boolean;
    is_available: boolean;
  };
}

export interface AgentSelectionOption {
  id: string;
  organization_id: string;
  settings_id: string;
  option_type: AgentSelectionOptionType;
  user_id?: string | null;
  team_id?: string | null;
  label: string;
  description?: string;
  is_enabled: boolean;
  sort_order: number;
  action?: string;
}

export interface AgentSelectionRenderedOption {
  number: number;
  option_id: string;
  type: AgentSelectionOptionType;
  label: string;
  description?: string;
  user_id?: string;
  team_id?: string;
  action?: string;
  response?: string;
}

export interface AgentSelectionMenuPreview {
  text: string;
  options: AgentSelectionRenderedOption[];
}

export interface AgentSelectionAuditEvent {
  id: string;
  event_type: string;
  actor_type: string;
  contact_id?: string | null;
  session_id?: string | null;
  selected_agent_id?: string | null;
  selected_team_id?: string | null;
  reason?: string;
  created_at: string;
}

export interface AgentSelectionSession {
  id: string;
  contact_id: string;
  status: string;
  whatsapp_account: string;
  prompt_due_at: string;
  menu_sent_at?: string | null;
  expires_at?: string | null;
  invalid_attempts: number;
  created_at: string;
}

export const agentSelectionService = {
  getSettings: (params?: { instance_id?: string }) =>
    api.get<{ settings: AgentSelectionSettings }>("/agent-selection/settings", {
      params,
    }),
  updateSettings: (data: Partial<AgentSelectionSettings>) =>
    api.put<{ settings: AgentSelectionSettings }>(
      "/agent-selection/settings",
      data,
    ),
  deleteSettings: (id: string) =>
    api.delete<{ deleted: boolean }>(`/agent-selection/settings/${id}`),
  listParticipants: (params?: { settings_id?: string }) =>
    api.get<{ participants: AgentSelectionParticipant[] }>(
      "/agent-selection/participants",
      { params },
    ),
  createParticipant: (
    data: Omit<
      AgentSelectionParticipant,
      "id" | "organization_id" | "user"
    >,
  ) => api.post<{ participant: AgentSelectionParticipant }>(
    "/agent-selection/participants",
    data,
  ),
  updateParticipant: (
    id: string,
    data: Partial<AgentSelectionParticipant>,
  ) =>
    api.put<{ participant: AgentSelectionParticipant }>(
      `/agent-selection/participants/${id}`,
      data,
    ),
  deleteParticipant: (id: string) =>
    api.delete(`/agent-selection/participants/${id}`),
  listOptions: (params?: { settings_id?: string }) =>
    api.get<{ options: AgentSelectionOption[] }>("/agent-selection/options", {
      params,
    }),
  createOption: (
    data: Omit<AgentSelectionOption, "id" | "organization_id">,
  ) => api.post<{ option: AgentSelectionOption }>("/agent-selection/options", data),
  updateOption: (id: string, data: Partial<AgentSelectionOption>) =>
    api.put<{ option: AgentSelectionOption }>(
      `/agent-selection/options/${id}`,
      data,
    ),
  deleteOption: (id: string) => api.delete(`/agent-selection/options/${id}`),
  preview: (data: { settings_id?: string; contact_id?: string }) =>
    api.post<{ menu: AgentSelectionMenuPreview }>(
      "/agent-selection/preview",
      data,
    ),
  testSend: (data: { settings_id?: string; contact_id: string }) =>
    api.post<{
      sent: boolean;
      whatsapp_account: string;
      contact_id: string;
      menu_text: string;
      option_count: number;
      outbound_message_id?: string;
    }>("/agent-selection/test-send", data),
  listAudit: (params?: {
    event_type?: string;
    contact_id?: string;
    session_id?: string;
    page?: number;
    limit?: number;
  }) =>
    api.get<{ events: AgentSelectionAuditEvent[]; total?: number }>(
      "/agent-selection/audit",
      { params },
    ),
  listSessions: (params?: { status?: string; page?: number; limit?: number }) =>
    api.get<{ sessions: AgentSelectionSession[]; total?: number }>(
      "/agent-selection/sessions",
      { params },
    ),
  cancelSession: (id: string) =>
    api.post<{ session: AgentSelectionSession }>(
      `/agent-selection/sessions/${id}/cancel`,
    ),
};

export interface CannedResponse {
  id: string;
  name: string;
  shortcut: string;
  content: string;
  attachments: CannedResponseAttachment[];
  category: string;
  is_active: boolean;
  usage_count: number;
  created_at: string;
  updated_at: string;
}

export interface CannedResponseAttachment {
  id: string;
  type: "image" | "video";
  mime_type: string;
  file_name: string;
  file_path: string;
  file_size: number;
  created_at?: string;
}

export const cannedResponsesService = {
  list: (params?: {
    category?: string;
    search?: string;
    active_only?: string;
    page?: number;
    limit?: number;
  }) =>
    api.get<{ canned_responses: CannedResponse[]; total?: number }>(
      "/canned-responses",
      { params },
    ),
  create: (
    data:
      | FormData
      | {
          name: string;
          shortcut?: string;
          content: string;
          category?: string;
          is_active?: boolean;
          keep_attachment_ids?: string[];
        },
  ) =>
    api.post(
      "/canned-responses",
      data,
      data instanceof FormData
        ? {
            headers: { "Content-Type": "multipart/form-data" },
          }
        : undefined,
    ),
  update: (
    id: string,
    data:
      | FormData
      | {
          name?: string;
          shortcut?: string;
          content?: string;
          category?: string;
          is_active?: boolean;
          keep_attachment_ids?: string[];
        },
  ) =>
    api.put(
      `/canned-responses/${id}`,
      data,
      data instanceof FormData
        ? {
            headers: { "Content-Type": "multipart/form-data" },
          }
        : undefined,
    ),
  send: (
    id: string,
    data: {
      contact_id: string;
      content?: string;
      instance_id?: string;
      reply_to_message_id?: string;
      whatsapp_account?: string;
    },
  ) => api.post(`/canned-responses/${id}/send`, data),
  delete: (id: string) => api.delete(`/canned-responses/${id}`),
  use: (id: string) => api.post(`/canned-responses/${id}/use`),
};

export interface SavedContent {
  id: string;
  name: string;
  body: string;
  variables: string[];
  category: string;
  preview: string;
  media_id?: string;
  media_filename?: string;
  media_mime_type?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export const savedContentsService = {
  list: (params?: {
    category?: string;
    search?: string;
    page?: number;
    limit?: number;
  }) =>
    api.get<{ saved_contents: SavedContent[]; total?: number }>(
      "/saved-contents",
      { params },
    ),
  create: (data: {
    name: string;
    body: string;
    category?: string;
  }) => api.post("/saved-contents", data),
  get: (id: string) =>
    api.get<{ saved_content: SavedContent }>(`/saved-contents/${id}`),
  update: (
    id: string,
    data: { name?: string; body?: string; category?: string },
  ) => api.put(`/saved-contents/${id}`, data),
  delete: (id: string) => api.delete(`/saved-contents/${id}`),
  categories: () =>
    api.get<{ categories: string[] }>("/saved-contents/categories"),
  preview: (id: string) =>
    api.get<{ preview: string; variables: string[] }>(
      `/saved-contents/${id}/preview`,
    ),
  import: (items: { name: string; body: string; category?: string }[]) =>
    api.post("/saved-contents/import", items),
  uploadMedia: (id: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    const csrfToken = getCookie("whm_csrf");
    return axios.post(
      `${api.defaults.baseURL}/saved-contents/${id}/media`,
      formData,
      {
        withCredentials: true,
        headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
      },
    );
  },
  getMedia: (id: string) =>
    api.get(`/saved-contents/${id}/media`, { responseType: "arraybuffer" }),
};

export const agentAnalyticsService = {
  getSummary: (params?: {
    from?: string;
    to?: string;
    agent_id?: string;
    instance_id?: string;
    min_rating?: number;
    max_rating?: number;
  }) => api.get("/analytics/agents", { params }),
  exportRatings: (params?: {
    from?: string;
    to?: string;
    agent_id?: string;
    instance_id?: string;
    min_rating?: number;
    max_rating?: number;
  }) =>
    api.get("/analytics/agents/ratings/export", {
      params,
      responseType: "blob",
    }),
};

// Meta WhatsApp Analytics Types
export type MetaAnalyticsType =
  | "analytics"
  | "conversation_analytics"
  | "pricing_analytics"
  | "template_analytics"
  | "call_analytics";

export type MetaGranularity = "HALF_HOUR" | "DAY" | "MONTH";

export interface MetaAnalyticsAccount {
  id: string;
  name: string;
  phone_id: string;
}

export interface MetaMessagingDataPoint {
  start: number;
  end: number;
  sent: number;
  delivered: number;
}

interface MetaConversationDataPoint {
  start: number;
  end: number;
  conversation: number;
  conversation_type: string;
  conversation_direction: string;
  conversation_category: string;
  cost: number;
}

export interface MetaPricingDataPoint {
  start: number;
  end: number;
  volume: number;
  cost: number;
  country?: string; // Country code (IN, US, etc.)
  pricing_type?: string; // FREE_CUSTOMER_SERVICE, FREE_ENTRY_POINT, REGULAR
  pricing_category?: string; // MARKETING, UTILITY, AUTHENTICATION, SERVICE, etc.
  tier?: string; // Pricing tier
}

interface MetaTemplateCostItem {
  type: string; // amount_spent, cost_per_delivered, cost_per_url_button_click
  value?: number; // The cost value
}

interface MetaTemplateClickItem {
  type: string; // quick_reply_button, unique_url_button
  button_content: string; // The button text
  count: number; // Number of clicks
}

export interface MetaTemplateDataPoint {
  start: number;
  end: number;
  template_id: string;
  sent: number;
  delivered: number;
  read: number;
  replied?: number;
  clicked?: MetaTemplateClickItem[]; // Array of button click details
  cost?: MetaTemplateCostItem[];
}

export interface MetaCallDataPoint {
  start: number;
  end: number;
  total_calls: number;
  call_duration: number;
  call_type: string;
  call_direction: string;
}

interface MetaAnalyticsData {
  id: string;
  analytics?: {
    granularity: string;
    data_points: MetaMessagingDataPoint[];
  };
  conversation_analytics?: {
    granularity: string;
    data_points: MetaConversationDataPoint[];
  };
  pricing_analytics?: {
    granularity: string;
    data_points: MetaPricingDataPoint[];
  };
  template_analytics?: {
    granularity: string;
    data_points: MetaTemplateDataPoint[];
  };
  call_analytics?: {
    granularity: string;
    data_points: MetaCallDataPoint[];
  };
}

export interface MetaAnalyticsResponse {
  account_id: string;
  account_name: string;
  data: MetaAnalyticsData | null;
  template_names?: Record<string, string>; // meta_template_id -> template name
}

export const metaAnalyticsService = {
  get: (params: {
    account_id?: string;
    analytics_type: MetaAnalyticsType;
    start: string;
    end: string;
    granularity?: MetaGranularity;
    template_ids?: string;
  }) =>
    api.get<{ accounts: MetaAnalyticsResponse[]; cached: boolean }>(
      "/analytics/meta",
      { params },
    ),

  getAccounts: () =>
    api.get<{ accounts: MetaAnalyticsAccount[] }>("/analytics/meta/accounts"),

  refresh: () => api.post("/analytics/meta/refresh"),
};

// Dashboard Widgets (customizable analytics)
export interface DashboardWidget {
  id: string;
  name: string;
  description: string;
  data_source: string;
  metric: string;
  field: string;
  filters: Array<{ field: string; operator: string; value: string }>;
  display_type: string;
  chart_type: string;
  group_by_field: string;
  show_change: boolean;
  color: string;
  size: string;
  display_order: number;
  grid_x: number;
  grid_y: number;
  grid_w: number;
  grid_h: number;
  config: Record<string, unknown>;
  is_shared: boolean;
  is_default: boolean;
  is_owner: boolean;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface WidgetData {
  widget_id: string;
  value: number;
  change: number;
  prev_value: number;
  chart_data: Array<{ label: string; value: number }>;
  data_points: Array<{ label: string; value: number; color?: string }>;
  grouped_series?: {
    labels: string[];
    datasets: Array<{ label: string; data: number[] }>;
  };
  table_rows?: Array<{
    id: string;
    contact_id?: string;
    label: string;
    sub_label: string;
    status: string;
    direction?: string;
    created_at: string;
  }>;
}

interface DataSourceInfo {
  name: string;
  label: string;
  fields: string[];
}

export interface LayoutItem {
  id: string;
  grid_x: number;
  grid_y: number;
  grid_w: number;
  grid_h: number;
}

export const widgetsService = {
  list: () => api.get<{ widgets: DashboardWidget[] }>("/widgets"),
  create: (data: {
    name: string;
    description?: string;
    data_source: string;
    metric: string;
    field?: string;
    filters?: Array<{ field: string; operator: string; value: string }>;
    display_type?: string;
    chart_type?: string;
    group_by_field?: string;
    show_change?: boolean;
    color?: string;
    size?: string;
    config?: Record<string, unknown>;
    is_shared?: boolean;
  }) => api.post<DashboardWidget>("/widgets", data),
  update: (
    id: string,
    data: Partial<{
      name: string;
      description: string;
      data_source: string;
      metric: string;
      field: string;
      filters: Array<{ field: string; operator: string; value: string }>;
      display_type: string;
      chart_type: string;
      group_by_field: string;
      show_change: boolean;
      color: string;
      size: string;
      config: Record<string, unknown>;
      is_shared: boolean;
    }>,
  ) => api.put<DashboardWidget>(`/widgets/${id}`, data),
  delete: (id: string) => api.delete(`/widgets/${id}`),
  getAllData: (params?: { from?: string; to?: string }) =>
    api.get<{ data: Record<string, WidgetData> }>("/widgets/data", { params }),
  getDataSources: () =>
    api.get<{
      data_sources: DataSourceInfo[];
      metrics: string[];
      display_types: string[];
      operators: Array<{ value: string; label: string }>;
    }>("/widgets/data-sources"),
  saveLayout: (layout: LayoutItem[]) => api.post("/widgets/layout", { layout }),
};

export const organizationService = {
  getSettings: () => api.get("/org/settings"),
  updateSettings: (data: {
    mask_phone_numbers?: boolean;
    strict_sending_restrictions_enabled?: boolean;
    uploads_cleanup_retention_days?: number;
    uploads_cleanup_schedule_hour?: number;
    outbound_mode?: "inbound_only" | "mixed";
    strict_sending_apply_to_system?: boolean;
    campaign_draft_only?: boolean;
    strict_rollout_mode?: "audit" | "enforce";
    strict_rollout_enforce_at?: string | null;
    timezone?: string;
    date_format?: string;
    name?: string;
    slug?: string;
  }) => api.put("/org/settings", data),
  runUploadsCleanupNow: () =>
    api.post<{
      message: string;
      deleted_files: number;
      retention_days: number;
      instances: PerInstanceRunRow[];
    }>("/org/uploads-cleanup/run"),
};

// Organizations
export interface Organization {
  id: string;
  name: string;
  slug?: string;
  created_at: string;
}

export interface OrganizationMember {
  id: string;
  user_id: string;
  organization_id: string;
  role_id?: string | null;
  email: string;
  full_name: string;
  role_name?: string;
  is_active?: boolean;
  created_at?: string;
}

export const organizationsService = {
  list: () => api.get<{ organizations: Organization[] }>("/organizations"),
  getCurrent: () => api.get<{ organization: Organization }>("/organizations/current"),
  create: (data: { name: string; slug?: string }) => api.post("/organizations", data),
  delete: (id: string) => api.delete(`/organizations/${id}`),
  // Members
  listMembers: () => api.get<{ members: OrganizationMember[] }>("/organizations/members"),
  addMember: (data: { user_id?: string; email?: string; role_id?: string }) =>
    api.post("/organizations/members", data),
  updateMember: (memberId: string, data: { role_id: string }) =>
    api.put(`/organizations/members/${memberId}`, data),
  removeMember: (memberId: string) =>
    api.delete(`/organizations/members/${memberId}`),
};

export interface Webhook {
  id: string;
  name: string;
  url: string;
  events: string[];
  headers: Record<string, string>;
  is_active: boolean;
  has_secret: boolean;
  created_at: string;
  updated_at: string;
}

export interface WebhookEvent {
  value: string;
  label: string;
  description: string;
}

export interface Team {
  id: string;
  name: string;
  description: string;
  assignment_strategy: "round_robin" | "load_balanced" | "manual";
  is_active: boolean;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface TeamMember {
  id: string;
  team_id?: string;
  user_id: string;
  role: "manager" | "agent";
  last_assigned_at: string | null;
  // Flat structure from API
  full_name: string;
  email: string;
  is_available: boolean;
  // Optional nested user for local additions
  user?: {
    id: string;
    full_name: string;
    email: string;
    is_available: boolean;
  };
}

export const teamsService = {
  list: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get<{ teams: Team[] }>("/teams", { params }),
  get: (id: string) => api.get<{ team: Team }>(`/teams/${id}`),
  create: (data: {
    name: string;
    description?: string;
    assignment_strategy?: "round_robin" | "load_balanced" | "manual";
  }) => api.post<{ team: Team }>("/teams", data),
  update: (
    id: string,
    data: {
      name?: string;
      description?: string;
      assignment_strategy?: "round_robin" | "load_balanced" | "manual";
      is_active?: boolean;
    },
  ) => api.put<{ team: Team }>(`/teams/${id}`, data),
  delete: (id: string) => api.delete(`/teams/${id}`),
  // Members
  listMembers: (teamId: string) =>
    api.get<{ members: TeamMember[] }>(`/teams/${teamId}/members`),
  addMember: (
    teamId: string,
    data: { user_id: string; role?: "manager" | "agent" },
  ) => api.post<{ member: TeamMember }>(`/teams/${teamId}/members`, data),
  removeMember: (teamId: string, userId: string) =>
    api.delete(`/teams/${teamId}/members/${userId}`),
};

export const webhooksService = {
  list: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get<{
      webhooks: Webhook[];
      available_events: WebhookEvent[];
      total?: number;
    }>("/webhooks", { params }),
  get: (id: string) => api.get<Webhook>(`/webhooks/${id}`),
  create: (data: {
    name: string;
    url: string;
    events: string[];
    headers?: Record<string, string>;
    secret?: string;
  }) => api.post<Webhook>("/webhooks", data),
  update: (
    id: string,
    data: {
      name?: string;
      url?: string;
      events?: string[];
      headers?: Record<string, string>;
      secret?: string;
      is_active?: boolean;
    },
  ) => api.put<Webhook>(`/webhooks/${id}`, data),
  delete: (id: string) => api.delete(`/webhooks/${id}`),
  test: (id: string) => api.post(`/webhooks/${id}/test`),
};

export interface CustomAction {
  id: string;
  name: string;
  icon: string;
  action_type: "webhook" | "url" | "javascript";
  config: {
    url?: string;
    method?: string;
    headers?: Record<string, string>;
    body?: string;
    open_in_new_tab?: boolean;
    code?: string;
  };
  is_active: boolean;
  display_order: number;
  created_at: string;
  updated_at: string;
}

export interface ActionResult {
  success: boolean;
  message?: string;
  redirect_url?: string;
  clipboard?: string;
  toast?: {
    message: string;
    type: "success" | "error" | "info" | "warning";
  };
  data?: Record<string, unknown>;
}

export const customActionsService = {
  list: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get<{ custom_actions: CustomAction[]; total?: number }>(
      "/custom-actions",
      { params },
    ),
  get: (id: string) => api.get<CustomAction>(`/custom-actions/${id}`),
  create: (data: {
    name: string;
    icon?: string;
    action_type: "webhook" | "url" | "javascript";
    config: Record<string, unknown>;
    is_active?: boolean;
    display_order?: number;
  }) => api.post<CustomAction>("/custom-actions", data),
  update: (
    id: string,
    data: {
      name?: string;
      icon?: string;
      action_type?: "webhook" | "url" | "javascript";
      config?: Record<string, unknown>;
      is_active?: boolean;
      display_order?: number;
    },
  ) => api.put<CustomAction>(`/custom-actions/${id}`, data),
  delete: (id: string) => api.delete(`/custom-actions/${id}`),
  execute: (id: string, contactId: string) =>
    api.post<ActionResult>(`/custom-actions/${id}/execute`, {
      contact_id: contactId,
    }),
};

// Roles and Permissions
export interface Role {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
  is_default: boolean;
  permissions: string[]; // ["resource:action", ...]
  user_count: number;
  created_at: string;
  updated_at: string;
}

export const rolesService = {
  list: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get<{ roles: Role[] }>("/roles", { params }),
  get: (id: string) => api.get<Role>(`/roles/${id}`),
  create: (data: {
    name: string;
    description?: string;
    is_default?: boolean;
    permissions: string[];
  }) => api.post<Role>("/roles", data),
  update: (
    id: string,
    data: {
      name?: string;
      description?: string;
      is_default?: boolean;
      permissions?: string[];
    },
  ) => api.put<Role>(`/roles/${id}`, data),
  delete: (id: string) => api.delete(`/roles/${id}`),
};

export const permissionsService = {
  list: () => api.get<{ permissions: Permission[] }>("/permissions"),
};

// Tags
export interface Tag {
  name: string;
  color: string;
  created_at: string;
  updated_at: string;
}

export const tagsService = {
  list: (params?: { search?: string; page?: number; limit?: number }) =>
    api.get<{ tags: Tag[]; total?: number; page?: number; limit?: number }>(
      "/tags",
      { params },
    ),
  create: (data: { name: string; color?: string }) =>
    api.post<Tag>("/tags", data),
  update: (name: string, data: { name?: string; color?: string }) =>
    api.put<Tag>(`/tags/${encodeURIComponent(name)}`, data),
  delete: (name: string) => api.delete(`/tags/${encodeURIComponent(name)}`),
};

// Conversation Notes
export interface ConversationNote {
  id: string;
  contact_id: string;
  created_by_id: string;
  created_by_name: string;
  content: string;
  created_at: string;
  updated_at: string;
}

export const notesService = {
  list: (contactId: string, params?: { limit?: number; before?: string }) =>
    api.get<{ notes: ConversationNote[]; total: number; has_more: boolean }>(
      `/contacts/${contactId}/notes`,
      { params },
    ),
  create: (contactId: string, data: { content: string }) =>
    api.post<ConversationNote>(`/contacts/${contactId}/notes`, data),
  update: (contactId: string, noteId: string, data: { content: string }) =>
    api.put<ConversationNote>(`/contacts/${contactId}/notes/${noteId}`, data),
  delete: (contactId: string, noteId: string) =>
    api.delete(`/contacts/${contactId}/notes/${noteId}`),
};

// WhatsApp Filter
export interface WhatsAppFilterBatch {
  id: string;
  organization_id: string;
  created_by: string;
  whatsapp_account: string;
  instance_id?: string | null;
  status: "pending" | "processing" | "completed" | "failed";
  total_numbers: number;
  valid_numbers: number;
  invalid_numbers: number;
  error_message?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface WhatsAppFilterResult {
  id: string;
  batch_id: string;
  phone_number: string;
  contact_name?: string;
  is_valid: boolean;
  error_message?: string;
  checked_at?: string;
  created_at: string;
}

export interface WhatsAppFilterResultsPage {
  data: WhatsAppFilterResult[];
  total: number;
  page: number;
  limit: number;
}

export function unwrapWhatsAppFilterResultsPage(payload: any): WhatsAppFilterResultsPage {
  const page = payload?.status === "success" && payload.data ? payload.data : payload;
  if (Array.isArray(page)) {
    return {
      data: page,
      total: page.length,
      page: 1,
      limit: page.length,
    };
  }
  return {
    data: Array.isArray(page?.data) ? page.data : [],
    total: Number(page?.total ?? 0),
    page: Number(page?.page ?? 1),
    limit: Number(page?.limit ?? 0),
  };
}

export const whatsappFilterService = {
  list: (params?: { page?: number; limit?: number }) =>
    api.get<{ data: WhatsAppFilterBatch[]; total: number; page: number; limit: number }>("/whatsapp-filter/batches", { params }),
  get: (id: string) =>
    api.get<WhatsAppFilterBatch>(`/whatsapp-filter/batches/${id}`),
  listResults: (id: string, params?: { page?: number; limit?: number; status?: "all" | "valid" | "invalid"; q?: string }) =>
    api.get<WhatsAppFilterResultsPage>(`/whatsapp-filter/batches/${id}/results`, { params })
      .then((response) => {
        response.data = unwrapWhatsAppFilterResultsPage(response.data);
        return response;
      }),
  createJSON: (data: { connection_id: string; phones: string[]; names?: string[] }) =>
    api.post<WhatsAppFilterBatch>("/whatsapp-filter/batches", data),
  createCSV: (connectionId: string, file: File) => {
    const formData = new FormData();
    formData.append("connection_id", connectionId);
    formData.append("file", file);
    return api.post<WhatsAppFilterBatch>("/whatsapp-filter/batches", formData, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
    });
  },
  exportCSV: async (id: string, params?: { status?: "all" | "valid" | "invalid"; q?: string }) => {
    return api.get(`/whatsapp-filter/batches/${id}/export`, {
      params,
      responseType: "blob",
    });
  },
  delete: (id: string) =>
    api.delete<{ success: boolean }>(`/whatsapp-filter/batches/${id}`),
};

export const groupDirectoryService = {
  search: (params: {
    q?: string;
    country?: string;
    category?: string;
    page?: number;
    limit?: number;
  }) => api.get("/groups/directory", { params }),
  create: (data: {
    group_jid: string;
    name: string;
    description?: string;
    country?: string;
    language?: string;
    category?: string;
    image_url?: string;
    join_link?: string;
    participant_count?: number;
  }) => api.post("/groups/directory", data),
  update: (id: string, data: Partial<{
    name: string;
    description: string;
    country: string;
    language: string;
    category: string;
    image_url: string;
    join_link: string;
    participant_count: number;
  }>) => api.put(`/groups/directory/${id}`, data),
  delete: (id: string) => api.delete(`/groups/directory/${id}`),
  getCategories: () => api.get<string[]>("/groups/directory/categories"),
  getCountries: () => api.get<string[]>("/groups/directory/countries"),
  previewFromLink: (instanceId: string, inviteLink: string) =>
    api.post("/groups/directory/preview", { instance_id: instanceId, invite_link: inviteLink }),
  importToCampaign: (campaignId: string, groupIds: string[]) =>
    api.post("/groups/directory/import", { campaign_id: campaignId, group_ids: groupIds }),
};

export interface GroupParticipant {
  jid: string;
  phone_number: string;
  is_admin: boolean;
  is_super_admin: boolean;
}

export const groupParticipantsService = {
  list: (instanceId: string, groupJid: string) =>
    api.get<{ participants: GroupParticipant[]; total: number }>(
      "/groups/participants",
      { params: { instance_id: instanceId, group_jid: groupJid } },
    ),
  add: (instanceId: string, groupJid: string, participants: string[]) =>
    api.post<{ action: string; participants: GroupParticipant[]; affected: number }>(
      "/groups/participants/add",
      { instance_id: instanceId, group_jid: groupJid, participants },
    ),
  remove: (instanceId: string, groupJid: string, participants: string[]) =>
    api.post<{ action: string; participants: GroupParticipant[]; affected: number }>(
      "/groups/participants/remove",
      { instance_id: instanceId, group_jid: groupJid, participants },
    ),
  promote: (instanceId: string, groupJid: string, participants: string[]) =>
    api.post<{ action: string; participants: GroupParticipant[]; affected: number }>(
      "/groups/participants/promote",
      { instance_id: instanceId, group_jid: groupJid, participants },
    ),
  demote: (instanceId: string, groupJid: string, participants: string[]) =>
    api.post<{ action: string; participants: GroupParticipant[]; affected: number }>(
      "/groups/participants/demote",
      { instance_id: instanceId, group_jid: groupJid, participants },
    ),
};

// Group Join Campaigns
export interface GroupJoinCampaign {
  id: string;
  organization_id: string;
  name: string;
  accounts: string[];
  speed: "slow" | "fast";
  status: "draft" | "processing" | "paused" | "completed" | "failed" | "cancelled";
  total_recipients: number;
  joined_count: number;
  failed_count: number;
  skipped_count: number;
  started_at?: string;
  completed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface GroupJoinRecipient {
  id: string;
  campaign_id: string;
  invite_link: string;
  group_name: string;
  group_jid: string;
  participant_count: number;
  status: "pending" | "joined" | "failed" | "skipped" | "duplicate";
  error_message: string;
  processed_at?: string;
  created_at: string;
  updated_at: string;
}

export const groupJoinCampaignsService = {
  list: (params?: { status?: string; search?: string; page?: number; limit?: number }) =>
    api.get<{ data: GroupJoinCampaign[]; total: number }>("/group-join-campaigns", { params }),
  create: (data: { name: string; accounts: string[]; speed: string }) =>
    api.post<GroupJoinCampaign>("/group-join-campaigns", data),
  get: (id: string) => api.get<GroupJoinCampaign>(`/group-join-campaigns/${id}`),
  update: (id: string, data: { name?: string; accounts?: string[]; speed?: string }) =>
    api.put(`/group-join-campaigns/${id}`, data),
  delete: (id: string) => api.delete(`/group-join-campaigns/${id}`),
  start: (id: string) => api.post(`/group-join-campaigns/${id}/start`),
  pause: (id: string) => api.post(`/group-join-campaigns/${id}/pause`),
  getStats: (id: string) =>
    api.get<{ total_recipients: number; joined_count: number; failed_count: number; skipped_count: number; status: string }>(
      `/group-join-campaigns/${id}/stats`,
    ),
  getRecipients: (id: string, params?: { status?: string; page?: number; limit?: number }) =>
    api.get<{ data: GroupJoinRecipient[]; total: number }>(`/group-join-campaigns/${id}/recipients`, { params }),
  addRecipients: (id: string, data: { invite_links: string[] }) =>
    api.post(`/group-join-campaigns/${id}/recipients`, data),
  uploadRecipientsCSV: (id: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return api.post(`/group-join-campaigns/${id}/recipients`, formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },
  deleteRecipient: (campaignId: string, recipientId: string) =>
    api.delete(`/group-join-campaigns/${campaignId}/recipients/${recipientId}`),
  importDirectory: (id: string, groupIds: string[]) =>
    api.post(`/group-join-campaigns/${id}/import-directory`, { campaign_id: id, group_ids: groupIds }),
};

export interface ExtractionCampaign {
  id: string;
  organization_id: string;
  name: string;
  instance_id: string;
  instance_name: string;
  status: string;
  total_chats?: number;
  total_groups?: number;
  total_members?: number;
  extracted_count: number;
  failed_count: number;
  group_jid?: string;
  group_name?: string;
  started_at?: string;
  completed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface MessageExtractionResult {
  id: string;
  campaign_id: string;
  chat_jid: string;
  phone_number: string;
  profile_name: string;
  push_name: string;
  is_group: boolean;
  group_name: string;
  group_jid: string;
  unread_count: number;
  is_me: boolean;
  last_message_at?: string;
  status: string;
  created_at: string;
}

export interface GroupExtractionResult {
  id: string;
  campaign_id: string;
  group_jid: string;
  group_name: string;
  participant_count: number;
  is_admin: boolean;
  description: string;
  status: string;
  created_at: string;
}

export interface MemberExtractionResult {
  id: string;
  campaign_id: string;
  participant_jid: string;
  phone_number: string;
  push_name: string;
  is_admin: boolean;
  is_super_admin: boolean;
  status: string;
  created_at: string;
}

export const messageExtractionService = {
  list: (params?: { status?: string; search?: string; page?: number; limit?: number }) =>
    api.get<{ data: ExtractionCampaign[]; total: number }>("/message-extraction-campaigns", { params }),
  create: (data: { name: string; instance_id: string }) =>
    api.post<ExtractionCampaign>("/message-extraction-campaigns", data),
  get: (id: string) => api.get<ExtractionCampaign>(`/message-extraction-campaigns/${id}`),
  update: (id: string, data: { name?: string; instance_id?: string }) =>
    api.put(`/message-extraction-campaigns/${id}`, data),
  delete: (id: string) => api.delete(`/message-extraction-campaigns/${id}`),
  start: (id: string) => api.post(`/message-extraction-campaigns/${id}/start`),
  pause: (id: string) => api.post(`/message-extraction-campaigns/${id}/pause`),
  getStats: (id: string) =>
    api.get<{ total_chats: number; extracted_count: number; failed_count: number; status: string }>(
      `/message-extraction-campaigns/${id}/stats`,
    ),
  getResults: (id: string, params?: { status?: string; search?: string; page?: number; limit?: number }) =>
    api.get<{ data: MessageExtractionResult[]; total: number }>(`/message-extraction-campaigns/${id}/results`, { params }),
  exportCSV: (id: string) =>
    api.get(`/message-extraction-campaigns/${id}/export`, { responseType: "blob" }),
};

export const groupExtractionService = {
  list: (params?: { status?: string; search?: string; page?: number; limit?: number }) =>
    api.get<{ data: ExtractionCampaign[]; total: number }>("/group-extraction-campaigns", { params }),
  create: (data: { name: string; instance_id: string }) =>
    api.post<ExtractionCampaign>("/group-extraction-campaigns", data),
  get: (id: string) => api.get<ExtractionCampaign>(`/group-extraction-campaigns/${id}`),
  update: (id: string, data: { name?: string; instance_id?: string }) =>
    api.put(`/group-extraction-campaigns/${id}`, data),
  delete: (id: string) => api.delete(`/group-extraction-campaigns/${id}`),
  start: (id: string) => api.post(`/group-extraction-campaigns/${id}/start`),
  pause: (id: string) => api.post(`/group-extraction-campaigns/${id}/pause`),
  getStats: (id: string) =>
    api.get<{ total_groups: number; extracted_count: number; failed_count: number; status: string }>(
      `/group-extraction-campaigns/${id}/stats`,
    ),
  getResults: (id: string, params?: { status?: string; search?: string; page?: number; limit?: number }) =>
    api.get<{ data: GroupExtractionResult[]; total: number }>(`/group-extraction-campaigns/${id}/results`, { params }),
  exportCSV: (id: string) =>
    api.get(`/group-extraction-campaigns/${id}/export`, { responseType: "blob" }),
};

export const memberExtractionService = {
  list: (params?: { status?: string; search?: string; page?: number; limit?: number }) =>
    api.get<{ data: ExtractionCampaign[]; total: number }>("/member-extraction-campaigns", { params }),
  create: (data: { name: string; instance_id: string; group_jid: string }) =>
    api.post<ExtractionCampaign>("/member-extraction-campaigns", data),
  get: (id: string) => api.get<ExtractionCampaign>(`/member-extraction-campaigns/${id}`),
  update: (id: string, data: { name?: string; instance_id?: string; group_jid?: string }) =>
    api.put(`/member-extraction-campaigns/${id}`, data),
  delete: (id: string) => api.delete(`/member-extraction-campaigns/${id}`),
  start: (id: string) => api.post(`/member-extraction-campaigns/${id}/start`),
  pause: (id: string) => api.post(`/member-extraction-campaigns/${id}/pause`),
  getStats: (id: string) =>
    api.get<{ total_members: number; extracted_count: number; failed_count: number; status: string }>(
      `/member-extraction-campaigns/${id}/stats`,
    ),
  getResults: (id: string, params?: { status?: string; search?: string; page?: number; limit?: number }) =>
    api.get<{ data: MemberExtractionResult[]; total: number }>(`/member-extraction-campaigns/${id}/results`, { params }),
  exportCSV: (id: string) =>
    api.get(`/member-extraction-campaigns/${id}/export`, { responseType: "blob" }),
};

export default api;
