import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Dashboard page exposes its primary features",
  path: "/",
  heading: "Dashboard",
  permission: "analytics",
  instructions:
    "Explore the dashboard widgets, open the date range control and inspect its preset/custom options, then inspect the widget/layout controls. Restore any temporary selection and leave the dashboard layout unchanged.",
});
