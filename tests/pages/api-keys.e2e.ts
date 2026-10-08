import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "API keys page exposes its primary features",
  path: "/settings/api-keys",
  heading: "API Keys",
  permission: "api_keys",
  instructions:
    "Inspect API-key search/status/expiration information and the create flow. Open its form and cancel; do not generate, reveal, rotate, or delete a key.",
});
