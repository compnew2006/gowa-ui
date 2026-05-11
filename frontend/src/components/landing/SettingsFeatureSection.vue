<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Badge } from "@/components/ui/badge";
import {
  Settings,
  Users,
  Shield,
  UserCog,
  Smartphone,
  Key,
  Link,
  Webhook,
  FileText,
  Tags,
  MessageSquareText,
  Bot,
  Palette,
  Bell,
  Layers,
  Activity,
  Lock,
  CheckCheck,
} from "lucide-vue-next";

const { t } = useI18n();

const settingsFeatures = computed(() => [
  {
    icon: Users,
    title: t("landingSettings.usersTitle"),
    description: t("landingSettings.usersDesc"),
    details: [
      t("landingSettings.usersD1"),
      t("landingSettings.usersD2"),
      t("landingSettings.usersD3"),
      t("landingSettings.usersD4"),
    ],
  },
  {
    icon: Shield,
    title: t("landingSettings.rbacTitle"),
    description: t("landingSettings.rbacDesc"),
    details: [
      t("landingSettings.rbacD1"),
      t("landingSettings.rbacD2"),
      t("landingSettings.rbacD3"),
    ],
  },
  {
    icon: Smartphone,
    title: t("landingSettings.instancesTitle"),
    description: t("landingSettings.instancesDesc"),
    details: [
      t("landingSettings.instancesD1"),
      t("landingSettings.instancesD2"),
      t("landingSettings.instancesD3"),
      t("landingSettings.instancesD4"),
    ],
  },
  {
    icon: UserCog,
    title: t("landingSettings.teamsTitle"),
    description: t("landingSettings.teamsDesc"),
    details: [
      t("landingSettings.teamsD1"),
      t("landingSettings.teamsD2"),
      t("landingSettings.teamsD3"),
    ],
  },
  {
    icon: FileText,
    title: t("landingSettings.templatesTitle"),
    description: t("landingSettings.templatesDesc"),
    details: [
      t("landingSettings.templatesD1"),
      t("landingSettings.templatesD2"),
      t("landingSettings.templatesD3"),
    ],
  },
  {
    icon: MessageSquareText,
    title: t("landingSettings.cannedTitle"),
    description: t("landingSettings.cannedDesc"),
    details: [
      t("landingSettings.cannedD1"),
      t("landingSettings.cannedD2"),
      t("landingSettings.cannedD3"),
    ],
  },
  {
    icon: Key,
    title: t("landingSettings.integrationsTitle"),
    description: t("landingSettings.integrationsDesc"),
    details: [
      t("landingSettings.integrationsD1"),
      t("landingSettings.integrationsD2"),
      t("landingSettings.integrationsD3"),
      t("landingSettings.integrationsD4"),
    ],
  },
  {
    icon: Palette,
    title: t("landingSettings.orgTitle"),
    description: t("landingSettings.orgDesc"),
    details: [
      t("landingSettings.orgD1"),
      t("landingSettings.orgD2"),
      t("landingSettings.orgD3"),
      t("landingSettings.orgD4"),
    ],
  },
]);

const quickCapabilities = computed(() => [
  { icon: Users, label: t("landingSettings.capUsers") },
  { icon: Shield, label: t("landingSettings.capRoles") },
  { icon: UserCog, label: t("landingSettings.capTeams") },
  { icon: Smartphone, label: t("landingSettings.capInstances") },
  { icon: Activity, label: t("landingSettings.capHealth") },
  { icon: FileText, label: t("landingSettings.capTemplates") },
  { icon: Layers, label: t("landingSettings.capFlows") },
  { icon: MessageSquareText, label: t("landingSettings.capCanned") },
  { icon: Tags, label: t("landingSettings.capTags") },
  { icon: Key, label: t("landingSettings.capAPIKeys") },
  { icon: Webhook, label: t("landingSettings.capWebhooks") },
  { icon: Link, label: t("landingSettings.capSSO") },
  { icon: Bot, label: t("landingSettings.capChatbot") },
  { icon: Lock, label: t("landingSettings.capLicense") },
  { icon: Bell, label: t("landingSettings.capNotifs") },
  { icon: Palette, label: t("landingSettings.capAppearance") },
]);
</script>

<template>
  <section class="border-t border-border/50 bg-muted/10 py-20 sm:py-28">
    <div class="mx-auto max-w-6xl px-4 sm:px-6">
      <!-- Section Header -->
      <div class="mx-auto max-w-2xl text-center">
        <Badge variant="outline" class="mb-4 gap-1.5 border-emerald-500/25 bg-emerald-500/10 px-3 py-1 text-xs text-emerald-600">
          <Settings class="h-3 w-3" />
          {{ $t("landingSettings.badge") }}
        </Badge>
        <h2 class="text-2xl font-bold tracking-tight sm:text-3xl">
          {{ $t("landingSettings.title") }}
        </h2>
        <p class="mt-3 text-sm leading-7 text-muted-foreground sm:text-base">
          {{ $t("landingSettings.subtitle") }}
        </p>
      </div>

      <!-- Feature Cards -->
      <div class="mt-12 grid gap-5 sm:mt-16 md:grid-cols-2">
        <div
          v-for="feature in settingsFeatures"
          :key="feature.title"
          class="group relative rounded-[calc(var(--radius)+0.5rem)] border border-border bg-card p-6 transition-all duration-200 hover:border-emerald-500/20 hover:shadow-md"
        >
          <div class="flex items-start gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10">
              <component :is="feature.icon" class="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
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
                  <CheckCheck class="mt-0.5 h-3 w-3 shrink-0 text-emerald-500/70" />
                  <span>{{ detail }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div
            class="absolute inset-x-0 bottom-0 h-px mx-4 bg-gradient-to-r from-transparent via-emerald-500/40 to-transparent opacity-0 transition-opacity duration-200 group-hover:opacity-100"
          />
        </div>
      </div>

      <!-- Quick Capabilities -->
      <div class="mt-16">
        <h3 class="text-center text-lg font-semibold text-foreground">
          {{ $t("landingSettings.capabilitiesTitle") }}
        </h3>
        <p class="mt-2 text-center text-sm text-muted-foreground">
          {{ $t("landingSettings.capabilitiesSubtitle") }}
        </p>
        <div
          class="mt-8 grid grid-cols-3 gap-2.5 sm:grid-cols-4 md:grid-cols-4 lg:grid-cols-8"
        >
          <div
            v-for="cap in quickCapabilities"
            :key="cap.label"
            class="flex flex-col items-center gap-2 rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card p-3 text-center transition-colors hover:border-emerald-500/20 hover:bg-accent/50"
          >
            <component :is="cap.icon" class="h-4 w-4 text-emerald-500/80" />
            <span class="text-[11px] font-medium leading-tight text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
