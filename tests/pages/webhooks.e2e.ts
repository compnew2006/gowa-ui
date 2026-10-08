import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Webhooks page exposes its primary features",
  path: "/settings/webhooks",
  heading: "Webhooks",
  permission: "webhooks",
  instructions:
    "Inspect webhook search, status, and event information. Open the add-webhook flow and cancel; do not test, enable, disable, or delete a webhook.",
});
