import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Role detail page exposes its primary features",
  path: "/settings/roles/new",
  heading: "Create Role",
  permission: "roles",
  instructions:
    "Review role fields and the permission matrix. Check that permission groups can be inspected, restore any temporary toggles, and do not save or delete.",
});
