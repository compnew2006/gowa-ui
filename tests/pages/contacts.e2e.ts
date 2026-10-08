import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Contacts page exposes its primary features",
  path: "/settings/contacts",
  heading: "Contacts",
  fixture: "contacts-list",
  permission: "contacts.manage",
  instructions:
    "Inspect contact search, filters, selection, and import/export controls. Use a no-match search and clear it; do not import, export, create, edit, or delete contacts.",
});
