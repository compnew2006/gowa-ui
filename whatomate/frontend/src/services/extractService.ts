import { api } from "./api";

export interface ExtractContact {
  id: string;
  phone_number: string;
  profile_name: string;
  last_message_at: string | null;
  message_count: number;
  unread_count: number;
  instance_id: string | null;
}

export interface ExtractStats {
  instance_id: string;
  instance_name: string;
  phone_number: string;
  total_contacts: number;
  total_messages: number;
  last_sync_at: string | null;
  status: string;
}

export interface ExtractContactsResponse {
  data: ExtractContact[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface ExtractStatsResponse {
  stats: ExtractStats[];
}

export const extractService = {
  listContacts: (params?: {
    instance_id?: string;
    search?: string;
    page?: number;
    limit?: number;
  }) => api.get<ExtractContactsResponse>("/extract/contacts", { params }),

  exportContacts: (data: { instance_id?: string; search?: string }) =>
    api.post("/extract/contacts/export", data, {
      responseType: "blob",
    }),

  getStats: (params?: { instance_id?: string }) =>
    api.get<ExtractStatsResponse>("/extract/stats", { params }),

  triggerSync: (data: { instance_id: string }) =>
    api.post("/extract/sync", data),
};

/**
 * Unwraps the {@code {"status":"success","data":...}} envelope from an Axios response
 * so callers receive the inner payload directly.
 *
 * The {@code T} passed to {@code api.get<T>()} is the *envelope's* `.data` type,
 * but Axios stores the full HTTP body in {@code response.data}.  This helper
 * performs {@code response.data.data ?? response.data}.
 */
export function unwrapEnvelope<T>(response: { data: unknown }): T {
  const body = response.data as Record<string, unknown>;
  return ((body && typeof body === "object" && "data" in body ? body.data : body) ?? body) as T;
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(url);
}
