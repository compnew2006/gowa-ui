import { defineStore } from "pinia";
import { ref } from "vue";
import { fbAccountsService } from "@/services/api";
import { useAuthStore } from "@/stores/auth";
import type { FacebookAccount } from "@/types/facebook";
import { toast } from "vue-sonner";
import { i18n } from "@/i18n";
import { unwrapResponse } from "@/lib/api-utils";

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
      accounts.value = unwrapResponse<FacebookAccount[]>(response);
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
      status?: "active" | "inactive" | "closed";
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
  };
});
