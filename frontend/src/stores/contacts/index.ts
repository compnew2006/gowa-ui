import type { ChatTypeFilter, Contact, Message } from "@/types/contacts";

export type { ChatTypeFilter, Contact, Message };

export { useChatFiltersStore } from "./chat-filters";
export { useMessagesStore } from "./messages";

import { useContactsStore as _useContactsCoreStore } from "./contacts";
import { useChatFiltersStore } from "./chat-filters";
import { useMessagesStore } from "./messages";

type PiniaInternalKeys =
  | "$id"
  | "$state"
  | "$patch"
  | "$reset"
  | "$subscribe"
  | "$onAction"
  | "$dispose"
  | "_customProperties";

type StripPinia<T> = Omit<T, PiniaInternalKeys>;

type ContactsStoreFacade = StripPinia<ReturnType<typeof _useContactsCoreStore>> &
  StripPinia<ReturnType<typeof useMessagesStore>> &
  StripPinia<ReturnType<typeof useChatFiltersStore>>;

export function useContactsStore(): ContactsStoreFacade {
  const coreStore = _useContactsCoreStore();
  const messagesStore = useMessagesStore();
  const filtersStore = useChatFiltersStore();

  const messageProps: ReadonlySet<string> = new Set([
    "messages",
    "isLoadingMessages",
    "isLoadingOlderMessages",
    "isMessageAccessRestricted",
    "messageLoadError",
    "hasMoreMessages",
    "replyingTo",
    "fetchMessages",
    "fetchOlderMessages",
    "sendMessage",
    "addMessage",
    "updateMessageStatus",
    "patchMessage",
    "setReplyingTo",
    "clearReplyingTo",
    "setAccountFilter",
    "clearMessages",
    "updateMessageReactions",
  ]);

  const filterProps: ReadonlySet<string> = new Set([
    "searchQuery",
    "selectedTags",
    "selectedInstanceId",
    "selectedChatTypes",
  ]);

  const handler: ProxyHandler<Record<string, unknown>> = {
    get(_target, prop: string) {
      if (messageProps.has(prop)) return (messagesStore as any)[prop];
      if (filterProps.has(prop)) return (filtersStore as any)[prop];
      return (coreStore as any)[prop];
    },
    set(_target, prop: string, value) {
      if (messageProps.has(prop)) {
        (messagesStore as any)[prop] = value;
        return true;
      }
      if (filterProps.has(prop)) {
        (filtersStore as any)[prop] = value;
        return true;
      }
      (coreStore as any)[prop] = value;
      return true;
    },
  };

  return new Proxy({} as Record<string, unknown>, handler) as unknown as ContactsStoreFacade;
}
