import { ref, type ComputedRef } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue-sonner";
import { useRouter } from "vue-router";
import {
  contactsService,
  chatbotService,
  customActionsService,
  type CustomAction,
  type ActionResult,
} from "@/services/api";
import {
  Ticket,
  User,
  BarChart,
  Link,
  Phone,
  Mail,
  FileText,
  ExternalLink,
  Zap,
  Globe,
  Code,
} from "lucide-vue-next";
import type { Contact } from "@/stores/contacts";

export function useChatActions(
  currentContact: ComputedRef<Contact | null>,
  options: {
    isAdminUser: ComputedRef<boolean>;
    refreshContactsSidebar: () => Promise<void>;
    canReadCustomActions: ComputedRef<boolean>;
  },
) {
  const { t } = useI18n();
  const router = useRouter();

  const customActions = ref<CustomAction[]>([]);
  const executingActionId = ref<string | null>(null);
  const isTransferring = ref(false);
  const isResuming = ref(false);
  const isAssignDialogOpen = ref(false);
  const assignSearchQuery = ref("");
  const isClaimingCurrentChat = ref(false);
  const isClosingCurrentChat = ref(false);
  const isReopeningCurrentChat = ref(false);
  const isUpdatingCurrentChatPublic = ref(false);

  const actionIconMap: Record<string, any> = {
    ticket: Ticket,
    user: User,
    "bar-chart": BarChart,
    link: Link,
    phone: Phone,
    mail: Mail,
    "file-text": FileText,
    "external-link": ExternalLink,
    zap: Zap,
    globe: Globe,
    code: Code,
  };

  function getActionIcon(iconName: string) {
    return actionIconMap[iconName] || Zap;
  }

  async function fetchCustomActions() {
    try {
      const response = await customActionsService.list();
      const data = (response.data as any).data || response.data;
      customActions.value = (data.custom_actions || []).filter(
        (a: CustomAction) => a.is_active,
      );
    } catch (error) {
      console.error("Failed to fetch custom actions:", error);
    }
  }

  async function executeCustomAction(action: CustomAction) {
    if (!currentContact.value || executingActionId.value) return;

    executingActionId.value = action.id;
    try {
      const response = await customActionsService.execute(
        action.id,
        currentContact.value.id,
      );
      let result: ActionResult = (response.data as any).data || response.data;

      if (result.redirect_url) {
        let redirectUrl = result.redirect_url;
        if (!redirectUrl.startsWith("/api/")) {
          redirectUrl = "";
        }
        if (redirectUrl) {
          const basePath = ((window as any).__BASE_PATH__ ?? "").replace(
            /\/$/,
            "",
          );
          redirectUrl = basePath + redirectUrl;
          window.open(redirectUrl, "_blank", "noopener,noreferrer");
        }
      }

      if (result.clipboard) {
        await navigator.clipboard.writeText(result.clipboard);
        toast.success(t("common.copiedToClipboard"));
      }

      if (result.toast) {
        if (result.toast.type === "success") {
          toast.success(result.toast.message);
        } else if (result.toast.type === "error") {
          toast.error(result.toast.message);
        } else {
          toast.info(result.toast.message);
        }
      } else if (result.success && !result.redirect_url && !result.clipboard) {
        toast.success(result.message || t("chat.actionExecuted"));
      } else if (!result.success) {
        toast.error(result.message || t("chat.actionFailed"));
      }
    } catch (error: any) {
      const message = error.response?.data?.message || "Failed to execute action";
      toast.error(message);
    } finally {
      executingActionId.value = null;
    }
  }

  async function transferToAgent() {
    if (!currentContact.value) return;

    isTransferring.value = true;
    try {
      await chatbotService.createTransfer({
        contact_id: currentContact.value.id,
        whatsapp_account: (currentContact.value as any).whatsapp_account,
        source: "manual",
      });
      toast.success(t("chat.transferSuccess"), {
        description: t("chat.transferSuccessDesc"),
      });
    } catch (error: any) {
      const message = error.response?.data?.message || t("chat.transferFailed");
      toast.error(message);
    } finally {
      isTransferring.value = false;
    }
  }

  async function resumeChatbot(activeTransferId: string | null) {
    if (!activeTransferId) return;

    const currentContactId = currentContact.value?.id;
    isResuming.value = true;
    try {
      await chatbotService.resumeTransfer(activeTransferId);
      toast.success(t("chat.resumeSuccess"), {
        description: t("chat.resumeSuccessDesc"),
      });
      await options.refreshContactsSidebar();

      if (currentContactId) {
        const { useContactsStore } = await import("@/stores/contacts");
        const contactsStore = useContactsStore();
        const stillExists = contactsStore.contacts.some(
          (c) => c.id === currentContactId,
        );
        if (!stillExists) {
          contactsStore.setCurrentContact(null);
          contactsStore.clearMessages();
          router.push("/chat");
        }
      }
    } catch (error: any) {
      const message = error.response?.data?.message || t("chat.resumeFailed");
      toast.error(message);
    } finally {
      isResuming.value = false;
    }
  }

  async function assignContactToUser(userId: string | null) {
    if (!currentContact.value) return;

    try {
      await contactsService.assign(currentContact.value.id, userId);
      toast.success(
        userId ? t("chat.contactAssigned") : t("chat.contactUnassigned"),
      );
      const { useContactsStore } = await import("@/stores/contacts");
      const contactsStore = useContactsStore();
      contactsStore.currentContact = {
        ...contactsStore.currentContact!,
        assigned_user_id: userId || undefined,
        status: userId ? "open" : "pending",
      };
      await options.refreshContactsSidebar();
    } catch (error: any) {
      const message = error.response?.data?.message || t("chat.assignFailed");
      toast.error(message);
    }
  }

  async function claimCurrentChat() {
    if (!currentContact.value || isClaimingCurrentChat.value) return;
    isClaimingCurrentChat.value = true;
    try {
      const { useContactsStore } = await import("@/stores/contacts");
      const contactsStore = useContactsStore();
      const updated = await contactsStore.claimChat(currentContact.value.id);
      if (!updated) {
        toast.error("Failed to claim chat");
        return;
      }
      toast.success("Chat claimed successfully");
      contactsStore.setActiveChatTab("assigned");
      await options.refreshContactsSidebar();
      return updated;
    } catch (error: any) {
      const message = error?.response?.data?.message || "Failed to claim chat";
      toast.error(message);
      return null;
    } finally {
      isClaimingCurrentChat.value = false;
    }
  }

  async function closeCurrentChat() {
    if (!currentContact.value || isClosingCurrentChat.value) return;
    isClosingCurrentChat.value = true;
    try {
      const { useContactsStore } = await import("@/stores/contacts");
      const contactsStore = useContactsStore();
      const updated = await contactsStore.closeChat(currentContact.value.id);
      if (!updated) {
        toast.error("Failed to close chat");
        return;
      }
      toast.success("Chat closed");
      await options.refreshContactsSidebar();
      contactsStore.setCurrentContact(null);
      contactsStore.clearMessages();
      router.push("/chat");
    } catch (error: any) {
      const message = error?.response?.data?.message || "Failed to close chat";
      toast.error(message);
    } finally {
      isClosingCurrentChat.value = false;
    }
  }

  async function reopenCurrentChat() {
    if (!currentContact.value || isReopeningCurrentChat.value) return;
    isReopeningCurrentChat.value = true;
    try {
      const { useContactsStore } = await import("@/stores/contacts");
      const contactsStore = useContactsStore();
      const updated = await contactsStore.reopenChat(currentContact.value.id);
      if (!updated) {
        toast.error("Failed to reopen chat");
        return;
      }
      toast.success("Chat reopened and moved to pending queue");
      contactsStore.setActiveChatTab("pending");
      await options.refreshContactsSidebar();
      return updated;
    } catch (error: any) {
      const message = error?.response?.data?.message || "Failed to reopen chat";
      toast.error(message);
      return null;
    } finally {
      isReopeningCurrentChat.value = false;
    }
  }

  async function toggleCurrentChatPublicVisibility(canToggle: boolean) {
    if (!currentContact.value || !canToggle || isUpdatingCurrentChatPublic.value) {
      return;
    }

    isUpdatingCurrentChatPublic.value = true;
    const nextIsPublic = currentContact.value.is_public !== true;
    try {
      const { useContactsStore } = await import("@/stores/contacts");
      const contactsStore = useContactsStore();
      const updated = await contactsStore.setChatPublic(
        currentContact.value.id,
        nextIsPublic,
      );
      if (!updated) {
        toast.error("Failed to update public chat setting");
        return;
      }
      toast.success(
        nextIsPublic ? t("chat.publicChatEnabled") : t("chat.publicChatDisabled"),
      );
      await options.refreshContactsSidebar();
    } catch (error: any) {
      const message =
        error?.response?.data?.message || t("chat.publicChatUpdateFailed");
      toast.error(message);
    } finally {
      isUpdatingCurrentChatPublic.value = false;
    }
  }

  return {
    customActions,
    executingActionId,
    isTransferring,
    isResuming,
    isAssignDialogOpen,
    assignSearchQuery,
    isClaimingCurrentChat,
    isClosingCurrentChat,
    isReopeningCurrentChat,
    isUpdatingCurrentChatPublic,
    getActionIcon,
    fetchCustomActions,
    executeCustomAction,
    transferToAgent,
    resumeChatbot,
    assignContactToUser,
    claimCurrentChat,
    closeCurrentChat,
    reopenCurrentChat,
    toggleCurrentChatPublicVisibility,
  };
}
