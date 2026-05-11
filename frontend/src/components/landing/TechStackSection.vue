<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Badge } from "@/components/ui/badge";
import { Code2 } from "lucide-vue-next";

const { t } = useI18n();

const categories = computed(() => [
  {
    title: t("landingTech.catBackend"),
    items: [
      { name: "Go", desc: "1.25", detail: t("landingTech.goDesc") },
      { name: "fasthttp", desc: "", detail: t("landingTech.fasthttpDesc") },
      { name: "GORM", desc: "", detail: t("landingTech.gormDesc") },
      { name: "PostgreSQL", desc: "17", detail: t("landingTech.pgDesc") },
    ],
  },
  {
    title: t("landingTech.catFrontend"),
    items: [
      { name: "Vue 3", desc: "Composition API", detail: t("landingTech.vueDesc") },
      { name: "TypeScript", desc: "", detail: t("landingTech.tsDesc") },
      { name: "Vite", desc: "", detail: t("landingTech.viteDesc") },
      { name: "Tailwind CSS", desc: "v3", detail: t("landingTech.twDesc") },
    ],
  },
  {
    title: t("landingTech.catInfra"),
    items: [
      { name: "Redis", desc: "7", detail: t("landingTech.redisDesc") },
      { name: "WebSocket", desc: "", detail: t("landingTech.wsDesc") },
      { name: "Docker", desc: "", detail: t("landingTech.dockerDesc") },
      { name: "GitHub Actions", desc: "", detail: t("landingTech.ciDesc") },
    ],
  },
  {
    title: t("landingTech.catWhatsApp"),
    items: [
      { name: "Meta Cloud API", desc: "", detail: t("landingTech.metaDesc") },
      { name: "whatsmeow", desc: "", detail: t("landingTech.whatsmeowDesc") },
      { name: "goja", desc: "", detail: t("landingTech.gojaDesc") },
      { name: "MCP Server", desc: "Node.js", detail: t("landingTech.mcpDesc") },
    ],
  },
  {
    title: t("landingTech.catUILibs"),
    items: [
      { name: "shadcn-vue", desc: "", detail: t("landingTech.shadcnDesc") },
      { name: "Pinia", desc: "", detail: t("landingTech.piniaDesc") },
      { name: "vue-i18n", desc: "", detail: t("landingTech.i18nDesc") },
      { name: "Chart.js", desc: "", detail: t("landingTech.chartDesc") },
    ],
  },
  {
    title: t("landingTech.catTooling"),
    items: [
      { name: "koanf", desc: "", detail: t("landingTech.koanfDesc") },
      { name: "golangci-lint", desc: "", detail: t("landingTech.lintDesc") },
      { name: "Playwright", desc: "", detail: t("landingTech.playwrightDesc") },
      { name: "Vitest", desc: "", detail: t("landingTech.vitestDesc") },
    ],
  },
]);
</script>

<template>
  <section class="border-t border-border/40 bg-muted/30 py-24 sm:py-32">
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-2xl text-center">
        <Badge variant="outline" class="mb-5 gap-2 rounded-full border-sky-500/20 bg-sky-500/5 px-4 py-1.5 text-xs text-sky-600">
          <Code2 class="h-3.5 w-3.5" aria-hidden="true" />
          {{ $t("landingTech.badge") }}
        </Badge>
        <h2 class="text-3xl font-extrabold tracking-tight sm:text-4xl">{{ $t("landingTech.title") }}</h2>
        <p class="mt-4 text-base leading-7 text-muted-foreground sm:text-lg">{{ $t("landingTech.subtitle") }}</p>
      </div>
      <div class="mt-16 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="category in categories"
          :key="category.title"
          class="rounded-xl border border-border bg-card p-6"
        >
          <h3 class="mb-5 text-sm font-semibold uppercase tracking-wider text-muted-foreground">{{ category.title }}</h3>
          <div class="space-y-4">
            <div v-for="item in category.items" :key="item.name" class="flex items-start gap-3">
              <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-sky-500/10 text-xs font-bold text-sky-600 dark:text-sky-400">
                {{ item.name.substring(0, 2).toUpperCase() }}
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-baseline gap-2">
                  <span class="text-sm font-semibold text-foreground">{{ item.name }}</span>
                  <span v-if="item.desc" class="text-xs text-muted-foreground">{{ item.desc }}</span>
                </div>
                <p class="mt-0.5 text-xs leading-5 text-muted-foreground">{{ item.detail }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
