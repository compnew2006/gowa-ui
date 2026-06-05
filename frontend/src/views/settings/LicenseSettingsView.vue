<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  KeyRound,
  Loader2,
  LockKeyhole,
  ShieldCheck,
  Clock,
  Users,
  HardDrive,
  MessageSquare,
  Building2,
} from "lucide-vue-next";
import { PageHeader } from "@/components/shared";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useLicenseActivation } from "@/composables/useLicenseActivation";

const { t } = useI18n();
const {
  licenseStore,
  securityKey,
  bootstrapPending,
  statusVariant,
  statusLabel,
  entitlementLabel,
  licenseMetaLabel,
  quotaCards,
  formatBytes,
  formatQuotaValue,
  usagePercentage,
  loadBootstrap,
  copyHWID,
  submitActivation,
} = useLicenseActivation();

const quotaIcons: Record<string, typeof Building2> = {
  organizations: Building2,
  users: Users,
  whatsapp: MessageSquare,
  storage: HardDrive,
};

function parseDurationDays(durationLabel?: string | null): number | null {
  if (!durationLabel || durationLabel === "lifetime") {
    return null;
  }

  const dayMatch = durationLabel.match(/^(\d+)d$/i);
  if (!dayMatch) {
    return null;
  }

  const parsed = Number(dayMatch[1]);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

function getExpiryMessage(daysUntilExpiry: number | null) {
  if (daysUntilExpiry === 0) {
    return t("licenseSettings.summary.expiresToday");
  }
  if (daysUntilExpiry === 1) {
    return t("licenseSettings.summary.expiresInOneDay");
  }
  if (daysUntilExpiry !== null && daysUntilExpiry > 1) {
    return t("licenseSettings.summary.expiresInDays", {
      count: daysUntilExpiry,
    });
  }
  return t("licenseSettings.summary.healthy");
}

function getQuotaUsageLabel(
  key: string,
  limit: number,
  overQuota: boolean,
  overage: number,
) {
  if (licenseStore.isDisabled || limit <= 0) {
    return t("licenseSettings.quota.notEnforced");
  }
  if (overQuota) {
    return t("licenseSettings.quota.overBy", {
      count: formatQuotaValue(key, overage),
    });
  }
  return t("licenseSettings.quota.withinCapacity");
}

function getQuotaLimitLabel(key: string, limit: number) {
  if (licenseStore.isDisabled || limit <= 0) {
    return t("licenseSettings.quota.notEnforcedShort");
  }
  return formatQuotaValue(key, limit);
}

function getOrganizationUsageLabel(
  userCount: number,
  endpointCount: number,
  storageBytes: number,
) {
  return t("licenseSettings.organizationUsage.summary", {
    users: userCount,
    endpoints: endpointCount,
    storage: formatBytes(storageBytes),
  });
}

const graceDeadlineLabel = computed(
  () =>
    licenseStore.state.grace_deadline ||
    t("licenseSettings.placeholders.deadline"),
);

const shortIdLabel = computed(
  () =>
    licenseStore.state.hwid_short || t("licenseSettings.placeholders.pending"),
);

const statusSummary = computed(() => {
  if (licenseStore.isDisabled) {
    return t("licenseSettings.summary.disabled");
  }
  if (licenseStore.showQuotaOverage) {
    return t("licenseSettings.summary.paused");
  }
  if (licenseStore.isGrace) {
    return t("licenseSettings.summary.grace");
  }
  if (licenseStore.showExpiryWarning) {
    return getExpiryMessage(licenseStore.daysUntilExpiry);
  }
  return t("licenseSettings.summary.healthy");
});

const totalDurationDays = computed(() =>
  parseDurationDays(licenseStore.state.duration_label),
);

const remainingDurationDays = computed(() => {
  const total = totalDurationDays.value;
  const remaining = Math.max(licenseStore.daysUntilExpiry ?? 0, 0);
  if (total === null) {
    return remaining;
  }
  return Math.min(remaining, total);
});

const durationHeadline = computed(() => {
  if (licenseStore.isDisabled) {
    return t("licenseSettings.duration.disabled");
  }

  if (licenseStore.state.duration_label === "lifetime") {
    return t("licenseSettings.duration.lifetime");
  }

  if (totalDurationDays.value !== null) {
    return `${remainingDurationDays.value}/${totalDurationDays.value}`;
  }

  return t("licenseSettings.duration.unavailable");
});

const durationProgressValue = computed(() => {
  if (licenseStore.isDisabled) {
    return 0;
  }

  if (licenseStore.state.duration_label === "lifetime") {
    return 100;
  }

  if (totalDurationDays.value === null) {
    return 0;
  }

  return usagePercentage(remainingDurationDays.value, totalDurationDays.value);
});

const durationUsageLabel = computed(() => {
  if (licenseStore.isDisabled) {
    return t("licenseSettings.duration.disabledDescription");
  }

  if (licenseStore.state.duration_label === "lifetime") {
    return t("licenseSettings.duration.noExpiry");
  }

  if (totalDurationDays.value === null) {
    return t("licenseSettings.duration.unavailable");
  }

  if (remainingDurationDays.value <= 0) {
    return t("licenseSettings.banner.remainingExpiresToday");
  }

  if (remainingDurationDays.value === 1) {
    return t("licenseSettings.banner.remainingOneDay");
  }

  return t("licenseSettings.banner.remainingDays", {
    count: remainingDurationDays.value,
  });
});

const isInitialLoad = computed(
  () => bootstrapPending.value && !licenseStore.state.enabled,
);

onMounted(() => {
  void loadBootstrap();
});

async function refreshStatus() {
  await loadBootstrap(true);
}

async function handleActivationSubmit() {
  await submitActivation({
    onSuccess: async () => {
      await loadBootstrap(true);
    },
  });
}
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="t('licenseSettings.title')"
      :subtitle="t('licenseSettings.subtitle')"
      :icon="ShieldCheck"
      icon-gradient="bg-primary text-primary-foreground shadow-none"
    >
      <template #actions>
        <Button
          variant="outline"
          size="sm"
          :disabled="bootstrapPending"
          @click="refreshStatus"
        >
          <Loader2 v-if="bootstrapPending" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
          {{ t("licenseSettings.actions.refresh") }}
        </Button>
      </template>
    </PageHeader>

    <ScrollArea class="flex-1">
      <div class="mx-auto max-w-5xl space-y-6 p-6">
        <template v-if="isInitialLoad">
          <Card class="border-border/70 bg-card/95">
            <CardHeader class="space-y-4">
              <div class="flex flex-wrap items-center gap-3">
                <Skeleton class="h-6 w-28 rounded-full" />
                <Skeleton class="h-6 w-24 rounded-full" />
              </div>
              <Skeleton class="h-4 w-96 max-w-full" />
            </CardHeader>
            <CardContent class="space-y-5">
              <div class="space-y-3">
                <Skeleton v-for="i in 4" :key="i" class="h-14 w-full rounded-lg" />
              </div>
            </CardContent>
          </Card>
          <div class="grid gap-6 md:grid-cols-2">
            <Skeleton class="h-48 w-full rounded-xl" />
            <Skeleton class="h-48 w-full rounded-xl" />
          </div>
        </template>

        <template v-else>
          <Card class="border-border/70 bg-card/95">
            <CardHeader class="space-y-4">
              <div
                class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between"
              >
                <div class="space-y-1.5">
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t("licenseSettings.sections.deploymentStatus") }}
                  </p>
                  <CardTitle class="text-xl">
                    {{ t("licenseSettings.sections.overview") }}
                  </CardTitle>
                </div>
                <div class="flex flex-wrap gap-1.5">
                  <Badge :variant="statusVariant">{{ statusLabel }}</Badge>
                  <Badge v-if="licenseStore.state.tier" variant="outline">
                    {{ entitlementLabel }}
                  </Badge>
                  <Badge v-if="licenseMetaLabel" variant="outline">
                    {{ licenseMetaLabel }}
                  </Badge>
                  <Badge
                    v-if="licenseStore.showQuotaOverage"
                    variant="warning"
                  >
                    {{ t("licenseSettings.status.quotaOverage") }}
                  </Badge>
                </div>
              </div>
              <p class="max-w-3xl text-sm leading-6 text-muted-foreground">
                {{ statusSummary }}
              </p>
            </CardHeader>
            <CardContent class="space-y-5">
              <Alert
                v-if="licenseStore.isGrace"
                variant="warning"
                class="border-amber-500/35"
              >
                <AlertTriangle class="h-4 w-4" />
                <AlertTitle>
                  {{ t("licenseSettings.alerts.graceTitle") }}
                </AlertTitle>
                <AlertDescription>
                  {{
                    t("licenseSettings.alerts.graceDescription", {
                      deadline: graceDeadlineLabel,
                    })
                  }}
                </AlertDescription>
              </Alert>

              <Alert
                v-else-if="licenseStore.showQuotaOverage"
                variant="warning"
                class="border-amber-500/35"
              >
                <AlertTriangle class="h-4 w-4" />
                <AlertTitle>
                  {{ t("licenseSettings.alerts.quotaTitle") }}
                </AlertTitle>
                <AlertDescription>
                  {{ t("licenseSettings.alerts.quotaDescription") }}
                </AlertDescription>
              </Alert>

              <Alert
                v-else-if="licenseStore.showExpiryWarning"
                variant="info"
                class="border-primary/25"
              >
                <ShieldCheck class="h-4 w-4" />
                <AlertTitle>
                  {{ t("licenseSettings.alerts.expiryTitle") }}
                </AlertTitle>
                <AlertDescription>
                  {{ getExpiryMessage(licenseStore.daysUntilExpiry) }}
                </AlertDescription>
              </Alert>

              <div class="space-y-0 divide-y divide-border/50 rounded-lg border border-border/50 bg-background/60">
                <div
                  v-for="item in quotaCards"
                  :key="item.key"
                  class="flex flex-col gap-2 px-4 py-3.5 transition-colors hover:bg-accent/50 first:rounded-t-lg last:rounded-b-lg sm:flex-row sm:items-center sm:justify-between sm:gap-4"
                >
                  <div class="flex items-center gap-2.5">
                    <component
                      :is="quotaIcons[item.key] ?? Building2"
                      class="h-4 w-4 shrink-0 text-muted-foreground/70"
                    />
                    <span class="text-sm font-medium text-foreground">
                      {{ item.label }}
                    </span>
                  </div>
                  <div class="flex flex-1 flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-4">
                    <div class="flex items-baseline gap-1.5 text-sm tabular-nums">
                      <span class="font-semibold text-foreground">
                        {{ formatQuotaValue(item.key, item.usage.current) }}
                      </span>
                      <span class="text-muted-foreground">/</span>
                      <span class="text-muted-foreground">
                        {{ getQuotaLimitLabel(item.key, item.usage.limit) }}
                      </span>
                    </div>
                    <div class="flex items-center gap-3 sm:w-40">
                      <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-muted/80">
                        <div
                          class="h-full rounded-full transition-all duration-300"
                          :class="
                            item.usage.over_quota
                              ? 'bg-amber-500 dark:bg-amber-400'
                              : item.usage.limit > 0 &&
                                  usagePercentage(item.usage.current, item.usage.limit) >= 80
                                ? 'bg-primary/70'
                                : 'bg-primary/40'
                          "
                          :style="{
                            width:
                              item.usage.limit > 0
                                ? `${Math.min(100, usagePercentage(item.usage.current, item.usage.limit))}%`
                                : '0%',
                          }"
                        />
                      </div>
                      <span
                        class="shrink-0 text-[11px] tabular-nums"
                        :class="
                          item.usage.over_quota
                            ? 'font-medium text-amber-600 dark:text-amber-400'
                            : 'text-muted-foreground'
                        "
                      >
                        {{
                          getQuotaUsageLabel(
                            item.key,
                            item.usage.limit,
                            item.usage.over_quota,
                            item.usage.overage,
                          )
                        }}
                      </span>
                    </div>
                  </div>
                </div>

                <div class="flex flex-col gap-2 px-4 py-3.5 transition-colors hover:bg-accent/50 last:rounded-b-lg sm:flex-row sm:items-center sm:justify-between sm:gap-4">
                  <div class="flex items-center gap-2.5">
                    <Clock class="h-4 w-4 shrink-0 text-muted-foreground/70" />
                    <span class="text-sm font-medium text-foreground">
                      {{ t("licenseSettings.quota.subscriptionDays") }}
                    </span>
                  </div>
                  <div class="flex flex-1 flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-4">
                    <div class="flex items-baseline gap-1.5 text-sm tabular-nums">
                      <span class="font-semibold text-foreground">
                        {{ durationHeadline }}
                      </span>
                    </div>
                    <div class="flex items-center gap-3 sm:w-40">
                      <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-muted/80">
                        <div
                          class="h-full rounded-full transition-all duration-300"
                          :class="
                            durationProgressValue <= 20 && durationProgressValue > 0
                              ? 'bg-amber-500 dark:bg-amber-400'
                              : 'bg-primary/40'
                          "
                          :style="{
                            width: `${durationProgressValue}%`,
                          }"
                        />
                      </div>
                      <span class="shrink-0 text-[11px] text-muted-foreground">
                        {{ durationUsageLabel }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              <div
                v-if="
                  licenseStore.state.usage.organization_details.length > 0
                "
                class="space-y-2"
              >
                <h3 class="flex items-center gap-2 text-sm font-medium text-foreground">
                  <CheckCircle2 class="h-3.5 w-3.5 text-emerald-500" />
                  {{ t("licenseSettings.sections.currentOrgUsage") }}
                </h3>
                <p class="text-xs text-muted-foreground">
                  {{ t("licenseSettings.descriptions.orgUsage") }}
                </p>
                <div class="space-y-1.5">
                  <div
                    v-for="org in licenseStore.state.usage
                      .organization_details"
                    :key="org.organization_id"
                    class="flex flex-col gap-1 rounded-lg border border-border/40 bg-card/60 px-3 py-2.5 text-sm transition-colors hover:bg-accent/30 md:flex-row md:items-center md:justify-between"
                  >
                    <span class="font-medium text-foreground">
                      {{ org.organization_name }}
                    </span>
                    <span class="text-muted-foreground">
                      {{
                        getOrganizationUsageLabel(
                          org.user_count,
                          org.whatsapp_endpoint_count,
                          org.storage_bytes,
                        )
                      }}
                    </span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <div class="grid gap-6 md:grid-cols-2">
            <Card class="border-border/50 bg-muted/30">
              <CardHeader>
                <CardTitle class="text-base">
                  {{ t("licenseSettings.sections.serverIdentity") }}
                </CardTitle>
                <CardDescription>
                  {{ t("licenseSettings.descriptions.serverIdentity") }}
                </CardDescription>
              </CardHeader>
              <CardContent class="space-y-3">
                <Label for="license-hwid" class="text-xs">
                  {{ t("licenseSettings.fields.hwid") }}
                </Label>
                <div class="flex flex-col gap-2">
                  <Input
                    id="license-hwid"
                    :model-value="licenseStore.state.hwid_full"
                    readonly
                    class="font-mono text-[11px] tracking-[0.02em]"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    class="justify-start"
                    :disabled="
                      bootstrapPending || !licenseStore.state.hwid_full
                    "
                    @click="copyHWID"
                  >
                    <Copy class="mr-1.5 h-3.5 w-3.5" />
                    {{ t("licenseSettings.actions.copyHwid") }}
                  </Button>
                </div>
                <p class="text-xs text-muted-foreground">
                  {{
                    t("licenseSettings.fields.shortIdValue", {
                      value: shortIdLabel,
                    })
                  }}
                </p>
              </CardContent>
            </Card>

            <Card class="border-border/50 bg-muted/30">
              <CardHeader>
                <CardTitle class="text-base">
                  {{ t("licenseSettings.sections.activation") }}
                </CardTitle>
                <CardDescription>
                  {{ t("licenseSettings.descriptions.activation") }}
                </CardDescription>
              </CardHeader>
              <CardContent class="space-y-4">
                <div class="space-y-2">
                  <Label for="license-security-key" class="text-xs">
                    {{ t("licenseSettings.fields.securityKey") }}
                  </Label>
                  <Textarea
                    id="license-security-key"
                    v-model="securityKey"
                    :rows="6"
                    :disabled="licenseStore.activationPending"
                    class="min-h-[160px] font-mono text-xs"
                    :placeholder="t('licenseSettings.placeholders.securityKey')"
                  />
                </div>

                <Alert
                  v-if="licenseStore.error"
                  variant="destructive"
                  class="border-destructive/35"
                >
                  <LockKeyhole class="h-4 w-4" />
                  <AlertTitle>
                    {{ t("licenseSettings.errors.activationBlocked") }}
                  </AlertTitle>
                  <AlertDescription>
                    {{ licenseStore.error }}
                  </AlertDescription>
                </Alert>

                <div class="flex flex-col gap-2 sm:flex-row">
                  <Button
                    size="sm"
                    :disabled="
                      licenseStore.activationPending || bootstrapPending
                    "
                    @click="handleActivationSubmit"
                  >
                    <Loader2
                      v-if="licenseStore.activationPending"
                      class="mr-1.5 h-3.5 w-3.5 animate-spin"
                    />
                    <KeyRound v-else class="mr-1.5 h-3.5 w-3.5" />
                    {{ t("licenseSettings.actions.activateLicense") }}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    :disabled="bootstrapPending"
                    @click="refreshStatus"
                  >
                    <Loader2
                      v-if="bootstrapPending"
                      class="mr-1.5 h-3.5 w-3.5 animate-spin"
                    />
                    {{ t("licenseSettings.actions.refreshStatus") }}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </template>
      </div>
    </ScrollArea>
  </div>
</template>
