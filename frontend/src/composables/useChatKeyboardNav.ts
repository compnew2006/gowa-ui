import { ref, type Ref, onMounted, onUnmounted, nextTick, type ComputedRef } from "vue";
import type { SidebarContactEntry } from "@/lib/chat-sidebar-unifier";

export interface ChatKeyboardNavOptions {
  sidebarContacts: ComputedRef<SidebarContactEntry[]>;
  isInfoPanelOpen: Ref<boolean>;
  isNotesPanelOpen: Ref<boolean>;
  isAssignDialogOpen: Ref<boolean>;
  isMediaDialogOpen: Ref<boolean>;
  isAddContactOpen: Ref<boolean>;
  isProfilePhotoDialogOpen: Ref<boolean>;
  isChatMediaViewerOpen: Ref<boolean>;
  cannedPickerOpen: Ref<boolean>;
  emojiPickerOpen: Ref<boolean>;
  replyingTo: ComputedRef<any>;
  onContactSelect: (entry: SidebarContactEntry) => void;
  onEscapeReply: () => void;
  onEscapeCanned: () => void;
  onEscapeEmoji: () => void;
}

export function useChatKeyboardNav(options: ChatKeyboardNavOptions) {
  const {
    sidebarContacts,
    isInfoPanelOpen,
    isNotesPanelOpen,
    isAssignDialogOpen,
    isMediaDialogOpen,
    isAddContactOpen,
    isProfilePhotoDialogOpen,
    isChatMediaViewerOpen,
    cannedPickerOpen,
    emojiPickerOpen,
    replyingTo,
    onContactSelect,
    onEscapeReply,
    onEscapeCanned,
    onEscapeEmoji,
  } = options;

  const focusedSidebarIndex = ref(-1);

  function scrollToSidebarEntry(index: number) {
    nextTick(() => {
      const entries = document.querySelectorAll(
        "[data-testid='chat-sidebar-entry']",
      );
      if (entries[index]) {
        entries[index].scrollIntoView({ block: "nearest", behavior: "smooth" });
      }
    });
  }

  function isAnyDialogOpen(): boolean {
    return (
      isAssignDialogOpen.value ||
      isMediaDialogOpen.value ||
      isAddContactOpen.value ||
      isProfilePhotoDialogOpen.value ||
      isChatMediaViewerOpen.value
    );
  }

  function handleGlobalKeydown(event: KeyboardEvent) {
    const target = event.target as HTMLElement;
    const isInputFocused =
      target.tagName === "INPUT" ||
      target.tagName === "TEXTAREA" ||
      target.tagName === "SELECT" ||
      target.isContentEditable;

    if (isAnyDialogOpen()) {
      return;
    }

    switch (event.key) {
      case "Escape": {
        event.preventDefault();

        if (emojiPickerOpen.value) {
          onEscapeEmoji();
          return;
        }
        if (cannedPickerOpen.value) {
          onEscapeCanned();
          return;
        }
        if (replyingTo.value) {
          onEscapeReply();
          return;
        }
        if (isInfoPanelOpen.value) {
          isInfoPanelOpen.value = false;
          return;
        }
        if (isNotesPanelOpen.value) {
          isNotesPanelOpen.value = false;
          return;
        }
        focusedSidebarIndex.value = -1;
        return;
      }

      case "ArrowUp": {
        if (isInputFocused) return;

        event.preventDefault();
        const contacts = sidebarContacts.value;
        if (contacts.length === 0) return;

        const newIndex =
          focusedSidebarIndex.value <= 0
            ? contacts.length - 1
            : focusedSidebarIndex.value - 1;
        focusedSidebarIndex.value = newIndex;
        scrollToSidebarEntry(newIndex);
        return;
      }

      case "ArrowDown": {
        if (isInputFocused) return;

        event.preventDefault();
        const contacts = sidebarContacts.value;
        if (contacts.length === 0) return;

        const newIndex =
          focusedSidebarIndex.value >= contacts.length - 1
            ? 0
            : focusedSidebarIndex.value + 1;
        focusedSidebarIndex.value = newIndex;
        scrollToSidebarEntry(newIndex);
        return;
      }

      case "Enter": {
        if (isInputFocused) return;

        const idx = focusedSidebarIndex.value;
        const contacts = sidebarContacts.value;
        if (idx >= 0 && idx < contacts.length) {
          event.preventDefault();
          onContactSelect(contacts[idx]);
          focusedSidebarIndex.value = -1;
        }
        return;
      }
    }
  }

  onMounted(() => {
    window.addEventListener("keydown", handleGlobalKeydown);
  });

  onUnmounted(() => {
    window.removeEventListener("keydown", handleGlobalKeydown);
  });

  return {
    focusedSidebarIndex,
  };
}
