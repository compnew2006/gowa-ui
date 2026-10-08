import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Team detail page exposes its primary features",
  path: "/settings/teams/new",
  heading: "New Team",
  permission: "teams",
  instructions:
    "Inspect team name/description and member selection controls. Open selectors and close them without assigning users or saving.",
});
