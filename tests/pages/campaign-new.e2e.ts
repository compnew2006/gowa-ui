import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "New campaign form exposes its primary features",
  path: "/campaigns/new",
  heading: "New Campaign",
  fixture: "campaign-form",
  permission: "campaigns",
  instructions:
    "Inspect the campaign name, WhatsApp account, and template fields. Select only the E2E fixture account and one of its E2E fixture templates to verify the dependent selector, then leave without saving. This new-campaign form has no recipient or scheduling controls. Do not create, save, or trigger a campaign action.",
});
