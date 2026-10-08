import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "New canned response form exposes its primary features",
  path: "/settings/canned-responses/new",
  heading: "Create Canned Response",
  permission: "canned_responses",
  instructions:
    "Scroll through the complete new-response form and inspect name, shortcut, category, content, and message-button controls, including the allowed reply or URL button types. If useful, add and remove a button locally to inspect its fields and type rules. Note which fields are marked required. Do not enter or submit response data; the new-response route has no active-state control.",
  regularUserInstructions:
    "Inspect the new-response form and verify name, shortcut, category, content, and message-button controls are disabled or read-only for this role. Do not attempt to edit or save.",
});

definePageExploration({
  name: "Saved canned response detail page exposes its primary features",
  path: "/settings/canned-responses",
  heading: "",
  fixture: "canned-response-detail",
  permission: "canned_responses",
  instructions:
    "Inspect the saved response fields, button configuration, active state, metadata, and activity log. Do not edit, save, or delete this response.",
  regularUserInstructions:
    "Inspect the saved response and verify its fields are read-only for this role. Do not edit, save, or delete it.",
});
