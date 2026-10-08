import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Tags page exposes its primary features",
  path: "/settings/tags",
  heading: "Tags",
  permission: "tags",
  instructions:
    "Search for a no-match tag and clear the search. Open the create-tag dialog, inspect its name/color fields and live preview, and cancel without saving.",
  regularUserInstructions:
    "Search for a no-match tag and clear the search. Inspect the list and verify there is no create, edit, or delete control for this read-only role; do not try to bypass those permissions.",
});
