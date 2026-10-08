import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Settings page exposes its primary features",
  path: "/settings",
  heading: "Settings",
  permission: "settings.general",
  instructions:
    "Visit both General and Notifications tabs. Inspect timezone/date selectors and notification toggles, restore any temporary state, and do not save settings.",
});
