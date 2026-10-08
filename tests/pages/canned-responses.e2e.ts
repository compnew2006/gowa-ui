import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Canned responses page exposes its primary features",
  path: "/settings/canned-responses",
  heading: "Canned Responses",
  fixture: "canned-responses-list",
  permission: "canned_responses",
  instructions:
    "Use response search and the category selector, then inspect the list or empty state and the sortable status column if rows are present. The page has no status filter. Do not create, edit, or delete a response.",
  regularUserInstructions:
    "Use response search and the category selector, then inspect the list or empty state and confirm the read-only actions available to this role. The page has no status filter, and creating or editing responses is restricted. Do not attempt those actions.",
});
