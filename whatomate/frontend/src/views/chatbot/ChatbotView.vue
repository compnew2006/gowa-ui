<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { chatbotService } from "@/services/api";
import { toast } from "vue-sonner";
import { PageHeader } from "@/components/shared";
import { getErrorMessage } from "@/lib/api-utils";
import {
  Bot,
  Key,
  Sparkles,
  Power,
  Settings,
  TrendingUp,
  Users,
  MessageSquare,
  Clock,
} from "lucide-vue-next";

const { t } = useI18n();

interface ChatbotSettings {
  enabled: boolean;
  greeting_message: string;
  fallback_message: string;
  session_timeout_minutes: number;
  ai_enabled: boolean;
  ai_provider: string;
}

interface Stats {
  total_sessions: number;
  active_sessions: number;
  messages_handled: number;
  ai_responses: number;
  agent_transfers: number;
  keywords_count: number;
  ai_contexts_count: number;
}

const settings = ref<ChatbotSettings>({
  enabled: false,
  greeting_message: "",
  fallback_message: "",
  session_timeout_minutes: 30,
  ai_enabled: false,
  ai_provider: "",
});

const stats = ref<Stats>({
  total_sessions: 0,
  active_sessions: 0,
  messages_handled: 0,
  ai_responses: 0,
  agent_transfers: 0,
  keywords_count: 0,
  ai_contexts_count: 0,
});

const isLoading = ref(true);
const isToggling = ref(false);

onMounted(async () => {
  try {
    const response = await chatbotService.getSettings();
    // API response is wrapped in { status: "success", data: { settings: {...}, stats: {...} } }
    const data = response.data.data || response.data;
    settings.value = data.settings || settings.value;
    stats.value = data.stats || stats.value;
  } catch (error) {
    console.error("Failed to load chatbot settings:", error);
    // Keep default values on error
  } finally {
    isLoading.value = false;
  }
});

async function toggleChatbot() {
  isToggling.value = true;
  try {
    const newState = !settings.value.enabled;
    await chatbotService.updateSettings({ enabled: newState });
    settings.value.enabled = newState;
    toast.success(
      newState
        ? t("common.enabledSuccess", { resource: t("resources.Chatbot") })
        : t("common.disabledSuccess", { resource: t("resources.Chatbot") }),
    );
  } catch (error: any) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedToggle", { resource: t("resources.chatbot") }),
      ),
    );
  } finally {
    isToggling.value = false;
  }
}

const statCards = computed(() => [
  {
    title: t("chatbot.totalSessions"),
    key: "total_sessions",
    icon: Users,
    color: "text-blue-500",
  },
  {
    title: t("chatbot.activeSessions"),
    key: "active_sessions",
    icon: MessageSquare,
    color: "text-green-500",
  },
  {
    title: t("chatbot.messagesHandled"),
    key: "messages_handled",
    icon: TrendingUp,
    color: "text-purple-500",
  },
  {
    title: t("chatbot.aiResponses"),
    key: "ai_responses",
    icon: Sparkles,
    color: "text-orange-500",
  },
]);
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('chatbot.title')"
      :description="$t('chatbot.subtitle')"
      :icon="Bot"
      icon-gradient="bg-gradient-to-br from-purple-500 to-pink-600 shadow-purple-500/20"
    >
      <template #actions>
        <div class="flex items-center gap-3">
          <Badge
            :class="
              settings.enabled
                ? 'bg-primary/12 text-primary'
                : 'bg-muted text-muted-foreground'
            "
          >
            {{
              settings.enabled ? $t("chatbot.active") : $t("chatbot.inactive")
            }}
          </Badge>
          <Button
            variant="outline"
            size="sm"
            @click="toggleChatbot"
            :disabled="isToggling"
            :class="
              settings.enabled
                ? 'border-red-500/50 text-red-400 hover:bg-red-500/10'
                : 'border-emerald-500/50 text-emerald-400 hover:bg-emerald-500/10'
            "
          >
            <Power class="h-4 w-4 mr-2" />
            {{
              settings.enabled ? $t("chatbot.disable") : $t("chatbot.enable")
            }}
          </Button>
        </div>
      </template>
    </PageHeader>

    <!-- Content -->
    <ScrollArea class="flex-1">
      <div class="p-6 space-y-6">
        <!-- Stats -->
        <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <!-- Skeleton Loading State -->
          <template v-if="isLoading">
            <div
              v-for="i in 4"
              :key="i"
              class="rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 p-6 shadow-sm"
            >
              <div
                class="flex flex-row items-center justify-between space-y-0 pb-2"
              >
                <Skeleton class="h-4 w-24 bg-accent/80" />
                <Skeleton class="h-10 w-10 rounded-lg bg-accent/80" />
              </div>
              <div class="pt-2">
                <Skeleton class="h-8 w-16 bg-accent/80" />
              </div>
            </div>
          </template>
          <!-- Actual Stats -->
          <template v-else>
            <div
              v-for="card in statCards"
              :key="card.key"
              class="card-depth rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 p-6 shadow-sm"
            >
              <div
                class="flex flex-row items-center justify-between space-y-0 pb-2"
              >
                <span class="text-sm font-medium text-muted-foreground">{{
                  card.title
                }}</span>
                <div
                  :class="[
                    'h-10 w-10 rounded-lg flex items-center justify-center',
                    card.key === 'total_sessions' ? 'bg-blue-500/20' : '',
                    card.key === 'active_sessions' ? 'bg-emerald-500/20' : '',
                    card.key === 'messages_handled' ? 'bg-purple-500/20' : '',
                    card.key === 'ai_responses' ? 'bg-orange-500/20' : '',
                  ]"
                >
                  <component
                    :is="card.icon"
                    :class="[
                      'h-5 w-5',
                      card.key === 'total_sessions' ? 'text-blue-400' : '',
                      card.key === 'active_sessions' ? 'text-emerald-400' : '',
                      card.key === 'messages_handled' ? 'text-purple-400' : '',
                      card.key === 'ai_responses' ? 'text-orange-400' : '',
                    ]"
                  />
                </div>
              </div>
              <div class="pt-2">
                <div class="text-3xl font-bold text-foreground">
                  {{ stats[card.key as keyof Stats].toLocaleString() }}
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- Quick Actions -->
        <div class="grid gap-4 md:grid-cols-3">
          <RouterLink
            to="/chatbot/keywords"
            class="card-interactive h-full rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 shadow-sm"
          >
            <div class="p-6">
              <div class="flex items-center gap-3">
                <div
                  class="h-10 w-10 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center shadow-lg shadow-blue-500/20"
                >
                  <Key class="h-5 w-5 text-white" />
                </div>
                <div>
                  <h3 class="text-lg font-semibold text-foreground">
                    {{ $t("chatbot.keywordRules") }}
                  </h3>
                  <p class="text-sm text-muted-foreground">
                    {{
                      $t("chatbot.rulesConfigured", {
                        count: stats.keywords_count,
                      })
                    }}
                  </p>
                </div>
              </div>
            </div>
            <div class="px-6 pb-6">
              <p class="text-sm text-muted-foreground">
                {{ $t("chatbot.keywordRulesDesc") }}
              </p>
            </div>
          </RouterLink>

          <RouterLink
            to="/chatbot/ai"
            class="card-interactive h-full rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 shadow-sm"
          >
            <div class="p-6">
              <div class="flex items-center gap-3">
                <div
                  class="h-10 w-10 rounded-lg bg-gradient-to-br from-orange-500 to-amber-600 flex items-center justify-center shadow-lg shadow-orange-500/20"
                >
                  <Sparkles class="h-5 w-5 text-white" />
                </div>
                <div>
                  <h3 class="text-lg font-semibold text-foreground">
                    {{ $t("chatbot.aiContexts") }}
                  </h3>
                  <p class="text-sm text-muted-foreground">
                    {{
                      $t("chatbot.contextsActive", {
                        count: stats.ai_contexts_count,
                      })
                    }}
                  </p>
                </div>
              </div>
            </div>
            <div class="px-6 pb-6">
              <p class="text-sm text-muted-foreground">
                {{ $t("chatbot.aiContextsDesc") }}
              </p>
            </div>
          </RouterLink>
        </div>

        <!-- Current Settings -->
        <div
          class="rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card/95 shadow-sm"
        >
          <div class="p-6">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-lg font-semibold text-foreground">
                  {{ $t("chatbot.currentConfiguration") }}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {{ $t("chatbot.configOverview") }}
                </p>
              </div>
              <RouterLink to="/settings/chatbot">
                <Button variant="outline" size="sm">
                  <Settings class="h-4 w-4 mr-2" />
                  {{ $t("chatbot.editSettings") }}
                </Button>
              </RouterLink>
            </div>
          </div>
          <div class="px-6 pb-6">
            <div class="grid gap-4 md:grid-cols-2">
              <div class="space-y-2">
                <h4 class="text-sm font-medium text-foreground/80">
                  {{ $t("chatbot.greetingMessage") }}
                </h4>
                <p
                  class="rounded-[calc(var(--radius)-0.1rem)] bg-input p-3 text-sm text-muted-foreground"
                >
                  {{ settings.greeting_message || $t("chatbot.notConfigured") }}
                </p>
              </div>
              <div class="space-y-2">
                <h4 class="text-sm font-medium text-foreground/80">
                  {{ $t("chatbot.fallbackMessage") }}
                </h4>
                <p
                  class="rounded-[calc(var(--radius)-0.1rem)] bg-input p-3 text-sm text-muted-foreground"
                >
                  {{ settings.fallback_message || $t("chatbot.notConfigured") }}
                </p>
              </div>
              <div class="space-y-2">
                <h4 class="text-sm font-medium text-foreground/80">
                  {{ $t("chatbot.sessionTimeout") }}
                </h4>
                <div
                  class="flex items-center gap-2 text-sm text-muted-foreground"
                >
                  <Clock class="h-4 w-4" />
                  {{
                    $t("chatbot.minutes", {
                      count: settings.session_timeout_minutes,
                    })
                  }}
                </div>
              </div>
              <div class="space-y-2">
                <h4 class="text-sm font-medium text-foreground/80">
                  {{ $t("chatbot.aiProvider") }}
                </h4>
                <div class="flex items-center gap-2">
                  <Badge
                    v-if="settings.ai_enabled"
                    class="bg-primary/12 text-primary"
                  >
                    {{ settings.ai_provider || $t("chatbot.notConfigured") }}
                  </Badge>
                  <Badge v-else class="bg-muted text-muted-foreground">{{
                    $t("chatbot.disabled")
                  }}</Badge>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>
