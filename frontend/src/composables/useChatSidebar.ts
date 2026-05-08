import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { localeDirectionManager } from "@/i18n/locale-direction";
import {
  ChatSidebarUnifier,
  type ChatSidebarViewMode,
  type SidebarContactEntry,
} from "@/lib/chat-sidebar-unifier";
import { useInstancesStore } from "@/stores/instances";
import type { Contact } from "@/stores/contacts";

const ACCOUNT_TOGGLE_PREFIX = "acct:";
const CONTACT_TOGGLE_PREFIX = "contact:";

const CONTACTS_SIDEBAR_WIDTH_STORAGE_KEY = "chat.contactsSidebarWidth";
const CONTACTS_SIDEBAR_MIN_WIDTH = 280;
const CONTACTS_SIDEBAR_MAX_WIDTH = 500;
const CONTACTS_SIDEBAR_DEFAULT_WIDTH = 320;

function toAccountToggleKey(accountName: string): string {
  return `${ACCOUNT_TOGGLE_PREFIX}${accountName}`;
}

function toContactToggleKey(contactID: string): string {
  return `${CONTACT_TOGGLE_PREFIX}${contactID}`;
}

function accountFromToggleKey(toggleKey?: string | null): string {
  if (!toggleKey || !toggleKey.startsWith(ACCOUNT_TOGGLE_PREFIX)) {
    return "";
  }
  return toggleKey.slice(ACCOUNT_TOGGLE_PREFIX.length).trim();
}

function contactIDFromToggleKey(toggleKey?: string | null): string {
  if (!toggleKey || !toggleKey.startsWith(CONTACT_TOGGLE_PREFIX)) {
    return "";
  }
  return toggleKey.slice(CONTACT_TOGGLE_PREFIX.length).trim();
}

function selectedAccountFilter(toggleKey?: string | null): string | undefined {
  const account = accountFromToggleKey(toggleKey);
  return account || undefined;
}

function clampContactsSidebarWidth(value: number): number {
  return Math.min(
    CONTACTS_SIDEBAR_MAX_WIDTH,
    Math.max(CONTACTS_SIDEBAR_MIN_WIDTH, value),
  );
}

function readContactsSidebarWidth(): number {
  try {
    const stored = Number(
      localStorage.getItem(CONTACTS_SIDEBAR_WIDTH_STORAGE_KEY),
    );
    if (Number.isFinite(stored) && stored > 0) {
      return clampContactsSidebarWidth(stored);
    }
  } catch {
    // Ignore localStorage errors
  }
  return CONTACTS_SIDEBAR_DEFAULT_WIDTH;
}

export function useChatSidebar() {
  const { locale } = useI18n();
  const instancesStore = useInstancesStore();

  const isRTL = computed(() =>
    localeDirectionManager.isRTL(String(locale.value)),
  );

  const chatSidebarUnifier = new ChatSidebarUnifier();
  const chatSidebarViewMode = ref<ChatSidebarViewMode>(
    ChatSidebarUnifier.readViewMode(),
  );

  const selectedAccount = ref<string | null>(null);
  const contactAccounts = ref<string[]>([]);

  const contactsSidebarWidth = ref(readContactsSidebarWidth());
  const isContactsSidebarResizing = ref(false);
  const isContactsSidebarCompact = computed(
    () => contactsSidebarWidth.value <= 320,
  );
  const isContactsSidebarWide = computed(() => contactsSidebarWidth.value >= 420);
  let contactsSidebarResizeStartX = 0;
  let contactsSidebarResizeStartWidth = contactsSidebarWidth.value;

  const isSidebarUnifiedMode = computed(
    () => chatSidebarViewMode.value === "unified",
  );

  function setContactsSidebarWidth(value: number) {
    const nextWidth = clampContactsSidebarWidth(value);
    contactsSidebarWidth.value = nextWidth;
    try {
      localStorage.setItem(CONTACTS_SIDEBAR_WIDTH_STORAGE_KEY, String(nextWidth));
    } catch {
      // Ignore localStorage errors
    }
  }

  function onContactsSidebarResizeMove(event: MouseEvent) {
    if (!isContactsSidebarResizing.value) return;
    const deltaX = isRTL.value
      ? contactsSidebarResizeStartX - event.clientX
      : event.clientX - contactsSidebarResizeStartX;
    setContactsSidebarWidth(contactsSidebarResizeStartWidth + deltaX);
  }

  function stopContactsSidebarResize() {
    if (!isContactsSidebarResizing.value) return;
    isContactsSidebarResizing.value = false;
    window.removeEventListener("mousemove", onContactsSidebarResizeMove);
    window.removeEventListener("mouseup", stopContactsSidebarResize);
  }

  function startContactsSidebarResize(event: MouseEvent) {
    if (window.innerWidth < 768) return;
    isContactsSidebarResizing.value = true;
    contactsSidebarResizeStartX = event.clientX;
    contactsSidebarResizeStartWidth = contactsSidebarWidth.value;
    window.addEventListener("mousemove", onContactsSidebarResizeMove);
    window.addEventListener("mouseup", stopContactsSidebarResize);
    event.preventDefault();
  }

  function refreshChatSidebarViewModePreference() {
    chatSidebarViewMode.value = ChatSidebarUnifier.readViewMode();
  }

  function findSidebarEntrySourceContact(
    entry: SidebarContactEntry | null,
    contactID: string,
  ): Contact | null {
    if (!entry || !contactID) return null;
    return (
      entry.sourceContacts.find((contact) => contact.id === contactID) || null
    );
  }

  function resolveSourceContactForToggle(
    entry: SidebarContactEntry | null,
    toggleKey?: string | null,
  ): Contact | null {
    if (!entry || !toggleKey) return null;

    const contactID = contactIDFromToggleKey(toggleKey);
    if (contactID) {
      return findSidebarEntrySourceContact(entry, contactID);
    }

    const accountName = accountFromToggleKey(toggleKey);
    if (accountName && entry.contactsByAccount[accountName]) {
      return entry.contactsByAccount[accountName];
    }

    return null;
  }

  function resolveSelectedSourceContact(contact: Contact | null, currentSidebarEntry: SidebarContactEntry | null): Contact | null {
    if (!contact) return null;
    const selected = resolveSourceContactForToggle(currentSidebarEntry, selectedAccount.value);
    if (selected) return selected;
    return contact;
  }

  function resolveExplicitSourceContact(contact: Contact | null, currentSidebarEntry: SidebarContactEntry | null): Contact | null {
    if (!contact) return null;
    return resolveSourceContactForToggle(
      currentSidebarEntry,
      selectedAccount.value,
    );
  }

  function resolveInstanceToggleLabel(instanceID?: string): string {
    if (!instanceID) return "";
    const instance = instancesStore.instances.find(
      (item) => item.id === instanceID,
    );
    if (!instance) return "";
    if (typeof instance.name === "string" && instance.name.trim() !== "") {
      return instance.name.trim();
    }
    if (
      typeof (instance as Record<string, unknown>).phone_number === "string" &&
      String((instance as Record<string, unknown>).phone_number).trim() !== ""
    ) {
      return String((instance as Record<string, unknown>).phone_number).trim();
    }
    return "";
  }

  function resolveSidebarEntryInstanceIDs(entry: SidebarContactEntry): string[] {
    const instanceIDs: string[] = [];
    const seen = new Set<string>();

    const appendInstanceID = (rawValue?: string) => {
      const instanceID = (rawValue || "").trim();
      if (!instanceID || seen.has(instanceID)) return;
      seen.add(instanceID);
      instanceIDs.push(instanceID);
    };

    for (const sourceContact of entry.sourceContacts || []) {
      appendInstanceID(
        typeof sourceContact.instance_id === "string"
          ? sourceContact.instance_id
          : "",
      );
    }

    for (const instanceID of entry.sourceInstanceIDs || []) {
      appendInstanceID(instanceID);
    }

    if (instanceIDs.length === 0 && entry.displayContact.instance_id) {
      appendInstanceID(entry.displayContact.instance_id);
    }

    return instanceIDs;
  }

  function getSidebarEntryInstanceCount(entry: SidebarContactEntry): number {
    return resolveSidebarEntryInstanceIDs(entry).length;
  }

  function hasSidebarEntryMultipleInstances(entry: SidebarContactEntry): boolean {
    return getSidebarEntryInstanceCount(entry) > 1;
  }

  function getSidebarEntryPrimaryInstanceID(
    entry: SidebarContactEntry,
  ): string | undefined {
    return resolveSidebarEntryInstanceIDs(entry)[0];
  }

  function resolveSidebarEntryInstanceLabel(
    entry: SidebarContactEntry,
    instanceID?: string,
  ): string {
    const normalizedInstanceID = (instanceID || "").trim();
    if (!normalizedInstanceID) return "";

    for (const sourceContact of entry.sourceContacts || []) {
      const sourceInstanceID =
        typeof sourceContact.instance_id === "string"
          ? sourceContact.instance_id.trim()
          : "";
      if (sourceInstanceID !== normalizedInstanceID) {
        continue;
      }
      const accountLabel =
        typeof sourceContact.whatsapp_account === "string"
          ? sourceContact.whatsapp_account.trim()
          : "";
      if (accountLabel) {
        return accountLabel;
      }
    }

    const displayInstanceID =
      typeof entry.displayContact.instance_id === "string"
        ? entry.displayContact.instance_id.trim()
        : "";
    if (displayInstanceID === normalizedInstanceID) {
      const displayAccount =
        typeof entry.displayContact.whatsapp_account === "string"
          ? entry.displayContact.whatsapp_account.trim()
          : "";
      if (displayAccount) {
        return displayAccount;
      }
    }

    return "";
  }

  function getSidebarEntryPrimaryInstanceLabel(
    entry: SidebarContactEntry,
  ): string {
    return resolveSidebarEntryInstanceLabel(
      entry,
      getSidebarEntryPrimaryInstanceID(entry),
    );
  }

  function formatAccountToggleLabel(
    toggleKey: string,
    currentSidebarEntry: SidebarContactEntry | null,
    contacts: Contact[],
  ): string {
    const account = accountFromToggleKey(toggleKey);
    if (account) {
      return account;
    }

    const contactID = contactIDFromToggleKey(toggleKey);
    if (contactID) {
      const sourceContact =
        findSidebarEntrySourceContact(currentSidebarEntry, contactID) ||
        contacts.find((contact) => contact.id === contactID) ||
        null;
      if (sourceContact) {
        const instanceLabel = resolveInstanceToggleLabel(
          sourceContact.instance_id,
        );
        if (instanceLabel) {
          return instanceLabel;
        }
        const contactAccount = (sourceContact.whatsapp_account || "").trim();
        if (contactAccount) {
          return contactAccount;
        }
        if (sourceContact.instance_id) {
          return sourceContact.instance_id;
        }
        if (sourceContact.phone_number) {
          return sourceContact.phone_number;
        }
      }
    }

    return toggleKey;
  }

  function isSidebarEntryActive(entry: SidebarContactEntry, currentContactID?: string): boolean {
    if (!currentContactID) return false;
    return entry.sourceContactIDs.includes(currentContactID);
  }

  function getSidebarEntryPreferredContact(entry: SidebarContactEntry): Contact {
    const selectedAccountName = accountFromToggleKey(selectedAccount.value);
    if (selectedAccountName && entry.contactsByAccount[selectedAccountName]) {
      return entry.contactsByAccount[selectedAccountName];
    }

    const selectedContactID = contactIDFromToggleKey(selectedAccount.value);
    if (selectedContactID) {
      const selectedContact = findSidebarEntrySourceContact(
        entry,
        selectedContactID,
      );
      if (selectedContact) {
        return selectedContact;
      }
    }

    const displayAccount =
      typeof entry.displayContact.whatsapp_account === "string"
        ? entry.displayContact.whatsapp_account.trim()
        : "";
    if (displayAccount && entry.contactsByAccount[displayAccount]) {
      return entry.contactsByAccount[displayAccount];
    }

    if (entry.accountNames.length > 0) {
      const fallbackContact = entry.contactsByAccount[entry.accountNames[0]];
      if (fallbackContact) {
        return fallbackContact;
      }
    }

    return entry.displayContact;
  }

  return {
    isRTL,
    chatSidebarUnifier,
    chatSidebarViewMode,
    selectedAccount,
    contactAccounts,
    contactsSidebarWidth,
    isContactsSidebarResizing,
    isContactsSidebarCompact,
    isContactsSidebarWide,
    isSidebarUnifiedMode,
    setContactsSidebarWidth,
    startContactsSidebarResize,
    stopContactsSidebarResize,
    refreshChatSidebarViewModePreference,
    toAccountToggleKey,
    toContactToggleKey,
    accountFromToggleKey,
    contactIDFromToggleKey,
    selectedAccountFilter,
    findSidebarEntrySourceContact,
    resolveSourceContactForToggle,
    resolveSelectedSourceContact,
    resolveExplicitSourceContact,
    resolveInstanceToggleLabel,
    resolveSidebarEntryInstanceIDs,
    getSidebarEntryInstanceCount,
    hasSidebarEntryMultipleInstances,
    getSidebarEntryPrimaryInstanceID,
    resolveSidebarEntryInstanceLabel,
    getSidebarEntryPrimaryInstanceLabel,
    formatAccountToggleLabel,
    isSidebarEntryActive,
    getSidebarEntryPreferredContact,
  };
}
