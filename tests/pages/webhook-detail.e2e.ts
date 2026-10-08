import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Webhook detail page exposes its primary features",
  path: "/settings/webhooks/new",
  heading: "New Webhook",
  permission: "webhooks",
  instructions:
    "Inspect webhook URL, event, and secret fields. Do not reveal secrets, send a test request, or save the configuration.",
});
