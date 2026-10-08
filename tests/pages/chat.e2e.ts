import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Chat page exposes its primary features",
  path: "/chat",
  heading: "Select a conversation",
  permission: "chat",
  instructions:
    "Inspect the conversation list, search, account/private tabs, and available filters. Use only reversible searches and popovers; do not send messages, claim/close conversations, or change assignments.",
});
