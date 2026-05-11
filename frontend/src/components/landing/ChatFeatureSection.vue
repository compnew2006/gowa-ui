<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Badge } from "@/components/ui/badge";
import {
  MessageSquare,
  Send,
  Paperclip,
  Smile,
  FileText,
  Reply,
  RotateCcw,
  Tag,
  StickyNote,
  MessageSquareText,
  Users,
  Zap,
  Shield,
  Smartphone,
  Clock,
  CheckCheck,
  Search,
  Printer,
  ThumbsUp,
  Columns,
  Eye,
  Globe,
  Bot,
  Workflow,
  BarChart3,
  Megaphone,
} from "lucide-vue-next";

const { t } = useI18n();

const chatFeatures = computed(() => [
  {
    icon: Send,
    title: t("landingChat.realtimeTitle"),
    description: t("landingChat.realtimeDesc"),
    details: [
      t("landingChat.realtimeD1"),
      t("landingChat.realtimeD2"),
      t("landingChat.realtimeD3"),
    ],
  },
  {
    icon: Paperclip,
    title: t("landingChat.mediaTitle"),
    description: t("landingChat.mediaDesc"),
    details: [
      t("landingChat.mediaD1"),
      t("landingChat.mediaD2"),
      t("landingChat.mediaD3"),
      t("landingChat.mediaD4"),
    ],
  },
  {
    icon: Columns,
    title: t("landingChat.sidebarTitle"),
    description: t("landingChat.sidebarDesc"),
    details: [
      t("landingChat.sidebarD1"),
      t("landingChat.sidebarD2"),
      t("landingChat.sidebarD3"),
      t("landingChat.sidebarD4"),
    ],
  },
  {
    icon: Users,
    title: t("landingChat.assignTitle"),
    description: t("landingChat.assignDesc"),
    details: [
      t("landingChat.assignD1"),
      t("landingChat.assignD2"),
      t("landingChat.assignD3"),
      t("landingChat.assignD4"),
    ],
  },
  {
    icon: Bot,
    title: t("landingChat.smartTitle"),
    description: t("landingChat.smartDesc"),
    details: [
      t("landingChat.smartD1"),
      t("landingChat.smartD2"),
      t("landingChat.smartD3"),
      t("landingChat.smartD4"),
    ],
  },
  {
    icon: Eye,
    title: t("landingChat.contactInfoTitle"),
    description: t("landingChat.contactInfoDesc"),
    details: [
      t("landingChat.contactInfoD1"),
      t("landingChat.contactInfoD2"),
      t("landingChat.contactInfoD3"),
      t("landingChat.contactInfoD4"),
    ],
  },
  {
    icon: Smartphone,
    title: t("landingChat.statusTitle"),
    description: t("landingChat.statusDesc"),
    details: [
      t("landingChat.statusD1"),
      t("landingChat.statusD2"),
      t("landingChat.statusD3"),
    ],
  },
  {
    icon: BarChart3,
    title: t("landingChat.ratingsTitle"),
    description: t("landingChat.ratingsDesc"),
    details: [
      t("landingChat.ratingsD1"),
      t("landingChat.ratingsD2"),
      t("landingChat.ratingsD3"),
    ],
  },
]);

const quickCapabilities = computed(() => [
  { icon: Reply, label: t("landingChat.capReply") },
  { icon: ThumbsUp, label: t("landingChat.capReactions") },
  { icon: RotateCcw, label: t("landingChat.capRevoke") },
  { icon: Smile, label: t("landingChat.capEmoji") },
  { icon: MessageSquareText, label: t("landingChat.capCanned") },
  { icon: Printer, label: t("landingChat.capPrint") },
  { icon: Tag, label: t("landingChat.capTags") },
  { icon: StickyNote, label: t("landingChat.capNotes") },
  { icon: Search, label: t("landingChat.capSearch") },
  { icon: Globe, label: t("landingChat.capRTL") },
  { icon: Zap, label: t("landingChat.capCustomActions") },
  { icon: Shield, label: t("landingChat.capAccess") },
  { icon: Clock, label: t("landingChat.capAutoReset") },
  { icon: FileText, label: t("landingChat.capTemplates") },
  { icon: Workflow, label: t("landingChat.capFlows") },
  { icon: Megaphone, label: t("landingChat.capBulkOps") },
]);
</script>

<template>
  <section class="border-t border-border/50 bg-muted/10 py-20 sm:py-28">
    <div class="mx-auto max-w-6xl px-4 sm:px-6">
      <!-- Section Header -->
      <div class="mx-auto max-w-2xl text-center">
        <Badge variant="outline" class="mb-4 gap-1.5 border-primary/25 px-3 py-1 text-xs">
          <MessageSquare class="h-3 w-3 text-primary" />
          {{ $t("landingChat.badge") }}
        </Badge>
        <h2 class="text-2xl font-bold tracking-tight sm:text-3xl">
          {{ $t("landingChat.title") }}
        </h2>
        <p class="mt-3 text-sm leading-7 text-muted-foreground sm:text-base">
          {{ $t("landingChat.subtitle") }}
        </p>
      </div>

      <!-- Feature Cards -->
      <div class="mt-12 grid gap-5 sm:mt-16 md:grid-cols-2">
        <div
          v-for="feature in chatFeatures"
          :key="feature.title"
          class="group relative rounded-[calc(var(--radius)+0.5rem)] border border-border bg-card p-6 transition-all duration-200 hover:border-primary/20 hover:shadow-md"
        >
          <div class="flex items-start gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10">
              <component :is="feature.icon" class="h-5 w-5 text-primary" />
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
                  <CheckCheck class="mt-0.5 h-3 w-3 shrink-0 text-primary/70" />
                  <span>{{ detail }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div
            class="absolute inset-x-0 bottom-0 h-px mx-4 bg-gradient-to-r from-transparent via-primary/40 to-transparent opacity-0 transition-opacity duration-200 group-hover:opacity-100"
          />
        </div>
      </div>

      <!-- Quick Capabilities -->
      <div class="mt-16">
        <h3 class="text-center text-lg font-semibold text-foreground">
          {{ $t("landingChat.capabilitiesTitle") }}
        </h3>
        <p class="mt-2 text-center text-sm text-muted-foreground">
          {{ $t("landingChat.capabilitiesSubtitle") }}
        </p>
        <div
          class="mt-8 grid grid-cols-2 gap-2.5 sm:grid-cols-4 md:grid-cols-4 lg:grid-cols-8"
        >
          <div
            v-for="cap in quickCapabilities"
            :key="cap.label"
            class="flex flex-col items-center gap-2 rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card p-3 text-center transition-colors hover:border-primary/20 hover:bg-accent/50"
          >
            <component :is="cap.icon" class="h-4 w-4 text-primary/80" />
            <span class="text-[11px] font-medium leading-tight text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
