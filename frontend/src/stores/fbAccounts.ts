import { defineStore } from "pinia";
import { ref } from "vue";
import { fbAccountsService } from "@/services/api";
import { useAuthStore } from "@/stores/auth";
import type { FacebookAccount } from "@/types/facebook";
import { toast } from "vue-sonner";
import { i18n } from "@/i18n";
import { unwrapListResponse, unwrapResponse } from "@/lib/api-utils";

export const useFBAccountsStore = defineStore("fbAccounts", () => {
  const authStore = useAuthStore();
  const t = i18n.global.t;
  const accounts = ref<FacebookAccount[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  function canReadAccounts() {
    return authStore.hasPermission("accounts", "read");
  }

  function canWriteAccounts() {
    return authStore.hasPermission("accounts", "write");
  }

  async function fetchAccounts() {
    if (!canReadAccounts()) {
      accounts.value = [];
      error.value = null;
      return [];
    }

    loading.value = true;
    error.value = null;
    try {
      const response = await fbAccountsService.list();
      accounts.value = unwrapListResponse<FacebookAccount>(response, "accounts");
      return accounts.value;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.fetchFailed");
      error.value = message;
      toast.error(message);
    } finally {
      loading.value = false;
    }
  }

  async function createAccount(data: {
    name: string;
    account_uid?: string;
    method?: "cookies" | "credentials" | "oauth";
    cookies_text?: string;
    data?: Record<string, unknown>;
  }): Promise<FacebookAccount | null> {
    try {
      const response = await fbAccountsService.create(data);
      const account = unwrapResponse<FacebookAccount>(response);
      accounts.value.unshift(account);
      toast.success(t("fbAccounts.toast.created"));
      return account;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.createFailed");
      toast.error(message);
      return null;
    }
  }

  async function updateAccount(
    id: string,
    data: {
      name?: string;
      account_uid?: string;
      status?: "active" | "inactive" | "closed" | "expired" | "revoked";
      method?: "cookies" | "credentials" | "oauth";
      cookies_text?: string;
      data?: Record<string, unknown>;
    },
  ): Promise<FacebookAccount | null> {
    try {
      const response = await fbAccountsService.update(id, data);
      const updated = unwrapResponse<FacebookAccount>(response);
      const index = accounts.value.findIndex((a) => a.id === id);
      if (index !== -1) {
        accounts.value[index] = updated;
      }
      toast.success(t("fbAccounts.toast.updated"));
      return updated;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.updateFailed");
      toast.error(message);
      return null;
    }
  }

  async function deleteAccount(id: string): Promise<boolean> {
    try {
      await fbAccountsService.delete(id);
      accounts.value = accounts.value.filter((a) => a.id !== id);
      toast.success(t("fbAccounts.toast.deleted"));
      return true;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.deleteFailed");
      toast.error(message);
      return false;
    }
  }

  async function startOAuth(accountId?: string): Promise<string | null> {
    try {
      const response = accountId
        ? await fbAccountsService.renewOAuth(accountId)
        : await fbAccountsService.initOAuth({ action: "connect" });
      const payload = unwrapResponse<{ auth_url: string }>(response);
      return payload.auth_url;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.oauthInitFailed");
      toast.error(message);
      return null;
    }
  }

  function replaceAccount(updated: FacebookAccount) {
    const index = accounts.value.findIndex((a) => a.id === updated.id);
    if (index !== -1) {
      accounts.value[index] = updated;
    }
    return updated;
  }

  async function refreshPages(id: string): Promise<FacebookAccount | null> {
    try {
      const response = await fbAccountsService.refreshPages(id);
      const updated = replaceAccount(unwrapResponse<FacebookAccount>(response));
      toast.success(t("fbAccounts.toast.pagesRefreshed"));
      return updated;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.pagesRefreshFailed");
      toast.error(message);
      return null;
    }
  }

  async function connectPage(id: string, pageId: string): Promise<FacebookAccount | null> {
    try {
      const response = await fbAccountsService.connectPage(id, pageId);
      const updated = replaceAccount(unwrapResponse<FacebookAccount>(response));
      toast.success(t("fbAccounts.toast.pageConnected"));
      return updated;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.pageConnectFailed");
      toast.error(message);
      return null;
    }
  }

  async function disconnectPage(id: string, pageId: string): Promise<FacebookAccount | null> {
    try {
      const response = await fbAccountsService.disconnectPage(id, pageId);
      const updated = replaceAccount(unwrapResponse<FacebookAccount>(response));
      toast.success(t("fbAccounts.toast.pageDisconnected"));
      return updated;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.pageDisconnectFailed");
      toast.error(message);
      return null;
    }
  }

  async function removePage(id: string, pageId: string): Promise<FacebookAccount | null> {
    try {
      const response = await fbAccountsService.removePage(id, pageId);
      const updated = replaceAccount(unwrapResponse<FacebookAccount>(response));
      toast.success(t("fbAccounts.toast.pageRemoved"));
      return updated;
    } catch (err: any) {
      const message =
        err.response?.data?.message || t("fbAccounts.toast.pageRemoveFailed");
      toast.error(message);
      return null;
    }
  }

  return {
    accounts,
    loading,
    error,
    canReadAccounts,
    canWriteAccounts,
    fetchAccounts,
    createAccount,
    updateAccount,
    deleteAccount,
    startOAuth,
    refreshPages,
    connectPage,
    disconnectPage,
    removePage,
  };
});
