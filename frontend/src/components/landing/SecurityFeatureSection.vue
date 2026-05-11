<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Badge } from "@/components/ui/badge";
import {
  Shield,
  Lock,
  Key,
  Fingerprint,
  ShieldCheck,
  Globe,
  Gauge,
  FileLock,
  CheckCheck,
  Scan,
  UserCheck,
  Building,
  Hash,
  AlertTriangle,
  ShieldAlert,
  Network,
} from "lucide-vue-next";

const { t } = useI18n();

const securityFeatures = computed(() => [
  { icon: UserCheck, title: t("landingSecurity.ssoTitle"), description: t("landingSecurity.ssoDesc"), details: [t("landingSecurity.ssoD1"), t("landingSecurity.ssoD2"), t("landingSecurity.ssoD3"), t("landingSecurity.ssoD4")] },
  { icon: Key, title: t("landingSecurity.apiKeysTitle"), description: t("landingSecurity.apiKeysDesc"), details: [t("landingSecurity.apiKeysD1"), t("landingSecurity.apiKeysD2"), t("landingSecurity.apiKeysD3")] },
  { icon: FileLock, title: t("landingSecurity.encryptionTitle"), description: t("landingSecurity.encryptionDesc"), details: [t("landingSecurity.encryptionD1"), t("landingSecurity.encryptionD2"), t("landingSecurity.encryptionD3")] },
  { icon: ShieldCheck, title: t("landingSecurity.csrfTitle"), description: t("landingSecurity.csrfDesc"), details: [t("landingSecurity.csrfD1"), t("landingSecurity.csrfD2"), t("landingSecurity.csrfD3")] },
  { icon: Gauge, title: t("landingSecurity.rateLimitTitle"), description: t("landingSecurity.rateLimitDesc"), details: [t("landingSecurity.rateLimitD1"), t("landingSecurity.rateLimitD2"), t("landingSecurity.rateLimitD3"), t("landingSecurity.rateLimitD4")] },
  { icon: ShieldAlert, title: t("landingSecurity.headersTitle"), description: t("landingSecurity.headersDesc"), details: [t("landingSecurity.headersD1"), t("landingSecurity.headersD2"), t("landingSecurity.headersD3")] },
  { icon: Building, title: t("landingSecurity.multiOrgTitle"), description: t("landingSecurity.multiOrgDesc"), details: [t("landingSecurity.multiOrgD1"), t("landingSecurity.multiOrgD2"), t("landingSecurity.multiOrgD3")] },
  { icon: Hash, title: t("landingSecurity.rbacTitle"), description: t("landingSecurity.rbacDesc"), details: [t("landingSecurity.rbacD1"), t("landingSecurity.rbacD2"), t("landingSecurity.rbacD3"), t("landingSecurity.rbacD4")] },
]);

const quickCapabilities = computed(() => [
  { icon: UserCheck, label: t("landingSecurity.capSSO") },
  { icon: Key, label: t("landingSecurity.capAPIKeys") },
  { icon: FileLock, label: t("landingSecurity.capEncryption") },
  { icon: ShieldCheck, label: t("landingSecurity.capCSRF") },
  { icon: Gauge, label: t("landingSecurity.capRateLimit") },
  { icon: ShieldAlert, label: t("landingSecurity.capHeaders") },
  { icon: Building, label: t("landingSecurity.capMultiOrg") },
  { icon: Hash, label: t("landingSecurity.capRBAC") },
  { icon: Fingerprint, label: t("landingSecurity.capJWT") },
  { icon: Globe, label: t("landingSecurity.capCORS") },
  { icon: Scan, label: t("landingSecurity.capWebhookHMAC") },
  { icon: Lock, label: t("landingSecurity.capPasswordPolicy") },
  { icon: AlertTriangle, label: t("landingSecurity.capPanicRecovery") },
  { icon: Shield, label: t("landingSecurity.capPermissionGuard") },
  { icon: Network, label: t("landingSecurity.capTenantScope") },
  { icon: UserCheck, label: t("landingSecurity.capSignedInvites") },
]);
</script>

<template>
  <section class="border-t border-border/40 bg-muted/30 py-24 sm:py-32">
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-2xl text-center">
        <Badge variant="outline" class="mb-5 gap-2 rounded-full border-rose-500/20 bg-rose-500/5 px-4 py-1.5 text-xs text-rose-600">
          <Shield class="h-3.5 w-3.5" aria-hidden="true" />
          {{ $t("landingSecurity.badge") }}
        </Badge>
        <h2 class="text-3xl font-extrabold tracking-tight sm:text-4xl">{{ $t("landingSecurity.title") }}</h2>
        <p class="mt-4 text-base leading-7 text-muted-foreground sm:text-lg">{{ $t("landingSecurity.subtitle") }}</p>
      </div>
      <div class="mt-16 grid gap-5 md:grid-cols-2">
        <div v-for="feature in securityFeatures" :key="feature.title" class="group relative rounded-xl border border-border bg-card p-6 transition-[border-color,box-shadow] duration-300 hover:border-rose-500/20 hover:shadow-lg hover:shadow-rose-500/5">
          <div class="flex items-start gap-4">
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-rose-500/10">
              <component :is="feature.icon" class="h-5 w-5 text-rose-600 dark:text-rose-400" aria-hidden="true" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-base font-semibold text-foreground">{{ feature.title }}</h3>
              <p class="mt-1.5 text-sm leading-6 text-muted-foreground">{{ feature.description }}</p>
              <ul class="mt-3.5 space-y-2">
                <li v-for="detail in feature.details" :key="detail" class="flex items-start gap-2.5 text-sm leading-5 text-muted-foreground">
                  <CheckCheck class="mt-0.5 h-3.5 w-3.5 shrink-0 text-rose-500/60" aria-hidden="true" />
                  <span>{{ detail }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div class="absolute inset-x-0 bottom-0 h-0.5 rounded-b-xl bg-gradient-to-r from-transparent via-rose-500/40 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
        </div>
      </div>
      <div class="mt-20">
        <h3 class="text-center text-lg font-semibold text-foreground">{{ $t("landingSecurity.capabilitiesTitle") }}</h3>
        <p class="mt-2 text-center text-sm text-muted-foreground">{{ $t("landingSecurity.capabilitiesSubtitle") }}</p>
        <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
          <div v-for="cap in quickCapabilities" :key="cap.label" class="flex items-center gap-2 rounded-full border border-border bg-card px-4 py-2 text-sm transition-[border-color,background-color] duration-200 hover:border-rose-500/20 hover:bg-accent/50">
            <component :is="cap.icon" class="h-4 w-4 text-rose-500/70" aria-hidden="true" />
            <span class="font-medium text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
