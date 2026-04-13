import { computed, ref } from "vue";
import { defineStore } from "pinia";
import { api } from "@/services/api";
import { i18n } from "@/i18n";
import { unwrapResponse } from "@/lib/api-utils";

export interface LicenseMetricUsage {
  current: number;
  limit: number;
  over_quota: boolean;
  overage: number;
}

export interface LicenseOrganizationUsage {
  organization_id: string;
  organization_name: string;
  user_count: number;
  whatsapp_endpoint_count: number;
}

export interface LicenseUsageSnapshot {
  organizations: LicenseMetricUsage;
  users_per_org: LicenseMetricUsage;
  whatsapp_endpoints_per_org: LicenseMetricUsage;
  organization_details: LicenseOrganizationUsage[];
}

export interface LicenseBootstrapResponse {
  enabled: boolean;
  status: string;
  locked: boolean;
  reason?: string;
  license_id?: string;
  license_family_id?: string;
  revision: number;
  key_id?: string;
  hwid_full: string;
  hwid_short: string;
  hwid_hash: string;
  tier?: string;
  license_kind?: string;
  trial_days?: number;
  duration_label?: string;
  max_organizations: number;
  max_users_per_org: number;
  max_whatsapp_endpoints_per_org: number;
  max_workers: number;
  expires_at?: string | null;
  grace_deadline?: string | null;
  days_until_expiry?: number | null;
  expiring_soon: boolean;
  quota_overages: Record<string, number>;
  updated_at?: string;
  usage: LicenseUsageSnapshot;
}

const emptyUsageMetric = (): LicenseMetricUsage => ({
  current: 0,
  limit: 0,
  over_quota: false,
  overage: 0,
});

const emptyLicenseState = (): LicenseBootstrapResponse => ({
  enabled: true,
  status: "unlicensed",
  locked: true,
  reason: "license_required",
  revision: 0,
  hwid_full: "",
  hwid_short: "",
  hwid_hash: "",
  max_organizations: 0,
  max_users_per_org: 0,
  max_whatsapp_endpoints_per_org: 0,
  max_workers: 0,
  duration_label: "",
  expires_at: null,
  grace_deadline: null,
  days_until_expiry: null,
  expiring_soon: false,
  quota_overages: {},
  usage: {
    organizations: emptyUsageMetric(),
    users_per_org: emptyUsageMetric(),
    whatsapp_endpoints_per_org: emptyUsageMetric(),
    organization_details: [],
  },
});

function normalizeResponse(
  payload: Partial<LicenseBootstrapResponse> | null | undefined,
): LicenseBootstrapResponse {
  const fallback = emptyLicenseState();
  const usage = payload?.usage ?? fallback.usage;

  return {
    ...fallback,
    ...payload,
    expires_at: payload?.expires_at ?? null,
    grace_deadline: payload?.grace_deadline ?? null,
    days_until_expiry: payload?.days_until_expiry ?? null,
    quota_overages: payload?.quota_overages ?? {},
    usage: {
      organizations: usage.organizations ?? emptyUsageMetric(),
      users_per_org: usage.users_per_org ?? emptyUsageMetric(),
      whatsapp_endpoints_per_org:
        usage.whatsapp_endpoints_per_org ?? emptyUsageMetric(),
      organization_details: usage.organization_details ?? [],
    },
  };
}

export const useLicenseStore = defineStore("license", () => {
  const state = ref<LicenseBootstrapResponse>(emptyLicenseState());
  const loading = ref(false);
  const activationPending = ref(false);
  const loaded = ref(false);
  const error = ref<string | null>(null);
  let inflight: Promise<LicenseBootstrapResponse> | null = null;

  const status = computed(() => state.value.status);
  const isLocked = computed(() => Boolean(state.value.locked));
  const isDisabled = computed(() => status.value === "disabled");
  const isActive = computed(
    () => status.value === "active" || status.value === "grace",
  );
  const isGrace = computed(() => status.value === "grace");
  const isTrial = computed(() => state.value.license_kind === "trial");
  const daysUntilExpiry = computed<number | null>(
    () => state.value.days_until_expiry ?? null,
  );
  const showExpiryWarning = computed(
    () =>
      !isLocked.value &&
      daysUntilExpiry.value !== null &&
      daysUntilExpiry.value <= 14,
  );
  const showQuotaOverage = computed(
    () => Object.keys(state.value.quota_overages ?? {}).length > 0,
  );

  function setState(
    payload: Partial<LicenseBootstrapResponse> | null | undefined,
  ) {
    state.value = normalizeResponse(payload);
    loaded.value = true;
  }

  async function fetchBootstrap(
    force = false,
  ): Promise<LicenseBootstrapResponse> {
    if (inflight && !force) {
      return inflight;
    }
    if (loaded.value && !force) {
      return state.value;
    }

    loading.value = true;
    error.value = null;

    inflight = api
      .get("/license/bootstrap")
      .then((response) => {
        const payload = unwrapResponse<LicenseBootstrapResponse>(response);
        setState(payload);
        return state.value;
      })
      .catch((err: any) => {
        error.value =
          err?.response?.data?.message ??
          i18n.global.t("licenseSettings.toast.failedToLoadStatus");
        throw err;
      })
      .finally(() => {
        inflight = null;
        loading.value = false;
      });

    return inflight;
  }

  async function activate(token: string): Promise<LicenseBootstrapResponse> {
    activationPending.value = true;
    error.value = null;
    try {
      const response = await api.post("/license/activate", { token });
      const payload = unwrapResponse<LicenseBootstrapResponse>(response);
      setState(payload);
      return state.value;
    } catch (err: any) {
      error.value =
        err?.response?.data?.message ??
        i18n.global.t("licenseSettings.toast.activationFailed");
      throw err;
    } finally {
      activationPending.value = false;
    }
  }

  function markLocked(reason = "license_locked") {
    state.value = normalizeResponse({
      ...state.value,
      enabled: true,
      locked: true,
      status: "locked",
      reason,
    });
    loaded.value = true;
  }

  return {
    state,
    loading,
    activationPending,
    loaded,
    error,
    status,
    isLocked,
    isDisabled,
    isActive,
    isGrace,
    isTrial,
    daysUntilExpiry,
    showExpiryWarning,
    showQuotaOverage,
    fetchBootstrap,
    activate,
    markLocked,
    setState,
  };
});
