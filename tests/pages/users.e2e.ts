import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Users page exposes its primary features",
  path: "/settings/users",
  heading: "User Management",
  permission: "users",
  instructions:
    "Inspect user search, role/online filters, and account-access controls. Open the add-user dialog and cancel; do not create users or generate invitation links.",
});
