import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Teams page exposes its primary features",
  path: "/settings/teams",
  heading: "Teams",
  permission: "teams",
  fixture: "teams",
  instructions:
    "Inspect team search, member counts, and list actions. Open the create-team form and cancel without adding members or saving.",
});
