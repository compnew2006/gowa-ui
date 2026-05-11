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
  { icon: Globe, title: t("landingPlatform.i18nTitle"), description: t("landingPlatform.i18nDesc"), details: [t("landingPlatform.i18nD1"), t("landingPlatform.i18nD2"), t("landingPlatform.i18nD3")] },
  { icon: Palette, title: t("landingPlatform.themeTitle"), description: t("landingPlatform.themeDesc"), details: [t("landingPlatform.themeD1"), t("landingPlatform.themeD2"), t("landingPlatform.themeD3")] },
  { icon: Cpu, title: t("landingPlatform.scalingTitle"), description: t("landingPlatform.scalingDesc"), details: [t("landingPlatform.scalingD1"), t("landingPlatform.scalingD2"), t("landingPlatform.scalingD3"), t("landingPlatform.scalingD4")] },
  { icon: RefreshCw, title: t("landingPlatform.resilienceTitle"), description: t("landingPlatform.resilienceDesc"), details: [t("landingPlatform.resilienceD1"), t("landingPlatform.resilienceD2"), t("landingPlatform.resilienceD3"), t("landingPlatform.resilienceD4")] },
  { icon: HardDrive, title: t("landingPlatform.mediaTitle"), description: t("landingPlatform.mediaDesc"), details: [t("landingPlatform.mediaD1"), t("landingPlatform.mediaD2"), t("landingPlatform.mediaD3")] },
  { icon: KeyRound, title: t("landingPlatform.licenseTitle"), description: t("landingPlatform.licenseDesc"), details: [t("landingPlatform.licenseD1"), t("landingPlatform.licenseD2"), t("landingPlatform.licenseD3"), t("landingPlatform.licenseD4")] },
  { icon: Server, title: t("landingPlatform.infraTitle"), description: t("landingPlatform.infraDesc"), details: [t("landingPlatform.infraD1"), t("landingPlatform.infraD2"), t("landingPlatform.infraD3")] },
  { icon: Layers, title: t("landingPlatform.mcpTitle"), description: t("landingPlatform.mcpDesc"), details: [t("landingPlatform.mcpD1"), t("landingPlatform.mcpD2"), t("landingPlatform.mcpD3")] },
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
  <section class="border-t border-border/40 py-24 sm:py-32">
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-2xl text-center">
        <Badge variant="outline" class="mb-5 gap-2 rounded-full border-violet-500/20 bg-violet-500/5 px-4 py-1.5 text-xs text-violet-600">
          <Layers class="h-3.5 w-3.5" aria-hidden="true" />
          {{ $t("landingPlatform.badge") }}
        </Badge>
        <h2 class="text-3xl font-extrabold tracking-tight sm:text-4xl">{{ $t("landingPlatform.title") }}</h2>
        <p class="mt-4 text-base leading-7 text-muted-foreground sm:text-lg">{{ $t("landingPlatform.subtitle") }}</p>
      </div>
      <div class="mt-16 grid gap-5 md:grid-cols-2">
        <div v-for="feature in platformFeatures" :key="feature.title" class="group relative rounded-xl border border-border bg-card p-6 transition-[border-color,box-shadow] duration-300 hover:border-violet-500/20 hover:shadow-lg hover:shadow-violet-500/5">
          <div class="flex items-start gap-4">
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-violet-500/10">
              <component :is="feature.icon" class="h-5 w-5 text-violet-600 dark:text-violet-400" aria-hidden="true" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-base font-semibold text-foreground">{{ feature.title }}</h3>
              <p class="mt-1.5 text-sm leading-6 text-muted-foreground">{{ feature.description }}</p>
              <ul class="mt-3.5 space-y-2">
                <li v-for="detail in feature.details" :key="detail" class="flex items-start gap-2.5 text-sm leading-5 text-muted-foreground">
                  <CheckCheck class="mt-0.5 h-3.5 w-3.5 shrink-0 text-violet-500/60" aria-hidden="true" />
                  <span>{{ detail }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div class="absolute inset-x-0 bottom-0 h-0.5 rounded-b-xl bg-gradient-to-r from-transparent via-violet-500/40 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
        </div>
      </div>
      <div class="mt-20">
        <h3 class="text-center text-lg font-semibold text-foreground">{{ $t("landingPlatform.capabilitiesTitle") }}</h3>
        <p class="mt-2 text-center text-sm text-muted-foreground">{{ $t("landingPlatform.capabilitiesSubtitle") }}</p>
        <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
          <div v-for="cap in quickCapabilities" :key="cap.label" class="flex items-center gap-2 rounded-full border border-border bg-card px-4 py-2 text-sm transition-[border-color,background-color] duration-200 hover:border-violet-500/20 hover:bg-accent/50">
            <component :is="cap.icon" class="h-4 w-4 text-violet-500/70" aria-hidden="true" />
            <span class="font-medium text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
