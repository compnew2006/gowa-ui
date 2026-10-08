import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "API key detail page exposes its primary features",
  path: "/settings/api-keys/new",
  heading: "New API Key",
  permission: "api_keys",
  instructions:
    "Inspect the key name and expiration fields and their validation. Do not generate or reveal a key, or submit the form.",
});
