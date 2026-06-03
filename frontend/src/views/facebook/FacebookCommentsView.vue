<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { PageHeader } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
  ExternalLink,
  Loader2,
  MessageSquareReply,
  RefreshCw,
  Search,
  Send,
  Settings2,
  User,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { useI18n } from "vue-i18n";
import { facebookCommentsService } from "@/services/api";
import { unwrapItemResponse, unwrapListResponse, unwrapResponse } from "@/lib/api-utils";
import type {
  FacebookComment,
  FacebookCommentsListResponse,
  FacebookCommentSettings,
  FacebookCommentStatus,
} from "@/types/facebookComments";

const { t } = useI18n();

const comments = ref<FacebookComment[]>([]);
const selectedCommentId = ref<string>("");
const settings = ref<FacebookCommentSettings | null>(null);
const total = ref(0);
const page = ref(1);
const limit = ref(30);
const search = ref("");
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

const selectedComment = computed(() =>
  comments.value.find((comment) => comment.id === selectedCommentId.value) || comments.value[0] || null,
);

const statusCounts = computed(() => {
  const counts: Record<string, number> = { open: 0, replied: 0, closed: 0, archived: 0 };
  for (const comment of comments.value) {
    counts[comment.status] = (counts[comment.status] || 0) + 1;
  }
  return counts;
});

onMounted(async () => {
  await Promise.all([fetchSettings(), fetchComments()]);
});

async function fetchSettings() {
  try {
    const response = await facebookCommentsService.getSettings();
    settings.value = unwrapItemResponse<FacebookCommentSettings>(response, "settings");
    syncPostLimit.value = settings.value.default_sync_post_limit || 25;
    syncCommentsPerPost.value = settings.value.default_sync_comments_per_post || 50;
    syncRunAutoReply.value = settings.value.auto_reply_enabled;
    if (!replyText.value) replyText.value = settings.value.auto_comment_reply_text;
    if (!privateMessageText.value) privateMessageText.value = settings.value.auto_private_message_text;
  } catch (error: any) {
    toast.error(error.response?.data?.message || t("facebookComments.toast.settingsFailed"));
  }
}

async function fetchComments() {
  loading.value = true;
  try {
    const response = await facebookCommentsService.list({
      page: page.value,
      limit: limit.value,
      status: status.value,
      search: search.value || undefined,
    });
    const payload = unwrapResponse<FacebookCommentsListResponse>(response);
    comments.value = unwrapListResponse<FacebookComment>(response, "comments");
    total.value = payload.total || comments.value.length;
    if (!selectedCommentId.value && comments.value.length > 0) {
      selectedCommentId.value = comments.value[0].id;
    }
    if (selectedCommentId.value && !comments.value.some((comment) => comment.id === selectedCommentId.value)) {
      selectedCommentId.value = comments.value[0]?.id || "";
    }
  } catch (error: any) {
    toast.error(error.response?.data?.message || t("facebookComments.toast.fetchFailed"));
  } finally {
    loading.value = false;
  }
}

async function syncComments() {
  syncing.value = true;
  try {
    const response = await facebookCommentsService.sync({
      post_limit: syncPostLimit.value,
      comments_per_post: syncCommentsPerPost.value,
      run_auto_reply: syncRunAutoReply.value,
    });
    const result = unwrapResponse<{ synced: number; created: number; auto_replies: number; failures: string[] }>(response);
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
    const response = await facebookCommentsService.updateSettings(settings.value as unknown as Record<string, unknown>);
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
    await fetchComments();
  } catch (error: any) {
    toast.error(error.response?.data?.message || t("facebookComments.toast.replyFailed"));
  } finally {
    sendingReply.value = false;
  }
}

async function updateStatus(nextStatus: FacebookCommentStatus) {
  const comment = selectedComment.value;
  if (!comment) return;
  try {
    await facebookCommentsService.updateStatus(comment.id, nextStatus);
    comment.status = nextStatus;
    toast.success(t("facebookComments.toast.statusUpdated"));
  } catch (error: any) {
    toast.error(error.response?.data?.message || t("facebookComments.toast.statusFailed"));
  }
}

function selectComment(comment: FacebookComment) {
  selectedCommentId.value = comment.id;
  replyText.value = settings.value?.auto_comment_reply_text || "تم الرد خاص";
  privateMessageText.value = settings.value?.auto_private_message_text || "اهلا كيف اقدر اساعدك";
}

function statusVariant(commentStatus: string) {
  switch (commentStatus) {
    case "open":
      return "default";
    case "replied":
      return "secondary";
    case "closed":
      return "outline";
    default:
      return "destructive";
  }
}

function formatDate(value?: string) {
  if (!value) return "";
  return new Date(value).toLocaleString();
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <PageHeader
      :title="$t('facebookComments.title')"
      :subtitle="$t('facebookComments.subtitle')"
      :breadcrumbs="[
        { label: $t('nav.facebookTools'), href: '/facebook/page-search' },
        { label: $t('facebookComments.title') },
      ]"
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="settingsOpen = true">
          <Settings2 class="mr-2 h-4 w-4" />
          {{ $t("facebookComments.settings") }}
        </Button>
        <Button size="sm" :disabled="syncing" @click="syncOpen = true">
          <Loader2 v-if="syncing" class="mr-2 h-4 w-4 animate-spin" />
          <RefreshCw v-else class="mr-2 h-4 w-4" />
          {{ $t("facebookComments.sync") }}
        </Button>
      </template>
    </PageHeader>

    <div class="grid min-h-0 flex-1 gap-4 p-4 lg:grid-cols-[360px_minmax(0,1fr)_320px]">
      <Card class="min-h-0">
        <CardHeader class="space-y-3 pb-3">
          <div class="flex items-center justify-between">
            <CardTitle class="text-base">{{ $t("facebookComments.inbox") }}</CardTitle>
            <Badge variant="outline">{{ total }}</Badge>
          </div>
          <div class="flex gap-2">
            <div class="relative min-w-0 flex-1">
              <Search class="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                v-model="search"
                class="pl-8"
                :placeholder="$t('facebookComments.search')"
                @keyup.enter="fetchComments"
              />
            </div>
            <Button variant="outline" size="sm" @click="fetchComments">
              <RefreshCw class="h-4 w-4" />
            </Button>
          </div>
          <Select v-model="status" @update:model-value="fetchComments">
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="open">{{ $t("facebookComments.status.open") }}</SelectItem>
              <SelectItem value="replied">{{ $t("facebookComments.status.replied") }}</SelectItem>
              <SelectItem value="closed">{{ $t("facebookComments.status.closed") }}</SelectItem>
              <SelectItem value="archived">{{ $t("facebookComments.status.archived") }}</SelectItem>
              <SelectItem value="all">{{ $t("common.all") }}</SelectItem>
            </SelectContent>
          </Select>
        </CardHeader>
        <CardContent class="min-h-0 p-0">
          <ScrollArea class="h-[calc(100vh-260px)]">
            <div class="space-y-2 p-3">
              <button
                v-for="comment in comments"
                :key="comment.id"
                class="w-full rounded-md border p-3 text-left transition hover:bg-muted/50"
                :class="comment.id === selectedComment?.id ? 'border-primary bg-muted/60' : 'border-border'"
                @click="selectComment(comment)"
              >
                <div class="mb-2 flex items-center justify-between gap-2">
                  <div class="min-w-0 font-medium">
                    {{ comment.from_name || $t("facebookComments.unknownUser") }}
                  </div>
                  <Badge :variant="statusVariant(comment.status)" class="shrink-0 text-xs">
                    {{ $t(`facebookComments.status.${comment.status}`) }}
                  </Badge>
                </div>
                <p class="line-clamp-2 text-sm text-foreground">{{ comment.message }}</p>
                <div class="mt-2 flex items-center justify-between gap-2 text-xs text-muted-foreground">
                  <span class="truncate">{{ comment.page_name || comment.page_id }}</span>
                  <span class="shrink-0">{{ formatDate(comment.commented_at) }}</span>
                </div>
              </button>
              <div v-if="loading" class="p-6 text-center text-sm text-muted-foreground">
                <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
                {{ $t("common.loading") }}
              </div>
              <div v-else-if="comments.length === 0" class="p-6 text-center text-sm text-muted-foreground">
                {{ $t("facebookComments.empty") }}
              </div>
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      <Card class="min-h-0">
        <CardContent v-if="selectedComment" class="flex h-full min-h-0 flex-col p-0">
          <div class="border-b p-4">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <Badge variant="outline">{{ selectedComment.page_name || selectedComment.page_id }}</Badge>
              <Badge :variant="statusVariant(selectedComment.status)">
                {{ $t(`facebookComments.status.${selectedComment.status}`) }}
              </Badge>
              <a
                v-if="selectedComment.permalink || selectedComment.post_permalink"
                :href="selectedComment.permalink || selectedComment.post_permalink"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1 text-xs text-primary"
              >
                <ExternalLink class="h-3 w-3" />
                {{ $t("facebookComments.openOnFacebook") }}
              </a>
            </div>
            <h2 class="text-lg font-semibold">{{ selectedComment.from_name }}</h2>
            <p class="text-xs text-muted-foreground">
              {{ $t("facebookComments.fromPost") }}:
              {{ selectedComment.post_message || selectedComment.post_id }}
            </p>
          </div>

          <ScrollArea class="min-h-0 flex-1">
            <div class="space-y-4 p-4">
              <div class="max-w-[80%] rounded-md bg-muted p-3">
                <div class="mb-1 flex items-center gap-2 text-xs text-muted-foreground">
                  <User class="h-3 w-3" />
                  {{ selectedComment.from_name }}
                  <span>{{ formatDate(selectedComment.commented_at) }}</span>
                </div>
                <p class="whitespace-pre-wrap text-sm">{{ selectedComment.message }}</p>
              </div>
              <div
                v-for="reply in selectedComment.replies"
                :key="reply.id"
                class="ml-auto max-w-[80%] rounded-md border border-primary/20 bg-primary/10 p-3"
              >
                <div class="mb-1 flex items-center justify-between gap-2 text-xs text-muted-foreground">
                  <span class="inline-flex items-center gap-1">
                    <MessageSquareReply class="h-3 w-3" />
                    {{ reply.is_auto ? $t("facebookComments.autoReply") : $t("facebookComments.agentReply") }}
                  </span>
                  <Badge :variant="reply.status === 'failed' ? 'destructive' : 'outline'" class="text-xs">
                    {{ reply.status }}
                  </Badge>
                </div>
                <p v-if="reply.reply_text" class="whitespace-pre-wrap text-sm">{{ reply.reply_text }}</p>
                <p v-if="reply.private_message_text" class="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">
                  {{ $t("facebookComments.privateMessage") }}: {{ reply.private_message_text }}
                </p>
                <p v-if="reply.error_message" class="mt-2 text-xs text-destructive">{{ reply.error_message }}</p>
              </div>
            </div>
          </ScrollArea>

          <div class="border-t p-4">
            <div class="mb-3 grid gap-3 md:grid-cols-2">
              <label class="flex items-center gap-2 text-sm">
                <Switch v-model:checked="sendCommentReply" />
                {{ $t("facebookComments.sendPublicReply") }}
              </label>
              <label class="flex items-center gap-2 text-sm">
                <Switch v-model:checked="sendPrivateMessage" />
                {{ $t("facebookComments.sendPrivateMessage") }}
              </label>
            </div>
            <div class="grid gap-3 md:grid-cols-2">
              <Textarea
                v-model="replyText"
                :disabled="!sendCommentReply"
                :placeholder="$t('facebookComments.replyPlaceholder')"
                class="min-h-[84px]"
              />
              <Textarea
                v-model="privateMessageText"
                :disabled="!sendPrivateMessage"
                :placeholder="$t('facebookComments.privatePlaceholder')"
                class="min-h-[84px]"
              />
            </div>
            <div class="mt-3 flex justify-between gap-2">
              <div class="flex gap-2">
                <Button variant="outline" size="sm" @click="updateStatus('closed')">
                  {{ $t("facebookComments.close") }}
                </Button>
                <Button variant="outline" size="sm" @click="updateStatus('open')">
                  {{ $t("facebookComments.reopen") }}
                </Button>
              </div>
              <Button size="sm" :disabled="sendingReply" @click="sendReply">
                <Loader2 v-if="sendingReply" class="mr-2 h-4 w-4 animate-spin" />
                <Send v-else class="mr-2 h-4 w-4" />
                {{ $t("facebookComments.sendReply") }}
              </Button>
            </div>
          </div>
        </CardContent>
        <CardContent v-else class="flex h-full items-center justify-center text-sm text-muted-foreground">
          {{ $t("facebookComments.selectComment") }}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-base">{{ $t("facebookComments.overview") }}</CardTitle>
        </CardHeader>
        <CardContent class="space-y-3 text-sm">
          <div class="flex justify-between">
            <span>{{ $t("facebookComments.status.open") }}</span>
            <Badge>{{ statusCounts.open }}</Badge>
          </div>
          <div class="flex justify-between">
            <span>{{ $t("facebookComments.status.replied") }}</span>
            <Badge variant="secondary">{{ statusCounts.replied }}</Badge>
          </div>
          <div class="flex justify-between">
            <span>{{ $t("facebookComments.autoReply") }}</span>
            <Badge :variant="settings?.auto_reply_enabled ? 'default' : 'outline'">
              {{ settings?.auto_reply_enabled ? $t("common.enabled") : $t("common.disabled") }}
            </Badge>
          </div>
          <div class="rounded-md border p-3 text-xs text-muted-foreground">
            {{ $t("facebookComments.productionNote") }}
          </div>
        </CardContent>
      </Card>
    </div>

    <Dialog :open="settingsOpen" @update:open="settingsOpen = $event">
      <DialogContent v-if="settings" class="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{{ $t("facebookComments.settings") }}</DialogTitle>
          <DialogDescription>{{ $t("facebookComments.settingsDesc") }}</DialogDescription>
        </DialogHeader>
        <div class="grid max-h-[70vh] gap-4 overflow-y-auto py-2">
          <label class="flex items-center justify-between gap-3 rounded-md border p-3">
            <span>{{ $t("facebookComments.enableFeature") }}</span>
            <Switch v-model:checked="settings.enabled" />
          </label>
          <label class="flex items-center justify-between gap-3 rounded-md border p-3">
            <span>{{ $t("facebookComments.enableAutoReply") }}</span>
            <Switch v-model:checked="settings.auto_reply_enabled" />
          </label>
          <div class="grid gap-2">
            <Label>{{ $t("facebookComments.autoCommentText") }}</Label>
            <Textarea v-model="settings.auto_comment_reply_text" class="min-h-[80px]" />
          </div>
          <div class="grid gap-2">
            <Label>{{ $t("facebookComments.autoPrivateText") }}</Label>
            <Textarea v-model="settings.auto_private_message_text" class="min-h-[80px]" />
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="flex items-center justify-between gap-3 rounded-md border p-3">
              <span>{{ $t("facebookComments.autoPublic") }}</span>
              <Switch v-model:checked="settings.auto_comment_reply_enabled" />
            </label>
            <label class="flex items-center justify-between gap-3 rounded-md border p-3">
              <span>{{ $t("facebookComments.autoPrivate") }}</span>
              <Switch v-model:checked="settings.auto_private_reply_enabled" />
            </label>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div class="grid gap-2">
              <Label>{{ $t("facebookComments.postLimit") }}</Label>
              <Input v-model.number="settings.default_sync_post_limit" type="number" min="1" max="100" />
            </div>
            <div class="grid gap-2">
              <Label>{{ $t("facebookComments.commentsPerPost") }}</Label>
              <Input v-model.number="settings.default_sync_comments_per_post" type="number" min="1" max="100" />
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

    <Dialog :open="syncOpen" @update:open="syncOpen = $event">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t("facebookComments.sync") }}</DialogTitle>
          <DialogDescription>{{ $t("facebookComments.syncDesc") }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-3 md:grid-cols-2">
            <div class="grid gap-2">
              <Label>{{ $t("facebookComments.postLimit") }}</Label>
              <Input v-model.number="syncPostLimit" type="number" min="1" max="100" />
            </div>
            <div class="grid gap-2">
              <Label>{{ $t("facebookComments.commentsPerPost") }}</Label>
              <Input v-model.number="syncCommentsPerPost" type="number" min="1" max="100" />
            </div>
          </div>
          <label class="flex items-center justify-between gap-3 rounded-md border p-3">
            <span>{{ $t("facebookComments.runAutoReply") }}</span>
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
