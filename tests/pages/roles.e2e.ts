import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Roles page exposes its primary features",
  path: "/settings/roles",
  heading: "Roles & Permissions",
  permission: "roles",
  instructions:
    "Inspect role search, permission summaries, and the create-role flow. Open the form, review its permission controls, then cancel without saving.",
});
