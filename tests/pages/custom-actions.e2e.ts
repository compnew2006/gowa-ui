import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Custom actions page exposes its primary features",
  path: "/settings/custom-actions",
  heading: "Custom Actions",
  permission: "custom_actions",
  instructions:
    "Inspect action search/status and the action-type form. Open and cancel the form; do not save, run scripts, or trigger webhook/URL actions.",
});
