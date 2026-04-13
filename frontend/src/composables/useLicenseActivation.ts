import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue-sonner";
import { useLicenseStore } from "@/stores/license";

interface SubmitActivationOptions {
  onSuccess?: () => Promise<void> | void;
}

export function useLicenseActivation() {
  const { t } = useI18n();
  const licenseStore = useLicenseStore();
  const securityKey = ref("");
  const bootstrapPending = ref(false);

  function formatLicenseKind(licenseKind?: string): string {
    switch (licenseKind) {
      case "trial":
        return t("licenseSettings.licenseKind.trial");
      case "paid":
        return t("licenseSettings.licenseKind.paid");
      default:
        return licenseKind ?? "";
    }
  }

  function formatDurationLabel(durationLabel?: string): string {
    if (!durationLabel) {
      return "";
    }

    if (durationLabel === "lifetime") {
      return t("licenseSettings.duration.lifetime");
    }

    const dayMatch = durationLabel.match(/^(\d+)d$/i);
    if (dayMatch) {
      const count = Number(dayMatch[1]);
      if (count === 1) {
        return t("licenseSettings.duration.oneDay");
      }
      return t("licenseSettings.duration.days", { count });
    }

    return durationLabel;
  }

  const statusVariant = computed(() => {
    switch (licenseStore.status) {
      case "active":
        return "success";
      case "grace":
        return "warning";
      case "disabled":
        return "secondary";
      case "locked":
      case "unlicensed":
        return "destructive";
      default:
        return "secondary";
    }
  });

  const statusLabel = computed(() => {
    switch (licenseStore.status) {
      case "active":
        return t("licenseSettings.status.active");
      case "grace":
        return t("licenseSettings.status.grace");
      case "disabled":
        return t("licenseSettings.status.disabled");
      case "locked":
        return t("licenseSettings.status.locked");
      case "unlicensed":
        return t("licenseSettings.status.activationRequired");
      default:
        return t("licenseSettings.status.unknown");
    }
  });

  const entitlementLabel = computed(
    () => licenseStore.state.tier || t("licenseSettings.status.licensed"),
  );

  const licenseMetaLabel = computed(() => {
    const licenseKind = formatLicenseKind(licenseStore.state.license_kind);
    const durationLabel = formatDurationLabel(
      licenseStore.state.duration_label,
    );

    if (!licenseKind) {
      return "";
    }
    if (durationLabel) {
      return `${licenseKind} • ${durationLabel}`;
    }
    return licenseKind;
  });

  const quotaCards = computed(() => [
    {
      key: "organizations",
      label: t("licenseSettings.quota.organizations"),
      usage: licenseStore.state.usage.organizations,
    },
    {
      key: "users",
      label: t("licenseSettings.quota.usersPerOrg"),
      usage: licenseStore.state.usage.users_per_org,
    },
    {
      key: "whatsapp",
      label: t("licenseSettings.quota.whatsAppEndpointsPerOrg"),
      usage: licenseStore.state.usage.whatsapp_endpoints_per_org,
    },
  ]);

  function usagePercentage(current: number, limit: number): number {
    if (!limit || limit <= 0) {
      return 0;
    }
    return Math.min(100, Math.round((current / limit) * 100));
  }

  async function loadBootstrap(force = false) {
    bootstrapPending.value = true;
    try {
      await licenseStore.fetchBootstrap(force);
    } catch (error: any) {
      toast.error(
        error?.response?.data?.message ||
          t("licenseSettings.toast.failedToLoadStatus"),
      );
    } finally {
      bootstrapPending.value = false;
    }
  }

  async function copyHWID() {
    if (!licenseStore.state.hwid_full) {
      toast.error(t("licenseSettings.toast.hwidUnavailable"));
      return;
    }
    try {
      await navigator.clipboard.writeText(licenseStore.state.hwid_full);
      toast.success(t("licenseSettings.toast.hwidCopied"));
    } catch {
      toast.error(t("licenseSettings.toast.failedToCopyHwid"));
    }
  }

  async function submitActivation(options: SubmitActivationOptions = {}) {
    if (!securityKey.value.trim()) {
      toast.error(t("licenseSettings.toast.pasteSecurityKey"));
      return false;
    }

    try {
      await licenseStore.activate(securityKey.value);
      toast.success(t("licenseSettings.toast.activationSuccess"));
      securityKey.value = "";
      await options.onSuccess?.();
      return true;
    } catch (error: any) {
      toast.error(
        error?.response?.data?.message ||
          t("licenseSettings.toast.activationFailed"),
      );
      return false;
    }
  }

  return {
    licenseStore,
    securityKey,
    bootstrapPending,
    statusVariant,
    statusLabel,
    entitlementLabel,
    licenseMetaLabel,
    quotaCards,
    usagePercentage,
    loadBootstrap,
    copyHWID,
    submitActivation,
  };
}
