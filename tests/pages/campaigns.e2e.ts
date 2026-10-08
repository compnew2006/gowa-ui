import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Campaigns page exposes its primary features",
  path: "/campaigns",
  heading: "Campaigns",
  fixture: "campaigns-list",
  permission: "campaigns",
  instructions:
    "Exercise the campaign search and available status/time filters, then inspect any read-only list or empty state. Do not create, edit, delete, start, pause, or send a campaign.",
});
