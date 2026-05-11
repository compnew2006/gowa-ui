<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import ChatFeatureSection from "@/components/landing/ChatFeatureSection.vue";
import CampaignFeatureSection from "@/components/landing/CampaignFeatureSection.vue";
import SettingsFeatureSection from "@/components/landing/SettingsFeatureSection.vue";
import ChatbotFeatureSection from "@/components/landing/ChatbotFeatureSection.vue";
import AnalyticsFeatureSection from "@/components/landing/AnalyticsFeatureSection.vue";
import SecurityFeatureSection from "@/components/landing/SecurityFeatureSection.vue";
import PlatformFeatureSection from "@/components/landing/PlatformFeatureSection.vue";
import TechStackSection from "@/components/landing/TechStackSection.vue";
import {
  MessageSquare,
  Bot,
  LayoutDashboard,
  Megaphone,
  FileText,
  BarChart3,
  Users,
  ShieldCheck,
  Workflow,
  Key,
  Webhook,
  Smartphone,
  Zap,
  Globe,
  Lock,
  ArrowRight,
  Menu,
  X,
  Sparkles,
  Settings,
  Tags,
  MessageSquareText,
  ChevronRight,
  Check,
  Star,
  Heart,
} from "lucide-vue-next";

const { t, locale } = useI18n();
const router = useRouter();
const mobileMenuOpen = ref(false);
const scrolled = ref(false);

const isRTL = computed(() => locale.value === "ar");

const features = computed(() => [
  {
    icon: MessageSquare,
    title: t("landing.featureChatTitle"),
    description: t("landing.featureChatDesc"),
    bg: "bg-emerald-500/10",
    text: "text-emerald-600 dark:text-emerald-400",
    number: "01",
  },
  {
    icon: Bot,
    title: t("landing.featureChatbotTitle"),
    description: t("landing.featureChatbotDesc"),
    bg: "bg-purple-500/10",
    text: "text-purple-600 dark:text-purple-400",
    number: "02",
  },
  {
    icon: LayoutDashboard,
    title: t("landing.featureDashboardTitle"),
    description: t("landing.featureDashboardDesc"),
    bg: "bg-blue-500/10",
    text: "text-blue-600 dark:text-blue-400",
    number: "03",
  },
  {
    icon: Megaphone,
    title: t("landing.featureCampaignsTitle"),
    description: t("landing.featureCampaignsDesc"),
    bg: "bg-orange-500/10",
    text: "text-orange-600 dark:text-orange-400",
    number: "04",
  },
  {
    icon: Workflow,
    title: t("landing.featureFlowsTitle"),
    description: t("landing.featureFlowsDesc"),
    bg: "bg-indigo-500/10",
    text: "text-indigo-600 dark:text-indigo-400",
    number: "05",
  },
  {
    icon: BarChart3,
    title: t("landing.featureAnalyticsTitle"),
    description: t("landing.featureAnalyticsDesc"),
    bg: "bg-teal-500/10",
    text: "text-teal-600 dark:text-teal-400",
    number: "06",
  },
  {
    icon: Users,
    title: t("landing.featureTeamsTitle"),
    description: t("landing.featureTeamsDesc"),
    bg: "bg-violet-500/10",
    text: "text-violet-600 dark:text-violet-400",
    number: "07",
  },
  {
    icon: ShieldCheck,
    title: t("landing.featureSecurityTitle"),
    description: t("landing.featureSecurityDesc"),
    bg: "bg-rose-500/10",
    text: "text-rose-600 dark:text-rose-400",
    number: "08",
  },
]);

const stats = computed(() => [
  { value: "10M+", label: t("landing.statMessages"), icon: MessageSquare },
  { value: "99.9%", label: t("landing.statUptime"), icon: Zap },
  { value: "50+", label: t("landing.statCountries"), icon: Globe },
  { value: "24/7", label: t("landing.statSupport"), icon: Heart },
]);

const capabilities = computed(() => [
  { icon: FileText, label: t("landing.capTemplates") },
  { icon: Key, label: t("landing.capAPIKeys") },
  { icon: Webhook, label: t("landing.capWebhooks") },
  { icon: Smartphone, label: t("landing.capMultiProvider") },
  { icon: Globe, label: t("landing.capMultiLang") },
  { icon: Lock, label: t("landing.capRBAC") },
  { icon: Tags, label: t("landing.capTags") },
  { icon: MessageSquareText, label: t("landing.capCanned") },
  { icon: Zap, label: t("landing.capCustomActions") },
  { icon: Settings, label: t("landing.capSSO") },
]);

const trustItems = computed(() => [
  { label: t("landing.trustEncryption"), icon: Lock },
  { label: t("landing.trustOpenSource"), icon: Star },
  { label: t("landing.trustGDPR"), icon: ShieldCheck },
  { label: t("landing.trustEnterprise"), icon: Check },
]);

const handleScroll = () => {
  scrolled.value = window.scrollY > 20;
};

onMounted(() => {
  window.addEventListener("scroll", handleScroll, { passive: true });
});

onUnmounted(() => {
  window.removeEventListener("scroll", handleScroll);
});

const goToLogin = () => {
  router.push("/login");
};

const goToDashboard = () => {
  router.push("/dashboard");
};
</script>

<template>
  <div class="min-h-screen bg-background text-foreground antialiased">
    <!-- Header -->
    <header
      class="fixed top-0 left-0 right-0 z-50 transition-all duration-300"
      :class="
        scrolled
          ? 'border-b border-border/40 bg-background/95 shadow-sm backdrop-blur-xl'
          : 'bg-transparent'
      "
    >
      <div class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <div class="flex items-center gap-2.5">
          <div
            class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm"
          >
            <MessageSquare class="h-4 w-4 text-white" />
          </div>
          <span class="text-lg font-bold tracking-tight">Whatomate</span>
        </div>

        <nav class="hidden items-center gap-6 lg:flex">
          <a
            href="#features"
            class="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            {{ $t("landing.navFeatures") }}
          </a>
          <a
            href="#deep-dive"
            class="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            {{ $t("landing.navDeepDive") }}
          </a>
          <a
            href="#tech-stack"
            class="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            {{ $t("landing.navTechStack") }}
          </a>
          <a
            href="#capabilities"
            class="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            {{ $t("landing.navCapabilities") }}
          </a>
        </nav>

        <div class="hidden items-center gap-3 sm:flex">
          <Button variant="ghost" size="sm" class="text-sm" @click="goToLogin">
            {{ $t("auth.signIn") }}
          </Button>
          <Button size="sm" class="gap-2" @click="goToLogin">
            {{ $t("auth.signUp") }}
            <ArrowRight class="h-3.5 w-3.5" />
          </Button>
        </div>

        <Button
          variant="ghost"
          size="icon"
          class="sm:hidden"
          @click="mobileMenuOpen = !mobileMenuOpen"
        >
          <X v-if="mobileMenuOpen" class="h-5 w-5" />
          <Menu v-else class="h-5 w-5" />
        </Button>
      </div>

      <!-- Mobile menu -->
      <div
        v-if="mobileMenuOpen"
        class="border-t border-border/40 bg-background/98 backdrop-blur-xl sm:hidden"
      >
        <div class="space-y-1 px-4 py-4">
          <a
            href="#features"
            class="block rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-foreground"
            @click="mobileMenuOpen = false"
          >
            {{ $t("landing.navFeatures") }}
          </a>
          <a
            href="#deep-dive"
            class="block rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-foreground"
            @click="mobileMenuOpen = false"
          >
            {{ $t("landing.navDeepDive") }}
          </a>
          <a
            href="#tech-stack"
            class="block rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-foreground"
            @click="mobileMenuOpen = false"
          >
            {{ $t("landing.navTechStack") }}
          </a>
          <a
            href="#capabilities"
            class="block rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-foreground"
            @click="mobileMenuOpen = false"
          >
            {{ $t("landing.navCapabilities") }}
          </a>
          <div class="flex gap-2 pt-3">
            <Button variant="outline" size="sm" class="flex-1" @click="goToLogin">
              {{ $t("auth.signIn") }}
            </Button>
            <Button size="sm" class="flex-1" @click="goToLogin">
              {{ $t("auth.signUp") }}
            </Button>
          </div>
        </div>
      </div>
    </header>

    <!-- Hero -->
    <section class="relative overflow-hidden pt-16">
      <div class="absolute inset-0 -z-10">
        <div
          class="absolute inset-0"
          :style="{
            background:
              'radial-gradient(ellipse 80% 60% at 50% -10%, rgb(var(--primary) / 0.15), transparent 70%)',
          }"
        />
        <div
          class="absolute top-0 left-1/2 h-[600px] w-[600px] -translate-x-1/2"
          :style="{
            background:
              'radial-gradient(circle, rgb(var(--primary) / 0.06), transparent 70%)',
          }"
        />
      </div>

      <div class="mx-auto max-w-7xl px-4 pb-16 pt-20 sm:px-6 sm:pt-28 lg:px-8 lg:pt-36">
        <div class="mx-auto max-w-3xl text-center">
          <div class="mb-6 inline-flex items-center justify-center sm:mb-8">
            <Badge
              variant="outline"
              class="gap-2 rounded-full border-primary/20 bg-primary/5 px-4 py-1.5 text-xs"
            >
              <Sparkles class="h-3.5 w-3.5 text-primary" />
              {{ $t("landing.heroBadge") }}
            </Badge>
          </div>

          <h1
            class="text-4xl font-extrabold leading-[1.1] tracking-tight sm:text-5xl lg:text-6xl"
          >
            {{ $t("landing.heroTitle") }}
            <span
              class="bg-gradient-to-r from-primary via-primary/80 to-primary/50 bg-clip-text text-transparent"
            >
              {{ $t("landing.heroHighlight") }}
            </span>
          </h1>

          <p
            class="mx-auto mt-6 max-w-2xl text-lg leading-8 text-muted-foreground sm:text-xl"
          >
            {{ $t("landing.heroSubtitle") }}
          </p>

          <div class="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
            <Button
              size="lg"
              class="h-12 gap-2 rounded-lg px-8 text-base shadow-lg shadow-primary/25"
              @click="goToDashboard"
            >
              {{ $t("landing.heroCTA") }}
              <ArrowRight class="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="lg"
              class="h-12 gap-2 rounded-lg px-8 text-base"
              @click="goToLogin"
            >
              {{ $t("landing.heroSecondaryCTA") }}
            </Button>
          </div>

          <!-- Trust badges -->
          <div
            class="mt-12 flex flex-wrap items-center justify-center gap-6 text-muted-foreground"
          >
            <div
              v-for="item in trustItems"
              :key="item.label"
              class="flex items-center gap-2 text-xs font-medium"
            >
              <component :is="item.icon" class="h-3.5 w-3.5 text-primary/60" />
              {{ item.label }}
            </div>
          </div>
        </div>
      </div>

      <!-- Stats -->
      <div class="mx-auto max-w-7xl px-4 pb-20 sm:px-6 lg:px-8">
        <div class="rounded-2xl border border-border/60 bg-card/50 p-8 shadow-sm backdrop-blur-sm">
          <div class="grid grid-cols-2 gap-8 sm:grid-cols-4">
            <div v-for="stat in stats" :key="stat.label" class="text-center">
              <div class="mb-2 flex justify-center">
                <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10">
                  <component :is="stat.icon" class="h-5 w-5 text-primary" />
                </div>
              </div>
              <div class="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl">
                {{ stat.value }}
              </div>
              <div class="mt-1 text-sm font-medium text-muted-foreground">
                {{ stat.label }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Features -->
    <section id="features" class="border-t border-border/40 bg-muted/30 py-24 sm:py-32">
      <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div class="mx-auto max-w-2xl text-center">
          <Badge variant="secondary" class="mb-4 rounded-full px-3 py-1">
            {{ $t("landing.featuresBadge") }}
          </Badge>
          <h2 class="text-3xl font-extrabold tracking-tight sm:text-4xl">
            {{ $t("landing.featuresTitle") }}
          </h2>
          <p class="mt-4 text-base leading-7 text-muted-foreground sm:text-lg">
            {{ $t("landing.featuresSubtitle") }}
          </p>
        </div>

        <div class="mt-16 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="feature in features"
            :key="feature.title"
            class="group relative overflow-hidden rounded-xl border border-border bg-card p-6 transition-all duration-300 hover:border-primary/20 hover:shadow-lg hover:shadow-primary/5"
          >
            <div class="absolute top-4 right-4 text-3xl font-extrabold text-muted-foreground/10">
              {{ feature.number }}
            </div>
            <div
              :class="['mb-4 flex h-12 w-12 items-center justify-center rounded-xl', feature.bg]"
            >
              <component :is="feature.icon" :class="['h-6 w-6', feature.text]" />
            </div>
            <h3 class="text-base font-semibold text-foreground">{{ feature.title }}</h3>
            <p class="mt-2 text-sm leading-6 text-muted-foreground">
              {{ feature.description }}
            </p>
            <div
              class="absolute inset-x-0 bottom-0 h-0.5 bg-gradient-to-r from-transparent via-primary/50 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100"
            />
          </div>
        </div>
      </div>
    </section>

    <!-- Deep Dives -->
    <div id="deep-dive">
      <ChatFeatureSection />
      <CampaignFeatureSection />
      <SettingsFeatureSection />
      <ChatbotFeatureSection />
      <AnalyticsFeatureSection />
      <SecurityFeatureSection />
      <PlatformFeatureSection />
    </div>

    <!-- Tech Stack -->
    <div id="tech-stack">
      <TechStackSection />
    </div>

    <!-- Capabilities -->
    <section id="capabilities" class="border-t border-border/40 py-24 sm:py-32">
      <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div class="mx-auto max-w-2xl text-center">
          <Badge variant="secondary" class="mb-4 rounded-full px-3 py-1">
            {{ $t("landing.capBadge") }}
          </Badge>
          <h2 class="text-3xl font-extrabold tracking-tight sm:text-4xl">
            {{ $t("landing.capTitle") }}
          </h2>
          <p class="mt-4 text-base leading-7 text-muted-foreground sm:text-lg">
            {{ $t("landing.capSubtitle") }}
          </p>
        </div>

        <div
          class="mt-16 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5"
        >
          <div
            v-for="cap in capabilities"
            :key="cap.label"
            class="flex items-center gap-3 rounded-xl border border-border bg-card p-4 transition-all duration-200 hover:border-primary/20 hover:shadow-sm"
          >
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10">
              <component :is="cap.icon" class="h-4 w-4 text-primary" />
            </div>
            <span class="text-sm font-medium text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA -->
    <section class="relative overflow-hidden border-t border-border/40 py-24 sm:py-32">
      <div class="absolute inset-0 -z-10 bg-gradient-to-b from-muted/50 to-muted/20" />
      <div class="absolute inset-0 -z-10 opacity-30">
        <div
          class="absolute inset-0"
          :style="{
            background:
              'radial-gradient(ellipse 60% 40% at 50% 50%, rgb(var(--primary) / 0.08), transparent)',
          }"
        />
      </div>
      <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div class="mx-auto max-w-2xl text-center">
          <div class="mb-8 flex justify-center">
            <div
              class="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10 shadow-lg shadow-primary/10"
            >
              <MessageSquare class="h-8 w-8 text-primary" />
            </div>
          </div>
          <h2 class="text-3xl font-extrabold tracking-tight sm:text-4xl">
            {{ $t("landing.ctaTitle") }}
          </h2>
          <p class="mt-4 text-base leading-7 text-muted-foreground sm:text-lg">
            {{ $t("landing.ctaSubtitle") }}
          </p>
          <div class="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
            <Button
              size="lg"
              class="h-12 gap-2 rounded-lg px-10 text-base shadow-lg shadow-primary/25"
              @click="goToLogin"
            >
              {{ $t("landing.ctaPrimary") }}
              <ArrowRight class="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="lg"
              class="h-12 gap-2 rounded-lg px-10 text-base"
              @click="goToDashboard"
            >
              {{ $t("landing.ctaSecondary") }}
              <ChevronRight class="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="border-t border-border/40 bg-muted/20">
      <div class="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div class="flex flex-col items-center justify-between gap-6 sm:flex-row">
          <div class="flex items-center gap-2.5">
            <div
              class="flex h-7 w-7 items-center justify-center rounded-lg bg-primary text-primary-foreground"
            >
              <MessageSquare class="h-3.5 w-3.5 text-white" />
            </div>
            <span class="text-sm font-bold">Whatomate</span>
          </div>
          <div class="flex items-center gap-6 text-xs text-muted-foreground">
            <span>{{ $t("landing.footerPrivacy") }}</span>
            <span class="text-border">|</span>
            <span>{{ $t("landing.footerTerms") }}</span>
            <span class="text-border">|</span>
            <span>{{ $t("landing.footer") }}</span>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>
