import { ref } from "vue";
import { messagesService } from "@/services/api";
import { resolvePreferredOutboundInstanceID } from "@/lib/chat-outbound-instance";
import type { Contact, Message } from "@/stores/contacts";

type TypingPresenceState = "composing" | "paused";

const TYPING_COMPOSING_THROTTLE_MS = 2500;
const TYPING_IDLE_PAUSE_MS = 3500;

export function useTypingPresence() {
  const typingLastComposeAt = ref(0);
  const typingLastState = ref<TypingPresenceState | null>(null);
  const typingLastContactID = ref<string | null>(null);
  let typingPauseTimer: ReturnType<typeof setTimeout> | null = null;

  function clearTypingPauseTimer() {
    if (typingPauseTimer) {
      clearTimeout(typingPauseTimer);
      typingPauseTimer = null;
    }
  }

  function resetTypingPresenceState() {
    typingLastComposeAt.value = 0;
    typingLastState.value = null;
    typingLastContactID.value = null;
  }

  function isTypingPresenceEligibleContact(contact: Contact | null): boolean {
    if (!contact) return false;
    if (contact.is_group_chat === true) return false;

    const metadata = contact.metadata || {};
    if (metadata.is_group_chat === true || metadata.is_channel_chat === true) {
      return false;
    }

    const phone = String(contact.phone_number || "")
      .trim()
      .toLowerCase();
    if (!phone) return false;
    if (phone.endsWith("@g.us") || phone.endsWith("@newsletter")) return false;

    return true;
  }

  function resolveTypingInstanceID(
    _contact: Contact,
    options: {
      messages: Message[];
      selectedSourceContact: Contact | null;
      currentContact: Contact;
      selectedInstanceID: string;
    },
  ): string | undefined {
    return resolvePreferredOutboundInstanceID({
      messages: options.messages,
      selectedSourceContact: options.selectedSourceContact,
      currentContact: options.currentContact,
      selectedInstanceID: options.selectedInstanceID,
    });
  }

  async function sendTypingPresenceForContact(
    contact: Contact | null,
    state: TypingPresenceState,
    options?: {
      force?: boolean;
      messages?: Message[];
      selectedSourceContact?: Contact | null;
      selectedInstanceID?: string;
    },
  ) {
    if (!isTypingPresenceEligibleContact(contact)) return;
    if (!contact) return;

    const contactID = contact.id;
    const force = options?.force === true;

    if (!force) {
      if (
        state === "paused" &&
        typingLastState.value === "paused" &&
        typingLastContactID.value === contactID
      ) {
        return;
      }

      if (
        state === "composing" &&
        typingLastContactID.value === contactID &&
        Date.now() - typingLastComposeAt.value < TYPING_COMPOSING_THROTTLE_MS
      ) {
        return;
      }
    }

    if (state === "composing") {
      typingLastComposeAt.value = Date.now();
    }
    typingLastState.value = state;
    typingLastContactID.value = contactID;

    try {
      await messagesService.sendTyping(contactID, {
        state,
        instance_id: options
          ? resolveTypingInstanceID(contact, {
              messages: options.messages || [],
              selectedSourceContact: options.selectedSourceContact ?? null,
              currentContact: contact,
              selectedInstanceID: options.selectedInstanceID || "",
            })
          : undefined,
      });
    } catch {
      // Typing presence is best-effort and should not interrupt chat UX.
    }
  }

  function scheduleTypingPaused(
    contact: Contact | null,
    options?: {
      messages?: Message[];
      selectedSourceContact?: Contact | null;
      selectedInstanceID?: string;
    },
  ) {
    clearTypingPauseTimer();
    if (!isTypingPresenceEligibleContact(contact)) return;

    typingPauseTimer = setTimeout(() => {
      void sendTypingPresenceForContact(contact, "paused", options);
      clearTypingPauseTimer();
    }, TYPING_IDLE_PAUSE_MS);
  }

  function stopTypingForContact(
    contact: Contact | null,
    options?: {
      force?: boolean;
      messages?: Message[];
      selectedSourceContact?: Contact | null;
      selectedInstanceID?: string;
    },
  ) {
    clearTypingPauseTimer();
    void sendTypingPresenceForContact(contact, "paused", options);
  }

  return {
    typingLastComposeAt,
    typingLastState,
    typingLastContactID,
    clearTypingPauseTimer,
    resetTypingPresenceState,
    isTypingPresenceEligibleContact,
    sendTypingPresenceForContact,
    scheduleTypingPaused,
    stopTypingForContact,
  };
}
