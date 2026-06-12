<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useFBAccountsStore } from "@/stores/fbAccounts";
import { DeleteConfirmDialog, PageHeader } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Facebook,
  Plus,
  Loader2,
  MoreVertical,
  Pencil,
  Trash2,
  Cookie,
  Clipboard,
  ExternalLink,
  User,
  ShieldCheck,
  ShieldX,
  ShieldOff,
  RefreshCw,
  Link2,
  Unlink,
} from "lucide-vue-next";
import { useI18n } from "vue-i18n";
import { toast } from "vue-sonner";
import type { FacebookAccount, FacebookAccountPage } from "@/types/facebook";

const fbAccountsStore = useFBAccountsStore();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const {
  fetchAccounts,
  createAccount,
  updateAccount,
  deleteAccount,
  startOAuth,
  refreshPages: refreshAccountPages,
  connectPage: connectAccountPage,
  disconnectPage: disconnectAccountPage,
  removePage: removeAccountPage,
} = fbAccountsStore;

const createDialogOpen = ref(false);
const newAccountName = ref("");
const newAccountUID = ref("");
const newAccountCookies = ref("");
const newAccountPlatform = ref<"facebook" | "instagram">("facebook");
const isCreating = ref(false);
const isReadingClipboard = ref(false);
const isStartingOAuth = ref(false);

const editDialogOpen = ref(false);
const editAccountId = ref("");
const editAccountName = ref("");
const editAccountUID = ref("");
const editAccountStatus = ref<"active" | "inactive" | "closed" | "expired" | "revoked">("active");
const editAccountCookies = ref("");
const isUpdating = ref(false);

const deleteDialogOpen = ref(false);
const deletingAccount = ref<FacebookAccount | null>(null);
const isDeleting = ref(false);

const pageActionKey = ref("");
const removePageDialogOpen = ref(false);
const removingPage = ref<{ account: FacebookAccount; page: FacebookAccountPage } | null>(null);
const isRemovingPage = ref(false);

onMounted(async () => {
  handleOAuthCallbackToast();
  await fetchAccounts();
});

const loginUrl = computed(() =>
  newAccountPlatform.value === "instagram"
    ? "https://www.instagram.com/accounts/login/"
    : "https://www.facebook.com/login",
);

function getStatusBadge(status: string) {
  switch (status) {
    case "active":
      return { variant: "default" as const, icon: ShieldCheck, label: t("fbAccounts.status.active") };
    case "inactive":
      return { variant: "secondary" as const, icon: ShieldOff, label: t("fbAccounts.status.inactive") };
    case "closed":
      return { variant: "destructive" as const, icon: ShieldX, label: t("fbAccounts.status.closed") };
    case "expired":
      return { variant: "destructive" as const, icon: ShieldX, label: t("fbAccounts.status.expired") };
    case "revoked":
      return { variant: "destructive" as const, icon: ShieldX, label: t("fbAccounts.status.revoked") };
    default:
      return { variant: "outline" as const, icon: ShieldOff, label: status };
  }
}

function getMethodLabel(method: string) {
  switch (method) {
    case "oauth":
      return t("fbAccounts.method.oauth");
    case "credentials":
      return t("fbAccounts.method.credentials");
    default:
      return t("fbAccounts.method.cookies");
  }
}

function getLinkedPages(account: FacebookAccount): FacebookAccountPage[] {
  const pages = account.data?.pages;
  if (!Array.isArray(pages)) {
    return [];
  }

  return pages
    .map((page) => ({
      ...page,
      id: String(page?.id || ""),
      name: String(page?.name || ""),
      category: String(page?.category || ""),
      connected: page?.connected !== false,
    }))
    .filter((page) => page.id || page.name);
}

function isPageConnected(page: FacebookAccountPage) {
  return page.connected !== false;
}

function pageLabel(page: FacebookAccountPage) {
  return page.name || page.id || t("fbAccounts.unknownPage");
}

function pageDetail(page: FacebookAccountPage) {
  return page.category || page.id || t("fbAccounts.unknownPage");
}

function pageImageUrl(page: FacebookAccountPage) {
  return page.picture?.data?.url || "";
}

function pageActionLoading(accountId: string, pageId: string, action: string) {
  return pageActionKey.value === `${accountId}:${pageId}:${action}`;
}

async function handleRefreshPages(account: FacebookAccount) {
  const actionKey = `${account.id}:refresh`;
  pageActionKey.value = actionKey;
  try {
    await refreshAccountPages(account.id);
  } finally {
    if (pageActionKey.value === actionKey) {
      pageActionKey.value = "";
    }
  }
}

async function handleConnectPage(account: FacebookAccount, page: FacebookAccountPage) {
  if (!page.id) return;
  const actionKey = `${account.id}:${page.id}:connect`;
  pageActionKey.value = actionKey;
  try {
    await connectAccountPage(account.id, page.id);
  } finally {
    if (pageActionKey.value === actionKey) {
      pageActionKey.value = "";
    }
  }
}

async function handleDisconnectPage(account: FacebookAccount, page: FacebookAccountPage) {
  if (!page.id) return;
  const actionKey = `${account.id}:${page.id}:disconnect`;
  pageActionKey.value = actionKey;
  try {
    await disconnectAccountPage(account.id, page.id);
  } finally {
    if (pageActionKey.value === actionKey) {
      pageActionKey.value = "";
    }
  }
}

function openRemovePageDialog(account: FacebookAccount, page: FacebookAccountPage) {
  if (!page.id) return;
  removingPage.value = { account, page };
  removePageDialogOpen.value = true;
}

function closeRemovePageDialog() {
  if (isRemovingPage.value) return;
  removePageDialogOpen.value = false;
  removingPage.value = null;
}

async function handleRemovePage() {
  if (!removingPage.value || isRemovingPage.value) return;
  const { account, page } = removingPage.value;
  if (!page.id) return;

  const actionKey = `${account.id}:${page.id}:remove`;
  isRemovingPage.value = true;
  pageActionKey.value = actionKey;
  try {
    await removeAccountPage(account.id, page.id);
    removePageDialogOpen.value = false;
    removingPage.value = null;
  } finally {
    isRemovingPage.value = false;
    if (pageActionKey.value === actionKey) {
      pageActionKey.value = "";
    }
  }
}

async function handleStartOAuth(account?: FacebookAccount) {
  isStartingOAuth.value = true;
  try {
    const authUrl = await startOAuth(account?.id);
    if (authUrl) {
      window.location.href = authUrl;
    }
  } finally {
    isStartingOAuth.value = false;
  }
}

function handleOAuthCallbackToast() {
  const status = route.query.facebook_oauth;
  if (status === "connected") {
    toast.success(t("fbAccounts.toast.oauthConnected"));
  } else if (status === "renewed") {
    toast.success(t("fbAccounts.toast.oauthRenewed"));
  } else if (status === "error") {
    toast.error(
      typeof route.query.message === "string"
        ? route.query.message
        : t("fbAccounts.toast.oauthCallbackFailed"),
    );
  } else {
    return;
  }

  const nextQuery = { ...route.query };
  delete nextQuery.facebook_oauth;
  delete nextQuery.message;
  router.replace({ query: nextQuery });
}

async function handleCreate() {
  const trimmedName = newAccountName.value.trim();
  if (!trimmedName) return;
  isCreating.value = true;
  try {
    await createAccount({
      name: trimmedName,
      account_uid: newAccountUID.value.trim() || undefined,
      method: "cookies",
      cookies_text: normalizeCookiesInput(newAccountCookies.value) || undefined,
      data: {
        platform: newAccountPlatform.value,
        cookie_import_source: "login_tab_manual_paste",
      },
    });
    createDialogOpen.value = false;
    newAccountName.value = "";
    newAccountUID.value = "";
    newAccountCookies.value = "";
    newAccountPlatform.value = "facebook";
  } finally {
    isCreating.value = false;
  }
}

function openLoginTab() {
  window.open(loginUrl.value, "_blank", "noopener,noreferrer,width=1180,height=820");
}

async function pasteCookiesFromClipboard() {
  if (!navigator.clipboard?.readText) {
    toast.error(t("fbAccounts.toast.clipboardUnavailable"));
    return;
  }

  isReadingClipboard.value = true;
  try {
    const text = await navigator.clipboard.readText();
    newAccountCookies.value = normalizeCookiesInput(text);
    if (!newAccountCookies.value) {
      toast.error(t("fbAccounts.toast.clipboardEmpty"));
      return;
    }
    toast.success(t("fbAccounts.toast.cookiesPasted"));
  } catch {
    toast.error(t("fbAccounts.toast.clipboardFailed"));
  } finally {
    isReadingClipboard.value = false;
  }
}

function normalizeCookiesInput(input: string) {
  const trimmed = input.trim();
  if (!trimmed) return "";

  try {
    const parsed = JSON.parse(trimmed);
    if (Array.isArray(parsed)) {
      const cookieHeader = parsed
        .filter((cookie) => cookie && typeof cookie.name === "string")
        .map((cookie) => `${cookie.name}=${cookie.value ?? ""}`)
        .join("; ");
      return cookieHeader || JSON.stringify(parsed);
    }
    if (parsed && typeof parsed === "object") {
      const cookieHeader = Object.entries(parsed as Record<string, unknown>)
        .map(([name, value]) => `${name}=${value ?? ""}`)
        .join("; ");
      return cookieHeader || trimmed;
    }
  } catch {
    // Keep raw cookie headers and Netscape-style exports as pasted.
  }

  return trimmed;
}

function openEditDialog(account: FacebookAccount) {
  editAccountId.value = account.id;
  editAccountName.value = account.name;
  editAccountUID.value = account.account_uid;
  editAccountStatus.value = account.status;
  editAccountCookies.value = "";
  editDialogOpen.value = true;
}

function closeEditDialog() {
  editDialogOpen.value = false;
  editAccountId.value = "";
  editAccountName.value = "";
  editAccountUID.value = "";
  editAccountStatus.value = "active";
  editAccountCookies.value = "";
}

async function handleUpdate() {
  const trimmedName = editAccountName.value.trim();
  if (!editAccountId.value || !trimmedName) return;
  isUpdating.value = true;
  try {
    const updateData: any = {
      name: trimmedName,
      account_uid: editAccountUID.value.trim() || undefined,
      status: editAccountStatus.value,
    };
    if (editAccountCookies.value.trim()) {
      updateData.cookies_text = editAccountCookies.value.trim();
    }
    await updateAccount(editAccountId.value, updateData);
    closeEditDialog();
  } finally {
    isUpdating.value = false;
  }
}

function openDeleteDialog(account: FacebookAccount) {
  deletingAccount.value = account;
  deleteDialogOpen.value = true;
}

function closeDeleteDialog() {
  if (isDeleting.value) return;
  deleteDialogOpen.value = false;
  deletingAccount.value = null;
}

async function handleDelete() {
  if (!deletingAccount.value || isDeleting.value) return;
  isDeleting.value = true;
  try {
    await deleteAccount(deletingAccount.value.id);
    deleteDialogOpen.value = false;
    deletingAccount.value = null;
  } finally {
    isDeleting.value = false;
  }
}
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('fbAccounts.title')"
      :subtitle="$t('fbAccounts.subtitle')"
      :icon="Facebook"
      icon-gradient="bg-primary text-primary-foreground shadow-none"
      :breadcrumbs="[
        { label: $t('nav.facebookTools'), href: '/facebook' },
        { label: $t('fbAccounts.title') },
      ]"
    >
      <template #actions>
        <Button size="sm" :disabled="isStartingOAuth" @click="handleStartOAuth()">
          <Loader2 v-if="isStartingOAuth" class="h-4 w-4 mr-2 animate-spin" />
          <ExternalLink v-else class="h-4 w-4 mr-2" />
          {{ $t("fbAccounts.connectOAuth") }}
        </Button>
        <Button size="sm" variant="outline" @click="createDialogOpen = true">
          <Plus class="h-4 w-4 mr-2" />
          {{ $t("fbAccounts.addManualAccount") }}
        </Button>
      </template>
    </PageHeader>

    <ScrollArea class="flex-1">
      <div class="p-6">
        <div class="flex w-full flex-col gap-6">
          <div
            v-if="
              fbAccountsStore.loading && fbAccountsStore.accounts.length === 0
            "
            class="flex min-h-[320px] items-center justify-center rounded-[calc(var(--radius)+0.35rem)] border border-dashed border-border bg-card/70"
          >
            <div class="flex items-center gap-3 text-sm text-muted-foreground">
              <Loader2 class="h-5 w-5 animate-spin text-primary" />
              <span>{{ $t("common.loading") }}</span>
            </div>
          </div>

          <div
            v-else-if="fbAccountsStore.accounts.length === 0"
            class="rounded-[calc(var(--radius)+0.35rem)] border border-dashed border-border bg-card/80"
          >
            <EmptyState
              :icon="Facebook"
              :title="$t('fbAccounts.noAccounts')"
              :description="$t('fbAccounts.noAccountsDesc')"
              class="py-16"
            >
              <template #action>
                <Button variant="outline" @click="createDialogOpen = true">
                  {{ $t("fbAccounts.addManualAccount") }}
                </Button>
              </template>
            </EmptyState>
          </div>

          <div
            v-else
            class="grid gap-6 [grid-template-columns:repeat(auto-fit,minmax(min(100%,20rem),1fr))]"
          >
            <Card
              v-for="account in fbAccountsStore.accounts"
              :key="account.id"
              class="group relative overflow-hidden border-border/60 bg-card/50 backdrop-blur-sm transition-all hover:border-primary/30 hover:shadow-lg"
            >
              <CardContent class="p-5">
                <div class="flex items-start justify-between mb-4">
                  <div class="flex items-center gap-3">
                    <div
                      class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-600/10 text-blue-600"
                    >
                      <Facebook class="h-5 w-5" />
                    </div>
                    <div>
                      <h3 class="text-sm font-semibold text-foreground leading-tight">
                        {{ account.name }}
                      </h3>
                      <p class="text-xs text-muted-foreground mt-0.5">
                        {{ account.account_uid || $t("fbAccounts.noUid") }}
                      </p>
                    </div>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger as-child>
                      <Button
                        variant="ghost"
                        size="icon"
                        class="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <MoreVertical class="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" class="w-40">
                      <DropdownMenuItem @click="openEditDialog(account)">
                        <Pencil class="mr-2 h-4 w-4" />
                        {{ $t("common.edit") }}
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        v-if="account.method === 'oauth'"
                        @click="handleStartOAuth(account)"
                      >
                        <ExternalLink class="mr-2 h-4 w-4" />
                        {{ $t("fbAccounts.renewOAuth") }}
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        class="text-destructive focus:text-destructive"
                        @click="openDeleteDialog(account)"
                      >
                        <Trash2 class="mr-2 h-4 w-4" />
                        {{ $t("common.delete") }}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>

                <div class="flex flex-wrap items-center gap-2 mb-3">
                  <Badge
                    :variant="getStatusBadge(account.status).variant"
                    class="text-xs gap-1"
                  >
                    <component
                      :is="getStatusBadge(account.status).icon"
                      class="h-3 w-3"
                    />
                    {{ getStatusBadge(account.status).label }}
                  </Badge>
                  <Badge variant="outline" class="text-xs gap-1">
                    {{ getMethodLabel(account.method) }}
                  </Badge>
                  <Badge
                    v-if="account.oauth_connected"
                    variant="outline"
                    class="text-xs gap-1 border-blue-500/30 text-blue-600 dark:text-blue-400"
                  >
                    <ExternalLink class="h-3 w-3" />
                    {{ $t("fbAccounts.oauthConnected") }}
                  </Badge>
                  <Badge
                    v-if="account.has_cookies"
                    variant="outline"
                    class="text-xs gap-1 border-green-500/30 text-green-600 dark:text-green-400"
                  >
                    <Cookie class="h-3 w-3" />
                    {{ $t("fbAccounts.hasCookies") }}
                  </Badge>
                </div>

                <div class="flex items-center gap-2 text-xs text-muted-foreground">
                  <User class="h-3 w-3" />
                  <span>{{ $t("fbAccounts.uid") }}: {{ account.account_uid || "—" }}</span>
                </div>
                <div
                  v-if="account.method === 'oauth'"
                  class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground"
                >
                  <span>{{ $t("fbAccounts.pages") }}: {{ account.page_count || 0 }}</span>
                  <span v-if="account.token_expires_at">
                    {{ $t("fbAccounts.expires") }}:
                    {{ new Date(account.token_expires_at).toLocaleDateString() }}
                  </span>
                </div>
                <div
                  v-if="account.method === 'oauth'"
                  class="mt-4 overflow-hidden rounded-lg border border-border/70 bg-muted/20"
                >
                  <div
                    class="flex flex-col gap-3 border-b border-border/70 p-3 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div>
                      <p class="text-sm font-medium text-foreground">
                        {{ $t("fbAccounts.pages") }}
                      </p>
                      <p class="text-xs text-muted-foreground">
                        {{ getLinkedPages(account).length }} {{ $t("fbAccounts.linkedPages") }}
                      </p>
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      :disabled="pageActionKey === `${account.id}:refresh`"
                      @click="handleRefreshPages(account)"
                    >
                      <Loader2
                        v-if="pageActionKey === `${account.id}:refresh`"
                        class="mr-2 h-4 w-4 animate-spin"
                      />
                      <RefreshCw v-else class="mr-2 h-4 w-4" />
                      {{ $t("fbAccounts.refreshPages") }}
                    </Button>
                  </div>

                  <div
                    v-if="getLinkedPages(account).length === 0"
                    class="p-3 text-sm text-muted-foreground"
                  >
                    {{ $t("fbAccounts.noPages") }}
                  </div>
                  <div v-else class="divide-y divide-border/70">
                    <div
                      v-for="page in getLinkedPages(account)"
                      :key="page.id || page.name"
                      class="flex flex-col gap-3 p-3 sm:flex-row sm:items-center sm:justify-between"
                    >
                      <div class="flex min-w-0 items-center gap-3">
                        <img
                          v-if="pageImageUrl(page)"
                          :src="pageImageUrl(page)"
                          :alt="pageLabel(page)"
                          class="h-9 w-9 rounded-md border border-border object-cover"
                        />
                        <div
                          v-else
                          class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-blue-600/10 text-blue-600"
                        >
                          <Facebook class="h-4 w-4" />
                        </div>
                        <div class="min-w-0">
                          <div class="flex flex-wrap items-center gap-2">
                            <p class="truncate text-sm font-medium text-foreground">
                              {{ pageLabel(page) }}
                            </p>
                            <Badge
                              :variant="isPageConnected(page) ? 'default' : 'secondary'"
                              class="text-xs"
                            >
                              {{
                                isPageConnected(page)
                                  ? $t("fbAccounts.pageConnected")
                                  : $t("fbAccounts.pageDisconnected")
                              }}
                            </Badge>
                          </div>
                          <p class="truncate text-xs text-muted-foreground">
                            {{ pageDetail(page) }}
                          </p>
                        </div>
                      </div>

                      <div class="flex flex-wrap gap-2">
                        <Button
                          v-if="!isPageConnected(page)"
                          size="sm"
                          variant="outline"
                          :disabled="pageActionLoading(account.id, page.id, 'connect')"
                          @click="handleConnectPage(account, page)"
                        >
                          <Loader2
                            v-if="pageActionLoading(account.id, page.id, 'connect')"
                            class="mr-2 h-4 w-4 animate-spin"
                          />
                          <Link2 v-else class="mr-2 h-4 w-4" />
                          {{ $t("fbAccounts.connectPage") }}
                        </Button>
                        <Button
                          v-else
                          size="sm"
                          variant="outline"
                          :disabled="pageActionLoading(account.id, page.id, 'disconnect')"
                          @click="handleDisconnectPage(account, page)"
                        >
                          <Loader2
                            v-if="pageActionLoading(account.id, page.id, 'disconnect')"
                            class="mr-2 h-4 w-4 animate-spin"
                          />
                          <Unlink v-else class="mr-2 h-4 w-4" />
                          {{ $t("fbAccounts.disconnectPage") }}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          class="text-destructive hover:text-destructive"
                          :disabled="pageActionLoading(account.id, page.id, 'remove')"
                          @click="openRemovePageDialog(account, page)"
                        >
                          <Loader2
                            v-if="pageActionLoading(account.id, page.id, 'remove')"
                            class="mr-2 h-4 w-4 animate-spin"
                          />
                          <Trash2 v-else class="mr-2 h-4 w-4" />
                          {{ $t("fbAccounts.removePage") }}
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </ScrollArea>

    <!-- Create Account Dialog -->
    <Dialog :open="createDialogOpen" @update:open="createDialogOpen = $event">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{{ $t("fbAccounts.dialog.addTitle") }}</DialogTitle>
          <DialogDescription>
            {{ $t("fbAccounts.dialog.addDesc") }}
          </DialogDescription>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid gap-2">
            <Label for="fb-platform">{{ $t("fbAccounts.dialog.platform") }}</Label>
            <Select v-model="newAccountPlatform">
              <SelectTrigger id="fb-platform">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="facebook">Facebook</SelectItem>
                <SelectItem value="instagram">Instagram</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="rounded-lg border border-border bg-muted/30 p-3">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <p class="text-sm text-muted-foreground">
                {{ $t("fbAccounts.dialog.loginTabHint") }}
              </p>
              <Button variant="outline" size="sm" @click="openLoginTab">
                <ExternalLink class="mr-2 h-4 w-4" />
                {{ $t("fbAccounts.dialog.openLoginTab") }}
              </Button>
            </div>
          </div>
          <div class="grid gap-2">
            <Label for="fb-name">{{ $t("fbAccounts.dialog.accountName") }}</Label>
            <Input
              id="fb-name"
              v-model="newAccountName"
              :placeholder="$t('fbAccounts.dialog.namePlaceholder')"
            />
          </div>
          <div class="grid gap-2">
            <Label for="fb-uid">{{ $t("fbAccounts.dialog.accountUid") }}</Label>
            <Input
              id="fb-uid"
              v-model="newAccountUID"
              :placeholder="$t('fbAccounts.dialog.uidPlaceholder')"
            />
          </div>
          <div class="grid gap-2">
            <div class="flex items-center justify-between gap-3">
              <Label for="fb-cookies">{{ $t("fbAccounts.dialog.cookies") }}</Label>
              <Button
                variant="ghost"
                size="sm"
                type="button"
                :disabled="isReadingClipboard"
                @click="pasteCookiesFromClipboard"
              >
                <Loader2 v-if="isReadingClipboard" class="mr-2 h-4 w-4 animate-spin" />
                <Clipboard v-else class="mr-2 h-4 w-4" />
                {{ $t("fbAccounts.dialog.pasteCookies") }}
              </Button>
            </div>
            <Textarea
              id="fb-cookies"
              v-model="newAccountCookies"
              :placeholder="$t('fbAccounts.dialog.cookiesPlaceholder')"
              class="min-h-[110px]"
            />
            <p class="text-xs text-muted-foreground">
              {{ $t("fbAccounts.dialog.cookiesHint") }}
            </p>
            <p class="text-xs text-muted-foreground">
              {{ $t("fbAccounts.dialog.browserLimitHint") }}
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="createDialogOpen = false">
            {{ $t("common.cancel") }}
          </Button>
          <Button
            @click="handleCreate"
            :disabled="isCreating || !newAccountName.trim()"
          >
            <Loader2 v-if="isCreating" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t("common.create") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Edit Account Dialog -->
    <Dialog
      :open="editDialogOpen"
      @update:open="(open) => !open && closeEditDialog()"
    >
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{{ $t("fbAccounts.dialog.editTitle") }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid gap-2">
            <Label for="edit-fb-name">{{ $t("fbAccounts.dialog.accountName") }}</Label>
            <Input
              id="edit-fb-name"
              v-model="editAccountName"
              :placeholder="$t('fbAccounts.dialog.namePlaceholder')"
            />
          </div>
          <div class="grid gap-2">
            <Label for="edit-fb-uid">{{ $t("fbAccounts.dialog.accountUid") }}</Label>
            <Input
              id="edit-fb-uid"
              v-model="editAccountUID"
              :placeholder="$t('fbAccounts.dialog.uidPlaceholder')"
            />
          </div>
          <div class="grid gap-2">
            <Label for="edit-fb-status">{{ $t("fbAccounts.dialog.status") }}</Label>
            <Select v-model="editAccountStatus">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">
                  {{ $t("fbAccounts.status.active") }}
                </SelectItem>
                <SelectItem value="inactive">
                  {{ $t("fbAccounts.status.inactive") }}
                </SelectItem>
                <SelectItem value="closed">
                  {{ $t("fbAccounts.status.closed") }}
                </SelectItem>
                <SelectItem value="expired">
                  {{ $t("fbAccounts.status.expired") }}
                </SelectItem>
                <SelectItem value="revoked">
                  {{ $t("fbAccounts.status.revoked") }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid gap-2">
            <Label for="edit-fb-cookies">{{ $t("fbAccounts.dialog.cookies") }}</Label>
            <Textarea
              id="edit-fb-cookies"
              v-model="editAccountCookies"
              :placeholder="$t('fbAccounts.dialog.cookiesUpdateHint')"
              class="min-h-[80px]"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="closeEditDialog">
            {{ $t("common.cancel") }}
          </Button>
          <Button
            @click="handleUpdate"
            :disabled="isUpdating || !editAccountName.trim()"
          >
            <Loader2 v-if="isUpdating" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t("common.update") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('fbAccounts.dialog.deleteTitle')"
      :item-name="deletingAccount?.name"
      :confirm-label="$t('fbAccounts.dialog.confirmDelete')"
      :cancel-label="$t('fbAccounts.dialog.cancelKeep')"
      @confirm="handleDelete"
      @cancel="closeDeleteDialog"
    >
      <template #description>
        {{ $t("fbAccounts.dialog.deleteDesc") }}
      </template>
    </DeleteConfirmDialog>

    <DeleteConfirmDialog
      v-model:open="removePageDialogOpen"
      :title="$t('fbAccounts.dialog.removePageTitle')"
      :item-name="removingPage ? pageLabel(removingPage.page) : undefined"
      :confirm-label="$t('fbAccounts.dialog.confirmRemovePage')"
      :cancel-label="$t('common.cancel')"
      @confirm="handleRemovePage"
      @cancel="closeRemovePageDialog"
    >
      <template #description>
        {{ $t("fbAccounts.dialog.removePageDesc") }}
      </template>
    </DeleteConfirmDialog>
  </div>
</template>
