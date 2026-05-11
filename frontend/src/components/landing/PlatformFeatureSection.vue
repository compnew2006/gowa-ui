<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Badge } from "@/components/ui/badge";
import {
  Globe,
  Layers,
  Cpu,
  HardDrive,
  Timer,
  RefreshCw,
  Palette,
  Server,
  CheckCheck,
  Monitor,
  Languages,
  Database,
  Zap,
  KeyRound,
  Activity,
  ArrowRightLeft,
} from "lucide-vue-next";

const { t } = useI18n();

const platformFeatures = computed(() => [
  {
    icon: Globe,
    title: t("landingPlatform.i18nTitle"),
    description: t("landingPlatform.i18nDesc"),
    details: [
      t("landingPlatform.i18nD1"),
      t("landingPlatform.i18nD2"),
      t("landingPlatform.i18nD3"),
    ],
  },
  {
    icon: Palette,
    title: t("landingPlatform.themeTitle"),
    description: t("landingPlatform.themeDesc"),
    details: [
      t("landingPlatform.themeD1"),
      t("landingPlatform.themeD2"),
      t("landingPlatform.themeD3"),
    ],
  },
  {
    icon: Cpu,
    title: t("landingPlatform.scalingTitle"),
    description: t("landingPlatform.scalingDesc"),
    details: [
      t("landingPlatform.scalingD1"),
      t("landingPlatform.scalingD2"),
      t("landingPlatform.scalingD3"),
      t("landingPlatform.scalingD4"),
    ],
  },
  {
    icon: RefreshCw,
    title: t("landingPlatform.resilienceTitle"),
    description: t("landingPlatform.resilienceDesc"),
    details: [
      t("landingPlatform.resilienceD1"),
      t("landingPlatform.resilienceD2"),
      t("landingPlatform.resilienceD3"),
      t("landingPlatform.resilienceD4"),
    ],
  },
  {
    icon: HardDrive,
    title: t("landingPlatform.mediaTitle"),
    description: t("landingPlatform.mediaDesc"),
    details: [
      t("landingPlatform.mediaD1"),
      t("landingPlatform.mediaD2"),
      t("landingPlatform.mediaD3"),
    ],
  },
  {
    icon: KeyRound,
    title: t("landingPlatform.licenseTitle"),
    description: t("landingPlatform.licenseDesc"),
    details: [
      t("landingPlatform.licenseD1"),
      t("landingPlatform.licenseD2"),
      t("landingPlatform.licenseD3"),
      t("landingPlatform.licenseD4"),
    ],
  },
  {
    icon: Server,
    title: t("landingPlatform.infraTitle"),
    description: t("landingPlatform.infraDesc"),
    details: [
      t("landingPlatform.infraD1"),
      t("landingPlatform.infraD2"),
      t("landingPlatform.infraD3"),
    ],
  },
  {
    icon: Layers,
    title: t("landingPlatform.mcpTitle"),
    description: t("landingPlatform.mcpDesc"),
    details: [
      t("landingPlatform.mcpD1"),
      t("landingPlatform.mcpD2"),
      t("landingPlatform.mcpD3"),
    ],
  },
]);

const quickCapabilities = computed(() => [
  { icon: Globe, label: t("landingPlatform.capMultiLang") },
  { icon: Monitor, label: t("landingPlatform.capDarkMode") },
  { icon: Languages, label: t("landingPlatform.capRTL") },
  { icon: Cpu, label: t("landingPlatform.capAutoScale") },
  { icon: Database, label: t("landingPlatform.capRedis") },
  { icon: RefreshCw, label: t("landingPlatform.capRetry") },
  { icon: Timer, label: t("landingPlatform.capDLQ") },
  { icon: HardDrive, label: t("landingPlatform.capMediaRetention") },
  { icon: KeyRound, label: t("landingPlatform.capLicense") },
  { icon: Zap, label: t("landingPlatform.capHWID") },
  { icon: Activity, label: t("landingPlatform.capHealthCheck") },
  { icon: Server, label: t("landingPlatform.capPrometheus") },
  { icon: ArrowRightLeft, label: t("landingPlatform.capWebSocket") },
  { icon: Palette, label: t("landingPlatform.capEmbedSPA") },
  { icon: Layers, label: t("landingPlatform.capMCPSidecar") },
  { icon: Globe, label: t("landingPlatform.capMarketing") },
]);
</script>

<template>
  <section class="border-t border-border/50 bg-muted/10 py-20 sm:py-28">
    <div class="mx-auto max-w-6xl px-4 sm:px-6">
      <div class="mx-auto max-w-2xl text-center">
        <Badge variant="outline" class="mb-4 gap-1.5 border-violet-500/25 bg-violet-500/10 px-3 py-1 text-xs text-violet-600">
          <Layers class="h-3 w-3" />
          {{ $t("landingPlatform.badge") }}
        </Badge>
        <h2 class="text-2xl font-bold tracking-tight sm:text-3xl">
          {{ $t("landingPlatform.title") }}
        </h2>
        <p class="mt-3 text-sm leading-7 text-muted-foreground sm:text-base">
          {{ $t("landingPlatform.subtitle") }}
        </p>
      </div>

      <div class="mt-12 grid gap-5 sm:mt-16 md:grid-cols-2">
        <div
          v-for="feature in platformFeatures"
          :key="feature.title"
          class="group relative rounded-[calc(var(--radius)+0.5rem)] border border-border bg-card p-6 transition-all duration-200 hover:border-violet-500/20 hover:shadow-md"
        >
          <div class="flex items-start gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-violet-500/10">
              <component :is="feature.icon" class="h-5 w-5 text-violet-600 dark:text-violet-400" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-sm font-semibold text-foreground">{{ feature.title }}</h3>
              <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ feature.description }}</p>
              <ul class="mt-3 space-y-1.5">
                <li
                  v-for="detail in feature.details"
                  :key="detail"
                  class="flex items-start gap-2 text-xs leading-5 text-muted-foreground"
                >
                  <CheckCheck class="mt-0.5 h-3 w-3 shrink-0 text-violet-500/70" />
                  <span>{{ detail }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div
            class="absolute inset-x-0 bottom-0 h-px mx-4 bg-gradient-to-r from-transparent via-violet-500/40 to-transparent opacity-0 transition-opacity duration-200 group-hover:opacity-100"
          />
        </div>
      </div>

      <div class="mt-16">
        <h3 class="text-center text-lg font-semibold text-foreground">
          {{ $t("landingPlatform.capabilitiesTitle") }}
        </h3>
        <p class="mt-2 text-center text-sm text-muted-foreground">
          {{ $t("landingPlatform.capabilitiesSubtitle") }}
        </p>
        <div
          class="mt-8 grid grid-cols-3 gap-2.5 sm:grid-cols-4 md:grid-cols-4 lg:grid-cols-8"
        >
          <div
            v-for="cap in quickCapabilities"
            :key="cap.label"
            class="flex flex-col items-center gap-2 rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card p-3 text-center transition-colors hover:border-violet-500/20 hover:bg-accent/50"
          >
            <component :is="cap.icon" class="h-4 w-4 text-violet-500/80" />
            <span class="text-[11px] font-medium leading-tight text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
