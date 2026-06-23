import { api } from "@/services/api";
import { unwrapEnvelope } from "@/services/extractService";

export interface AuditEvent {
  id: string;
  created_at: string;
  organization_id?: string;
  category: string;
  action: string;
  source: string;
  actor_user_id?: string;
  actor_email?: string;
  actor_role?: string;
  target_type?: string;
  target_id?: string;
  success: boolean;
  reason?: string;
  details?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
}

export interface AuditEventFilters {
  category?: string;
  action?: string;
  source?: string;
  actor_user_id?: string;
  target_id?: string;
  target_type?: string;
  success?: boolean;
  q?: string;
  date_from?: string;
  date_to?: string;
  organization_id?: string;
  page?: number;
  per_page?: number;
}

export interface AuditEventListResponse {
  events: AuditEvent[];
  total: number;
  page: number;
  per_page: number;
}

/**
 * auditService wraps the shared api client for the admin-only audit log API.
 * Every call goes through api.ts so auth refresh/interceptors stay centralized
 * (per AGENTS.md frontend conventions — no direct fetch/axios).
 */
export const auditService = {
  async list(filters: AuditEventFilters = {}): Promise<AuditEventListResponse> {
    // Drop undefined/empty params so they don't pollute the query string.
    const params: Record<string, string | number | boolean> = {};
    for (const [k, v] of Object.entries(filters)) {
      if (v === undefined || v === null || v === "") continue;
      params[k] = v;
    }
    const res = await api.get("/audit-events", { params });
    return unwrapEnvelope<AuditEventListResponse>(res);
  },
};
