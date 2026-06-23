import { defineStore } from "pinia";
import { ref, reactive } from "vue";
import {
  auditService,
  type AuditEvent,
  type AuditEventFilters,
} from "@/services/audit";

/**
 * Pinia store for the admin-only audit log. Mirrors the established store
 * pattern (composition API, loading/error state, fetch/filter/pagination actions).
 */
export const useAuditStore = defineStore("audit", () => {
  const events = ref<AuditEvent[]>([]);
  const total = ref(0);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // Default filter state; page resets to 1 whenever a filter changes.
  const filters = reactive<AuditEventFilters>({
    page: 1,
    per_page: 50,
  });

  async function fetch(): Promise<void> {
    loading.value = true;
    error.value = null;
    try {
      const res = await auditService.list(filters);
      events.value = res.events ?? [];
      total.value = res.total ?? 0;
    } catch (err: unknown) {
      const e = err as { response?: { data?: { message?: string } } };
      error.value = e?.response?.data?.message ?? "Failed to load audit events";
    } finally {
      loading.value = false;
    }
  }

  /** Set a single filter key and reset to the first page. */
  function setFilter<K extends keyof AuditEventFilters>(
    key: K,
    value: AuditEventFilters[K],
  ): void {
    filters[key] = value;
    filters.page = 1;
  }

  /** Clear all filters back to defaults. */
  function resetFilters(): void {
    for (const k of Object.keys(filters) as (keyof AuditEventFilters)[]) {
      delete filters[k];
    }
    filters.page = 1;
    filters.per_page = 50;
  }

  /** Navigate to page n and reload. */
  async function goToPage(n: number): Promise<void> {
    filters.page = n;
    await fetch();
  }

  const totalPages = ref(1);
  function recalcTotalPages(): void {
    totalPages.value =
      total.value <= 0 ? 1 : Math.ceil(total.value / (filters.per_page ?? 50));
  }

  return {
    events,
    total,
    loading,
    error,
    filters,
    totalPages,
    fetch,
    setFilter,
    resetFilters,
    goToPage,
    recalcTotalPages,
  };
});
