import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Templates page exposes its primary features",
  path: "/settings/templates",
  heading: "Templates",
  fixture: "templates",
  permission: "templates",
  instructions:
    "Inspect template search, status/category filters, and the available create/edit affordance. Do not submit a template to WhatsApp or change/delete existing templates.",
});
