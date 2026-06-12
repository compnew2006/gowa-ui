<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, useTemplateRef } from "vue";
import { watchDebounced } from "@vueuse/core";
import { PageHeader } from "@/components/shared";
import { SearchInput } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState } from "@/components/ui/empty-state";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Inbox,
  ExternalLink,
  Loader2,
  MessageSquareReply,
  RefreshCw,
  Send,
  Settings2,
  Sparkles,
  User,
  X,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { useI18n } from "vue-i18n";
import { facebookCommentsService } from "@/services/api";
import { unwrapItemResponse, unwrapListResponse, unwrapResponse } from "@/lib/api-utils";
import {
  wsService,
  WS_TYPE_FACEBOOK_COMMENT_CREATED,
  WS_TYPE_FACEBOOK_COMMENT_UPDATED,
} from "@/services/websocket";
import {
  applyCommentCreated,
  applyCommentUpdated,
  isReplySkipped,
} from "./facebookCommentsMerge";
import type {
  FacebookComment,
  FacebookCommentsListResponse,
  FacebookCommentSettings,
  FacebookCommentStatus,
} from "@/types/facebookComments";
import { cn, getAvatarGradient, getInitials, formatDateTime } from "@/lib/utils";

const { t } = useI18n();

const comments = ref<FacebookComment[]>([]);
const selectedCommentId = ref<string>("");
const settings = ref<FacebookCommentSettings | null>(null);
const total = ref(0);
const page = ref(1);
const limit = ref(100);
const search = ref("");
const pageIdFilter = ref("all");
const status = ref<FacebookCommentStatus | "all">("open");
const loading = ref(false);
const syncing = ref(false);
const savingSettings = ref(false);
const sendingReply = ref(false);
const settingsOpen = ref(false);
const syncOpen = ref(false);
const replyText = ref("");
const privateMessageText = ref("");
const sendCommentReply = ref(true);
const sendPrivateMessage = ref(false);
const syncPostLimit = ref(25);
const syncCommentsPerPost = ref(50);
const syncRunAutoReply = ref(true);

const inboxListRef = useTemplateRef<HTMLElement>("inboxListRef");
let activeController: AbortController | null = null;

const selectedComment = computed(
  () =>
    comments.value.find((comment) => comment.id === selectedCommentId.value) ||
    comments.value[0] ||
    null,
);

const getCommentLink = (comment: FacebookComment | null) => {
  if (!comment) return "";
  const url = comment.permalink || comment.post_permalink;
  if (!url) return "";
  const parts = comment.external_id ? comment.external_id.split("_") : [];
  const commentId = parts.length > 0 ? parts[parts.length - 1] : "";
  if (commentId) {
    try {
      const parsedUrl = new URL(url);
      if (!parsedUrl.searchParams.has("comment_id")) {
        parsedUrl.searchParams.set("comment_id", commentId);
      }
      return parsedUrl.toString();
    } catch (e) {
      if (url.includes("?")) {
        if (!url.includes("comment_id=")) {
          return `${url}&comment_id=${commentId}`;
        }
      } else {
        return `${url}?comment_id=${commentId}`;
      }
    }
  }
  return url;
};

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / limit.value)));

const statusCounts = computed(() => {
  const counts: Record<string, number> = { open: 0, replied: 0, closed: 0, archived: 0 };
  for (const comment of comments.value) {
    counts[comment.status] = (counts[comment.status] || 0) + 1;
  }
  return counts;
});

const statusChips = computed(() =>
  (["open", "replied", "closed", "archived"] as FacebookCommentStatus[]).map((value) => ({
    value,
    label: t(`facebookComments.status.${value}`),
    count: statusCounts.value[value] || 0,
  })),
);

const allStatusChips = computed(() => [
  { value: "all" as const, label: t("common.all"), count: total.value },
  ...statusChips.value,
]);

const availablePages = computed(() => {
  const seen = new Map<string, string>();
  for (const comment of comments.value) {
    if (comment.page_id && !seen.has(comment.page_id)) {
      seen.set(comment.page_id, comment.page_name || comment.page_id);
    }
  }
  return Array.from(seen.entries()).map(([id, name]) => ({ id, name }));
});

onMounted(async () => {
  await Promise.all([fetchSettings(), fetchComments()]);
  wsService.subscribe(WS_TYPE_FACEBOOK_COMMENT_CREATED, handleCommentCreated);
  wsService.subscribe(WS_TYPE_FACEBOOK_COMMENT_UPDATED, handleCommentUpdated);
});

onUnmounted(() => {
  wsService.unsubscribe(WS_TYPE_FACEBOOK_COMMENT_CREATED, handleCommentCreated);
  wsService.unsubscribe(WS_TYPE_FACEBOOK_COMMENT_UPDATED, handleCommentUpdated);
});

function handleCommentCreated(payload: FacebookComment) {
  const result = applyCommentCreated(comments.value, payload);
  if (!result.appended) return;
  comments.value = result.comments;
  total.value += 1;
  if (!selectedCommentId.value) {
    selectedCommentId.value = payload.id;
  }
  nextTick(() => {
    inboxListRef.value
      ?.querySelector<HTMLElement>(`[data-comment-id="${payload.id}"]`)
      ?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  });
}

function handleCommentUpdated(payload: FacebookComment) {
  const result = applyCommentUpdated(comments.value, payload);
  comments.value = result.comments;
}

async function fetchSettings() {
  try {
    const response = await facebookCommentsService.getSettings();
    settings.value = unwrapItemResponse<FacebookCommentSettings>(response, "settings");
    syncPostLimit.value = settings.value.default_sync_post_limit || 25;
    syncCommentsPerPost.value = settings.value.default_sync_comments_per_post || 50;
    syncRunAutoReply.value = settings.value.auto_reply_enabled;
    if (!replyText.value) replyText.value = settings.value.auto_comment_reply_text;
    if (!privateMessageText.value)
      privateMessageText.value = settings.value.auto_private_message_text;
  } catch (error: any) {
    toast.error(error.response?.data?.message || t("facebookComments.toast.settingsFailed"));
  }
}

async function fetchComments() {
  if (activeController) {
    activeController.abort();
  }
  const controller = new AbortController();
  activeController = controller;
  loading.value = true;
  try {
    const response = await facebookCommentsService.list(
      {
        page: page.value,
        limit: limit.value,
        status: status.value,
        search: search.value || undefined,
        page_id: pageIdFilter.value !== "all" ? pageIdFilter.value : undefined,
      },
      { signal: controller.signal },
    );
    if (activeController !== controller) return;
    const payload = unwrapResponse<FacebookCommentsListResponse>(response);
    comments.value = unwrapListResponse<FacebookComment>(response, "comments");
    total.value = payload.total || comments.value.length;
    if (!selectedCommentId.value && comments.value.length > 0) {
      selectedCommentId.value = comments.value[0].id;
    }
    if (
      selectedCommentId.value &&
      !comments.value.some((comment) => comment.id === selectedCommentId.value)
    ) {
      selectedCommentId.value = comments.value[0]?.id || "";
    }
  } catch (error: any) {
    if (controller.signal.aborted) return;
    if (activeController !== controller) return;
    toast.error(error.response?.data?.message || t("facebookComments.toast.fetchFailed"));
  } finally {
    if (activeController === controller) {
      loading.value = false;
      activeController = null;
    }
  }
}

async function resetAndFetchComments() {
  page.value = 1;
  await fetchComments();
}

function setStatusFilter(next: FacebookCommentStatus | "all") {
  if (status.value === next) return;
  status.value = next;
  void resetAndFetchComments();
}

async function goToCommentsPage(nextPage: number) {
  const safePage = Math.min(Math.max(1, nextPage), totalPages.value);
  if (safePage === page.value) return;
  page.value = safePage;
  await fetchComments();
}

async function syncComments() {
  syncing.value = true;
  try {
    const response = await facebookCommentsService.sync({
      post_limit: syncPostLimit.value,
      comments_per_post: syncCommentsPerPost.value,
      run_auto_reply: syncRunAutoReply.value,
    });
    const result = unwrapResponse<{
      synced: number;
      created: number;
      auto_replies: number;
      failures: string[];
    }>(response);
    toast.success(
      t("facebookComments.toast.synced", {
        synced: result.synced || 0,
        created: result.created || 0,
        auto: result.auto_replies || 0,
      }),
    );
    if (result.failures?.length) {
      toast.warning(result.failures.slice(0, 2).join(" | "));
    }
    syncOpen.value = false;
    await fetchComments();
  } catch (error: any) {
    toast.error(error.response?.data?.message || t("facebookComments.toast.syncFailed"));
  } finally {
    syncing.value = false;
  }
}

async function saveSettings() {
  if (!settings.value) return;
  savingSettings.value = true;
  try {
    const response = await facebookCommentsService.updateSettings(
      settings.value as unknown as Record<string, unknown>,
    );
    settings.value = unwrapItemResponse<FacebookCommentSettings>(response, "settings");
    toast.success(t("facebookComments.toast.settingsSaved"));
    settingsOpen.value = false;
  } catch (error: any) {
    toast.error(error.response?.data?.message || t("facebookComments.toast.settingsSaveFailed"));
  } finally {
    savingSettings.value = false;
  }
}

async function sendReply() {
  const comment = selectedComment.value;
  if (!comment) return;
  sendingReply.value = true;
  try {
    await facebookCommentsService.reply(comment.id, {
      reply_text: replyText.value,
      private_message_text: privateMessageText.value,
      send_comment_reply: sendCommentReply.value,
      send_private_message: sendPrivateMessage.value,
    });
    toast.success(t("facebookComments.toast.replySent"));
  } catch (error: any) {
    toast.error(error.response?.data?.message || t("facebookComments.toast.replyFailed"));
  } finally {
    sendingReply.value = false;
  }
}

async function updateStatus(nextStatus: FacebookCommentStatus) {
  const comment = selectedComment.value;
  if (!comment || comment.status === nextStatus) return;
  const previous = comment.status;
  const previousLabel = t(`facebookComments.status.${previous}`);
  comment.status = nextStatus;
  try {
    await facebookCommentsService.updateStatus(comment.id, nextStatus);
    toast.success(t("facebookComments.toast.statusUpdated"), {
      description: t("facebookComments.toast.statusUpdatedDesc", {
        from: previousLabel,
        to: t(`facebookComments.status.${nextStatus}`),
      }),
      action: {
        label: t("common.undo"),
        onClick: () => {
          void revertStatus(comment.id, previous);
        },
      },
      duration: 6000,
    });
  } catch (error: any) {
    comment.status = previous;
    toast.error(error.response?.data?.message || t("facebookComments.toast.statusFailed"));
  }
}

async function revertStatus(commentId: string, previousStatus: FacebookCommentStatus) {
  const comment = comments.value.find((c) => c.id === commentId);
  if (!comment || comment.status === previousStatus) return;
  const optimistic = comment.status;
  comment.status = previousStatus;
  try {
    await facebookCommentsService.updateStatus(commentId, previousStatus);
    toast.success(t("facebookComments.toast.statusReverted"));
  } catch (error: any) {
    comment.status = optimistic;
    toast.error(error.response?.data?.message || t("facebookComments.toast.statusFailed"));
  }
}

function selectComment(comment: FacebookComment) {
  selectedCommentId.value = comment.id;
  if (settings.value) {
    replyText.value = settings.value.auto_comment_reply_text;
    privateMessageText.value = settings.value.auto_private_message_text;
  }
}

function statusVariant(commentStatus: string): "default" | "secondary" | "outline" | "destructive" | "success" | "warning" | "info" {
  switch (commentStatus) {
    case "open":
      return "info";
    case "replied":
      return "success";
    case "closed":
      return "secondary";
    case "archived":
      return "outline";
    default:
      return "destructive";
  }
}

function replyStatusVariant(replyStatus: string): "default" | "secondary" | "outline" | "destructive" | "success" | "warning" {
  switch (replyStatus) {
    case "sent":
      return "success";
    case "partial":
      return "warning";
    case "failed":
      return "destructive";
    case "skipped":
      return "secondary";
    default:
      return "outline";
  }
}

function getFallbackName(comment: FacebookComment | null): string {
  if (!comment) return t("facebookComments.unknownUser");
  if (comment.from_name) {
    return comment.from_name;
  }
  const idParts = (comment.external_id || "").split("_");
  const commentSuffix = idParts.pop() || "";
  if (commentSuffix) {
    const len = commentSuffix.length;
    const suffix = len > 6 ? commentSuffix.substring(len - 6) : commentSuffix;
    return `${t("facebookComments.unknownUser")} (${suffix})`;
  }
  return t("facebookComments.unknownUser");
}

function avatarGradient(name: string) {
  return `bg-gradient-to-br ${getAvatarGradient(name)} text-white`;
}

watchDebounced(
  search,
  () => {
    page.value = 1;
    void fetchComments();
  },
  { debounce: 350 },
);
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <PageHeader
      :title="$t('facebookComments.title')"
      :subtitle="$t('facebookComments.subtitle')"
      :breadcrumbs="[
        { label: $t('nav.facebookTools'), href: '/facebook' },
        { label: $t('facebookComments.title') },
      ]"
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="settingsOpen = true">
          <Settings2 class="mr-2 h-4 w-4" />
          <span class="hidden sm:inline">{{ $t("facebookComments.settings") }}</span>
        </Button>
        <Button size="sm" :disabled="syncing" @click="syncOpen = true">
          <Loader2 v-if="syncing" class="mr-2 h-4 w-4 animate-spin" />
          <RefreshCw v-else class="mr-2 h-4 w-4" />
          <span class="hidden sm:inline">{{ $t("facebookComments.sync") }}</span>
        </Button>
      </template>
    </PageHeader>

    <div
      class="grid min-h-0 flex-1 gap-4 p-4 lg:grid-cols-[360px_minmax(0,1fr)_320px] lg:grid-rows-[minmax(0,1fr)]"
    >
      <!-- Inbox column -->
      <Card class="flex h-full min-h-0 flex-col overflow-hidden">
        <CardHeader class="space-y-3 border-b py-3">
          <div class="flex items-center justify-between">
            <CardTitle class="text-base font-medium">
              {{ $t("facebookComments.inbox") }}
            </CardTitle>
            <Badge variant="outline" class="tabular-nums">{{ total }}</Badge>
          </div>

          <div
            class="-mx-1 flex items-center gap-1 overflow-x-auto px-1"
            role="tablist"
            :aria-label="$t('facebookComments.inbox')"
          >
            <button
              v-for="chip in allStatusChips"
              :key="chip.value"
              type="button"
              role="tab"
              :aria-selected="status === chip.value"
              :class="
                cn(
                  'inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium transition',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background',
                  status === chip.value
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground',
                )
              "
              @click="setStatusFilter(chip.value)"
            >
              <span>{{ chip.label }}</span>
              <span
                :class="
                  cn(
                    'rounded-full px-1.5 text-[10px] tabular-nums',
                    status === chip.value
                      ? 'bg-primary-foreground/20 text-primary-foreground'
                      : 'bg-background/60 text-muted-foreground',
                  )
                "
              >
                {{ chip.count }}
              </span>
            </button>
          </div>

          <div class="flex items-center gap-2">
            <SearchInput
              v-model="search"
              :placeholder="$t('facebookComments.search')"
              class="flex-1"
            />
            <Select v-model="pageIdFilter" @update:model-value="resetAndFetchComments()">
              <SelectTrigger class="w-[180px]">
                <SelectValue :placeholder="$t('facebookComments.allPages')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {{ $t("facebookComments.allPages") }}
                </SelectItem>
                <SelectItem
                  v-for="page in availablePages"
                  :key="page.id"
                  :value="page.id"
                >
                  {{ page.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>

        <CardContent class="flex min-h-0 flex-1 flex-col p-0">
          <ScrollArea class="min-h-0 flex-1">
            <div
              ref="inboxListRef"
              class="space-y-2 p-2"
              role="list"
              :aria-busy="loading"
              :aria-label="$t('facebookComments.inbox')"
            >
              <template v-if="loading && comments.length === 0">
                <div
                  v-for="i in 5"
                  :key="i"
                  class="rounded-lg border border-border p-3"
                >
                  <div class="mb-2 flex items-center gap-2">
                    <div class="h-7 w-7 animate-pulse rounded-full bg-muted" />
                    <div class="h-3 w-24 animate-pulse rounded bg-muted" />
                  </div>
                  <div class="h-3 w-full animate-pulse rounded bg-muted" />
                  <div class="mt-1.5 h-3 w-2/3 animate-pulse rounded bg-muted" />
                </div>
              </template>

              <button
                v-for="comment in comments"
                :key="comment.id"
                :data-comment-id="comment.id"
                type="button"
                role="listitem"
                :aria-current="comment.id === selectedComment?.id ? 'true' : undefined"
                :class="
                  cn(
                    'group w-full rounded-lg border p-3 text-left transition',
                    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background',
                    comment.id === selectedComment?.id
                      ? 'border-primary/60 bg-primary/5'
                      : 'border-border card-interactive hover:border-border/80',
                  )
                "
                @click="selectComment(comment)"
              >
                <div class="mb-2 flex items-center justify-between gap-2">
                  <div class="flex min-w-0 items-center gap-2">
                    <span
                      :class="
                        cn(
                          'flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold',
                          avatarGradient(getFallbackName(comment)),
                        )
                      "
                      aria-hidden="true"
                    >
                      {{ getInitials(getFallbackName(comment)) }}
                    </span>
                    <span class="truncate text-sm font-medium">
                      {{ getFallbackName(comment) }}
                    </span>
                    <Badge
                      v-if="comment.is_admin_reply"
                      variant="secondary"
                      class="shrink-0 text-[10px] uppercase tracking-wide"
                    >
                      {{ $t("facebookComments.adminReply") }}
                    </Badge>
                  </div>
                  <Badge
                    :variant="statusVariant(comment.status)"
                    class="shrink-0 text-[10px] uppercase tracking-wide"
                  >
                    {{ $t(`facebookComments.status.${comment.status}`) }}
                  </Badge>
                </div>
                <p class="line-clamp-2 text-sm text-foreground/80">
                  {{ comment.message }}
                </p>
                <div class="mt-2 flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
                  <span class="truncate">{{ comment.page_name || comment.page_id }}</span>
                  <span class="shrink-0 tabular-nums">
                    {{ formatDateTime(comment.commented_at) }}
                  </span>
                </div>
              </button>

              <EmptyState
                v-if="!loading && comments.length === 0"
                :icon="Inbox"
                :title="$t('facebookComments.empty')"
                size="compact"
                variant="muted"
                animated
              />
            </div>
          </ScrollArea>

          <div
            v-if="total > limit"
            class="flex items-center justify-between border-t px-3 py-2 text-xs text-muted-foreground"
          >
            <Button
              variant="ghost"
              size="sm"
              :disabled="page <= 1 || loading"
              @click="goToCommentsPage(page - 1)"
            >
              {{ $t("common.previous") }}
            </Button>
            <span class="tabular-nums">{{ page }} / {{ totalPages }}</span>
            <Button
              variant="ghost"
              size="sm"
              :disabled="page >= totalPages || loading"
              @click="goToCommentsPage(page + 1)"
            >
              {{ $t("common.next") }}
            </Button>
          </div>
        </CardContent>
      </Card>

      <!-- Thread column -->
      <Card class="flex h-full min-h-0 flex-col overflow-hidden">
        <CardContent v-if="selectedComment" class="flex min-h-0 flex-1 flex-col p-0">
          <div class="border-b p-4">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <Badge variant="outline" class="max-w-[12rem] truncate">
                {{ selectedComment.page_name || selectedComment.page_id }}
              </Badge>
              <Badge :variant="statusVariant(selectedComment.status)">
                {{ $t(`facebookComments.status.${selectedComment.status}`) }}
              </Badge>
              <Button
                v-if="selectedComment.permalink || selectedComment.post_permalink"
                variant="link"
                size="sm"
                as="a"
                :href="getCommentLink(selectedComment)"
                target="_blank"
                rel="noopener noreferrer"
                class="h-auto px-1 py-0 text-xs"
              >
                <ExternalLink class="mr-1 h-3 w-3" />
                {{ $t("facebookComments.openOnFacebook") }}
              </Button>
            </div>
            <div class="flex items-start gap-3">
              <span
                :class="
                  cn(
                    'flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm font-semibold',
                    avatarGradient(getFallbackName(selectedComment)),
                  )
                "
                aria-hidden="true"
              >
                {{ getInitials(getFallbackName(selectedComment)) }}
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <h2 class="truncate text-lg font-medium">
                    {{ getFallbackName(selectedComment) }}
                  </h2>
                  <Badge
                    v-if="selectedComment.is_admin_reply"
                    variant="secondary"
                    class="shrink-0 text-[10px] uppercase tracking-wide"
                  >
                    {{ $t("facebookComments.adminReply") }}
                  </Badge>
                </div>
                <p class="truncate text-xs text-muted-foreground">
                  {{ $t("facebookComments.fromPost") }}:
                  {{ selectedComment.post_message || selectedComment.post_id }}
                </p>
              </div>
            </div>
          </div>

          <ScrollArea class="min-h-0 flex-1">
            <div class="space-y-4 p-4">
              <div
                :class="
                  cn(
                    'chat-bubble chat-bubble-incoming flex flex-col gap-1 p-3 message-in-left',
                  )
                "
              >
                <div class="flex items-center gap-2 text-xs text-muted-foreground">
                  <User class="h-3 w-3" />
                  <span class="font-medium">
                    {{ getFallbackName(selectedComment) }}
                  </span>
                  <span class="chat-bubble-time">
                    {{ formatDateTime(selectedComment.commented_at) }}
                  </span>
                </div>
                <p class="whitespace-pre-wrap text-sm leading-relaxed">
                  {{ selectedComment.message }}
                </p>
              </div>

              <div
                v-for="reply in selectedComment.replies"
                :key="reply.id"
                :class="
                  cn(
                    'chat-bubble chat-bubble-outgoing flex flex-col gap-1.5 p-3',
                    reply.status === 'failed' && 'ring-1 ring-destructive/30',
                  )
                "
              >
                <div class="flex items-center justify-between gap-2 text-xs text-primary-foreground/80">
                  <span class="inline-flex items-center gap-1 font-medium">
                    <MessageSquareReply class="h-3 w-3" />
                    {{ reply.is_auto ? $t("facebookComments.autoReply") : $t("facebookComments.agentReply") }}
                  </span>
                  <Badge
                    :variant="replyStatusVariant(reply.status)"
                    class="text-[10px] uppercase tracking-wide"
                  >
                    {{ $t(`facebookComments.replyStatus.${reply.status}`, reply.status) }}
                  </Badge>
                </div>
                <p v-if="reply.reply_text" class="whitespace-pre-wrap text-sm leading-relaxed text-primary-foreground">
                  {{ reply.reply_text }}
                </p>
                <p
                  v-if="reply.private_message_text"
                  class="mt-1 whitespace-pre-wrap text-sm text-primary-foreground/80"
                >
                  {{ $t("facebookComments.privateMessage") }}: {{ reply.private_message_text }}
                </p>
                <p
                  v-if="reply.error_message"
                  class="mt-1 rounded-md bg-destructive/10 px-2 py-1 text-xs text-destructive"
                >
                  {{ reply.error_message }}
                </p>
                <p
                  v-else-if="isReplySkipped(reply)"
                  class="text-xs italic text-primary-foreground/70"
                >
                  {{ $t("facebookComments.dmNotAvailable") }}
                </p>
              </div>

              <EmptyState
                v-if="!selectedComment.replies?.length"
                :icon="MessageSquareReply"
                :title="$t('facebookComments.selectComment')"
                size="compact"
                variant="muted"
              />
            </div>
          </ScrollArea>

          <div class="space-y-4 border-t p-4">
            <div class="grid gap-3 md:grid-cols-2">
              <label
                :class="
                  cn(
                    'flex items-center gap-3 rounded-lg border p-3 text-sm transition',
                    sendCommentReply ? 'border-primary/40 bg-primary/5' : 'border-border',
                  )
                "
              >
                <Switch
                  v-model:checked="sendCommentReply"
                  :aria-label="$t('facebookComments.sendPublicReply')"
                />
                <span class="font-medium">{{ $t("facebookComments.sendPublicReply") }}</span>
              </label>
              <label
                :class="
                  cn(
                    'flex items-center gap-3 rounded-lg border p-3 text-sm transition',
                    sendPrivateMessage ? 'border-primary/40 bg-primary/5' : 'border-border',
                  )
                "
              >
                <Switch
                  v-model:checked="sendPrivateMessage"
                  :aria-label="$t('facebookComments.sendPrivateMessage')"
                />
                <span class="font-medium">{{ $t("facebookComments.sendPrivateMessage") }}</span>
              </label>
            </div>
            <div class="grid gap-3 md:grid-cols-2">
              <Textarea
                v-model="replyText"
                :disabled="!sendCommentReply"
                :placeholder="$t('facebookComments.replyPlaceholder')"
                class="min-h-[88px] resize-none"
                :aria-label="$t('facebookComments.sendPublicReply')"
              />
              <Textarea
                v-model="privateMessageText"
                :disabled="!sendPrivateMessage"
                :placeholder="$t('facebookComments.privatePlaceholder')"
                class="min-h-[88px] resize-none"
                :aria-label="$t('facebookComments.sendPrivateMessage')"
              />
            </div>
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="flex flex-wrap items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  :disabled="selectedComment.status === 'closed'"
                  @click="updateStatus('closed')"
                >
                  <X class="mr-1.5 h-3.5 w-3.5" />
                  {{ $t("facebookComments.close") }}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  :disabled="selectedComment.status === 'open'"
                  @click="updateStatus('open')"
                >
                  <RefreshCw class="mr-1.5 h-3.5 w-3.5" />
                  {{ $t("facebookComments.reopen") }}
                </Button>
                <Select
                  :model-value="selectedComment.status"
                  @update:model-value="(v: any) => v && updateStatus(v as FacebookCommentStatus)"
                >
                  <SelectTrigger class="h-9 w-[140px]" :aria-label="$t('common.status')">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="open">{{ $t("facebookComments.status.open") }}</SelectItem>
                    <SelectItem value="replied">{{ $t("facebookComments.status.replied") }}</SelectItem>
                    <SelectItem value="closed">{{ $t("facebookComments.status.closed") }}</SelectItem>
                    <SelectItem value="archived">{{ $t("facebookComments.status.archived") }}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button size="sm" :disabled="sendingReply" @click="sendReply">
                <Loader2 v-if="sendingReply" class="mr-2 h-4 w-4 animate-spin" />
                <Send v-else class="mr-2 h-4 w-4" />
                {{ $t("facebookComments.sendReply") }}
              </Button>
            </div>
          </div>
        </CardContent>

        <CardContent
          v-else
          class="flex min-h-0 flex-1 items-center justify-center"
        >
          <EmptyState
            :icon="MessageSquareReply"
            :title="$t('facebookComments.selectComment')"
            :description="$t('facebookComments.subtitle')"
            size="compact"
            variant="muted"
            animated
          />
        </CardContent>
      </Card>

      <!-- Overview column -->
      <Card class="flex h-full min-h-0 flex-col overflow-hidden">
        <CardHeader class="border-b py-3">
          <CardTitle class="text-base font-medium">
            {{ $t("facebookComments.overview") }}
          </CardTitle>
        </CardHeader>
        <CardContent class="flex-1 overflow-y-auto space-y-3 p-4 text-sm">
          <div class="card-depth grid grid-cols-2 gap-2.5 rounded-lg border border-border p-2.5">
            <button
              v-for="chip in statusChips"
              :key="chip.value"
              type="button"
              :aria-label="`${chip.label}: ${chip.count}`"
              :class="
                cn(
                  'group flex flex-col items-start gap-1 rounded-md px-2.5 py-2 text-left transition',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background',
                  status === chip.value ? 'bg-primary/5' : 'hover:bg-muted/60',
                )
              "
              @click="setStatusFilter(chip.value)"
            >
              <span class="text-[11px] text-muted-foreground">{{ chip.label }}</span>
              <span class="flex items-baseline gap-1">
                <span class="text-lg font-semibold tabular-nums leading-none">{{ chip.count }}</span>
              </span>
            </button>
          </div>

          <div
            :class="
              cn(
                'flex items-center justify-between rounded-lg border p-3',
                settings?.auto_reply_enabled
                  ? 'border-primary/30 bg-primary/5'
                  : 'border-border bg-muted/30',
              )
            "
          >
            <div class="flex items-center gap-2">
              <Sparkles
                :class="
                  cn(
                    'h-4 w-4',
                    settings?.auto_reply_enabled ? 'text-primary' : 'text-muted-foreground',
                  )
                "
              />
              <span class="font-medium">{{ $t("facebookComments.autoReply") }}</span>
            </div>
            <Badge :variant="settings?.auto_reply_enabled ? 'success' : 'outline'">
              {{ settings?.auto_reply_enabled ? $t("common.enabled") : $t("common.disabled") }}
            </Badge>
          </div>

          <div
            class="rounded-lg border border-dashed border-border bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground"
            role="note"
          >
            {{ $t("facebookComments.productionNote") }}
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Settings dialog -->
    <Dialog :open="settingsOpen" @update:open="settingsOpen = $event">
      <DialogContent v-if="settings" class="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{{ $t("facebookComments.settings") }}</DialogTitle>
          <DialogDescription>{{ $t("facebookComments.settingsDesc") }}</DialogDescription>
        </DialogHeader>
        <div class="grid max-h-[70vh] gap-4 overflow-y-auto py-2">
          <label class="flex items-center justify-between gap-3 rounded-lg border p-3">
            <div>
              <span class="text-sm font-medium">{{ $t("facebookComments.enableFeature") }}</span>
            </div>
            <Switch v-model:checked="settings.enabled" />
          </label>
          <label class="flex items-center justify-between gap-3 rounded-lg border p-3">
            <div>
              <span class="text-sm font-medium">{{ $t("facebookComments.enableAutoReply") }}</span>
            </div>
            <Switch v-model:checked="settings.auto_reply_enabled" />
          </label>
          <div class="grid gap-2">
            <Label :for="'auto-comment'">{{ $t("facebookComments.autoCommentText") }}</Label>
            <Textarea
              id="auto-comment"
              v-model="settings.auto_comment_reply_text"
              class="min-h-[88px] resize-none"
            />
          </div>
          <div class="grid gap-2">
            <Label :for="'auto-private'">{{ $t("facebookComments.autoPrivateText") }}</Label>
            <Textarea
              id="auto-private"
              v-model="settings.auto_private_message_text"
              class="min-h-[88px] resize-none"
            />
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="flex items-center justify-between gap-3 rounded-lg border p-3">
              <span class="text-sm font-medium">{{ $t("facebookComments.autoPublic") }}</span>
              <Switch v-model:checked="settings.auto_comment_reply_enabled" />
            </label>
            <label class="flex items-center justify-between gap-3 rounded-lg border p-3">
              <span class="text-sm font-medium">{{ $t("facebookComments.autoPrivate") }}</span>
              <Switch v-model:checked="settings.auto_private_reply_enabled" />
            </label>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div class="grid gap-2">
              <Label :for="'post-limit'">{{ $t("facebookComments.postLimit") }}</Label>
              <Input
                id="post-limit"
                v-model.number="settings.default_sync_post_limit"
                type="number"
                min="1"
                max="100"
              />
            </div>
            <div class="grid gap-2">
              <Label :for="'comments-per-post'">{{ $t("facebookComments.commentsPerPost") }}</Label>
              <Input
                id="comments-per-post"
                v-model.number="settings.default_sync_comments_per_post"
                type="number"
                min="1"
                max="100"
              />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="settingsOpen = false">{{ $t("common.cancel") }}</Button>
          <Button :disabled="savingSettings" @click="saveSettings">
            <Loader2 v-if="savingSettings" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t("common.save") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Sync dialog -->
    <Dialog :open="syncOpen" @update:open="syncOpen = $event">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t("facebookComments.sync") }}</DialogTitle>
          <DialogDescription>{{ $t("facebookComments.syncDesc") }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-3 md:grid-cols-2">
            <div class="grid gap-2">
              <Label :for="'sync-post-limit'">{{ $t("facebookComments.postLimit") }}</Label>
              <Input
                id="sync-post-limit"
                v-model.number="syncPostLimit"
                type="number"
                min="1"
                max="100"
              />
            </div>
            <div class="grid gap-2">
              <Label :for="'sync-comments-per-post'">
                {{ $t("facebookComments.commentsPerPost") }}
              </Label>
              <Input
                id="sync-comments-per-post"
                v-model.number="syncCommentsPerPost"
                type="number"
                min="1"
                max="100"
              />
            </div>
          </div>
          <label class="flex items-center justify-between gap-3 rounded-lg border p-3">
            <span class="text-sm font-medium">{{ $t("facebookComments.runAutoReply") }}</span>
            <Switch v-model:checked="syncRunAutoReply" />
          </label>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="syncOpen = false">{{ $t("common.cancel") }}</Button>
          <Button :disabled="syncing" @click="syncComments">
            <Loader2 v-if="syncing" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t("facebookComments.syncNow") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
